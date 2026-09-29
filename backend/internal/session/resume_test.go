package session

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/preferences"
)

// writeMockCLIResumeFails writes a CLI that appends its args (one line per
// invocation) to argsFile, exits 1 immediately when called with --resume (as
// the real CLI does for an unknown session) and otherwise stays alive like cat.
func writeMockCLIResumeFails(t *testing.T, path, argsFile string) {
	t.Helper()
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then echo 'mock 1.0.0'; exit 0; fi\n" +
		"{ printf '%s\\n' \"$@\"; echo ---; } >> \"" + argsFile + "\"\n" +
		"case \"$*\" in *--resume*) echo 'No conversation found'; exit 1;; esac\n" +
		"cat\n"
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write mock CLI: %v", err)
	}
}

func useMockAgent(t *testing.T, agentID, cliPath string) {
	t.Helper()
	for i := range agents.SupportedAgents {
		if agents.SupportedAgents[i].ID == agentID {
			orig := agents.SupportedAgents[i].CLI
			agents.SupportedAgents[i].CLI = cliPath
			t.Cleanup(func() { agents.SupportedAgents[i].CLI = orig })
			return
		}
	}
	t.Fatalf("unknown agent %s", agentID)
}

func fastProbe(t *testing.T) {
	t.Helper()
	orig := resumeProbeWindow
	resumeProbeWindow = 400 * time.Millisecond
	t.Cleanup(func() { resumeProbeWindow = orig })
}

func shProc(t *testing.T, script string) *Subprocess {
	t.Helper()
	cmd := exec.Command("sh", "-c", script)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	p := &Subprocess{cmd: cmd, stdin: stdin, stdout: stdout, agentID: "claude"}
	p.watchExit()
	return p
}

func TestStartWithResumeFallbackKeepsLiveResume(t *testing.T) {
	fastProbe(t)
	var calls []bool
	proc, id, lost, err := startWithResumeFallback(func(id string, resume bool) (*Subprocess, error) {
		calls = append(calls, resume)
		return shProc(t, "cat"), nil
	}, "sess-1", true)
	if err != nil || proc == nil {
		t.Fatalf("unexpected: %v", err)
	}
	if lost || id != "sess-1" || len(calls) != 1 || !calls[0] {
		t.Errorf("lost=%v id=%q calls=%v, want kept resume", lost, id, calls)
	}
}

func TestStartWithResumeFallbackRestartsOnEarlyExit(t *testing.T) {
	fastProbe(t)
	var ids []string
	var resumes []bool
	proc, id, lost, err := startWithResumeFallback(func(id string, resume bool) (*Subprocess, error) {
		ids = append(ids, id)
		resumes = append(resumes, resume)
		if resume {
			return shProc(t, "echo 'No conversation found'; exit 1"), nil
		}
		return shProc(t, "cat"), nil
	}, "sess-1", true)
	if err != nil || proc == nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !lost || id == "sess-1" || id == "" {
		t.Errorf("lost=%v id=%q, want a new id and contextLost", lost, id)
	}
	if len(resumes) != 2 || !resumes[0] || resumes[1] || ids[1] != id {
		t.Errorf("resumes=%v ids=%v", resumes, ids)
	}
}

func TestStartWithResumeFallbackRestartsOnStartError(t *testing.T) {
	n := 0
	_, _, lost, err := startWithResumeFallback(func(id string, resume bool) (*Subprocess, error) {
		n++
		if resume {
			return nil, io.ErrClosedPipe
		}
		return shProc(t, "cat"), nil
	}, "sess-1", false)
	if err != nil || !lost || n != 2 {
		t.Errorf("err=%v lost=%v calls=%d", err, lost, n)
	}
}

func TestExitedProcessOutputStaysReadable(t *testing.T) {
	// A process that writes and exits at once must not lose its output even
	// though its exit is reaped before anyone reads stdout.
	cmd := exec.Command("sh", "-c", "echo hello")
	pr, pw, _ := os.Pipe()
	cmd.Stdout = pw
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pw.Close()
	p := &Subprocess{cmd: cmd, stdout: pr, stdoutFile: pr}
	p.watchExit()
	if !p.exitedWithin(2 * time.Second) {
		t.Fatal("expected exit")
	}
	out, err := io.ReadAll(pr)
	if err != nil || string(out) != "hello\n" {
		t.Errorf("read %q, err %v", out, err)
	}
	_ = p.Wait()
	_ = p.Wait() // idempotent
}

