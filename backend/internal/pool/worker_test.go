package pool

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
)

// newPipeSubprocess builds a *session.Subprocess backed by in-memory pipes,
// so tests can drive its stdin/stdout without spawning a real process.
// Callers get back the writer side of stdin's pipe closed for them isn't
// needed here; instead they read from stdinR (what the code under test wrote)
// and write to stdoutW (what the code under test will read back).
func newPipeSubprocess() (proc *session.Subprocess, stdinR *io.PipeReader, stdoutW *io.PipeWriter) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	return session.NewTestSubprocess(inW, outR, "claude"), inR, outW
}

// fakeAutoRespondingSubprocess returns a *session.Subprocess that answers
// every line written to its stdin with a single stream-json "result" line,
// as if a real agent CLI had instantly completed each turn. It supports any
// number of turns (apply, then zero or more heals) transparently.
func fakeAutoRespondingSubprocess() *session.Subprocess {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	go func() {
		r := bufio.NewReader(inR)
		for {
			if _, err := r.ReadString('\n'); err != nil {
				_ = outW.Close()
				return
			}
			if _, err := outW.Write([]byte(`{"type":"result"}` + "\n")); err != nil {
				return
			}
		}
	}()
	return session.NewTestSubprocess(inW, outR, "claude")
}

// workingAgentStub returns a startSubprocessFn stub whose agent writes an
// uncommitted file in its worktree on every turn, then answers with a result:
// the way a real agent leaves its work behind without committing it.
func workingAgentStub() func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
	return func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
		inR, inW := io.Pipe()
		outR, outW := io.Pipe()
		go func() {
			r := bufio.NewReader(inR)
			for {
				if _, err := r.ReadString('\n'); err != nil {
					_ = outW.Close()
					return
				}
				_ = os.WriteFile(filepath.Join(workspacePath, "agent-output.txt"), []byte("work"), 0644)
				if _, err := outW.Write([]byte(`{"type":"result"}` + "\n")); err != nil {
					return
				}
			}
		}()
		return session.NewTestSubprocess(inW, outR, "claude"), nil
	}
}

