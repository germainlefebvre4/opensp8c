package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
	"github.com/go-chi/chi/v5"
)

type KanbanHandler struct {
	ws        *WorkspaceHandler
	prefs     *preferences.Service
	poolReg   *pool.Registry
	sessions  *session.Manager
	convStore *conversation.Store
	watcher   *watcher.WatcherService
	draftsDir string

	// cancelAndWait cancels the worker of a change and waits for its end;
	// a seam over the pool manager so handler tests can script its answer.
	cancelAndWait func(ctx context.Context, workspaceID, change string) (bool, pool.WorkerResult, bool)
}

func NewKanbanHandler(ws *WorkspaceHandler, prefs *preferences.Service, poolReg *pool.Registry, sessions *session.Manager, convStore *conversation.Store, watcherSvc *watcher.WatcherService, draftsDir string) *KanbanHandler {
	h := &KanbanHandler{
		ws:        ws,
		prefs:     prefs,
		poolReg:   poolReg,
		sessions:  sessions,
		convStore: convStore,
		watcher:   watcherSvc,
		draftsDir: draftsDir,
	}
	h.cancelAndWait = func(ctx context.Context, workspaceID, change string) (bool, pool.WorkerResult, bool) {
		if h.poolReg == nil {
			return false, pool.WorkerResult{}, false
		}
		return h.poolReg.For(workspaceID).CancelAndWaitForChange(ctx, change)
	}
	return h
}

// activeWorkerChanges returns the set of change names currently claimed by an
// Agent Pool worker for workspaceID, per that workspace's own pool manager
// in-memory state.
func (h *KanbanHandler) activeWorkerChanges(workspaceID string) map[string]bool {
	active := make(map[string]bool)
	if h.poolReg == nil {
		return active
	}
	_, _, workers := h.poolReg.For(workspaceID).Status(workspaceID)
	for _, w := range workers {
		active[w.ActiveChange] = true
	}
	return active
}

