package pool

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/session"
)

type journalLine struct {
	Dir  string          `json:"dir"`
	Data json.RawMessage `json:"data"`
}

func (l journalLine) marker() (typ, outcome, reason string) {
	var d struct {
		Type    string `json:"type"`
		Outcome string `json:"outcome"`
		Reason  string `json:"reason"`
	}
	_ = json.Unmarshal(l.Data, &d)
	return d.Type, d.Outcome, d.Reason
}

// loadSingleRun returns the parsed lines of the only "pool" run of change.
func loadSingleRun(t *testing.T, store *conversation.Store, ws, change string) (string, []journalLine) {
	t.Helper()
	runs, err := store.List(ws, change, "pool")
	if err != nil || len(runs) != 1 {
		t.Fatalf("expected exactly one pool run for %s, got %v (err=%v)", change, runs, err)
	}
	raw, err := store.Load(ws, change, "pool", runs[0].Ts)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	var lines []journalLine
	for _, r := range raw {
		var l journalLine
		if err := json.Unmarshal(r, &l); err != nil {
			t.Fatalf("bad journal line %q: %v", r, err)
		}
		lines = append(lines, l)
	}
	return runs[0].Ts, lines
}

func assertEndMarker(t *testing.T, lines []journalLine, wantOutcome string) (reason string) {
	t.Helper()
	if len(lines) < 2 {
		t.Fatalf("expected at least start and end markers, got %d lines", len(lines))
	}
	typ, _, _ := lines[0].marker()
	if lines[0].Dir != "meta" || typ != "pool_run_start" {
		t.Errorf("first line should be the start marker, got %s %s", lines[0].Dir, lines[0].Data)
	}
	last := lines[len(lines)-1]
	typ, outcome, reason := last.marker()
	if last.Dir != "meta" || typ != "pool_run_end" || outcome != wantOutcome {
		t.Errorf("expected end marker with outcome %q, got dir=%s %s", wantOutcome, last.Dir, last.Data)
	}
	return reason
}

type startFn = func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error)

func autoStub(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
	return fakeAutoRespondingSubprocess(), nil
}

func newJournalManager(t *testing.T, repoDir string, mode DelegationMode, stub startFn) (*Manager, *conversation.Store, *mockBroadcaster) {
	t.Helper()
	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, DelegationMode: mode, MaxAttempts: 1}, stub)
	store := conversation.NewStore(t.TempDir())
	bc := &mockBroadcaster{}
	m.convStore = store
	m.broadcaster = bc
	m.workspaceID = "ws1"
	return m, store, bc
}

func runJournaled(t *testing.T, m *Manager, ctx context.Context, change string) *Worker {
	t.Helper()
	w := &Worker{ID: 1, WorkspaceID: "ws1", ActiveChange: change, DelegationMode: m.config.DelegationMode}
	m.activeWorkers[1] = w
	m.runWorker(ctx, w)
	return w
}

func TestRunWorker_Journal_AwaitingReview(t *testing.T) {
	change := "journal-review"
	repoDir := newGoFixtureRepo(t, change, "- [x] done\n")
	m, store, bc := newJournalManager(t, repoDir, ModeHITLReview, autoStub)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := runJournaled(t, m, ctx, change)

	ts, lines := loadSingleRun(t, store, "ws1", change)
	if w.RunTS != ts {
		t.Errorf("worker RunTS %q should match the run ts %q", w.RunTS, ts)
	}
	assertEndMarker(t, lines, OutcomeAwaitingReview)
	var in, out int
	for _, l := range lines {
		switch l.Dir {
		case "in":
			in++
		case "out":
			out++
		}
	}
	if in != 1 || out != 1 {
		t.Errorf("expected 1 in and 1 out line, got in=%d out=%d", in, out)
	}
	if !strings.Contains(string(lines[0].Data), `"worker_id":1`) || !strings.Contains(string(lines[0].Data), change) {
		t.Errorf("start marker should carry worker and change: %s", lines[0].Data)
	}

	var got int
	for _, e := range bc.snapshot() {
		if e.ev.Type == "pool_run_appended" && e.ev.Name == change && e.workspaceID == "ws1" {
			got++
		}
	}
	if got < 2 {
		t.Errorf("expected at least the first and the final pool_run_appended, got %d", got)
	}
}

