package session

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/preferences"
)

func withMockClaude(t *testing.T) (tmpDir, argsFile string) {
	t.Helper()
	tmpDir = t.TempDir()
	mockCLIPath := filepath.Join(tmpDir, "mock-claude")
	argsFile = filepath.Join(tmpDir, "args.txt")
	writeMockCLI(t, mockCLIPath, argsFile)
	orig := agents.SupportedAgents[0].CLI
	agents.SupportedAgents[0].CLI = mockCLIPath
	t.Cleanup(func() { agents.SupportedAgents[0].CLI = orig })
	return tmpDir, argsFile
}

func TestManagerStartPassesChatLanguage(t *testing.T) {
	tmpDir, argsFile := withMockClaude(t)
	prefsSvc := preferences.NewService(filepath.Join(tmpDir, "preferences.json"))
	_ = prefsSvc.SetUILocale("fr")
	chat := "auto"
	_ = prefsSvc.SetAgentLanguages(preferences.AgentLanguagesUpdate{Chat: &chat})
	mgr := NewManager(prefsSvc, nil)

	sess, err := mgr.Start("ws1", "my-change", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Stop()

	data := string(waitForFile(t, argsFile))
	if !strings.Contains(data, "converse with the user in French") {
		t.Errorf("named exploration: chat language missing: %s", data)
	}
}

func TestManagerStartAnonymousPassesChatLanguage(t *testing.T) {
	tmpDir, argsFile := withMockClaude(t)
	prefsSvc := preferences.NewService(filepath.Join(tmpDir, "preferences.json"))
	_ = prefsSvc.SetUILocale("fr")
	mgr := NewManager(prefsSvc, nil)

	_, sess, err := mgr.StartAnonymous("ws1", t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Stop()

	data := string(waitForFile(t, argsFile))
	if !strings.Contains(data, "converse with the user in French") {
		t.Errorf("anonymous exploration: chat language missing: %s", data)
	}
}
