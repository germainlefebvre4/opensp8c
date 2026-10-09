package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

// verificationArtifact is one piece of evidence of a verification run.
type verificationArtifact struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// verificationReport is the body of GET .../verification/report. Run is the
// identifier of the run, which locates its artifacts.
type verificationReport struct {
	Run       string                 `json:"run"`
	Step      string                 `json:"step"`
	Verdict   string                 `json:"verdict"`
	Reason    string                 `json:"reason"`
	Report    string                 `json:"report"`
	StartedAt string                 `json:"started_at"`
	Artifacts []verificationArtifact `json:"artifacts"`
	// Verified are the tasks the verification checked off; Ignored the
	// TASK-VERIFIED lines that matched no task.
	Verified []string `json:"verified"`
	Ignored  []string `json:"ignored"`
	// Driver and AllowedTools describe how the UI step drove the browser; absent
	// for another step or for a run written before the drivers existed.
	Driver       string   `json:"driver,omitempty"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
}

// Limits of the evidence of a run.
const (
	maxArtifactSize  = 5 << 20
	maxArtifactCount = 50
)

// artifactTypes are the served evidence extensions and their content types.
var artifactTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
}

// artifactContentType returns the content type of an evidence file name, or ""
// when the name is not an acceptable image file name.
func artifactContentType(name string) string {
	if !validSegment(name) || strings.ContainsRune(name, 0) {
		return ""
	}
	return artifactTypes[strings.ToLower(filepath.Ext(name))]
}

// listArtifacts returns the evidence of the run directory dir: regular image
// files (links ignored) of at most maxArtifactSize bytes, alphabetical, at most
// maxArtifactCount.
func listArtifacts(dir string) []verificationArtifact {
	out := []verificationArtifact{}
	entries, err := os.ReadDir(dir) // sorted by name
	if err != nil {
		return out
	}
	for _, e := range entries {
		if len(out) == maxArtifactCount {
			break
		}
		if artifactContentType(e.Name()) == "" {
			continue
		}
		info, err := os.Lstat(filepath.Join(dir, e.Name()))
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxArtifactSize {
			continue
		}
		out = append(out, verificationArtifact{Name: e.Name(), Size: info.Size()})
	}
	return out
}

// runMarkers are the fields of a verify run marker the report reads.
type runMarkers struct {
	Type     string   `json:"type"`
	Step     string   `json:"step"`
	Verdict  string   `json:"verdict"`
	Reason   string   `json:"reason"`
	Report   string   `json:"report"`
	Verified []string `json:"verified"`
	Ignored  []string `json:"ignored"`
	// Driver and AllowedTools are written by the UI step only.
	Driver       string   `json:"driver"`
	AllowedTools []string `json:"allowed_tools"`
}

// readRunReport reads the markers of one verify run; ended tells whether the
// run reached its end marker.
func (h *VerificationHandler) readRunReport(id, name, ts string) (rep verificationReport, ended bool, err error) {
	lines, err := h.convStore.Load(id, name, "verify", ts)
	if err != nil {
		return rep, false, err
	}
	rep = verificationReport{Run: ts, StartedAt: startOfRun(ts), Artifacts: []verificationArtifact{}, Verified: []string{}, Ignored: []string{}}
	for _, raw := range lines {
		var env journalEnvelope
		if json.Unmarshal(raw, &env) != nil || env.Dir != "meta" {
			continue
		}
		var m runMarkers
		if json.Unmarshal(env.Data, &m) != nil {
			continue
		}
		switch m.Type {
		case "verify_run_start":
			if m.Step != "" {
				rep.Step = m.Step
			}
		case "verify_run_end":
			rep.Step, rep.Verdict, rep.Reason, rep.Report = m.Step, m.Verdict, m.Reason, m.Report
			if m.Verified != nil {
				rep.Verified = m.Verified
			}
			if m.Ignored != nil {
				rep.Ignored = m.Ignored
			}
			if m.Step == "ui" {
				rep.Driver, rep.AllowedTools = m.Driver, m.AllowedTools
			}
			ended = true
		}
	}
	rep.Artifacts = listArtifacts(h.convStore.RunDir(id, name, "verify", ts))
	return rep, ended, nil
}

// Report answers GET .../verification/report with the most recent verification
// run of the change (the one that decided the outcome), or with ?step= the most
// recent run of that step; 404 when none exists. A run still in flight falls
// back to the last finished one.
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
	step := r.URL.Query().Get("step")
	runs, err := h.convStore.List(id, name, "verify")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Most recent run first: the first matching run that reached its end
	// marker wins, else the newest matching run, which is still in flight.
	var out verificationReport
	found := false
	for _, run := range runs {
		rep, ended, err := h.readRunReport(id, name, run.Ts)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if step != "" && rep.Step != step {
			continue
		}
		if !found || ended {
			out, found = rep, true
		}
		if ended {
			break
		}
	}
	if !found {
		http.Error(w, "verification report not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// runIDLayout is the format of a run identifier (the journal's file name).
const runIDLayout = "2006-01-02T15-04-05Z"

// Artifact answers GET .../verification/artifacts/{run}/{file}: one image of
// the evidence of a run, typed by its extension. 400 for a run or a file name
// that is not a plain identifier or image name, 404 when absent.
func (h *VerificationHandler) Artifact(w http.ResponseWriter, r *http.Request) {
	id, name := chi.URLParam(r, "id"), chi.URLParam(r, "name")
	run, file := chi.URLParam(r, "run"), chi.URLParam(r, "file")
	if _, ok := h.ws.workspacePath(id); !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}
	if _, err := time.Parse(runIDLayout, run); err != nil || !validSegment(run) || !validSegment(name) {
		http.Error(w, "invalid run", http.StatusBadRequest)
		return
	}
	if !validSegment(file) || strings.ContainsRune(file, 0) {
		http.Error(w, "invalid artifact name", http.StatusBadRequest)
		return
	}
	ctype := artifactContentType(file)
	if ctype == "" { // not an image: not evidence
		http.Error(w, "artifact not found", http.StatusNotFound)
		return
	}
	if h.convStore == nil {
		http.Error(w, "artifact not found", http.StatusNotFound)
		return
	}
	dir := h.convStore.RunDir(id, name, "verify", run)
	path := filepath.Join(dir, file)
	if rel, err := filepath.Rel(dir, path); err != nil || rel != file {
		http.Error(w, "invalid artifact name", http.StatusBadRequest)
		return
	}
	info, err := os.Lstat(path) // a link is never served
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxArtifactSize {
		http.Error(w, "artifact not found", http.StatusNotFound)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "artifact not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeContent(w, r, file, info.ModTime(), f)
}

// startOfRun formats a run timestamp (the store's file name) as RFC 3339.
func startOfRun(ts string) string {
	t, err := time.Parse("2006-01-02T15-04-05Z", ts)
	if err != nil {
		return ts
	}
	return t.UTC().Format(time.RFC3339)
}