func TestRunWorker_Journal_Completed(t *testing.T) {
	change := "journal-completed"
	repoDir := newGoFixtureRepo(t, change, "- [x] done\n")
	m, store, _ := newJournalManager(t, repoDir, ModeFullAutonomy, autoStub)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runJournaled(t, m, ctx, change)

	_, lines := loadSingleRun(t, store, "ws1", change)
	assertEndMarker(t, lines, OutcomeCompleted)
}

func TestRunWorker_Journal_PausedOnSubprocessStartFailure(t *testing.T) {
	change := "journal-start-failure"
	repoDir := newGoFixtureRepo(t, change, "- [x] done\n")
	m, store, _ := newJournalManager(t, repoDir, ModeHITLReview,
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
			return nil, errors.New("boom: no such CLI")
		})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := runJournaled(t, m, ctx, change)

	_, lines := loadSingleRun(t, store, "ws1", change)
	reason := assertEndMarker(t, lines, OutcomePaused)
	if !strings.Contains(reason, "boom") || reason != w.BlockedReason {
		t.Errorf("end marker reason %q should be the worker's blocked reason %q", reason, w.BlockedReason)
	}
}

func TestRunWorker_Journal_PausedIncompleteTasks(t *testing.T) {
	change := "journal-incomplete"
	repoDir := newGoFixtureRepo(t, change, "- [x] a\n- [ ] b\n")
	m, store, _ := newJournalManager(t, repoDir, ModeFullAutonomy, autoStub)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runJournaled(t, m, ctx, change)

	_, lines := loadSingleRun(t, store, "ws1", change)
	if reason := assertEndMarker(t, lines, OutcomePaused); !strings.Contains(reason, "tâches restantes") {
		t.Errorf("unexpected reason %q", reason)
	}
}

func TestRunWorker_Journal_StoppedByPoolStop(t *testing.T) {
	change := "journal-stopped"
	repoDir := newGoFixtureRepo(t, change, "- [x] done\n")
	started := make(chan struct{})
	m, store, _ := newJournalManager(t, repoDir, ModeHITLReview,
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
			proc, stdinR, stdoutW := newPipeSubprocess()
			go func() { _, _ = io.Copy(io.Discard, stdinR) }()
			go func() {
				<-ctx.Done() // the agent dies when the worker context is cancelled
				_ = stdoutW.Close()
			}()
			close(started)
			return proc, nil
		})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-started
		cancel()
	}()
	runJournaled(t, m, ctx, change)

	_, lines := loadSingleRun(t, store, "ws1", change)
	assertEndMarker(t, lines, OutcomeStopped)
}

