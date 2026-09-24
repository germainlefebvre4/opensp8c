package handlers

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/preferences"
)

type TagsHandler struct {
	ws    *WorkspaceHandler
	prefs *preferences.Service
}

func NewTagsHandler(ws *WorkspaceHandler, prefs *preferences.Service) *TagsHandler {
	return &TagsHandler{ws: ws, prefs: prefs}
}

func (h *TagsHandler) Retag(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := chi.URLParam(r, "name")

	workspacePath, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	changesDir := filepath.Join(workspacePath, "openspec", "changes")
	changeRoot := filepath.Join(changesDir, name)
	if _, err := os.Stat(changeRoot); os.IsNotExist(err) {
		changeRoot = filepath.Join(changesDir, "archive", name)
		if _, err := os.Stat(changeRoot); os.IsNotExist(err) {
			http.Error(w, "change not found", http.StatusNotFound)
			return
		}
	}

	specializationVocabulary := specializationVocabularyFor(h.prefs)
	if err := openspec.TagChange(changeRoot, workspacePath, true, specializationVocabulary); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
