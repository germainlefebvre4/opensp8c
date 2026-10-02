package pool

import (
	"bufio"
	"context"
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
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/session"
)

// pipeAgent returns a startSubprocessFn stub whose in-memory agent runs
// before(worktree) on every turn then answers with resultLine. When hold is
// non-nil the agent waits for it before answering and closes its output
// without answering once released.
func pipeAgent(before func(ws string), resultLine string, hold <-chan struct{}) startFn {
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
				if hold != nil {
					<-hold
					_ = outW.Close()
					return
				}
				if before != nil {
					before(workspacePath)
				}
				if _, err := outW.Write([]byte(resultLine + "\n")); err != nil {
					return
				}
			}
		}()
		return session.NewTestSubprocess(inW, outR, "claude"), nil
	}
}

const okResult = `{"type":"result","subtype":"success"}`

func writeInWorktree(name string) func(string) {
	return func(ws string) { _ = os.WriteFile(filepath.Join(ws, name), []byte("work"), 0644) }
}

func branchExistsIn(t *testing.T, repo, branch string) bool {
	t.Helper()
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Dir = repo
	return cmd.Run() == nil
}

func runOnce(t *testing.T, m *Manager, change string) *Worker {
	t.Helper()
	w := &Worker{ID: 1, WorkspaceID: "ws1", ActiveChange: change}
	m.activeWorkers[1] = w
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.runWorker(ctx, w)
	return w
}

func pausedReason(m *Manager, id int) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.pausedWorkers[id]
	if !ok {
		return "", false
	}
	return p.BlockedReason, true
}

// The agent never commits: its file must still reach the current branch.
func TestFinalize_FullAutonomyKeepsUncommittedWork(t *testing.T) {
	change := "keep-work"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m, store, _ := newJournalManager(t, repo, ModeFullAutonomy, pipeAgent(writeInWorktree("agent-file.txt"), okResult, nil))

	w := runOnce(t, m, change)

	if b, err := os.ReadFile(filepath.Join(repo, "agent-file.txt")); err != nil || string(b) != "work" {
		t.Fatalf("agent work not merged into the current branch: %v %q", err, b)
	}
	if branchExistsIn(t, repo, "feature/"+change) {
		t.Error("branch should be deleted after the merge")
	}
	if _, err := os.Stat(w.WorktreePath); !os.IsNotExist(err) {
		t.Errorf("worktree should be removed: %v", err)
	}
	_, lines := loadSingleRun(t, store, "ws1", change)
	assertEndMarker(t, lines, OutcomeCompleted)
}

func TestFinalize_NoWorkProducedPauses(t *testing.T) {
	change := "no-work"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1}, pipeAgent(nil, okResult, nil))

	w := runOnce(t, m, change)

	reason, ok := pausedReason(m, w.ID)
	if !ok || !strings.Contains(reason, "Aucun travail produit") {
		t.Fatalf("paused=%v reason=%q", ok, reason)
	}
	if !branchExistsIn(t, repo, "feature/"+change) {
		t.Error("branch must be kept")
	}
}

func TestFinalize_CommitFailurePausesAndKeepsWork(t *testing.T) {
	change := "no-identity"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1}, pipeAgent(writeInWorktree("wip.txt"), okResult, nil))
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

	w := runOnce(t, m, change)

	reason, ok := pausedReason(m, w.ID)
	if !ok || !strings.Contains(reason, "Impossible de committer") {
		t.Fatalf("paused=%v reason=%q", ok, reason)
	}
	if _, err := os.Stat(filepath.Join(w.WorktreePath, "wip.txt")); err != nil {
		t.Errorf("uncommitted work must survive: %v", err)
	}
}

