package pool

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// startCancelableWorker runs a worker for change in the background with a
// done channel, like startWorker does.
func startCancelableWorker(m *Manager, id int, change string) (*Worker, context.CancelFunc, chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	w := &Worker{ID: id, WorkspaceID: "ws1", ActiveChange: change, CancelFunc: cancel, done: make(chan struct{})}
	m.mu.Lock()
	m.activeWorkers[id] = w
	m.mu.Unlock()
	finished := make(chan struct{})
	m.workers.Add(1)
	go func() {
		defer m.workers.Done()
		defer close(finished)
		m.runWorker(ctx, w)
	}()
	return w, cancel, finished
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// branchCommitted reports that feature/<change> carries a commit that HEAD lacks
// (the worker committed and is about to merge).
func branchCommitted(t *testing.T, repo, change string) bool {
	t.Helper()
	cmd := exec.Command("git", "rev-list", "--count", "HEAD..feature/"+change)
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return n > 0
}

func waitClosed(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(30 * time.Second):
		t.Fatal("worker did not finish")
	}
}

// 1.1: cancelled after a successful validation, before the commit.
func TestCancel_BeforeCommitDoesNotCommitOrMerge(t *testing.T) {
	change := "cancel-before-commit"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m, store, _ := newJournalManager(t, repo, ModeFullAutonomy, pipeAgent(writeInWorktree("wip.txt"), okResult, nil))
	headBefore := gitIn(t, repo, "rev-parse", "HEAD")

	ctx, cancel := context.WithCancel(context.Background())
	beforeCommitHook = cancel
	t.Cleanup(func() { beforeCommitHook = nil })

	w := runJournaled(t, m, ctx, change)

	if gitIn(t, repo, "rev-parse", "HEAD") != headBefore {
		t.Error("the target branch must not move")
	}
	if gitIn(t, repo, "rev-parse", "feature/"+change) != headBefore {
		t.Error("no commit must be added to the change branch")
	}
	if _, err := os.Stat(filepath.Join(w.WorktreePath, "wip.txt")); err != nil {
		t.Errorf("worktree must be left as is: %v", err)
	}
	if _, ok := pausedReason(m, w.ID); ok {
		t.Error("no ghost pause expected")
	}
	_, lines := loadSingleRun(t, store, "ws1", change)
	assertEndMarker(t, lines, OutcomeStopped)
}

// 1.2: cancelled while waiting for mergeMu.
func TestCancel_WhileWaitingForMergeLockDoesNotMerge(t *testing.T) {
	change := "cancel-wait-lock"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1}, pipeAgent(writeInWorktree("a.txt"), okResult, nil))
	headBefore := gitIn(t, repo, "rev-parse", "HEAD")

	m.mergeMu.Lock()
	w, cancel, finished := startCancelableWorker(m, 1, change)
	waitFor(t, "worker to commit and wait for the merge lock", func() bool { return branchCommitted(t, repo, change) })
	cancel()
	m.mergeMu.Unlock()
	waitClosed(t, finished)

	if gitIn(t, repo, "rev-parse", "HEAD") != headBefore {
		t.Error("the target branch must not move")
	}
	if !branchExistsIn(t, repo, "feature/"+change) {
		t.Error("branch must be kept")
	}
	if _, err := os.Stat(w.WorktreePath); err != nil {
		t.Errorf("worktree must be kept: %v", err)
	}
	if !m.mergeMu.TryLock() {
		t.Fatal("the merge lock must be released")
	}
	m.mergeMu.Unlock()
	if w.result.Outcome != OutcomeStopped || w.result.Merged {
		t.Errorf("result = %+v", w.result)
	}
}