func setupAnonManager(t *testing.T, agentID string) (*Manager, *preferences.Service, string) {
	t.Helper()
	fastProbe(t)
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args.txt")
	cli := filepath.Join(dir, "mock-cli")
	writeMockCLIResumeFails(t, cli, argsFile)
	useMockAgent(t, agentID, cli)
	prefs := preferences.NewService(filepath.Join(dir, "preferences.json"))
	if err := prefs.SetDefaultAgent(agentID); err != nil {
		t.Fatal(err)
	}
	return NewManager(prefs, nil), prefs, argsFile
}

func TestStartAnonymousFirstOpenGeneratesSessionID(t *testing.T) {
	mgr, _, argsFile := setupAnonManager(t, "claude")
	_, sess, err := mgr.StartAnonymous("ws1", t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Stop()
	if sess.ClaudeSessionID() == "" {
		t.Fatal("expected a claude session id")
	}
	args := string(waitForFile(t, argsFile))
	if !strings.Contains(args, "--session-id\n"+sess.ClaudeSessionID()) {
		t.Errorf("expected --session-id in args, got %s", args)
	}
	if sess.TakeContextLost() {
		t.Error("first open must not signal lost context")
	}
}

func TestStartAnonymousResumesExistingRecord(t *testing.T) {
	mgr, prefs, argsFile := setupAnonManager(t, "claude")
	_ = prefs.AddExploration(preferences.ExplorationRecord{ID: "g1", WorkspaceID: "ws1", Name: "n", ClaudeSessionId: "aaaaaaaa-1111-4111-8111-111111111111"})
	// Resuming a live-in-CLI session: the mock only fails on --resume, so use
	// a CLI that stays up for this case.
	stay := filepath.Join(t.TempDir(), "stay")
	writeMockCLI(t, stay, argsFile)
	useMockAgent(t, "claude", stay)

	_, sess, err := mgr.StartAnonymous("ws1", t.TempDir(), "g1")
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Stop()
	if !strings.Contains(string(waitForFile(t, argsFile)), "--resume\naaaaaaaa-1111-4111-8111-111111111111") {
		t.Errorf("expected --resume with the recorded id")
	}
	if sess.TakeContextLost() {
		t.Error("successful resume must not signal lost context")
	}
	if sess.ClaudeSessionID() != "aaaaaaaa-1111-4111-8111-111111111111" {
		t.Errorf("session id = %q", sess.ClaudeSessionID())
	}
}

func TestStartAnonymousResumeFailureSignalsAndPersistsNewID(t *testing.T) {
	mgr, prefs, _ := setupAnonManager(t, "claude")
	old := "aaaaaaaa-1111-4111-8111-111111111111"
	_ = prefs.AddExploration(preferences.ExplorationRecord{ID: "g1", WorkspaceID: "ws1", Name: "n", ClaudeSessionId: old})

	_, sess, err := mgr.StartAnonymous("ws1", t.TempDir(), "g1")
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Stop()
	if !sess.TakeContextLost() {
		t.Error("failed resume must signal lost context")
	}
	if sess.ClaudeSessionID() == old || sess.ClaudeSessionID() == "" {
		t.Errorf("expected a fresh id, got %q", sess.ClaudeSessionID())
	}
	if got := prefs.GetExploration("g1", "ws1").ClaudeSessionId; got != sess.ClaudeSessionID() {
		t.Errorf("persisted id = %q, want %q", got, sess.ClaudeSessionID())
	}
}

func TestStartAnonymousGhostWithoutIDGetsOneAndSignals(t *testing.T) {
	mgr, prefs, argsFile := setupAnonManager(t, "claude")
	_ = prefs.AddExploration(preferences.ExplorationRecord{ID: "g1", WorkspaceID: "ws1", Name: "n"})

	_, sess, err := mgr.StartAnonymous("ws1", t.TempDir(), "g1")
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Stop()
	if !strings.Contains(string(waitForFile(t, argsFile)), "--session-id\n"+sess.ClaudeSessionID()) {
		t.Error("expected --session-id for a ghost without id")
	}
	if !sess.TakeContextLost() {
		t.Error("ghost without id must signal lost context")
	}
	if got := prefs.GetExploration("g1", "ws1").ClaudeSessionId; got != sess.ClaudeSessionID() {
		t.Errorf("persisted id = %q, want %q", got, sess.ClaudeSessionID())
	}
}

func TestStartAnonymousUnsupportedAgentSignalsOnRestart(t *testing.T) {
	mgr, prefs, argsFile := setupAnonManager(t, "antigravity")
	_ = prefs.AddExploration(preferences.ExplorationRecord{ID: "g1", WorkspaceID: "ws1", Name: "n"})

	_, sess, err := mgr.StartAnonymous("ws1", t.TempDir(), "g1")
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Stop()
	_ = argsFile
	if !sess.TakeContextLost() {
		t.Error("agent without resume support must signal lost context on restart")
	}

	_, first, err := mgr.StartAnonymous("ws1", t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Stop()
	if first.TakeContextLost() {
		t.Error("first open must not signal")
	}
}

func TestManagerStartResumeExitingImmediatelyRestartsFresh(t *testing.T) {
	fastProbe(t)
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args.txt")
	cli := filepath.Join(dir, "mock-cli")
	writeMockCLIResumeFails(t, cli, argsFile)
	useMockAgent(t, "claude", cli)
	prefs := preferences.NewService(filepath.Join(dir, "preferences.json"))
	old := "aaaaaaaa-1111-4111-8111-111111111111"
	_ = prefs.SetSession("ws1", "chg", preferences.SessionEntry{Agent: "claude", ClaudeSessionId: old})

	sess, err := NewManager(prefs, nil).Start("ws1", "chg", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Stop()
	if sess.ClaudeSessionID() == old {
		t.Fatal("expected a new session id")
	}
	if got := prefs.GetSession("ws1", "chg").ClaudeSessionId; got != sess.ClaudeSessionID() {
		t.Errorf("persisted id = %q, want %q", got, sess.ClaudeSessionID())
	}
	runs := strings.Split(strings.TrimSpace(string(waitForFileContains(t, argsFile, "--session-id\n"+sess.ClaudeSessionID()))), "---\n")
	if len(runs) != 2 || !strings.Contains(runs[0], "--resume\n"+old) || strings.Contains(runs[1], "--resume") {
		t.Errorf("unexpected invocations: %q", runs)
	}
}

func TestStopIfCurrentIgnoresReplacedSession(t *testing.T) {
	mgr, _, _ := setupAnonManager(t, "claude")
	ws := t.TempDir()
	id, old, err := mgr.StartAnonymous("ws1", ws, "")
	if err != nil {
		t.Fatal(err)
	}
	mgr.StopAnonymous("ws1", id)
	_, fresh, err := mgr.StartAnonymous("ws1", ws, id)
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Stop()

	mgr.StopAnonymousIfCurrent("ws1", id, old) // expiry watcher of the replaced session
	if mgr.GetAnonymous("ws1", id) != fresh {
		t.Fatal("stale stop removed the new session")
	}
	select {
	case <-fresh.Done():
		t.Fatal("stale stop killed the new session")
	case <-time.After(300 * time.Millisecond):
	}

	mgr.StopAnonymousIfCurrent("ws1", id, fresh)
	if mgr.GetAnonymous("ws1", id) != nil {
		t.Error("current session should be stopped")
	}
}

func TestResumeProbeFromEnv(t *testing.T) {
	t.Setenv("OPENSP8C_RESUME_PROBE", "5s")
	if got := resumeProbeFromEnv(3 * time.Second); got != 5*time.Second {
		t.Errorf("got %v, want 5s", got)
	}
	for _, bad := range []string{"abc", "-1s", "0s"} {
		t.Setenv("OPENSP8C_RESUME_PROBE", bad)
		if got := resumeProbeFromEnv(3 * time.Second); got != 3*time.Second {
			t.Errorf("%q: got %v, want default", bad, got)
		}
	}
}
