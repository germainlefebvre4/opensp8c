package handlers

import (
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/glefebvre/opensp8c/internal/openspec"
)

type TaskHandler struct {
	ws *WorkspaceHandler
}

func NewTaskHandler(ws *WorkspaceHandler) *TaskHandler {
	return &TaskHandler{ws: ws}
}

func (h *TaskHandler) PatchTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := chi.URLParam(r, "name")
	indexStr := chi.URLParam(r, "index")

	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	index, err := strconv.Atoi(indexStr)
	if err != nil || index < 0 {
		http.Error(w, "invalid task index", http.StatusBadRequest)
		return
	}

	if err := openspec.ToggleTask(path, name, index); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
