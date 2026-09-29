package handlers

import (
	"bytes"
	"net/http"
	"os/exec"
	"path/filepath"

	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/go-chi/chi/v5"
)

type ArchiveHandler struct {
	ws    *WorkspaceHandler
	prefs *preferences.Service
}

func NewArchiveHandler(ws *WorkspaceHandler, prefs *preferences.Service) *ArchiveHandler {
	return &ArchiveHandler{ws: ws, prefs: prefs}
}

func (h *ArchiveHandler) Archive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := chi.URLParam(r, "name")

	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	var out bytes.Buffer
	cmd := exec.Command("openspec", "archive", name, "--yes")
	cmd.Dir = path
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write(out.Bytes())
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(out.Bytes())

	// Trigger async tagging after successful archive
	go func() {
		changesDir := filepath.Join(path, "openspec", "changes")
		changeRoot := filepath.Join(changesDir, "archive", name)
		specializationVocabulary := specializationVocabularyFor(h.prefs)
		_ = openspec.TagChange(changeRoot, path, false, specializationVocabulary)
	}()
}
