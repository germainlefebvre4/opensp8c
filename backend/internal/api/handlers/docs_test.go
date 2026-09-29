package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
	"github.com/glefebvre/opensp8c/internal/workspace"
	"github.com/go-chi/chi/v5"
)

// docsRequest builds an *http.Request carrying workspaceID (and optionally a
// page id) as chi URL params, mirroring how router.go dispatches
// /workspaces/{id}/docs*.
func docsRequest(method, path, workspaceID, page string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workspaceID)
	if page != "" {
		rctx.URLParams.Add("page", page)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// docsTestHandler wires a DocsHandler against a single configured workspace
// backed by a tmp dir, with a mock "claude" CLI installed so
// session.StartSubprocess can actually launch a fake subprocess.
func docsTestHandler(t *testing.T, workspacePath string) (h *DocsHandler, wsID string) {
	t.Helper()

	tmpDir := t.TempDir()
	mockCLIPath := filepath.Join(tmpDir, "mock-claude")
	writeMockDocsCLI(t, mockCLIPath, "")

	origCLI := agents.SupportedAgents[0].CLI
	if agents.SupportedAgents[0].ID != "claude" {
		t.Fatalf("expected SupportedAgents[0] to be claude, got %s", agents.SupportedAgents[0].ID)
	}
	agents.SupportedAgents[0].CLI = mockCLIPath
	t.Cleanup(func() { agents.SupportedAgents[0].CLI = origCLI })

	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: workspacePath}}}
	ws := NewWorkspaceHandler(cfg, "", nil)

	prefsSvc := preferences.NewService(filepath.Join(tmpDir, "preferences.json"))
	mgr := session.NewManager(prefsSvc, nil)
	watcherSvc := watcher.NewWatcherService()

	h = NewDocsHandler(ws, mgr, watcherSvc)
	wsID = workspace.StableID(mustAbs(t, workspacePath))
	return h, wsID
}

// writeMockDocsCLI writes an executable shell script that answers --version
// probes immediately (used by agents.Detect), and for any other invocation
// creates the generated doc pages directly under the process's working
// directory (cmd.Dir is set to the workspace path), simulating what a real
// agent run does when generating documentation.
func writeMockDocsCLI(t *testing.T, path, blockUntilFile string) {
	t.Helper()
	block := ""
	if blockUntilFile != "" {
		block = "while [ ! -f \"" + blockUntilFile + "\" ]; do sleep 0.02; done\n"
	}
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then\n" +
		"  echo 'mock 1.0.0'\n" +
		"  exit 0\n" +
		"fi\n" +
		block +
		"mkdir -p docs/opensp8c\n" +
		"echo 'overview content' > docs/opensp8c/overview.md\n" +
		"echo 'architecture content' > docs/opensp8c/architecture.md\n" +
		"echo 'domain model content' > docs/opensp8c/domain-model.md\n"
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write mock CLI: %v", err)
	}
}

func waitForCondition(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for condition")
}

// TestTriggerGenerate_WritesExpectedFiles verifies that triggering a
// generation run against a fake agent binary results in the expected files
// being created directly under <workspace path>/docs/opensp8c/.
func TestTriggerGenerate_WritesExpectedFiles(t *testing.T) {
	workspacePath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspacePath, "openspec", "specs", "capability-a"), 0755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspacePath, "openspec", "specs", "capability-a", "spec.md"), []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write spec.md: %v", err)
	}

	h, wsID := docsTestHandler(t, workspacePath)

	rec := httptest.NewRecorder()
	h.TriggerGenerate(rec, docsRequest(http.MethodPost, "/workspaces/"+wsID+"/docs/generate", wsID, ""))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 accepted, got %d: %s", rec.Code, rec.Body.String())
	}

	waitForCondition(t, 3*time.Second, func() bool {
		_, err := os.Stat(filepath.Join(workspacePath, "docs", "opensp8c", "overview.md"))
		return err == nil
	})

	for _, page := range []string{"overview.md", "architecture.md", "domain-model.md"} {
		data, err := os.ReadFile(filepath.Join(workspacePath, "docs", "opensp8c", page))
		if err != nil {
			t.Fatalf("expected %s to be created: %v", page, err)
		}
		if len(strings.TrimSpace(string(data))) == 0 {
			t.Fatalf("expected %s to have content", page)
		}
	}

	waitForCondition(t, 3*time.Second, func() bool { return !h.isRunning(wsID) })
}

