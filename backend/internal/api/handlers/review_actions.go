package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/go-chi/chi/v5"
)

// ReviewActionsHandler serves the two actions on a change in review: approve
// (merge) and request a correction.
type ReviewActionsHandler struct {
	ws      *WorkspaceHandler
	poolReg *pool.Registry
}

func NewReviewActionsHandler(ws *WorkspaceHandler, poolReg *pool.Registry) *ReviewActionsHandler {
	return &ReviewActionsHandler{ws: ws, poolReg: poolReg}
}

// reviewActionError is the JSON body of a refused or failed review action.
type reviewActionError struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
	Output  string `json:"output,omitempty"`
	// Remaining is the number of unchecked tasks of a tasks_pending refusal.
	Remaining int `json:"remaining,omitempty"`
	// Target and Files describe an integration_conflict refusal: the branch
	// integrated and the files in conflict.
	Target string   `json:"target,omitempty"`
	Files  []string `json:"files,omitempty"`
}

func writeReviewAction(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// resolve checks the workspace and the change, writing the error itself.
func (h *ReviewActionsHandler) resolve(w http.ResponseWriter, r *http.Request) (id, path, name string, ok bool) {
	return resolveChange(h.ws, w, r)
}

// resolveChange checks the workspace and the change of a request, writing the
// error itself.
func resolveChange(ws *WorkspaceHandler, w http.ResponseWriter, r *http.Request) (id, path, name string, ok bool) {
	id, name = chi.URLParam(r, "id"), chi.URLParam(r, "name")
	path, found := ws.workspacePath(id)
	if !found {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return "", "", "", false
	}
	if !validSegment(name) {
		http.Error(w, "change not found", http.StatusNotFound)
		return "", "", "", false
	}
	if _, err := os.Stat(filepath.Join(path, "openspec", "changes", name)); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "change not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return "", "", "", false
	}
	return id, path, name, true
}

// writeRefusal maps the errors shared by both actions; it reports whether err
// was one of them.
func writeRefusal(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, pool.ErrNotInReview):
		writeReviewAction(w, http.StatusConflict, reviewActionError{Code: "not_in_review", Message: err.Error()})
	case errors.Is(err, pool.ErrWorkerActive):
		writeReviewAction(w, http.StatusConflict, reviewActionError{Code: "worker_active", Message: err.Error()})
	default:
		return false
	}
	return true
}

// approveWarning is the "warning" member of a successful approval whose
// cleanup is incomplete.
type approveWarning struct {
	Code      string   `json:"code"`
	Message   string   `json:"message"`
	Remaining []string `json:"remaining"`
}

// approveResponse is the 200 body of an approval; warning is omitted when the
// cleanup was complete.
type approveResponse struct {
	Target  string          `json:"target"`
	Warning *approveWarning `json:"warning,omitempty"`
}

// Approve answers POST .../review/approve: 200 {"target"} on success, with a
// "warning" (code cleanup_incomplete) when the merge happened but the cleanup
// did not complete.
func (h *ReviewActionsHandler) Approve(w http.ResponseWriter, r *http.Request) {
	id, path, name, ok := h.resolve(w, r)
	if !ok {
		return
	}
	res, err := h.poolReg.For(id).ApproveReview(r.Context(), id, path, name)
	if err == nil {
		body := approveResponse{Target: res.Target}
		if res.Warning != nil {
			body.Warning = &approveWarning{Code: "cleanup_incomplete", Message: res.Warning.Message, Remaining: res.Warning.Remaining}
		}
		writeReviewAction(w, http.StatusOK, body)
		return
	}
	status, body := approveFailure(err)
	writeReviewAction(w, status, body)
}

// approveFailure maps an ApproveReview error to its HTTP status and body.
func approveFailure(err error) (int, reviewActionError) {
	var integration *pool.IntegrationConflictError
	var moving *pool.TargetMovingError
	var validation *pool.ValidationFailedError
	var env *pool.ValidationEnvError
	var pending *pool.TasksPendingError
	switch {
	case errors.As(err, &pending):
		return http.StatusConflict, reviewActionError{Code: "tasks_pending", Message: err.Error(), Remaining: pending.Remaining}
	case errors.Is(err, pool.ErrNotInReview):
		return http.StatusConflict, reviewActionError{Code: "not_in_review", Message: err.Error()}
	case errors.Is(err, pool.ErrWorkerActive):
		return http.StatusConflict, reviewActionError{Code: "worker_active", Message: err.Error()}
	case errors.Is(err, pool.ErrMergeInProgress):
		return http.StatusConflict, reviewActionError{Code: "merge_in_progress", Message: err.Error()}
	case errors.Is(err, pool.ErrBaseBranchMismatch):
		return http.StatusConflict, reviewActionError{Code: "base_branch_mismatch", Message: err.Error()}
	case errors.As(err, &integration):
		return http.StatusConflict, reviewActionError{Code: "integration_conflict", Message: err.Error(), Target: integration.Target, Files: integration.Files}
	case errors.As(err, &moving):
		return http.StatusConflict, reviewActionError{Code: "target_moving", Message: err.Error()}
	case errors.As(err, &validation):
		return http.StatusUnprocessableEntity, reviewActionError{Code: "validation_failed", Message: "la validation a échoué", Output: validation.Error()}
	case errors.As(err, &env):
		return http.StatusUnprocessableEntity, reviewActionError{Code: "validation_failed", Message: env.Reason, Output: env.Reason}
	}
	return http.StatusInternalServerError, reviewActionError{Code: "approve_failed", Message: err.Error()}
}

// RequestCorrection answers POST .../review/request-correction (body
// {"feedback", "reopen_human_tasks"?}): 204 on success.
func (h *ReviewActionsHandler) RequestCorrection(w http.ResponseWriter, r *http.Request) {
	id, path, name, ok := h.resolve(w, r)
	if !ok {
		return
	}
	var body struct {
		Feedback         string `json:"feedback"`
		ReopenHumanTasks bool   `json:"reopen_human_tasks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeReviewAction(w, http.StatusBadRequest, reviewActionError{Code: "empty_feedback", Message: "corps invalide"})
		return
	}
	err := h.poolReg.For(id).RequestCorrection(r.Context(), id, path, name, body.Feedback, pool.CorrectionOptions{ReopenHumanTasks: body.ReopenHumanTasks})
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, pool.ErrEmptyFeedback):
		writeReviewAction(w, http.StatusBadRequest, reviewActionError{Code: "empty_feedback", Message: err.Error()})
	case writeRefusal(w, err):
	default:
		writeReviewAction(w, http.StatusInternalServerError, reviewActionError{Code: "correction_failed", Message: err.Error()})
	}
}