// 1.3: Stop then Start while an old worker still waits for the lock.
func TestStop_StartQuicklyOldWorkerDoesNotMerge(t *testing.T) {
	change := "stop-start"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1}, pipeAgent(writeInWorktree("a.txt"), okResult, nil))
	m.workspaceID = "ws1"
	m.isRunning = true
	headBefore := gitIn(t, repo, "rev-parse", "HEAD")

	m.mergeMu.Lock()
	_, _, oldFinished := startCancelableWorker(m, 1, change)
	waitFor(t, "old worker waiting for the lock", func() bool { return branchCommitted(t, repo, change) })

	m.Stop()
	// A new worker takes over the same change (same branch and worktree).
	newW := &Worker{ID: 1, WorkspaceID: "ws1", ActiveChange: change, Status: StatusWorking}
	m.mu.Lock()
	m.activeWorkers[1] = newW
	m.mu.Unlock()

	m.mergeMu.Unlock()
	waitClosed(t, oldFinished)

	if gitIn(t, repo, "rev-parse", "HEAD") != headBefore {
		t.Error("the old worker must not merge")
	}
	if !branchExistsIn(t, repo, "feature/"+change) {
		t.Error("the new worker's branch must still exist")
	}
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	if _, err := os.Stat(wt.worktreePath(change)); err != nil {
		t.Errorf("the new worker's worktree must still exist: %v", err)
	}
	m.mu.Lock()
	same := m.activeWorkers[1] == newW
	m.mu.Unlock()
	if !same {
		t.Error("the newer worker's entry must be kept")
	}
}

// 1.4: cancellation during a merge that already started lets it finish cleanly.
func TestCancel_DuringStartedMergeCompletes(t *testing.T) {
	change := "cancel-in-merge"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	hook := filepath.Join(repo, ".git", "hooks", "pre-merge-commit")
	started := filepath.Join(t.TempDir(), "merge-started")
	script := "#!/bin/sh\ntouch " + started + "\nsleep 1\n"
	if err := os.WriteFile(hook, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	m, store, _ := newJournalManager(t, repo, ModeFullAutonomy, pipeAgent(writeInWorktree("a.txt"), okResult, nil))

	w := &Worker{ID: 1, WorkspaceID: "ws1", ActiveChange: change, DelegationMode: ModeFullAutonomy, done: make(chan struct{})}
	m.activeWorkers[1] = w
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		m.runWorker(ctx, w)
	}()
	waitFor(t, "merge to start", func() bool { _, err := os.Stat(started); return err == nil })
	cancel()
	waitClosed(t, finished)

	for _, f := range []string{"MERGE_HEAD", "index.lock"} {
		if _, err := os.Stat(filepath.Join(repo, ".git", f)); err == nil {
			t.Errorf("%s left behind", f)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, "a.txt")); err != nil {
		t.Errorf("the merge must complete: %v", err)
	}
	if !w.result.Merged || w.result.Outcome != OutcomeCompleted {
		t.Errorf("result = %+v", w.result)
	}
	_, lines := loadSingleRun(t, store, "ws1", change)
	assertEndMarker(t, lines, OutcomeCompleted)
}