func TestFinalize_MergeConflictPausesAndKeepsBranch(t *testing.T) {
	change := "conflicting"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	var once sync.Once
	stub := func(ctx context.Context, ws string, a agents.AgentConfig, e, c string, r bool, l *conversation.SessionLog, env map[string]string, n bool, d string) (*session.Subprocess, error) {
		// The current branch moves on after the worktree was created.
		once.Do(func() {
			_ = os.WriteFile(filepath.Join(repo, "main.go"), []byte("package main\n\n// main side\nfunc main() {}\n"), 0644)
			gitIn(t, repo, "commit", "-q", "-am", "main side")
		})
		return pipeAgent(func(ws string) {
			_ = os.WriteFile(filepath.Join(ws, "main.go"), []byte("package main\n\n// branch side\nfunc main() {}\n"), 0644)
		}, okResult, nil)(ctx, ws, a, e, c, r, l, env, n, d)
	}
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1}, stub)

	w := runOnce(t, m, change)

	reason, ok := pausedReason(m, w.ID)
	// The conflict is now detected by the integration step, before any merge.
	if !ok || !strings.Contains(reason, "Intégration de") {
		t.Fatalf("paused=%v reason=%q", ok, reason)
	}
	if !branchExistsIn(t, repo, "feature/"+change) {
		t.Error("branch must be kept")
	}
	if _, err := os.Stat(w.WorktreePath); err != nil {
		t.Errorf("worktree must be kept: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".git", "MERGE_HEAD")); err == nil {
		t.Error("the failed merge must be aborted")
	}
	if gitIn(t, w.WorktreePath, "status", "--porcelain") != "" {
		t.Error("the work must be committed in the branch")
	}
}

func TestFinalize_MergeInProgressPausesWithoutTouchingIt(t *testing.T) {
	change := "merge-busy"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	mergeHead := filepath.Join(repo, ".git", "MERGE_HEAD")
	stub := pipeAgent(func(ws string) {
		writeInWorktree("x.txt")(ws)
		_ = os.WriteFile(mergeHead, []byte(gitIn(t, repo, "rev-parse", "HEAD")+"\n"), 0644)
	}, okResult, nil)
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1}, stub)

	w := runOnce(t, m, change)

	reason, ok := pausedReason(m, w.ID)
	if !ok || !strings.Contains(reason, "merge est déjà en cours") {
		t.Fatalf("paused=%v reason=%q", ok, reason)
	}
	if _, err := os.Stat(mergeHead); err != nil {
		t.Errorf("the user's merge must be left alone: %v", err)
	}
	if !branchExistsIn(t, repo, "feature/"+change) {
		t.Error("branch must be kept")
	}
}

func TestFinalize_ConcurrentMergesAreSerialized(t *testing.T) {
	repo := newGoFixtureRepo(t, "change-one", "- [x] done\n")
	for _, name := range []string{"change-two", "change-three"} {
		writeFile(t, filepath.Join(repo, "openspec", "changes", name, "tasks.md"), "- [x] done\n")
	}
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "more changes")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 3, DelegationMode: ModeFullAutonomy, MaxAttempts: 1},
		pipeAgent(func(ws string) { writeInWorktree(filepath.Base(ws) + ".txt")(ws) }, okResult, nil))

	changes := []string{"change-one", "change-two", "change-three"}
	workers := make([]*Worker, len(changes))
	var wg sync.WaitGroup
	for i, c := range changes {
		workers[i] = &Worker{ID: i + 1, WorkspaceID: "ws1", ActiveChange: c}
		m.activeWorkers[i+1] = workers[i]
	}
	for i := range changes {
		wg.Add(1)
		go func(w *Worker) {
			defer wg.Done()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			m.runWorker(ctx, w)
		}(workers[i])
	}
	wg.Wait()

	for _, c := range changes {
		if _, err := os.Stat(filepath.Join(repo, "wt-"+c+".txt")); err != nil {
			t.Errorf("%s not merged: %v", c, err)
		}
		if branchExistsIn(t, repo, "feature/"+c) {
			t.Errorf("%s branch should be deleted", c)
		}
	}
	if len(m.pausedWorkers) != 0 {
		t.Errorf("no worker should pause: %+v", m.pausedWorkers)
	}
}

func TestFinalize_HITLReviewExcludesChangeFromDispatch(t *testing.T) {
	change := "review-me"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m, store, bc := newJournalManager(t, repo, ModeHITLReview, pipeAgent(writeInWorktree("r.txt"), okResult, nil))

	w := runOnce(t, m, change)

	_, lines := loadSingleRun(t, store, "ws1", change)
	assertEndMarker(t, lines, OutcomeAwaitingReview)
	if gitIn(t, w.WorktreePath, "status", "--porcelain") != "" {
		t.Error("the work must be committed before review")
	}
	if gitIn(t, repo, "rev-list", "--count", "HEAD..feature/"+change) != "1" {
		t.Error("expected one commit on the feature branch")
	}
	if has, err := NewWorktreeController(repo, "ws1", m.worktreesRoot).HasReview(change); err != nil || !has {
		t.Fatalf("review marker should be set: %v, %v", has, err)
	}
	if got := reviewStatus(t, repo, change); got != "to-review" {
		t.Errorf("status = %s, want to-review", got)
	}
	found := false
	for _, c := range bc.snapshot() {
		if c.ev.Type == "change_updated" && c.ev.Name == change && c.workspaceID == "ws1" {
			found = true
		}
	}
	if !found {
		t.Error("change_updated must be broadcast after the marker is set")
	}
}

