package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/docsgen"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
	"github.com/go-chi/chi/v5"
)

// DocsHandler triggers the single-run documentation generation, and exposes
// the generated pages under docs/opensp8c/ for a workspace, per the
// spec-documentation-generation and spec-documentation-view capabilities.
type DocsHandler struct {
	ws      *WorkspaceHandler
	mgr     *session.Manager
	watcher *watcher.WatcherService

	mu      sync.Mutex
	running map[string]struct{} // key: workspace id
}

func NewDocsHandler(ws *WorkspaceHandler, mgr *session.Manager, watcherSvc *watcher.WatcherService) *DocsHandler {
	return &DocsHandler{
		ws:      ws,
		mgr:     mgr,
		watcher: watcherSvc,
		running: make(map[string]struct{}),
	}
}

func (h *DocsHandler) isRunning(wsID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.running[wsID]
	return ok
}

func (h *DocsHandler) markRunning(wsID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.running[wsID] = struct{}{}
}

func (h *DocsHandler) markDone(wsID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.running, wsID)
}

// tryMarkRunning atomically checks-and-sets the running guard, returning
// false if a run was already in progress for this workspace (avoiding the
// isRunning/markRunning check-then-act race between concurrent requests).
func (h *DocsHandler) tryMarkRunning(wsID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.running[wsID]; ok {
		return false
	}
	h.running[wsID] = struct{}{}
	return true
}

type docsListResponse struct {
	Pages      []string `json:"pages"`
	IsStale    bool     `json:"is_stale"`
	Generating bool     `json:"generating"`
}

// ListDocs returns the documentation pages currently present under
// docs/opensp8c/ (fixed order, filtered to what exists on disk), whether the
// documentation is potentially stale, and whether a generation run is
// currently in progress for this workspace.
func (h *DocsHandler) ListDocs(w http.ResponseWriter, r *http.Request) {
	wsID := chi.URLParam(r, "id")
	path, ok := h.ws.workspacePath(wsID)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	pages, err := docsgen.ListPages(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stale, err := docsgen.IsStale(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(docsListResponse{
		Pages:      pages,
		IsStale:    stale,
		Generating: h.isRunning(wsID),
	})
}

type docPageResponse struct {
	Page    string `json:"page"`
	Content string `json:"content"`
}

// GetDocPage returns one generated page's raw Markdown content.
func (h *DocsHandler) GetDocPage(w http.ResponseWriter, r *http.Request) {
	wsID := chi.URLParam(r, "id")
	page := chi.URLParam(r, "page")

	path, ok := h.ws.workspacePath(wsID)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	content, err := docsgen.ReadPage(path, page)
	if err != nil {
		if errors.Is(err, docsgen.ErrPageNotFound) {
			http.Error(w, "doc page not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(docPageResponse{Page: page, Content: content})
}

// TriggerGenerate starts a single headless agent run producing the fixed
// documentation page set for this workspace, rejecting the request with 409
// if a run is already in progress for it.
func (h *DocsHandler) TriggerGenerate(w http.ResponseWriter, r *http.Request) {
	wsID := chi.URLParam(r, "id")

	workspacePath, ok := h.ws.workspacePath(wsID)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	if !h.tryMarkRunning(wsID) {
		http.Error(w, "docs generation already running for this workspace", http.StatusConflict)
		return
	}

	var cfg agents.AgentConfig
	if h.mgr != nil {
		cfg = h.mgr.ResolveAgentConfig(wsID, "")
	} else {
		var ok bool
		cfg, ok = agents.ByID("claude")
		if !ok {
			h.markDone(wsID)
			http.Error(w, "agent not found", http.StatusInternalServerError)
			return
		}
	}

	docsPrompt, err := docsgen.BuildPrompt(workspacePath)
	if err != nil {
		h.markDone(wsID)
		http.Error(w, "failed to assemble generation prompt: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var customEnv map[string]string
	if h.mgr != nil && h.mgr.Prefs() != nil {
		if p, err := h.mgr.Prefs().Load(); err == nil && p != nil {
			customEnv = p.Env
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	proc, err := session.StartSubprocess(ctx, workspacePath, cfg, docsPrompt, "", false, nil, customEnv, false)
	if err != nil {
		cancel()
		h.markDone(wsID)
		http.Error(w, "failed to start docs generation subprocess: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.watcher.Broadcast(wsID, watcher.Event{Type: "docs_generation_started"})

	initMsg := map[string]interface{}{
		"type": "user",
		"message": map[string]string{
			"role":    "user",
			"content": "Generate the documentation now, following the instructions above.",
		},
	}
	initBytes, _ := json.Marshal(initMsg)
	proc.Write(append(initBytes, '\n'))
	// This is a one-shot run (unlike the ongoing explore/ff chat sessions):
	// closing stdin right after the single turn tells the agent CLI no
	// further turns are coming, which is what lets it wrap up and exit
	// after finishing this one, instead of idling indefinitely waiting for
	// more stream-json input.
	proc.CloseStdin()

	go func() {
		defer cancel()
		defer h.markDone(wsID)

		scanner := bufio.NewScanner(proc.Stdout())
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			// Drain the subprocess's stdout: the run's outcome is the files
			// it writes directly to disk, there is no chat UI consuming this.
		}

		if err := proc.Wait(); err != nil {
			h.watcher.Broadcast(wsID, watcher.Event{
				Type:  "docs_generation_failed",
				Error: err.Error(),
			})
			return
		}
		h.watcher.Broadcast(wsID, watcher.Event{Type: "docs_generation_done"})
	}()

	w.WriteHeader(http.StatusAccepted)
}
