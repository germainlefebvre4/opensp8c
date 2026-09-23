package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
	"github.com/glefebvre/opensp8c/internal/workspace"
)

// docsEventsTestHandler is like docsTestHandler but also starts the watcher
// for the workspace so events broadcast on it can be observed via Subscribe,
// and lets the caller supply the mock CLI script body.
func docsEventsTestHandler(t *testing.T, workspacePath, mockScript string) (h *DocsHandler, wsID string, watcherSvc *watcher.WatcherService) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(workspacePath, "openspec", "changes"), 0755); err != nil {
		t.Fatalf("failed to create openspec dir: %v", err)
	}

	tmpDir := t.TempDir()
	mockCLIPath := filepath.Join(tmpDir, "mock-claude")
	if err := os.WriteFile(mockCLIPath, []byte(mockScript), 0755); err != nil {
		t.Fatalf("failed to write mock CLI: %v", err)
	}

	origCLI := agents.SupportedAgents[0].CLI
	agents.SupportedAgents[0].CLI = mockCLIPath
	t.Cleanup(func() { agents.SupportedAgents[0].CLI = origCLI })

	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: workspacePath}}}
	ws := NewWorkspaceHandler(cfg, "", nil)
	prefsSvc := preferences.NewService(filepath.Join(tmpDir, "preferences.json"))
	mgr := session.NewManager(prefsSvc, nil)

	watcherSvc = watcher.NewWatcherService()
	wsID = workspace.StableID(mustAbs(t, workspacePath))
	if err := watcherSvc.StartWatching(wsID, mustAbs(t, workspacePath)); err != nil {
		t.Fatalf("failed to start watching: %v", err)
	}
	t.Cleanup(func() { watcherSvc.StopWatching(wsID) })

	h = NewDocsHandler(ws, mgr, watcherSvc)
	return h, wsID, watcherSvc
}

const mockDocsCLISucceeds = "#!/bin/sh\n" +
	"if [ \"$1\" = \"--version\" ]; then\n echo 'mock 1.0.0'\n exit 0\nfi\n" +
	"mkdir -p docs/opensp8c\n" +
	"echo content > docs/opensp8c/overview.md\n"

const mockDocsCLIFails = "#!/bin/sh\n" +
	"if [ \"$1\" = \"--version\" ]; then\n echo 'mock 1.0.0'\n exit 0\nfi\n" +
	"exit 1\n"

func waitForEvent(t *testing.T, ch chan watcher.Event, want string, timeout time.Duration) watcher.Event {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case ev := <-ch:
			if ev.Type == want {
				return ev
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %q event", want)
		}
	}
}

func expectNoEvent(t *testing.T, ch chan watcher.Event, unwanted string, wait time.Duration) {
	t.Helper()
	deadline := time.After(wait)
	for {
		select {
		case ev := <-ch:
			if ev.Type == unwanted {
				t.Fatalf("expected no further %q event, got one", unwanted)
			}
		case <-deadline:
			return
		}
	}
}

// TestDocsEvents_SuccessEmitsStartedThenDone verifies a successful run
// broadcasts docs_generation_started then docs_generation_done.
func TestDocsEvents_SuccessEmitsStartedThenDone(t *testing.T) {
	workspacePath := t.TempDir()
	h, wsID, watcherSvc := docsEventsTestHandler(t, workspacePath, mockDocsCLISucceeds)
	ch := watcherSvc.Subscribe(wsID)

	rec := httptest.NewRecorder()
	h.TriggerGenerate(rec, docsRequest(http.MethodPost, "/x", wsID, ""))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	waitForEvent(t, ch, "docs_generation_started", 3*time.Second)
	waitForEvent(t, ch, "docs_generation_done", 3*time.Second)
}

// TestDocsEvents_FailureEmitsStartedThenFailed verifies a failing run
// broadcasts docs_generation_started then docs_generation_failed.
func TestDocsEvents_FailureEmitsStartedThenFailed(t *testing.T) {
	workspacePath := t.TempDir()
	h, wsID, watcherSvc := docsEventsTestHandler(t, workspacePath, mockDocsCLIFails)
	ch := watcherSvc.Subscribe(wsID)

	rec := httptest.NewRecorder()
	h.TriggerGenerate(rec, docsRequest(http.MethodPost, "/x", wsID, ""))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	waitForEvent(t, ch, "docs_generation_started", 3*time.Second)
	ev := waitForEvent(t, ch, "docs_generation_failed", 3*time.Second)
	if ev.Error == "" {
		t.Fatal("expected docs_generation_failed event to carry an error message")
	}
}

// TestDocsEvents_ConcurrentTriggerDoesNotDuplicateStarted verifies that a
// rejected concurrent trigger (409) never causes a second
// docs_generation_started to be broadcast for the still-running run.
func TestDocsEvents_ConcurrentTriggerDoesNotDuplicateStarted(t *testing.T) {
	workspacePath := t.TempDir()

	tmpDir := t.TempDir()
	blockFile := filepath.Join(tmpDir, "unblock")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then\n echo 'mock 1.0.0'\n exit 0\nfi\n" +
		"while [ ! -f \"" + blockFile + "\" ]; do sleep 0.02; done\n" +
		"mkdir -p docs/opensp8c\n" +
		"echo content > docs/opensp8c/overview.md\n"

	h, wsID, watcherSvc := docsEventsTestHandler(t, workspacePath, script)
	t.Cleanup(func() { _ = os.WriteFile(blockFile, []byte("go"), 0644) })
	ch := watcherSvc.Subscribe(wsID)

	rec1 := httptest.NewRecorder()
	h.TriggerGenerate(rec1, docsRequest(http.MethodPost, "/x", wsID, ""))
	if rec1.Code != http.StatusAccepted {
		t.Fatalf("expected first trigger to be accepted, got %d", rec1.Code)
	}
	waitForEvent(t, ch, "docs_generation_started", 3*time.Second)

	rec2 := httptest.NewRecorder()
	h.TriggerGenerate(rec2, docsRequest(http.MethodPost, "/x", wsID, ""))
	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected second concurrent trigger to be rejected with 409, got %d", rec2.Code)
	}

	expectNoEvent(t, ch, "docs_generation_started", 500*time.Millisecond)
}