func (h *KanbanHandler) ListChanges(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	changes, err := openspec.ListChanges(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if changes == nil {
		changes = []openspec.Change{}
	}

	activeWorkers := h.activeWorkerChanges(id)
	for i := range changes {
		changes[i].WorkerActive = activeWorkers[changes[i].Name]
	}

	// Merge ghost records (app-level explorations) into the changes list.
	if h.prefs != nil {
		draftsDir := filepath.Join(filepath.Dir(h.prefs.Path()), "drafts")
		for _, e := range h.prefs.ListExplorations(id) {
			tasksDone := 0
			tasksTotal := 0

			draftPath := filepath.Join(draftsDir, e.ID+".json")
			if data, err := os.ReadFile(draftPath); err == nil {
				var draft struct {
					Tasks []struct {
						Done bool `json:"done"`
					} `json:"tasks"`
				}
				if err := json.Unmarshal(data, &draft); err == nil {
					tasksTotal = len(draft.Tasks)
					for _, t := range draft.Tasks {
						if t.Done {
							tasksDone++
						}
					}
				}
			}

			changes = append(changes, openspec.Change{
				Name:         e.Name,
				KanbanStatus: "to-explore",
				Created:      e.CreatedAt,
				IsGhost:      true,
				GhostID:      e.ID,
				TasksDone:    tasksDone,
				TasksTotal:   tasksTotal,
			})
		}
	}

	json.NewEncoder(w).Encode(changes)
}

func (h *KanbanHandler) ListArchivedChanges(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	changes, err := openspec.ListArchivedChanges(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if changes == nil {
		changes = []openspec.Change{}
	}
	json.NewEncoder(w).Encode(changes)
}

func (h *KanbanHandler) GetChange(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := chi.URLParam(r, "name")

	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	detail, err := openspec.GetChangeDetail(path, name)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "change not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	detail.WorkerActive = h.activeWorkerChanges(id)[name]
	json.NewEncoder(w).Encode(detail)
}

// DeleteChange permanently deletes a change's folder. It refuses when an
// Agent Pool worker is actively working on the change, and cascades the
// deletion to any ghost exploration record sharing the change's name.
func (h *KanbanHandler) DeleteChange(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := chi.URLParam(r, "name")

	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	changeDir := filepath.Join(path, "openspec", "changes", name)
	if _, err := os.Stat(changeDir); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "change not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.activeWorkerChanges(id)[name] {
		http.Error(w, "a worker is active on this change", http.StatusConflict)
		return
	}

	if err := os.RemoveAll(changeDir); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.cascadeDeleteGhost(id, name)

	w.WriteHeader(http.StatusNoContent)
}

// Launch marks a change "launched", promoting it from Ready to To Do and
// making it eligible for Agent Pool pickup. Idempotent.
func (h *KanbanHandler) Launch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := chi.URLParam(r, "name")

	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	changeDir := filepath.Join(path, "openspec", "changes", name)
	if _, err := os.Stat(changeDir); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "change not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := openspec.SetLaunched(changeDir, true); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Unlaunch marks a change "not launched", demoting it from To Do to Ready.
// Refused while an Agent Pool worker is actively working on the change,
// unless ?force=true is provided, in which case the active worker is canceled.
func (h *KanbanHandler) Unlaunch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := chi.URLParam(r, "name")

	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	changeDir := filepath.Join(path, "openspec", "changes", name)
	if _, err := os.Stat(changeDir); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "change not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	force := r.URL.Query().Get("force") == "true"

	if h.activeWorkerChanges(id)[name] {
		if !force {
			http.Error(w, "a worker is active on this change", http.StatusConflict)
			return
		}
		// Wait for the worker's real end: the answer depends on whether it
		// merged the change before honoring the cancellation.
		_, result, timedOut := h.cancelAndWait(r.Context(), id, name)
		switch {
		case timedOut:
			w.Header().Set("Retry-After", "5")
			writeUnlaunchError(w, http.StatusServiceUnavailable, map[string]string{"code": "worker_still_running"})
			return
		case result.Merged:
			writeUnlaunchError(w, http.StatusConflict, map[string]string{"code": "change_already_merged", "target": result.Target})
			return
		}
	}

	if err := openspec.SetLaunched(changeDir, false); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeUnlaunchError(w http.ResponseWriter, status int, body map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// ReorderReady persists the priority rank of the Ready column's changes,
// sequentially numbered in the order given by the request body.
func (h *KanbanHandler) ReorderReady(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	var body struct {
		Order []string `json:"order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	changesDir := filepath.Join(path, "openspec", "changes")
	if err := openspec.ReorderReady(changesDir, body.Order); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// cascadeDeleteGhost removes the ghost exploration record matching changeName,
// if any, mirroring ExploreHandler.DeleteGhost's cleanup sequence.
func (h *KanbanHandler) cascadeDeleteGhost(workspaceID, changeName string) {
	if h.prefs == nil {
		return
	}
	var ghostID string
	for _, e := range h.prefs.ListExplorations(workspaceID) {
		if e.Name == changeName {
			ghostID = e.ID
			break
		}
	}
	if ghostID == "" {
		return
	}

	if h.sessions != nil {
		h.sessions.StopAnonymous(workspaceID, ghostID)
	}
	_ = h.prefs.DeleteExploration(ghostID)
	if h.draftsDir != "" {
		draftPath := filepath.Join(h.draftsDir, ghostID+".json")
		if err := os.Remove(draftPath); err != nil && !os.IsNotExist(err) {
			// best-effort cleanup, nothing actionable if it fails
		}
	}
	if h.convStore != nil {
		_ = h.convStore.DeleteExplorationLogs(workspaceID, ghostID)
	}
	if h.watcher != nil {
		h.watcher.Broadcast(workspaceID, watcher.Event{Type: "exploration_deleted", Name: ghostID})
	}
}
