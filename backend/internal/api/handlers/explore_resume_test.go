package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
	"github.com/glefebvre/opensp8c/internal/workspace"
)

func TestCreateGhostRecordPersistsClaudeSessionID(t *testing.T) {
	tmpDir := t.TempDir()
	installArgRecordingClaude(t, tmpDir)
	prefsPath := filepath.Join(tmpDir, "preferences.json")
	prefs := preferences.NewService(prefsPath)
	mgr := session.NewManager(prefs, nil)

	sid, sess, err := mgr.StartAnonymous("ws1", t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Stop()
	if sess.ClaudeSessionID() == "" {
		t.Fatal("anonymous session should carry a claude session id")
	}

	h := &ExploreHandler{prefs: prefs}
	h.createGhostRecord("ws1", sid, sess)

	raw, err := os.ReadFile(prefsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"claudeSessionId": "`+sess.ClaudeSessionID()+`"`) &&
		!strings.Contains(string(raw), `"claudeSessionId":"`+sess.ClaudeSessionID()+`"`) {
		t.Errorf("preferences.json lacks the session id %s: %s", sess.ClaudeSessionID(), raw)
	}
	if rec := prefs.GetExploration(sid, "ws1"); rec == nil || rec.ClaudeSessionId != sess.ClaudeSessionID() {
		t.Errorf("record = %+v", rec)
	}
}

func TestRunPromoteFFDoesNotForwardExplorationSessionID(t *testing.T) {
	workspacePath := t.TempDir()
	tmpDir := t.TempDir()
	argsFile := installArgRecordingClaude(t, tmpDir)
	prefs := frenchDocsPrefs(t, tmpDir)
	const ghostSession = "aaaaaaaa-1111-4111-8111-111111111111"
	wsID := workspace.StableID(mustAbs(t, workspacePath))
	if err := prefs.AddExploration(preferences.ExplorationRecord{ID: "ghost1", WorkspaceID: wsID, Name: "my-idea", ClaudeSessionId: ghostSession}); err != nil {
		t.Fatal(err)
	}
	h := &ExploreHandler{
		ws:      NewWorkspaceHandler(&config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: workspacePath}}}, "", nil),
		mgr:     session.NewManager(prefs, nil),
		prefs:   prefs,
		watcher: watcher.NewWatcherService(),
	}
	done := make(chan struct{})
	go func() { defer close(done); h.runPromoteFF(wsID, "ghost1", "my-idea", workspacePath, "some context") }()
	got := waitArgsContain(t, argsFile, "some context")
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
	for _, forbidden := range []string{ghostSession, "--resume", "--session-id"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("FF subprocess args must not contain %q: %s", forbidden, got)
		}
	}
}
