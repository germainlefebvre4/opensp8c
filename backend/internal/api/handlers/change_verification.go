package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/verification"
	"github.com/go-chi/chi/v5"
)

// changeVerification computes the verification view of a change. The
// change-level override is read from the main repository, never a worktree.
func (h *KanbanHandler) changeVerification(workspaceID, workspacePath, name string) (*openspec.ChangeVerification, error) {
	override, err := openspec.ReadVerification(workspacePath, name)
	if err != nil {
		return nil, err
	}
	var p *preferences.Preferences // nil resolves to the built-in values
	if h.prefs != nil {
		if p, err = h.prefs.Load(); err != nil {
			return nil, err
		}
	}
	out := &openspec.ChangeVerification{
		Inherited: p.ResolveVerification(workspaceID, nil),
		Resolved:  p.ResolveVerification(workspaceID, override),
	}
	if override != nil {
		out.Override = *override
	}
	return out, nil
}

// PatchVerification updates the change-level verification override of an
// active change; null resets a step to inheritance.
func (h *KanbanHandler) PatchVerification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := chi.URLParam(r, "name")

	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	var patch verification.Patch
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&patch); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	changesDir := filepath.Join(path, "openspec", "changes")
	changeDir := filepath.Join(changesDir, name)
	if name == "archive" || filepath.Base(name) != name {
		http.Error(w, "change not found", http.StatusNotFound)
		return
	}
	if _, err := os.Stat(changeDir); err != nil {
		if !os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, aerr := os.Stat(filepath.Join(changesDir, "archive", name)); aerr == nil {
			http.Error(w, "change is archived", http.StatusConflict)
			return
		}
		http.Error(w, "change not found", http.StatusNotFound)
		return
	}

	if _, err := openspec.SetVerification(changeDir, patch); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	view, err := h.changeVerification(id, path, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(view)
}
