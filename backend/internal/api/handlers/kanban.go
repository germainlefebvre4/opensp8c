package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
)

type KanbanHandler struct {
	ws        *WorkspaceHandler
	prefs     *preferences.Service
	poolMgr   *pool.Manager
	sessions  *session.Manager
	convStore *conversation.Store
	watcher   *watcher.WatcherService
	draftsDir string
}

func NewKanbanHandler(ws *WorkspaceHandler, prefs *preferences.Service, poolMgr *pool.Manager, sessions *session.Manager, convStore *conversation.Store, watcherSvc *watcher.WatcherService, draftsDir string) *KanbanHandler {
	return &KanbanHandler{
		ws:        ws,
		prefs:     prefs,
		poolMgr:   poolMgr,
		sessions:  sessions,
		convStore: convStore,
		watcher:   watcherSvc,
		draftsDir: draftsDir,
	}
}

// activeWorkerChanges returns the set of change names currently claimed by an
// Agent Pool worker, per the pool manager's in-memory state.
func (h *KanbanHandler) activeWorkerChanges() map[string]bool {
	active := make(map[string]bool)
	if h.poolMgr == nil {
		return active
	}
	_, _, workers := h.poolMgr.Status()
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

	activeWorkers := h.activeWorkerChanges()
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
	detail.WorkerActive = h.activeWorkerChanges()[name]
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

	if h.activeWorkerChanges()[name] {
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