// TestRunTurn_WritesTurnAndReturnsOnResult verifies that runTurn writes the
// expected stream-json user turn to the subprocess's stdin, extracts
// activity from content_block_delta lines into w.Activity as they stream in,
// and returns without error once a "result" line closes the turn.
func TestRunTurn_WritesTurnAndReturnsOnResult(t *testing.T) {
	proc, stdinR, stdoutW := newPipeSubprocess()

	var written []byte
	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		buf := make([]byte, 4096)
		n, _ := stdinR.Read(buf)
		written = buf[:n]
	}()

	go func() {
		_, _ = stdoutW.Write([]byte(`{"type":"content_block_delta","delta":{"text":"working..."}}` + "\n"))
		_, _ = stdoutW.Write([]byte(`{"type":"result","subtype":"success"}` + "\n"))
		_ = stdoutW.Close()
	}()

	m := &Manager{}
	w := &Worker{}
	if err := m.runTurn(w, proc, "/opsx:apply demo-change"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	<-writeDone

	var payload struct {
		Type    string `json:"type"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(written, &payload); err != nil {
		t.Fatalf("failed to parse written turn %q: %v", written, err)
	}
	if payload.Type != "user" || payload.Message.Role != "user" || payload.Message.Content != "/opsx:apply demo-change" {
		t.Errorf("unexpected turn written: %+v", payload)
	}
	if w.Activity != "working..." {
		t.Errorf("expected Activity to be updated from content_block_delta, got %q", w.Activity)
	}
}

// TestRunTurn_RecognizesGeminiTranslatedCompletion verifies that runTurn also
// treats the Gemini-translated "message_complete" event (see
// translateGeminiLine) as turn completion, not just the native "result" type.
func TestRunTurn_RecognizesGeminiTranslatedCompletion(t *testing.T) {
	proc, stdinR, stdoutW := newPipeSubprocess()
	go func() { _, _ = io.Copy(io.Discard, stdinR) }()
	go func() {
		_, _ = stdoutW.Write([]byte(`{"type":"message_complete","result":" "}` + "\n"))
		_ = stdoutW.Close()
	}()

	m := &Manager{}
	w := &Worker{}
	if err := m.runTurn(w, proc, "hello"); err != nil {
		t.Fatalf("expected message_complete to signal turn completion, got error: %v", err)
	}
}

// TestRunTurn_ErrorsWhenProcessEndsBeforeResult verifies that runTurn returns
// an error if the subprocess's stdout ends (EOF) before any completion line
// was seen, rather than silently succeeding.
func TestRunTurn_ErrorsWhenProcessEndsBeforeResult(t *testing.T) {
	proc, stdinR, stdoutW := newPipeSubprocess()
	go func() { _, _ = io.Copy(io.Discard, stdinR) }()
	go func() {
		_, _ = stdoutW.Write([]byte(`{"type":"content_block_delta","delta":{"text":"still going"}}` + "\n"))
		_ = stdoutW.Close()
	}()

	m := &Manager{}
	w := &Worker{}
	if err := m.runTurn(w, proc, "hello"); err == nil {
		t.Fatal("expected an error when the subprocess ends before signaling completion")
	}
}

// TestInvokeAgentHeal_WritesValidationErrorWithoutStartingNewSubprocess
// verifies that invokeAgentHeal writes the validation error's text as a
// follow-up turn on the given subprocess, and never starts a new one itself
// (only runWorker decides whether/when to start a subprocess).
func TestInvokeAgentHeal_WritesValidationErrorWithoutStartingNewSubprocess(t *testing.T) {
	proc, stdinR, stdoutW := newPipeSubprocess()

	origStart := startSubprocessFn
	callCount := 0
	startSubprocessFn = func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
		callCount++
		return nil, errors.New("invokeAgentHeal must never call this")
	}
	defer func() { startSubprocessFn = origStart }()

	var written []byte
	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		buf := make([]byte, 4096)
		n, _ := stdinR.Read(buf)
		written = buf[:n]
	}()
	go func() {
		_, _ = stdoutW.Write([]byte(`{"type":"result"}` + "\n"))
		_ = stdoutW.Close()
	}()

	m := &Manager{}
	w := &Worker{ActiveChange: "demo-change"}
	validationErr := fmt.Errorf("tests failed: boom")
	if err := m.invokeAgentHeal(w, proc, validationErr); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	<-writeDone

	if callCount != 0 {
		t.Errorf("expected invokeAgentHeal never to start a new subprocess, got %d call(s)", callCount)
	}
	if !strings.Contains(string(written), "tests failed: boom") {
		t.Errorf("expected the validation error text to be written as the follow-up turn, got %q", written)
	}
}

// TestExtractActivity covers extractActivity's fail-soft extraction: a valid
// content_block_delta line yields its delta text, an unparsable line falls
// back to itself (truncated), and an empty line yields no activity update.
func TestExtractActivity(t *testing.T) {
	cases := []struct {
		name string
		line string
		want string
	}{
		{"gemini-translated flat content_block_delta", `{"type":"content_block_delta","delta":{"text":"hello"}}`, "hello"},
		// This is the shape the real Claude CLI's own stream-json output
		// actually uses (confirmed against a live `claude` subprocess), unlike
		// the flat shape above which only Gemini's translated output produces.
		{"native claude stream_event content_block_delta (text)", `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"working on it"}}}`, "working on it"},
		{"native claude stream_event content_block_delta (thinking)", `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"pondering"}}}`, "pondering"},
		{"native claude stream_event with no text/thinking yields no activity", `{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"foo\""}}}`, ""},
		{"non-content_block_delta stream_event falls back to raw line", `{"type":"stream_event","event":{"type":"message_start"}}`, `{"type":"stream_event","event":{"type":"message_start"}}`},
		{"invalid json falls back to raw line", `not json at all`, "not json at all"},
		{"empty line yields no activity", "", ""},
		{"whitespace-only line yields no activity", "   ", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := extractActivity([]byte(c.line)); got != c.want {
				t.Errorf("extractActivity(%q) = %q, want %q", c.line, got, c.want)
			}
		})
	}
}

// TestExtractActivity_TruncatesLongRawLines verifies the raw-line fallback
// bounds its output rather than letting a huge line balloon Worker.Activity.
func TestExtractActivity_TruncatesLongRawLines(t *testing.T) {
	long := strings.Repeat("x", maxActivityLen+50)
	got := extractActivity([]byte(long))
	if len(got) != maxActivityLen {
		t.Errorf("expected truncated activity of length %d, got %d", maxActivityLen, len(got))
	}
}

// TestRunTurn_ThrottlesActivityBroadcasts sends many rapid activity updates
// within a turn and verifies notify() (and thus the pool_updated broadcast)
// fires far less often than once per line - once per second at most, not
// once per stdout line.
func TestRunTurn_ThrottlesActivityBroadcasts(t *testing.T) {
	bc := &mockBroadcaster{}
	m := NewManager(bc, nil, nil, nil, nil)
	m.workspaceID = "ws-1"

	proc, stdinR, stdoutW := newPipeSubprocess()
	go func() { _, _ = io.Copy(io.Discard, stdinR) }()
	go func() {
		for i := 0; i < 50; i++ {
			_, _ = stdoutW.Write([]byte(`{"type":"content_block_delta","delta":{"text":"chunk"}}` + "\n"))
		}
		_, _ = stdoutW.Write([]byte(`{"type":"result"}` + "\n"))
		_ = stdoutW.Close()
	}()

	w := &Worker{}
	if err := m.runTurn(w, proc, "/opsx:apply demo"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if events := bc.snapshot(); len(events) != 1 {
		t.Errorf("expected exactly 1 throttled broadcast for 50 rapid activity updates in well under a second, got %d", len(events))
	}
}

// newGoFixtureRepo creates a temp git repo containing a trivial Go module
// (so `go test ./...` passes without any real work) and a change directory
// with the given tasks.md content at its normal nested path
// (openspec/changes/<name>/tasks.md), then commits it all. It returns the
// repo's root path and the change name.
func newGoFixtureRepo(t *testing.T, changeName, tasksMd string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module fixture\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	changeDir := filepath.Join(repoDir, "openspec", "changes", changeName)
	if err := os.MkdirAll(changeDir, 0755); err != nil {
		t.Fatalf("mkdir change dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte(tasksMd), 0644); err != nil {
		t.Fatalf("write tasks.md: %v", err)
	}

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-q")
	// -c commit.gpgsign=false: this is a disposable fixture repo under
	// t.TempDir(), never pushed or part of real project history - only to
	// avoid failing on machines where commit signing needs an interactive
	// pinentry unavailable to the test runner.
	runGit("-c", "user.email=test@test.com", "-c", "user.name=test", "add", "-A")
	runGit("-c", "user.email=test@test.com", "-c", "user.name=test", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "init")

	return repoDir
}

// newWorkerTestManager builds a Manager wired directly (bypassing Start())
// for exercising runWorker synchronously in a test, with startSubprocessFn
// stubbed so no real agent CLI is ever invoked.
func newWorkerTestManager(t *testing.T, repoDir string, cfg AgentPoolConfig, stub func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error)) *Manager {
	t.Helper()
	origStart := startSubprocessFn
	startSubprocessFn = stub
	t.Cleanup(func() { startSubprocessFn = origStart })

	m := NewManager(nil, nil, nil, nil, nil)
	m.workspacePath = repoDir
	// Outside t.TempDir(): a background worker may still provision after the
	// test returns and must not race the temp dir cleanup.
	m.worktreesRoot = filepath.Join(os.Getenv("OPENSP8C_WORKTREES_DIR"), filepath.Base(repoDir))
	m.config = cfg
	return m
}

// TestRunWorker_StartsSubprocessExactlyOnce exercises runWorker end-to-end
// against a real (fixture) git worktree, and verifies that exactly one
// subprocess is started for the change it picks up (task 2.1) - even though
// invokeAgentApply and (potentially) invokeAgentHeal both run against it.
func TestRunWorker_StartsSubprocessExactlyOnce(t *testing.T) {
	repoDir := newGoFixtureRepo(t, "single-start-change", "- [x] done\n")

	callCount := 0
	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 3},
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
			callCount++
			return fakeAutoRespondingSubprocess(), nil
		})

	w := &Worker{ID: 1, ActiveChange: "single-start-change"}
	m.activeWorkers[1] = w
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.runWorker(ctx, w)

	if callCount != 1 {
		t.Errorf("expected exactly one subprocess start for the change taken up, got %d", callCount)
	}
}

// TestRunWorker_TerminatesSubprocessWhenItReturns verifies that the agent
// subprocess is actually closed down (stdin closed, process reaped) by the
// time runWorker returns (task 2.4), using a real child process rather than
// an in-memory fake so process termination is genuinely observable.
func TestRunWorker_TerminatesSubprocessWhenItReturns(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
	repoDir := newGoFixtureRepo(t, "leak-check-change", "- [x] done\n")

	scriptPath := filepath.Join(t.TempDir(), "fake-agent")
	script := "#!/bin/sh\nwhile IFS= read -r line; do\n  printf '{\"type\":\"result\"}\\n'\ndone\ntouch done-marker\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake agent script: %v", err)
	}

	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1},
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
			return session.StartSubprocess(ctx, workspacePath, agents.AgentConfig{ID: "fake", CLI: scriptPath}, extraSystemPrompt, claudeSessionID, resume, sessionLog, customEnv, nativeQuestionMode, languageDirective)
		})

	w := &Worker{ID: 1, ActiveChange: "leak-check-change"}
	m.activeWorkers[1] = w
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.runWorker(ctx, w)

	markerPath := filepath.Join(w.WorktreePath, "done-marker")
	if _, err := os.Stat(markerPath); err != nil {
		t.Errorf("expected the agent subprocess to observe stdin EOF and exit once runWorker returned, but %s is missing: %v", markerPath, err)
	}
}

// TestRunWorker_PausesWithoutFinalizingWhenTasksIncomplete verifies the
// "Finalisation refusée si des tâches restent ouvertes" scenario: validation
// passing does not authorize a full-autonomy merge when the worktree's
// tasks.md still has unchecked items (task 3.2).
func TestRunWorker_PausesWithoutFinalizingWhenTasksIncomplete(t *testing.T) {
	changeName := "partial-tasks-change"
	repoDir := newGoFixtureRepo(t, changeName, "- [x] one\n- [ ] two\n")

	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1},
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
			return fakeAutoRespondingSubprocess(), nil
		})

	w := &Worker{ID: 1, ActiveChange: changeName}
	m.activeWorkers[1] = w
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.runWorker(ctx, w)

	if w.Status != StatusPaused {
		t.Errorf("expected worker status to be %q when tasks.md is incomplete, got %q", StatusPaused, w.Status)
	}
	if !strings.Contains(w.BlockedReason, "tâches restantes incomplètes") {
		t.Errorf("expected BlockedReason to describe incomplete tasks.md, got %q", w.BlockedReason)
	}
	m.mu.Lock()
	paused, ok := m.pausedWorkers[w.ID]
	m.mu.Unlock()
	if !ok || paused.BlockedReason != w.BlockedReason {
		t.Errorf("expected worker to be snapshotted into pausedWorkers with its BlockedReason, got %+v (ok=%v)", paused, ok)
	}

	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/feature/"+changeName)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Errorf("expected the feature branch to still exist (no merge should have happened): %v", err)
	}
}

// TestRunWorker_PausesWithReason_SubprocessStartFailure verifies the first of
// the 4 paused paths (task 1.3): when the agent subprocess itself fails to
// start, the worker pauses with a BlockedReason describing that failure, and
// is snapshotted into pausedWorkers so it remains observable.
func TestRunWorker_PausesWithReason_SubprocessStartFailure(t *testing.T) {
	changeName := "subprocess-start-failure-change"
	repoDir := newGoFixtureRepo(t, changeName, "- [x] done\n")

	startErr := errors.New("boom: no such CLI")
	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1},
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
			return nil, startErr
		})

	w := &Worker{ID: 1, ActiveChange: changeName}
	m.activeWorkers[1] = w
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.runWorker(ctx, w)

	if w.Status != StatusPaused {
		t.Fatalf("expected worker status to be %q, got %q", StatusPaused, w.Status)
	}
	if w.BlockedReason == "" || !strings.Contains(w.BlockedReason, "démarrage du subprocess") {
		t.Errorf("expected BlockedReason to describe the subprocess start failure, got %q", w.BlockedReason)
	}
	m.mu.Lock()
	paused, ok := m.pausedWorkers[w.ID]
	m.mu.Unlock()
	if !ok || paused.BlockedReason != w.BlockedReason {
		t.Errorf("expected worker to be snapshotted into pausedWorkers with its BlockedReason, got %+v (ok=%v)", paused, ok)
	}
}

// TestRunWorker_PausesWithReason_ApplyInvocationFailure verifies the second
// paused path (task 1.3): when the agent subprocess ends before completing
// the initial apply turn, the worker pauses with a BlockedReason describing
// that invocation failure.
func TestRunWorker_PausesWithReason_ApplyInvocationFailure(t *testing.T) {
	changeName := "apply-invocation-failure-change"
	repoDir := newGoFixtureRepo(t, changeName, "- [x] done\n")

	// A subprocess whose stdout closes immediately, without ever writing a
	// completion line, so invokeAgentApply's runTurn returns an error.
	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1},
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
			inR, inW := io.Pipe()
			outR, outW := io.Pipe()
			go func() {
				_, _ = io.Copy(io.Discard, inR)
			}()
			_ = outW.Close()
			return session.NewTestSubprocess(inW, outR, "claude"), nil
		})

	w := &Worker{ID: 1, ActiveChange: changeName}
	m.activeWorkers[1] = w
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.runWorker(ctx, w)

	if w.Status != StatusPaused {
		t.Fatalf("expected worker status to be %q, got %q", StatusPaused, w.Status)
	}
	if w.BlockedReason == "" || !strings.Contains(w.BlockedReason, "invocation de l'agent") {
		t.Errorf("expected BlockedReason to describe the apply invocation failure, got %q", w.BlockedReason)
	}
	m.mu.Lock()
	paused, ok := m.pausedWorkers[w.ID]
	m.mu.Unlock()
	if !ok || paused.BlockedReason != w.BlockedReason {
		t.Errorf("expected worker to be snapshotted into pausedWorkers with its BlockedReason, got %+v (ok=%v)", paused, ok)
	}
}

// TestRunWorker_PausesWithReason_HealExhausted verifies the third paused path
// (task 1.3): when validation keeps failing until MaxAttempts is exhausted,
// the worker pauses with a BlockedReason describing that exhaustion.
func TestRunWorker_PausesWithReason_HealExhausted(t *testing.T) {
	changeName := "heal-exhausted-change"
	// A deliberately broken Go file: `go test ./...` will always fail here,
	// regardless of the (fake, no-op) heal turns the agent produces.
	repoDir := t.TempDir()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module fixture\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n\nfunc main() { this is not valid go }\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	changeDir := filepath.Join(repoDir, "openspec", "changes", changeName)
	if err := os.MkdirAll(changeDir, 0755); err != nil {
		t.Fatalf("mkdir change dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("- [x] done\n"), 0644); err != nil {
		t.Fatalf("write tasks.md: %v", err)
	}
	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-q")
	runGit("-c", "user.email=test@test.com", "-c", "user.name=test", "add", "-A")
	runGit("-c", "user.email=test@test.com", "-c", "user.name=test", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "init")
	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1},
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
			return fakeAutoRespondingSubprocess(), nil
		})

	w := &Worker{ID: 1, ActiveChange: changeName}
	m.activeWorkers[1] = w
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.runWorker(ctx, w)

	if w.Status != StatusPaused {
		t.Fatalf("expected worker status to be %q, got %q", StatusPaused, w.Status)
	}
	if w.BlockedReason == "" || !strings.Contains(w.BlockedReason, "réparation épuisées") {
		t.Errorf("expected BlockedReason to describe heal exhaustion, got %q", w.BlockedReason)
	}
	m.mu.Lock()
	paused, ok := m.pausedWorkers[w.ID]
	m.mu.Unlock()
	if !ok || paused.BlockedReason != w.BlockedReason {
		t.Errorf("expected worker to be snapshotted into pausedWorkers with its BlockedReason, got %+v (ok=%v)", paused, ok)
	}
}

// TestRunWorker_FinalizesWhenTasksComplete is the non-regression counterpart:
// once tasks.md is fully checked and validation passes, full-autonomy mode
// still merges and cleans up as before (task 3.3).
func TestRunWorker_FinalizesWhenTasksComplete(t *testing.T) {
	changeName := "complete-tasks-change"
	repoDir := newGoFixtureRepo(t, changeName, "- [x] one\n- [x] two\n")

	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1},
		workingAgentStub())

	w := &Worker{ID: 1, ActiveChange: changeName}
	m.activeWorkers[1] = w
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.runWorker(ctx, w)

	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/feature/"+changeName)
	cmd.Dir = repoDir
	if err := cmd.Run(); err == nil {
		t.Errorf("expected the feature branch to have been deleted after a successful full-autonomy merge")
	}
}

func TestIsTurnCompleteLine_Antigravity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Native Claude result",
			input:    `{"type":"result","subtype":"success"}`,
			expected: true,
		},
		{
			name:     "Gemini / Antigravity translated message_complete",
			input:    `{"type":"message_complete","result":" "}`,
			expected: true,
		},
		{
			name:     "Raw Antigravity event result",
			input:    `{"event":"result","result":{"status":"SUCCESS"}}`,
			expected: true,
		},
		{
			name:     "Active agent step",
			input:    `{"event":"step_update","step_update":{"state":"ACTIVE","step_type":"agent_response"}}`,
			expected: false,
		},
		{
			name:     "Content block delta",
			input:    `{"type":"content_block_delta","delta":{"text":"hello"}}`,
			expected: false,
		},
		{
			name:     "Plain non-json text",
			input:    `random text log`,
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isTurnCompleteLine([]byte(tc.input))
			if got != tc.expected {
				t.Errorf("expected %v, got %v for %s", tc.expected, got, tc.input)
			}
		})
	}
}

func TestExtractActivity_Antigravity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Translated Antigravity content_block_delta",
			input:    `{"type":"content_block_delta","delta":{"text":"implementing feature..."}}`,
			expected: "implementing feature...",
		},
		{
			name:     "Translated Antigravity tool_use",
			input:    `{"type":"content_block_start","content_block":{"type":"tool_use","id":"tool-1","name":"run_command"}}`,
			expected: "run_command",
		},
		{
			name:     "Raw Antigravity step_update text_delta",
			input:    `{"event":"step_update","step_update":{"state":"ACTIVE","step_type":"agent_response","text_delta":"analyzing codebase..."}}`,
			expected: "analyzing codebase...",
		},
		{
			name:     "Raw Antigravity step_update tool",
			input:    `{"event":"step_update","step_update":{"state":"ACTIVE","step_type":"tool","tool_name":"edit_file"}}`,
			expected: "edit_file",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractActivity([]byte(tc.input))
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestRunTurn_RecognizesAntigravityCompletion(t *testing.T) {
	t.Run("recognizes translated message_complete", func(t *testing.T) {
		proc, stdinR, stdoutW := newPipeSubprocess()
		go func() { _, _ = io.Copy(io.Discard, stdinR) }()
		go func() {
			_, _ = stdoutW.Write([]byte(`{"type":"message_complete","result":" "}` + "\n"))
			_ = stdoutW.Close()
		}()

		m := &Manager{}
		w := &Worker{}
		if err := m.runTurn(w, proc, "execute task"); err != nil {
			t.Fatalf("expected message_complete to complete turn, got: %v", err)
		}
	})

	t.Run("recognizes raw Antigravity result event", func(t *testing.T) {
		proc, stdinR, stdoutW := newPipeSubprocess()
		go func() { _, _ = io.Copy(io.Discard, stdinR) }()
		go func() {
			_, _ = stdoutW.Write([]byte(`{"event":"result","result":{"status":"SUCCESS"}}` + "\n"))
			_ = stdoutW.Close()
		}()

		m := &Manager{}
		w := &Worker{}
		if err := m.runTurn(w, proc, "execute task"); err != nil {
			t.Fatalf("expected event:result to complete turn, got: %v", err)
		}
	})
}

// TestRunWorker_InjectsOnlyConfiguredAgentEnv verifies the subprocess env is
// the global env overlaid with the resolved agent's own agentEnv only.
func TestRunWorker_InjectsOnlyConfiguredAgentEnv(t *testing.T) {
	repoDir := newGoFixtureRepo(t, "agent-env-change", "- [x] done\n")

	prefs := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	if err := prefs.SetAgentEnv(map[string]map[string]string{
		"claude": {"ONLY_CLAUDE": "1"},
		"codex":  {"ONLY_CODEX": "1"},
	}); err != nil {
		t.Fatal(err)
	}
	// Default agent is claude, and unavailable agents fall back to claude.
	sessMgr := session.NewManager(prefs, nil)

	var gotEnv map[string]string
	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 3},
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
			gotEnv = customEnv
			return fakeAutoRespondingSubprocess(), nil
		})
	m.sessionMgr = sessMgr
	m.prefs = prefs

	w := &Worker{ID: 1, ActiveChange: "agent-env-change"}
	m.activeWorkers[1] = w
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.runWorker(ctx, w)

	if gotEnv["ONLY_CLAUDE"] != "1" {
		t.Errorf("expected claude-specific var to be injected, got %v", gotEnv)
	}
	if _, ok := gotEnv["ONLY_CODEX"]; ok {
		t.Errorf("codex-specific var leaked into claude worker: %v", gotEnv)
	}
}
