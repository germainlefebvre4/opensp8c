package handlers

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/go-chi/chi/v5"
)

type TaskHandler struct {
	ws       *WorkspaceHandler
	actStore *activity.Store
	poolReg  *pool.Registry
}

func NewTaskHandler(ws *WorkspaceHandler, actStore *activity.Store, poolReg *pool.Registry) *TaskHandler {
	return &TaskHandler{
		ws:       ws,
		actStore: actStore,
		poolReg:  poolReg,
	}
}

// toggleRoot returns the root whose openspec/changes/<name>/tasks.md a toggle
// must edit: the worktree of the pool worker holding the change (active or
// paused) when that worktree has a usable task list, the workspace repository
// otherwise. It mirrors the read side (openspec.ApplyWorktreeProgress), so the
// index the client computed on the displayed list targets the same file.
func (h *TaskHandler) toggleRoot(workspaceID, workspacePath, change string) string {
	hw, held := activeWorkerChanges(h.poolReg, workspaceID)[change]
	if !held || hw.WorktreePath == "" {
		return workspacePath
	}
	tasksPath := filepath.Join(hw.WorktreePath, "openspec", "changes", change, "tasks.md")
	if _, total := openspec.ParseTaskProgress(tasksPath); total == 0 {
		return workspacePath
	}
	return hw.WorktreePath
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

	taskText, done, err := openspec.ToggleTask(h.toggleRoot(id, path, name), name, index)
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
