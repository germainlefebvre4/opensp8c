package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/preferences"
)

// writeMockCLI writes an executable shell script that answers --version
// probes (used by agents.Detect) immediately without touching argsFile, and
// for any other invocation records the args it was called with to argsFile,
// then echoes stdin back to stdout indefinitely (like `cat`) so the
// subprocess pipes stay open for the caller.
func writeMockCLI(t *testing.T, path, argsFile string) {
	t.Helper()
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then\n" +
		"  echo 'mock 1.0.0'\n" +
		"  exit 0\n" +
		"fi\n" +
		"printf '%s\\n' \"$@\" > \"" + argsFile + "\"\n" +
		"cat\n"
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write mock CLI: %v", err)
	}
}

func waitForFile(t *testing.T, path string) []byte {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			return data
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s to be written", path)
	return nil
}

// waitForFileContains polls path until its contents contain substr or times out.
func waitForFileContains(t *testing.T, path, substr string) []byte {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var last []byte
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(path); err == nil {
			last = data
			if strings.Contains(string(data), substr) {
				return data
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s to contain %q; last content: %s", path, substr, last)
	return nil
}

// writeMockCLICapturingStdin is like writeMockCLI but additionally appends
// everything written to its stdin into stdinFile, for tests that need to
// inspect what got written to the subprocess after startup.
func writeMockCLICapturingStdin(t *testing.T, path, argsFile, stdinFile string) {
	t.Helper()
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then\n" +
		"  echo 'mock 1.0.0'\n" +
		"  exit 0\n" +
		"fi\n" +
		"printf '%s\\n' \"$@\" > \"" + argsFile + "\"\n" +
		"cat >> \"" + stdinFile + "\"\n"
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write mock CLI: %v", err)
	}
}

// TestManagerStartUsesSharedExplorationPrompt verifies that a freshly created
// named session receives the shared cadrage/ghost_question system prompt as
// its extraSystemPrompt, instead of the previous empty string.
func TestManagerStartUsesSharedExplorationPrompt(t *testing.T) {
	tmpDir := t.TempDir()
	mockCLIPath := filepath.Join(tmpDir, "mock-claude")
	argsFile := filepath.Join(tmpDir, "args.txt")
	writeMockCLI(t, mockCLIPath, argsFile)

	// Temporarily point the "claude" agent config at our mock CLI.
	origCLI := agents.SupportedAgents[0].CLI
	if agents.SupportedAgents[0].ID != "claude" {
		t.Fatalf("expected SupportedAgents[0] to be claude, got %s", agents.SupportedAgents[0].ID)
	}
	agents.SupportedAgents[0].CLI = mockCLIPath
	defer func() { agents.SupportedAgents[0].CLI = origCLI }()

	prefsSvc := preferences.NewService(filepath.Join(tmpDir, "preferences.json"))
	mgr := NewManager(prefsSvc, nil)

	workDir := t.TempDir()
	sess, err := mgr.Start("ws1", "my-change", workDir)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer sess.Stop()

	data := waitForFile(t, argsFile)
	if !strings.Contains(string(data), "ghost_question") {
		t.Errorf("expected shared exploration prompt (ghost_question marker) in subprocess args, got: %s", data)
	}
}

// TestManagerStartInjectsResumeFollowUpForPendingQuestion verifies that when
// Manager.Start resumes a named session (an existing claudeSessionId is
// persisted) whose most recent prior conversation log ends on an unanswered
// ghost_question marker, it automatically writes a follow-up user message to
// the subprocess stdin asking the agent to rephrase and ask its question
// again, referencing the pending question's text.
func TestManagerStartInjectsResumeFollowUpForPendingQuestion(t *testing.T) {
	tmpDir := t.TempDir()
	mockCLIPath := filepath.Join(tmpDir, "mock-claude")
	argsFile := filepath.Join(tmpDir, "args.txt")
	stdinFile := filepath.Join(tmpDir, "stdin.log")
	writeMockCLICapturingStdin(t, mockCLIPath, argsFile, stdinFile)

	origCLI := agents.SupportedAgents[0].CLI
	if agents.SupportedAgents[0].ID != "claude" {
		t.Fatalf("expected SupportedAgents[0] to be claude, got %s", agents.SupportedAgents[0].ID)
	}
	agents.SupportedAgents[0].CLI = mockCLIPath
	defer func() { agents.SupportedAgents[0].CLI = origCLI }()

	workspaceID, changeName := "ws1", "my-change"
	convStore := conversation.NewStore(filepath.Join(tmpDir, "logs"))

	// Seed a prior run log that ends on an unanswered ghost_question marker.
	priorTs := "2026-01-01T00-00-00Z"
	f, err := convStore.OpenRun(workspaceID, changeName, "chat", priorTs)
	if err != nil {
		t.Fatalf("OpenRun: %v", err)
	}
	priorLog := conversation.NewSessionLog(f)
	priorLog.WriteLine("out", []byte(`{"event":"ghost_question","question":"Quel est le périmètre exact ?"}`))
	priorLog.Close()

	prefsSvc := preferences.NewService(filepath.Join(tmpDir, "preferences.json"))
	if err := prefsSvc.SetSession(workspaceID, changeName, preferences.SessionEntry{
		Agent:           "claude",
		ClaudeSessionId: "11111111-1111-4111-8111-111111111111",
	}); err != nil {
		t.Fatalf("SetSession: %v", err)
	}

	mgr := NewManager(prefsSvc, convStore)

	workDir := t.TempDir()
	sess, err := mgr.Start(workspaceID, changeName, workDir)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer sess.Stop()

	data := waitForFileContains(t, stdinFile, "Quel est le périmètre exact ?")

	var got struct {
		Type    string `json:"type"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal injected stdin message: %v (data: %s)", err, data)
	}
	if got.Type != "user" || got.Message.Role != "user" {
		t.Errorf("unexpected envelope: %+v", got)
	}
	if !strings.Contains(got.Message.Content, "Quel est le périmètre exact ?") {
		t.Errorf("expected follow-up content to reference the pending question, got: %q", got.Message.Content)
	}
}
