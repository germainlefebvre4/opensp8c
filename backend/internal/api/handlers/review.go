package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/go-chi/chi/v5"
)

// ReviewHandler serves, read-only, what a change in review brings: its changed
// files, and the patch and final content of each. Everything is computed from
// git references, so it works with the pool stopped and the worktree gone.
type ReviewHandler struct {
	ws *WorkspaceHandler
}

func NewReviewHandler(ws *WorkspaceHandler) *ReviewHandler {
	return &ReviewHandler{ws: ws}
}

type reviewResponse struct {
	Branch      string                 `json:"branch"`
	Base        string                 `json:"base"`
	TargetAhead bool                   `json:"target_ahead"`
	Files       []pool.ReviewFileEntry `json:"files"`
}

// resolve checks the workspace, the change and its review status, writing the
// error response itself; ok is false when it did.
func (h *ReviewHandler) resolve(w http.ResponseWriter, r *http.Request) (*pool.WorktreeController, string, bool) {
	id := chi.URLParam(r, "id")
	name := chi.URLParam(r, "name")
	path, found := h.ws.workspacePath(id)
	if !found {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return nil, "", false
	}
	if !validSegment(name) {
		http.Error(w, "change not found", http.StatusNotFound)
		return nil, "", false
	}
	if _, err := os.Stat(filepath.Join(path, "openspec", "changes", name)); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "change not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return nil, "", false
	}
	if !openspec.InReview(path, name) {
		writeReviewError(w, http.StatusConflict, "not_in_review")
		return nil, "", false
	}
	return pool.NewWorktreeController(path, id, ""), name, true
}

func writeReviewError(w http.ResponseWriter, status int, code string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code})
}

// writeReviewFailure maps the review errors to their HTTP status.
func writeReviewFailure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pool.ErrInvalidReviewPath):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, pool.ErrReviewPathNotInReview), errors.Is(err, pool.ErrReviewFileDeleted), errors.Is(err, pool.ErrReviewBranchMissing):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// List answers GET .../changes/{name}/review.
func (h *ReviewHandler) List(w http.ResponseWriter, r *http.Request) {
	wc, name, ok := h.resolve(w, r)
	if !ok {
		return
	}
	list, err := wc.ReviewFiles(name)
	if err != nil {
		writeReviewFailure(w, err)
		return
	}
	ahead, err := wc.TargetAhead(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(reviewResponse{
		Branch:      "feature/" + name,
		Base:        list.Base,
		TargetAhead: ahead,
		Files:       list.Files,
	})
}

// Diff answers GET .../review/diff?path=.
func (h *ReviewHandler) Diff(w http.ResponseWriter, r *http.Request) {
	wc, name, ok := h.resolve(w, r)
	if !ok {
		return
	}
	patch, err := wc.ReviewPatch(name, r.URL.Query().Get("path"))
	if err != nil {
		writeReviewFailure(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"path":      r.URL.Query().Get("path"),
		"patch":     patch.Content,
		"binary":    patch.Binary,
		"truncated": patch.Truncated,
	})
}

// File answers GET .../review/file?path=.
func (h *ReviewHandler) File(w http.ResponseWriter, r *http.Request) {
	wc, name, ok := h.resolve(w, r)
	if !ok {
		return
	}
	file, err := wc.ReviewFile(name, r.URL.Query().Get("path"))
	if err != nil {
		writeReviewFailure(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"path":      r.URL.Query().Get("path"),
		"content":   file.Content,
		"binary":    file.Binary,
		"truncated": file.Truncated,
	})
}
