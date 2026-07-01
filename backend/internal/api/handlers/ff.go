package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
)

type FFHandler struct {
	ws        *WorkspaceHandler
	convStore *conversation.Store
	watcher   *watcher.WatcherService

	mu      sync.Mutex
	running map[string]struct{} // key: wsID+"/"+changeName
}

func NewFFHandler(ws *WorkspaceHandler, _ interface{}, convStore *conversation.Store, watcherSvc *watcher.WatcherService) *FFHandler {
	return &FFHandler{
		ws:        ws,
		convStore: convStore,
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

	cfg, ok := agents.ByID("claude")
	if !ok {
		http.Error(w, "agent not found", http.StatusInternalServerError)
		return
	}

	ts := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	logFile, err := h.convStore.OpenRun(wsID, changeName, "ff", ts)
	if err != nil {
		http.Error(w, "failed to open conversation log: "+err.Error(), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	proc, err := session.StartSubprocess(ctx, workspacePath, cfg, "", "", false, nil)
	if err != nil {
		cancel()
		logFile.Close()
		http.Error(w, "failed to start ff subprocess: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.markRunning(wsID, changeName)
	h.watcher.Broadcast(wsID, watcher.Event{Type: "ff_started", Name: changeName})

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

	tasksPath := filepath.Join(workspacePath, "openspec", "changes", changeName, "tasks.md")
	if err := os.WriteFile(tasksPath, []byte{}, 0644); err != nil && !os.IsNotExist(err) {
		http.Error(w, "failed to reset tasks: "+err.Error(), http.StatusInternalServerError)
		return
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
	Ts       string          `json:"ts"`
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