func TestRunWorker_Journal_HealTurnsShareTheRun(t *testing.T) {
	change := "journal-heal"
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	repoDir := t.TempDir()
	write := func(p, c string) {
		full := filepath.Join(repoDir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module fixture\n\ngo 1.21\n")
	write("main.go", "package main\n\nfunc main() { this is not valid go }\n")
	write("openspec/changes/"+change+"/tasks.md", "- [x] done\n")
	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-q")
	runGit("-c", "user.email=t@t", "-c", "user.name=t", "add", "-A")
	runGit("-c", "user.email=t@t", "-c", "user.name=t", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "init")
	t.Cleanup(func() {
		home, _ := os.UserHomeDir()
		_ = os.RemoveAll(filepath.Join(home, ".opensp8c", "worktrees", "wt-"+change))
	})

	m, store, _ := newJournalManager(t, repoDir, ModeHITLReview, autoStub)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runJournaled(t, m, ctx, change)

	_, lines := loadSingleRun(t, store, "ws1", change)
	if reason := assertEndMarker(t, lines, OutcomePaused); !strings.Contains(reason, "réparation épuisées") {
		t.Errorf("unexpected reason %q", reason)
	}
	var in int
	for _, l := range lines {
		if l.Dir == "in" {
			in++
		}
	}
	if in != 2 {
		t.Errorf("expected the apply turn and one heal turn in the same run, got %d in-lines", in)
	}
}

// A journal that cannot be written must never interrupt the worker.
func TestRunTurn_JournalWriteFailureDoesNotInterrupt(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "run*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	sess := conversation.NewSessionLog(f)
	_ = sess.Close() // every further write now fails

	proc, stdinR, stdoutW := newPipeSubprocess()
	go func() { _, _ = io.Copy(io.Discard, stdinR) }()
	go func() {
		_, _ = stdoutW.Write([]byte(`{"type":"content_block_delta","delta":{"text":"hi"}}` + "\n"))
		_, _ = stdoutW.Write([]byte(`{"type":"result"}` + "\n"))
		_ = stdoutW.Close()
	}()

	bc := &mockBroadcaster{}
	m := &Manager{broadcaster: bc}
	w := &Worker{WorkspaceID: "ws1", ActiveChange: "c", runLog: &poolRunLog{sess: sess}}
	if err := m.runTurn(w, proc, "go"); err != nil {
		t.Fatalf("runTurn must succeed despite journal failures, got %v", err)
	}
	if !w.runLog.failed {
		t.Error("expected the failure to be recorded")
	}
	for _, e := range bc.snapshot() {
		if e.ev.Type == "pool_run_appended" {
			t.Error("no run event expected when nothing could be written")
		}
	}
}

func TestManager_StatusIsConsistentUnderConcurrentUpdates(t *testing.T) {
	m := NewManager(nil, nil, nil, nil, nil)
	m.workspaceID = "ws1"
	m.isRunning = true
	w := &Worker{ID: 1, WorkspaceID: "ws1", ActiveChange: "c"}
	m.activeWorkers[1] = w

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			m.setActivity(w, "activity")
			m.setStatus(w, StatusTesting)
			m.setRun(w, "ts", nil)
			m.setStatus(w, StatusHealing)
		}
	}()
	for i := 0; i < 500; i++ {
		_, _, workers := m.Status("ws1")
		if len(workers) != 1 {
			t.Fatalf("expected one worker, got %d", len(workers))
		}
	}
	close(stop)
	wg.Wait()
}

type fakeTimer struct{ stopped bool }

func (f *fakeTimer) Stop() bool { f.stopped = true; return true }

func TestNoteRunAppended_LimitsToOnePerSecondWithTrailingEvent(t *testing.T) {
	bc := &mockBroadcaster{}
	m := NewManager(bc, nil, nil, nil, nil)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var pending []func()
	m.clock = func() time.Time { return now }
	m.afterFunc = func(d time.Duration, f func()) stopper {
		pending = append(pending, f)
		return &fakeTimer{}
	}
	w := &Worker{WorkspaceID: "ws1", ActiveChange: "c1"}

	count := func() int {
		n := 0
		for _, e := range bc.snapshot() {
			if e.ev.Type == "pool_run_appended" && e.ev.Name == "c1" {
				n++
			}
		}
		return n
	}

	for i := 0; i < 200; i++ {
		m.noteRunAppended(w)
	}
	if got := count(); got != 1 {
		t.Fatalf("expected a single event during the burst, got %d", got)
	}
	if len(pending) != 1 {
		t.Fatalf("expected exactly one trailing event scheduled, got %d", len(pending))
	}
	now = now.Add(time.Second)
	pending[0]()
	if got := count(); got != 2 {
		t.Fatalf("expected a catch-up event after the burst, got %d", got)
	}

	// End of run: immediate event, pending timer cancelled.
	m.noteRunAppended(w)
	m.noteRunAppended(w)
	before := count()
	m.finishRunEvents(w)
	if count() != before+1 {
		t.Errorf("finishRunEvents should emit immediately")
	}
}
