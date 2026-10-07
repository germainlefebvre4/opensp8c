package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
	"github.com/glefebvre/opensp8c/internal/workspace"
)

// installArgRecordingClaude points the claude agent at a script that answers
// --version and records the args of any other invocation, then exits.
func installArgRecordingClaude(t *testing.T, dir string) (argsFile string) {
	t.Helper()
	argsFile = filepath.Join(dir, "args.txt")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then echo 'mock 1.0.0'; exit 0; fi\n" +
		"printf '%s\\n' \"$@\" >> \"" + argsFile + "\"\n" +
		"mkdir -p docs/opensp8c\n" +
		"echo x > docs/opensp8c/overview.md\n"
	cli := filepath.Join(dir, "mock-claude")
	if err := os.WriteFile(cli, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	orig := agents.SupportedAgents[0].CLI
	agents.SupportedAgents[0].CLI = cli
	t.Cleanup(func() { agents.SupportedAgents[0].CLI = orig })
	return argsFile
}

func waitArgsContain(t *testing.T, file, substr string) string {
	t.Helper()
	var last string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(file); err == nil {
			last = string(b)
			if strings.Contains(last, substr) {
				return last
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("args never contained %q; last: %q", substr, last)
	return ""
}

func frenchDocsPrefs(t *testing.T, dir string) *preferences.Service {
	t.Helper()
	svc := preferences.NewService(filepath.Join(dir, "preferences.json"))
	if err := svc.SetUILocale("fr"); err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestTriggerFFPassesDocumentationLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	argsFile := installArgRecordingClaude(t, tmpDir)
	if err := os.MkdirAll(filepath.Join(tmpDir, "openspec", "changes", "ff-change"), 0755); err != nil {
		t.Fatal(err)
	}
	wsID := workspace.StableID(mustAbs(t, tmpDir))
	ws := NewWorkspaceHandler(&config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: tmpDir}}}, "", nil)
	prefs := frenchDocsPrefs(t, tmpDir)
	convStore := conversation.NewStore(filepath.Join(tmpDir, "conversations"))
	h := NewFFHandler(ws, session.NewManager(prefs, convStore), convStore, activity.NewStore(filepath.Join(tmpDir, "activity"), nil), watcher.NewWatcherService(), nil)

	rec, req := triggerFFRequest(wsID, "ff-change")
	h.TriggerFF(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("code %d: %s", rec.Code, rec.Body.String())
	}
	got := waitArgsContain(t, argsFile, "in French")
	if !strings.Contains(got, "SHALL/MUST") {
		t.Errorf("formalism clause missing: %s", got)
	}
}

func TestTriggerGeneratePassesDocumentationLanguage(t *testing.T) {
	workspacePath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspacePath, "openspec", "specs", "cap"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspacePath, "openspec", "specs", "cap", "spec.md"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	argsFile := installArgRecordingClaude(t, tmpDir)
	ws := NewWorkspaceHandler(&config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: workspacePath}}}, "", nil)
	prefs := frenchDocsPrefs(t, tmpDir)
	h := NewDocsHandler(ws, session.NewManager(prefs, nil), watcher.NewWatcherService())
	wsID := workspace.StableID(mustAbs(t, workspacePath))

	rec := httptest.NewRecorder()
	h.TriggerGenerate(rec, docsRequest(http.MethodPost, "/workspaces/"+wsID+"/docs/generate", wsID, ""))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("code %d: %s", rec.Code, rec.Body.String())
	}
	got := waitArgsContain(t, argsFile, "in French")
	// The formalism prompt is still present and file names are unchanged.
	if !strings.Contains(got, "overview.md") || !strings.Contains(got, "docs/opensp8c/") {
		t.Errorf("formalism prompt lost: %s", got)
	}
	waitForCondition(t, 3*time.Second, func() bool { return !h.isRunning(wsID) })
}

func TestRunPromoteFFPassesDocumentationLanguage(t *testing.T) {
	workspacePath := t.TempDir()
	tmpDir := t.TempDir()
	argsFile := installArgRecordingClaude(t, tmpDir)
	prefs := frenchDocsPrefs(t, tmpDir)
	wsID := workspace.StableID(mustAbs(t, workspacePath))
	h := &ExploreHandler{
		ws:      NewWorkspaceHandler(&config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: workspacePath}}}, "", nil),
		mgr:     session.NewManager(prefs, nil),
		prefs:   prefs,
		watcher: watcher.NewWatcherService(),
	}
	done := make(chan struct{})
	go func() { defer close(done); h.runPromoteFF(wsID, "ghost1", "my-idea", workspacePath, "some context") }()
	waitArgsContain(t, argsFile, "in French")
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
}
