package handlers

import (
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/go-chi/chi/v5"
)

type TaskHandler struct {
	ws       *WorkspaceHandler
	actStore *activity.Store
}

func NewTaskHandler(ws *WorkspaceHandler, actStore *activity.Store) *TaskHandler {
	return &TaskHandler{
		ws:       ws,
		actStore: actStore,
	}
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

	taskText, done, err := openspec.ToggleTask(path, name, index)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.actStore != nil {
		summary := taskText
		if done {
			summary = "Completed: " + taskText
		} else {
			summary = "Uncompleted: " + taskText
		}
		_ = h.actStore.Append(id, name, activity.Entry{
			Type:     "kanban.task_toggled",
			Category: "kanban",
			Summary:  summary,
			Meta: map[string]any{
				"task":  taskText,
				"done":  done,
				"index": index,
			},
		})
	}

	w.WriteHeader(http.StatusOK)
}