// TestTriggerGenerate_ConcurrentRejectedWith409 verifies that a second
// generation request for the same workspace while one is already in
// progress is rejected with 409, without starting a second run.
func TestTriggerGenerate_ConcurrentRejectedWith409(t *testing.T) {
	workspacePath := t.TempDir()

	tmpDir := t.TempDir()
	blockFile := filepath.Join(tmpDir, "unblock")
	mockCLIPath := filepath.Join(tmpDir, "mock-claude")
	writeMockDocsCLI(t, mockCLIPath, blockFile)

	origCLI := agents.SupportedAgents[0].CLI
	agents.SupportedAgents[0].CLI = mockCLIPath
	t.Cleanup(func() { agents.SupportedAgents[0].CLI = origCLI })

	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: workspacePath}}}
	ws := NewWorkspaceHandler(cfg, "", nil)
	prefsSvc := preferences.NewService(filepath.Join(tmpDir, "preferences.json"))
	mgr := session.NewManager(prefsSvc, nil)
	watcherSvc := watcher.NewWatcherService()
	h := NewDocsHandler(ws, mgr, watcherSvc)
	wsID := workspace.StableID(mustAbs(t, workspacePath))

	t.Cleanup(func() { _ = os.WriteFile(blockFile, []byte("go"), 0644) })

	var wg sync.WaitGroup
	codes := make([]int, 2)
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func(idx int) {
			defer wg.Done()
			rec := httptest.NewRecorder()
			h.TriggerGenerate(rec, docsRequest(http.MethodPost, "/workspaces/"+wsID+"/docs/generate", wsID, ""))
			codes[idx] = rec.Code
		}(i)
	}
	wg.Wait()

	accepted, conflicts := 0, 0
	for _, c := range codes {
		switch c {
		case http.StatusAccepted:
			accepted++
		case http.StatusConflict:
			conflicts++
		}
	}
	if accepted != 1 || conflicts != 1 {
		t.Fatalf("expected exactly one 202 and one 409, got codes: %v", codes)
	}
}

// TestListDocs_OrderAndStaleness verifies GET .../docs reports the fixed
// page order/filtering and the is_stale flag.
func TestListDocs_OrderAndStaleness(t *testing.T) {
	workspacePath := t.TempDir()
	h, wsID := docsTestHandler(t, workspacePath)

	// No pages generated yet.
	rec := httptest.NewRecorder()
	h.ListDocs(rec, docsRequest(http.MethodGet, "/workspaces/"+wsID+"/docs", wsID, ""))
	var resp docsListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Pages) != 0 || resp.IsStale {
		t.Fatalf("expected no pages and no staleness, got %+v", resp)
	}

	// Generate pages, then add a newer spec to trigger staleness.
	docsDir := filepath.Join(workspacePath, "docs", "opensp8c")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("failed to create docs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "overview.md"), []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write overview.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "workflows.md"), []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write workflows.md: %v", err)
	}

	rec2 := httptest.NewRecorder()
	h.ListDocs(rec2, docsRequest(http.MethodGet, "/workspaces/"+wsID+"/docs", wsID, ""))
	var resp2 docsListResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp2.Pages) != 2 || resp2.Pages[0] != "overview" || resp2.Pages[1] != "workflows" {
		t.Fatalf("expected [overview workflows] in fixed order, got %v", resp2.Pages)
	}
}

// TestGetDocPage_NotFound verifies GET .../docs/{page} returns 404 for a
// page that hasn't been generated.
func TestGetDocPage_NotFound(t *testing.T) {
	workspacePath := t.TempDir()
	h, wsID := docsTestHandler(t, workspacePath)

	rec := httptest.NewRecorder()
	h.GetDocPage(rec, docsRequest(http.MethodGet, "/workspaces/"+wsID+"/docs/overview", wsID, "overview"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// TestGetDocPage_ReturnsContent verifies GET .../docs/{page} returns the raw
// Markdown content of an existing page.
func TestGetDocPage_ReturnsContent(t *testing.T) {
	workspacePath := t.TempDir()
	docsDir := filepath.Join(workspacePath, "docs", "opensp8c")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("failed to create docs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "overview.md"), []byte("# Overview\n"), 0644); err != nil {
		t.Fatalf("failed to write overview.md: %v", err)
	}

	h, wsID := docsTestHandler(t, workspacePath)

	rec := httptest.NewRecorder()
	h.GetDocPage(rec, docsRequest(http.MethodGet, "/workspaces/"+wsID+"/docs/overview", wsID, "overview"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp docPageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Content != "# Overview\n" {
		t.Fatalf("expected raw markdown content, got %q", resp.Content)
	}
}
