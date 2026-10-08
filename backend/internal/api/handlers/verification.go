package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/go-chi/chi/v5"
)

// VerificationHandler serves the actions on a change in the verification
// stage: rerun, finalize without verification, request a correction, and the
// report of the last verification.
type VerificationHandler struct {
	ws        *WorkspaceHandler
	poolReg   *pool.Registry
	convStore *conversation.Store
}

func NewVerificationHandler(ws *WorkspaceHandler, poolReg *pool.Registry, convStore *conversation.Store) *VerificationHandler {
	return &VerificationHandler{ws: ws, poolReg: poolReg, convStore: convStore}
}

// writeVerificationError maps the refusals of the verification actions; it
// reports whether err was one of them.
func writeVerificationError(w http.ResponseWriter, err error) bool {
	var incomplete *pool.ErrTasksIncomplete
	switch {
	case errors.Is(err, pool.ErrVerificationRunning):
		writeReviewAction(w, http.StatusConflict, reviewActionError{Code: "verification_running", Message: err.Error()})
	case errors.Is(err, pool.ErrVerificationNotFailed):
		writeReviewAction(w, http.StatusConflict, reviewActionError{Code: "not_failed", Message: err.Error()})
	case errors.Is(err, pool.ErrReviewBusy):
		writeReviewAction(w, http.StatusConflict, reviewActionError{Code: "review_busy", Message: err.Error()})
	case errors.As(err, &incomplete):
		writeReviewAction(w, http.StatusConflict, reviewActionError{Code: "tasks_incomplete", Message: err.Error(), Remaining: incomplete.Remaining})
	case errors.Is(err, pool.ErrWorkerActive):
		writeReviewAction(w, http.StatusConflict, reviewActionError{Code: "worker_active", Message: err.Error()})
	case errors.Is(err, pool.ErrEmptyFeedback):
		writeReviewAction(w, http.StatusBadRequest, reviewActionError{Code: "empty_feedback", Message: err.Error()})
	default:
		return false
	}
	return true
}

// Rerun answers POST .../verification/rerun: 204 on success.
func (h *VerificationHandler) Rerun(w http.ResponseWriter, r *http.Request) {
	id, path, name, ok := resolveChange(h.ws, w, r)
	if !ok {
		return
	}
	h.finish(w, h.poolReg.For(id).RerunVerification(id, path, name), "rerun_failed")
}

// Finalize answers POST .../verification/finalize: 204 on success.
func (h *VerificationHandler) Finalize(w http.ResponseWriter, r *http.Request) {
	id, path, name, ok := resolveChange(h.ws, w, r)
	if !ok {
		return
	}
	h.finish(w, h.poolReg.For(id).FinalizeWithoutVerification(id, path, name), "finalize_failed")
}

// RequestCorrection answers POST .../verification/request-correction (body
// {"feedback"}): 204 on success.
func (h *VerificationHandler) RequestCorrection(w http.ResponseWriter, r *http.Request) {
	id, path, name, ok := resolveChange(h.ws, w, r)
	if !ok {
		return
	}
	var body struct {
		Feedback string `json:"feedback"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeReviewAction(w, http.StatusBadRequest, reviewActionError{Code: "empty_feedback", Message: "corps invalide"})
		return
	}
	h.finish(w, h.poolReg.For(id).RequestVerificationCorrection(r.Context(), id, path, name, body.Feedback), "correction_failed")
}

func (h *VerificationHandler) finish(w http.ResponseWriter, err error, failureCode string) {
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case writeVerificationError(w, err):
	default:
		writeReviewAction(w, http.StatusInternalServerError, reviewActionError{Code: failureCode, Message: err.Error()})
	}
}

// verificationReport is the body of GET .../verification/report.
type verificationReport struct {
	Step      string `json:"step"`
	Verdict   string `json:"verdict"`
	Reason    string `json:"reason"`
	Report    string `json:"report"`
	StartedAt string `json:"started_at"`
}

// Report answers GET .../verification/report with the last verification run
// of the change; 404 when none exists.
func (h *VerificationHandler) Report(w http.ResponseWriter, r *http.Request) {
	id, name := chi.URLParam(r, "id"), chi.URLParam(r, "name")
	if _, ok := h.ws.workspacePath(id); !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}
	if !validSegment(name) || h.convStore == nil {
		http.Error(w, "verification report not found", http.StatusNotFound)
		return
	}
	runs, err := h.convStore.List(id, name, "verify")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(runs) == 0 {
		http.Error(w, "verification report not found", http.StatusNotFound)
		return
	}
	// Most recent run first: the first one that reached its end marker wins,
	// else the newest run, which is still in flight.
	var out verificationReport
	for i, run := range runs {
		lines, err := h.convStore.Load(id, name, "verify", run.Ts)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rep := verificationReport{StartedAt: startOfRun(run.Ts)}
		ended := false
		for _, raw := range lines {
			var env journalEnvelope
			if json.Unmarshal(raw, &env) != nil || env.Dir != "meta" {
				continue
			}
			var m struct {
				Type    string `json:"type"`
				Step    string `json:"step"`
				Verdict string `json:"verdict"`
				Reason  string `json:"reason"`
				Report  string `json:"report"`
			}
			if json.Unmarshal(env.Data, &m) == nil && m.Type == "verify_run_end" {
				rep.Step, rep.Verdict, rep.Reason, rep.Report = m.Step, m.Verdict, m.Reason, m.Report
				ended = true
			}
		}
		if i == 0 || ended {
			out = rep
		}
		if ended {
			break
		}
	}
	if out.StartedAt == "" {
		http.Error(w, "verification report not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// startOfRun formats a run timestamp (the store's file name) as RFC 3339.
func startOfRun(ts string) string {
	t, err := time.Parse("2006-01-02T15-04-05Z", ts)
	if err != nil {
		return ts
	}
	return t.UTC().Format(time.RFC3339)
}