func reviewStatus(t *testing.T, repo, change string) string {
	t.Helper()
	changes, err := openspec.ListChanges(repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range changes {
		if c.Name == change {
			return c.KanbanStatus
		}
	}
	t.Fatalf("change %s not listed", change)
	return ""
}

func TestFinalize_HITLMarkerFailurePausesWorker(t *testing.T) {
	change := "review-fail"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m, store, _ := newJournalManager(t, repo, ModeHITLReview, pipeAgent(func(ws string) {
		writeInWorktree("r.txt")(ws)
		// A stale lock makes every git config write fail.
		_ = os.WriteFile(filepath.Join(repo, ".git", "config.lock"), nil, 0644)
	}, okResult, nil))

	runOnce(t, m, change)

	reason, ok := pausedReason(m, 1)
	if !ok || !strings.Contains(reason, "état de revue") {
		t.Fatalf("worker should pause on marker failure, got %q (paused=%v)", reason, ok)
	}
	_, lines := loadSingleRun(t, store, "ws1", change)
	assertEndMarker(t, lines, OutcomePaused)
}

func TestTick_SkipsChangesAwaitingReviewAcrossRestart(t *testing.T) {
	repo := newGoFixtureRepo(t, "change-a", "- [ ] t\n")
	writeChangeForPoolTest(t, repo+"/openspec/changes", "change-a", boolPtr(true), intPtr(1))
	writeChangeForPoolTest(t, repo+"/openspec/changes", "change-b", boolPtr(true), intPtr(2))
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "-c", "user.email=t@t", "-c", "user.name=t", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "changes")
	gitIn(t, repo, "branch", "feature/change-a")
	gitIn(t, repo, "config", openspec.ReviewKey("change-a"), "2026-01-01T00:00:00Z")
	block := make(chan struct{})
	defer close(block)
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 2, MaxAttempts: 1},
		func(ctx context.Context, ws string, a agents.AgentConfig, e, c string, r bool, l *conversation.SessionLog, env map[string]string, n bool, d string) (*session.Subprocess, error) {
			<-block
			return nil, context.Canceled
		})
	m.workspaceID = "ws"
	m.isRunning = true

	assertOnlyB := func() {
		t.Helper()
		m.mu.Lock()
		defer m.mu.Unlock()
		for _, w := range m.activeWorkers {
			if w.ActiveChange == "change-a" {
				t.Fatalf("change awaiting review was redistributed")
			}
		}
		if len(m.activeWorkers) != 1 {
			t.Fatalf("expected only change-b to be dispatched, got %d workers", len(m.activeWorkers))
		}
	}
	m.tick()
	m.tick()
	assertOnlyB()

	m.Stop()
	if err := m.Start(AgentPoolConfig{Size: 2, MaxAttempts: 1}, "ws", "ws", repo); err != nil {
		t.Fatal(err)
	}
	defer m.Stop()
	m.tick()
	m.tick()
	assertOnlyB()
	if got := reviewStatus(t, repo, "change-a"); got != "to-review" {
		t.Errorf("status after restart = %s", got)
	}
}

func TestRunWorker_UncommittedChangePausesWithoutAgent(t *testing.T) {
	change := "not-committed"
	repo := newGoFixtureRepo(t, "other", "- [x] done\n")
	writeFile(t, filepath.Join(repo, "openspec", "changes", change, "tasks.md"), "- [ ] todo\n")
	calls := 0
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1},
		func(ctx context.Context, ws string, a agents.AgentConfig, e, c string, r bool, l *conversation.SessionLog, env map[string]string, n bool, d string) (*session.Subprocess, error) {
			calls++
			return nil, context.Canceled
		})

	w := runOnce(t, m, change)

	if calls != 0 {
		t.Fatalf("the agent must not start, got %d call(s)", calls)
	}
	reason, ok := pausedReason(m, w.ID)
	if !ok || !strings.Contains(reason, "doit être committé dans le dépôt avant d'être lancé") {
		t.Fatalf("paused=%v reason=%q", ok, reason)
	}
}

