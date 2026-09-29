package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
	"github.com/go-chi/chi/v5"
)

type FFHandler struct {
	ws        *WorkspaceHandler
	mgr       *session.Manager
	convStore *conversation.Store
	actStore  *activity.Store
	watcher   *watcher.WatcherService

	mu      sync.Mutex
	running map[string]struct{} // key: wsID+"/"+changeName
}

func NewFFHandler(ws *WorkspaceHandler, mgr *session.Manager, convStore *conversation.Store, actStore *activity.Store, watcherSvc *watcher.WatcherService) *FFHandler {
	return &FFHandler{
		ws:        ws,
		mgr:       mgr,
		convStore: convStore,
		actStore:  actStore,
		watcher:   watcherSvc,
		running:   make(map[string]struct{}),
	}
}

func (h *FFHandler) ffKey(wsID, changeName string) string {
	return wsID + "/" + changeName
}

func (h *FFHandler) isRunning(wsID, changeName string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.running[h.ffKey(wsID, changeName)]
	return ok
}

func (h *FFHandler) markRunning(wsID, changeName string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.running[h.ffKey(wsID, changeName)] = struct{}{}
}

func (h *FFHandler) markDone(wsID, changeName string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.running, h.ffKey(wsID, changeName))
}

func (h *FFHandler) TriggerFF(w http.ResponseWriter, r *http.Request) {
	wsID := chi.URLParam(r, "id")
	changeName := chi.URLParam(r, "name")

	workspacePath, ok := h.ws.workspacePath(wsID)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	if h.isRunning(wsID, changeName) {
		http.Error(w, "ff already running for this change", http.StatusConflict)
		return
	}

	var cfg agents.AgentConfig
	if h.mgr != nil {
		cfg = h.mgr.ResolveAgentConfig(wsID, changeName)
	} else {
		var ok bool
		cfg, ok = agents.ByID("claude")
		if !ok {
			http.Error(w, "agent not found", http.StatusInternalServerError)
			return
		}
	}

	ts := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	logFile, err := h.convStore.OpenRun(wsID, changeName, "ff", ts)
	if err != nil {
		http.Error(w, "failed to open conversation log: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var customEnv map[string]string
	if h.mgr != nil && h.mgr.Prefs() != nil {
		if p, err := h.mgr.Prefs().Load(); err == nil && p != nil {
			customEnv = p.EnvFor(cfg.ID)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	proc, err := session.StartSubprocess(ctx, workspacePath, cfg, "", "", false, nil, customEnv, false)
	if err != nil {
		cancel()
		logFile.Close()
		http.Error(w, "failed to start ff subprocess: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.markRunning(wsID, changeName)
	if h.watcher != nil {
		h.watcher.Broadcast(wsID, watcher.Event{Type: "ff_started", Name: changeName})
	}
	if h.actStore != nil {
		_ = h.actStore.Append(wsID, changeName, activity.Entry{
			Type:     "kanban.ff_triggered",
			Category: "kanban",
			Summary:  "Fast-forward triggered",
		})
	}

	initMsg := map[string]interface{}{
		"type": "user",
		"message": map[string]string{
			"role":    "user",
			"content": "/opsx:ff",
		},
	}
	initBytes, _ := json.Marshal(initMsg)
	proc.Write(append(initBytes, '\n'))

	go func() {
		defer cancel()
		defer logFile.Close()
		defer h.markDone(wsID, changeName)

		scanner := bufio.NewScanner(proc.Stdout())
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Bytes()
			b := make([]byte, len(line))
			copy(b, line)
			logFile.Write(b)
			logFile.Write([]byte("\n"))
		}

		if err := proc.Wait(); err != nil {
			h.watcher.Broadcast(wsID, watcher.Event{
				Type:  "ff_failed",
				Name:  changeName,
				Error: err.Error(),
			})
			return
		}

		changeDir := filepath.Join(workspacePath, "openspec", "changes", changeName)
		if err := openspec.SetLaunched(changeDir, false); err != nil {
			log.Printf("[ff] failed to set launched=false for %s: %v", changeName, err)
		}

		h.watcher.Broadcast(wsID, watcher.Event{Type: "ff_done", Name: changeName})
	}()

	w.WriteHeader(http.StatusAccepted)
}

func (h *FFHandler) ResetTasks(w http.ResponseWriter, r *http.Request) {
	wsID := chi.URLParam(r, "id")
	changeName := chi.URLParam(r, "name")

	workspacePath, ok := h.ws.workspacePath(wsID)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	if h.isRunning(wsID, changeName) {
		http.Error(w, "ff is running for this change", http.StatusConflict)
		return
	}

	changeDir := filepath.Join(workspacePath, "openspec", "changes", changeName)
	tasksPath := filepath.Join(changeDir, "tasks.md")
	if err := os.WriteFile(tasksPath, []byte{}, 0644); err != nil && !os.IsNotExist(err) {
		http.Error(w, "failed to reset tasks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := openspec.ClearKanbanState(changeDir); err != nil && !os.IsNotExist(err) {
		http.Error(w, "failed to reset kanban state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if h.actStore != nil {
		_ = h.actStore.Append(wsID, changeName, activity.Entry{
			Type:     "kanban.tasks_reset",
			Category: "kanban",
			Summary:  "Tasks reset",
		})
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *FFHandler) ListConversationRuns(w http.ResponseWriter, r *http.Request) {
	wsID := chi.URLParam(r, "id")
	changeName := chi.URLParam(r, "name")
	kind := chi.URLParam(r, "kind")

	if _, ok := h.ws.workspacePath(wsID); !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	runs, err := h.convStore.List(wsID, changeName, kind)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(runs)
}

type conversationRunResponse struct {
	Ts       string            `json:"ts"`
	Messages []json.RawMessage `json:"messages"`
}

func (h *FFHandler) GetConversationRun(w http.ResponseWriter, r *http.Request) {
	wsID := chi.URLParam(r, "id")
	changeName := chi.URLParam(r, "name")
	kind := chi.URLParam(r, "kind")
	ts := chi.URLParam(r, "ts")

	if _, ok := h.ws.workspacePath(wsID); !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	lines, err := h.convStore.Load(wsID, changeName, kind, ts)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "run not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	msgs := make([]json.RawMessage, len(lines))
	for i, l := range lines {
		msgs[i] = json.RawMessage(l)
	}
	json.NewEncoder(w).Encode(conversationRunResponse{Ts: ts, Messages: msgs})
}