// 2.1/2.2: Merged stays true when the cleanup fails after a successful merge,
// and a cancellation does not turn that run into "stopped".
func TestResult_MergedWhenCleanupFails(t *testing.T) {
	change := "cleanup-fails"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	// A locked worktree cannot be removed: the cleanup fails after the merge.
	hook := filepath.Join(repo, ".git", "hooks", "post-merge")
	script := "#!/bin/sh\ngit worktree list --porcelain | sed -n 's/^worktree //p' | grep -v \"^" + repo + "$\" | while read p; do git worktree lock \"$p\"; done\n"
	if err := os.WriteFile(hook, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	m, store, _ := newJournalManager(t, repo, ModeFullAutonomy, pipeAgent(writeInWorktree("a.txt"), okResult, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := &Worker{ID: 1, WorkspaceID: "ws1", ActiveChange: change, DelegationMode: ModeFullAutonomy, done: make(chan struct{})}
	m.activeWorkers[1] = w
	// Cancel as soon as the merge has landed.
	go func() {
		for gitOut(repo, "log", "--oneline", "--merges", "-1") == "" {
			time.Sleep(2 * time.Millisecond)
		}
		cancel()
	}()
	m.runWorker(ctx, w)

	if !w.result.Merged || w.result.Target == "" {
		t.Fatalf("Merged must be true after a successful merge: %+v", w.result)
	}
	if w.result.Outcome != OutcomePaused {
		t.Errorf("outcome = %q, want paused", w.result.Outcome)
	}
	_, lines := loadSingleRun(t, store, "ws1", change)
	assertEndMarker(t, lines, OutcomePaused)
}

func gitOut(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

// 2.2: a successful merge under cancellation is journaled completed.
// (Covered by TestCancel_DuringStartedMergeCompletes.) A cancelled worker
// without a run journal is normalized to stopped all the same.
func TestResult_CancelledWithoutJournalIsStopped(t *testing.T) {
	change := "no-journal"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1}, pipeAgent(writeInWorktree("a.txt"), okResult, nil))
	ctx, cancel := context.WithCancel(context.Background())
	beforeCommitHook = cancel
	t.Cleanup(func() { beforeCommitHook = nil })
	w := &Worker{ID: 1, WorkspaceID: "ws1", ActiveChange: change, done: make(chan struct{})}
	m.activeWorkers[1] = w
	m.runWorker(ctx, w)
	if w.result.Outcome != OutcomeStopped || w.result.Merged {
		t.Errorf("result = %+v", w.result)
	}
}

// 2.3: CancelAndWaitForChange.
func TestCancelAndWait_NoWorker(t *testing.T) {
	m := NewManager(nil, nil, nil, nil, nil)
	found, _, timedOut := m.CancelAndWaitForChange(context.Background(), "nope")
	if found || timedOut {
		t.Errorf("found=%v timedOut=%v", found, timedOut)
	}
}

func TestCancelAndWait_StoppedWorker(t *testing.T) {
	change := "wait-stopped"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1}, pipeAgent(writeInWorktree("a.txt"), okResult, nil))
	m.mergeMu.Lock()
	startCancelableWorker(m, 1, change)
	waitFor(t, "worker waiting for the lock", func() bool { return branchCommitted(t, repo, change) })
	go func() {
		time.Sleep(50 * time.Millisecond)
		m.mergeMu.Unlock()
	}()

	found, res, timedOut := m.CancelAndWaitForChange(context.Background(), change)
	if !found || timedOut || res.Outcome != OutcomeStopped || res.Merged {
		t.Errorf("found=%v timedOut=%v res=%+v", found, timedOut, res)
	}
	m.mu.Lock()
	_, still := m.activeWorkers[1]
	m.mu.Unlock()
	if still {
		t.Error("the worker must be out of activeWorkers once the wait returns")
	}
}

func TestCancelAndWait_MergedWorker(t *testing.T) {
	change := "wait-merged"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	hook := filepath.Join(repo, ".git", "hooks", "pre-merge-commit")
	started := filepath.Join(t.TempDir(), "started")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\ntouch "+started+"\nsleep 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1}, pipeAgent(writeInWorktree("a.txt"), okResult, nil))
	startCancelableWorker(m, 1, change)
	waitFor(t, "merge to start", func() bool { _, err := os.Stat(started); return err == nil })

	found, res, timedOut := m.CancelAndWaitForChange(context.Background(), change)
	if !found || timedOut || !res.Merged || res.Outcome != OutcomeCompleted || res.Target == "" {
		t.Errorf("found=%v timedOut=%v res=%+v", found, timedOut, res)
	}
}

func TestCancelAndWait_TimesOut(t *testing.T) {
	m := NewManager(nil, nil, nil, nil, nil)
	m.activeWorkers[1] = &Worker{ID: 1, ActiveChange: "stuck", CancelFunc: func() {}, done: make(chan struct{})}
	defer SetCancelWaitTimeout(50 * time.Millisecond)()

	found, _, timedOut := m.CancelAndWaitForChange(context.Background(), "stuck")
	if !found || !timedOut {
		t.Errorf("found=%v timedOut=%v", found, timedOut)
	}
}