func TestRunWorker_CompletionCheckIsStrict(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(tasks string)
		wantReason string // empty: the worker finalizes
	}{
		{"absent", func(p string) { _ = os.Remove(p) }, "absente ou vide"},
		{"empty", func(p string) { _ = os.WriteFile(p, []byte("no tasks here\n"), 0644) }, "absente ou vide"},
		{"partial", func(p string) { _ = os.WriteFile(p, []byte("- [x] a\n- [x] b\n- [ ] c\n"), 0644) }, "(2/3)"},
		{"complete", func(p string) { _ = os.WriteFile(p, []byte("- [x] a\n- [x] b\n- [x] c\n"), 0644) }, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			change := "strict-" + tc.name
			repo := newGoFixtureRepo(t, change, "- [ ] a\n")
			m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1},
				pipeAgent(func(ws string) {
					tc.mutate(filepath.Join(ws, "openspec", "changes", change, "tasks.md"))
				}, okResult, nil))

			w := runOnce(t, m, change)

			reason, paused := pausedReason(m, w.ID)
			if tc.wantReason == "" {
				if paused {
					t.Fatalf("should finalize, paused with %q", reason)
				}
				return
			}
			if !paused || !strings.Contains(reason, tc.wantReason) {
				t.Fatalf("paused=%v reason=%q, want %q", paused, reason, tc.wantReason)
			}
		})
	}
}

func TestClassifyTurnLine(t *testing.T) {
	cases := []struct {
		line   string
		kind   turnKind
		reason string
	}{
		{`{"type":"result","is_error":true,"subtype":"error_max_turns"}`, turnError, "error_max_turns"},
		{`{"type":"result","subtype":"error_during_execution"}`, turnError, "error_during_execution"},
		{`{"type":"result","is_error":true,"result":"Invalid API key"}`, turnError, "Invalid API key"},
		{`{"type":"result","subtype":"success"}`, turnOK, ""},
		{`{"type":"message_complete"}`, turnOK, ""},
		{`{"event":"result"}`, turnOK, ""},
		{`{"type":"assistant"}`, turnNone, ""},
		{`not json`, turnNone, ""},
	}
	for _, tc := range cases {
		kind, reason := classifyTurnLine([]byte(tc.line))
		if kind != tc.kind || (tc.reason != "" && !strings.Contains(reason, tc.reason)) {
			t.Errorf("%s => %v %q, want %v %q", tc.line, kind, reason, tc.kind, tc.reason)
		}
	}
}

func TestRunWorker_ErrorResultPausesWithoutValidation(t *testing.T) {
	change := "error-result"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1},
		pipeAgent(writeInWorktree("x.txt"), `{"type":"result","is_error":true,"subtype":"error_max_turns"}`, nil))

	w := runOnce(t, m, change)

	reason, ok := pausedReason(m, w.ID)
	if !ok || !strings.Contains(reason, "error_max_turns") {
		t.Fatalf("paused=%v reason=%q", ok, reason)
	}
	if w.Status == StatusTesting || w.Status == StatusHealing {
		t.Errorf("validation must not run, status = %s", w.Status)
	}
	if !branchExistsIn(t, repo, "feature/"+change) {
		t.Error("nothing must be finalized")
	}
}

// Stopping the pool during an agent turn leaves no ghost pause, and the old
// goroutine never removes a newer worker that reused its ID.
func TestStop_NoGhostPauseAndOldWorkerKeepsNewerEntry(t *testing.T) {
	change := "stop-midturn"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	release := make(chan struct{})
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1},
		pipeAgent(nil, okResult, release))
	m.workspaceID = "ws1"
	m.isRunning = true

	ctx, cancel := context.WithCancel(context.Background())
	w := &Worker{ID: 1, WorkspaceID: "ws1", ActiveChange: change, CancelFunc: cancel}
	m.activeWorkers[1] = w
	m.workers.Add(1)
	go func() {
		defer m.workers.Done()
		m.runWorker(ctx, w)
	}()

	deadline := time.Now().Add(10 * time.Second)
	for {
		m.mu.Lock()
		st := w.Status
		m.mu.Unlock()
		if st == StatusWorking {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("worker never started its turn")
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)

	m.Stop()
	newer := &Worker{ID: 1, ActiveChange: "newer"}
	m.mu.Lock()
	m.activeWorkers[1] = newer
	m.mu.Unlock()
	close(release)
	m.waitWorkers()

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activeWorkers[1] != newer {
		t.Error("the old worker removed the newer worker's entry")
	}
	if len(m.pausedWorkers) != 0 {
		t.Errorf("ghost pause after Stop: %+v", m.pausedWorkers)
	}
}
