package handlers

import (
	"errors"
	"net/http"
	"os"
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

// taskTarget is the file a toggle edits.
type taskTarget int

const (
	targetMain     taskTarget = iota // openspec/changes/<name>/tasks.md of the workspace repository
	targetWorktree                   // worktree of the pool worker holding the change, no commit
	targetBranch                     // feature/<name>, one commit per tick
)

// resolveTaskTarget picks the source of the tasks of a change, the same one
// the read side shows: the worktree of the pool worker holding the change
// (active or paused) when it has a usable task list, else feature/<change>
// when its tasks.md has a task and a pool exists to commit into it, else the
// workspace repository. root is the directory to edit for the worktree and
// main targets.
func (h *TaskHandler) resolveTaskTarget(workspaceID, workspacePath, change string) (taskTarget, string) {
	if hw, held := activeWorkerChanges(h.poolReg, workspaceID)[change]; held && openspec.WorktreeHasTasks(hw.WorktreePath, change) {
		return targetWorktree, hw.WorktreePath
	}
	if h.poolReg != nil && openspec.FeatureBranches(workspacePath)[change] {
		if content, ok := pool.NewWorktreeController(workspacePath, workspaceID, "").BranchTasks(change); ok && len(openspec.ParseTaskListContent(content)) > 0 {
			return targetBranch, ""
		}
	}
	return targetMain, workspacePath
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

	if h.poolReg != nil && h.poolReg.For(id).VerificationRunning(name) {
		writeReviewAction(w, http.StatusConflict, reviewActionError{Code: "verification_busy", Message: "une vérification est en cours sur ce changement"})
		return
	}

	var taskText string
	var done bool
	target, root := h.resolveTaskTarget(id, path, name)
	if target == targetBranch {
		taskText, done, err = h.poolReg.For(id).ToggleBranchTask(r.Context(), id, path, name, index)
	} else {
		taskText, done, err = openspec.ToggleTask(root, name, index)
	}
	if err != nil {
		switch {
		case errors.Is(err, os.ErrNotExist):
			http.Error(w, "task not found", http.StatusNotFound)
		case errors.Is(err, pool.ErrReviewBusy):
			writeReviewAction(w, http.StatusConflict, reviewActionError{Code: "review_busy", Message: err.Error()})
		case errors.Is(err, pool.ErrWorkerActive):
			writeReviewAction(w, http.StatusConflict, reviewActionError{Code: "worker_active", Message: err.Error()})
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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
