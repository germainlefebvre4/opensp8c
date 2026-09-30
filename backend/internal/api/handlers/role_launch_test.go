package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
	"github.com/glefebvre/opensp8c/internal/workspace"
)

func TestTriggerGenerateUsesDocumenterRole(t *testing.T) {
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
	prefs := preferences.NewService(filepath.Join(tmpDir, "preferences.json"))
	h := NewDocsHandler(ws, session.NewManager(prefs, nil), watcher.NewWatcherService())
	wsID := workspace.StableID(mustAbs(t, workspacePath))

	rec := httptest.NewRecorder()
	h.TriggerGenerate(rec, docsRequest(http.MethodPost, "/workspaces/"+wsID+"/docs/generate", wsID, ""))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("code %d: %s", rec.Code, rec.Body.String())
	}
	got := waitArgsContain(t, argsFile, "--effort")
	for _, want := range []string{"--model\nhaiku\n", "--effort\nlow\n"} {
		if !containsStr(got, want) {
			t.Errorf("documenter args missing %q: %q", want, got)
		}
	}
	waitForCondition(t, 3*time.Second, func() bool { return !h.isRunning(wsID) })
}

func TestTriggerFFUsesFFRoleWithWorkspaceOverride(t *testing.T) {
	tmpDir := t.TempDir()
	argsFile := installArgRecordingClaude(t, tmpDir)
	if err := os.MkdirAll(filepath.Join(tmpDir, "openspec", "changes", "ff-change"), 0755); err != nil {
		t.Fatal(err)
	}
	wsID := workspace.StableID(mustAbs(t, tmpDir))
	ws := NewWorkspaceHandler(&config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: tmpDir}}}, "", nil)
	prefs := preferences.NewService(filepath.Join(tmpDir, "preferences.json"))
	var patch preferences.AgentSettingsPatch
	if err := jsonUnmarshalString(`{"roles":{"ff":{"model":"opus"}}}`, &patch); err != nil {
		t.Fatal(err)
	}
	if err := prefs.PatchWorkspace(wsID, preferences.WorkspaceSettingsPatch{AgentSettings: &patch}); err != nil {
		t.Fatal(err)
	}
	convStore := conversation.NewStore(filepath.Join(tmpDir, "conversations"))
	h := NewFFHandler(ws, session.NewManager(prefs, convStore), convStore, activity.NewStore(filepath.Join(tmpDir, "activity"), nil), watcher.NewWatcherService())

	rec, req := triggerFFRequest(wsID, "ff-change")
	h.TriggerFF(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("code %d: %s", rec.Code, rec.Body.String())
	}
	got := waitArgsContain(t, argsFile, "--effort")
	// Workspace override of the model, preset effort for ff.
	for _, want := range []string{"--model\nopus\n", "--effort\nmedium\n"} {
		if !containsStr(got, want) {
			t.Errorf("ff args missing %q: %q", want, got)
		}
	}
}

func TestRunPromoteFFUsesFFRole(t *testing.T) {
	workspacePath := t.TempDir()
	tmpDir := t.TempDir()
	argsFile := installArgRecordingClaude(t, tmpDir)
	prefs := preferences.NewService(filepath.Join(tmpDir, "preferences.json"))
	wsID := workspace.StableID(mustAbs(t, workspacePath))
	h := &ExploreHandler{
		ws:      NewWorkspaceHandler(&config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: workspacePath}}}, "", nil),
		mgr:     session.NewManager(prefs, nil),
		prefs:   prefs,
		watcher: watcher.NewWatcherService(),
	}
	done := make(chan struct{})
	go func() { defer close(done); h.runPromoteFF(wsID, "ghost1", "my-idea", workspacePath, "some context") }()
	got := waitArgsContain(t, argsFile, "--effort")
	if !containsStr(got, "--model\nsonnet\n") {
		t.Errorf("promote ff args: %q", got)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
}

func containsStr(s, sub string) bool { return strings.Contains(s, sub) }

func jsonUnmarshalString(s string, v any) error { return json.Unmarshal([]byte(s), v) }
