package handlers

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/glefebvre/opensp8c/internal/watcher"
	"github.com/glefebvre/opensp8c/internal/workspace"
	"github.com/go-chi/chi/v5"
)

// TestPoolEvents_StartStopOverSSE drives pool/start then pool/stop over HTTP
// against a subscribed SSE stream, and verifies both transitions arrive as
// pool_updated events on /api/workspaces/{id}/events.
func TestPoolEvents_StartStopOverSSE(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "openspec", "changes"), 0755); err != nil {
		t.Fatalf("failed to create openspec/changes dir: %v", err)
	}

	absPath, err := filepath.Abs(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve abs path: %v", err)
	}
	workspaceID := workspace.StableID(absPath)

	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: tmpDir}}}

	watcherSvc := watcher.NewWatcherService()
	if err := watcherSvc.StartWatching(workspaceID, absPath); err != nil {
		t.Fatalf("failed to start watching: %v", err)
	}
	t.Cleanup(func() { watcherSvc.StopWatching(workspaceID) })

	poolReg := pool.NewRegistry(watcherSvc, nil, nil, nil)
	wsHandler := NewWorkspaceHandler(cfg, "", poolReg)
	poolHandler := NewPoolHandler(wsHandler, poolReg)
	eventsHandler := NewEventsHandler(wsHandler, watcherSvc)

	r := chi.NewRouter()
	r.Route("/api/workspaces/{id}", func(r chi.Router) {
		r.Get("/events", eventsHandler.HandleSSE)
		r.Post("/pool/start", poolHandler.StartPool)
		r.Post("/pool/stop", poolHandler.StopPool)
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	// The SSE handler doesn't write (and so doesn't flush headers) until its
	// first event, so http.Get would block the test until then. Issue it in
	// the background, on a cancellable request so we can unblock the
	// server's handler at the end of the test instead of hanging on
	// srv.Close(). Give the handler a brief moment to reach Subscribe before
	// triggering pool/start, so the first pool_updated isn't dropped on a
	// channel nobody is listening to yet.
	sseCtx, cancelSSE := context.WithCancel(context.Background())
	t.Cleanup(cancelSSE)

	events := make(chan string, 8)
	go func() {
		req, err := http.NewRequestWithContext(sseCtx, http.MethodGet, srv.URL+"/api/workspaces/"+workspaceID+"/events", nil)
		if err != nil {
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event: ") {
				events <- strings.TrimPrefix(line, "event: ")
			}
		}
	}()
	time.Sleep(200 * time.Millisecond)

	waitFor := func(want string) {
		t.Helper()
		deadline := time.After(5 * time.Second)
		for {
			select {
			case ev := <-events:
				if ev == want {
					return
				}
			case <-deadline:
				t.Fatalf("timed out waiting for %q event", want)
			}
		}
	}

	startBody := strings.NewReader(`{"size":1,"delegation_mode":"hitl-review","max_attempts":1}`)
	startResp, err := http.Post(srv.URL+"/api/workspaces/"+workspaceID+"/pool/start", "application/json", startBody)
	if err != nil {
		t.Fatalf("failed to POST pool/start: %v", err)
	}
	startResp.Body.Close()
	if startResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from pool/start, got %d", startResp.StatusCode)
	}
	waitFor("pool_updated")

	stopResp, err := http.Post(srv.URL+"/api/workspaces/"+workspaceID+"/pool/stop", "application/json", nil)
	if err != nil {
		t.Fatalf("failed to POST pool/stop: %v", err)
	}
	stopResp.Body.Close()
	if stopResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from pool/stop, got %d", stopResp.StatusCode)
	}
	waitFor("pool_updated")
}
