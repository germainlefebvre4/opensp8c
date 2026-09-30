package pool

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/session"
)

func waitUntil(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestManager_PauseThenResumeReusesWorktree replays the observed scenario on a
// temporary git repo whose Go module lives in backend/: validation now runs
// there and passes, a paused change is not relaunched by later ticks, and
// ResumeWorker relaunches it into the existing branch and worktree (with the
// agent's uncommitted work) without a branch error.
func TestManager_PauseThenResumeReusesWorktree(t *testing.T) {
	repo := newTestRepo(t) // also isolates HOME (worktrees dir)
	writeFile(t, filepath.Join(repo, "backend", "go.mod"), "module fixture\n\ngo 1.21\n")
	writeFile(t, filepath.Join(repo, "backend", "x.go"), "package fixture\n")
	writeChangeForPoolTest(t, filepath.Join(repo, "openspec", "changes"), "c1", boolPtr(true), intPtr(1))
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "fixture")

	var turns atomic.Int32
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1},
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
			return countingSubprocess(&turns)(ctx, workspacePath, agentCfg, extraSystemPrompt, claudeSessionID, resume, sessionLog, customEnv, nativeQuestionMode, languageDirective)
		})
	m.workspaceID = "ws"
	m.isRunning = true
	t.Cleanup(m.waitWorkers)

	pausedWorker := func() *Worker {
		m.mu.Lock()
		defer m.mu.Unlock()
		for _, w := range m.pausedWorkers {
			return w
		}
		return nil
	}

	// 1. The agent leaves a task unchecked: validation passes (go test in
	// backend/) and the worker pauses on the incomplete tasks.md - not on a
	// validation environment error.
	m.tick()
	waitUntil(t, "worker to pause", func() bool { return pausedWorker() != nil })
	first := pausedWorker()
	if !strings.Contains(first.BlockedReason, "tâches restantes incomplètes") {
		t.Fatalf("validation should have passed in backend/; pause reason: %q", first.BlockedReason)
	}
	turnsAfterFirstRun := turns.Load()

	// 2. A paused change is not relaunched by later ticks.
	m.tick()
	m.tick()
	time.Sleep(200 * time.Millisecond)
	m.mu.Lock()
	active := len(m.activeWorkers)
	m.mu.Unlock()
	if active != 0 || turns.Load() != turnsAfterFirstRun {
		t.Fatalf("paused change was relaunched (active=%d, turns %d -> %d)", active, turnsAfterFirstRun, turns.Load())
	}

	// 3. The agent's work (uncommitted) completes the tasks; the user resumes.
	tasks := filepath.Join(first.WorktreePath, "openspec", "changes", "c1", "tasks.md")
	if err := os.WriteFile(tasks, []byte("- [x] do the thing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := m.ResumeWorker(first.ID); err != nil {
		t.Fatal(err)
	}
	m.tick()
	waitUntil(t, "resumed worker to finish", func() bool {
		m.mu.Lock()
		defer m.mu.Unlock()
		return len(m.activeWorkers) == 0 && turns.Load() > turnsAfterFirstRun
	})

	if w := pausedWorker(); w != nil {
		t.Fatalf("resumed worker should finish without pausing again: %q", w.BlockedReason)
	}
	if b, err := os.ReadFile(tasks); err != nil || !strings.Contains(string(b), "[x]") {
		t.Fatalf("existing worktree (with uncommitted work) must be reused: %v %q", err, b)
	}
	if got := gitIn(t, first.WorktreePath, "rev-parse", "--abbrev-ref", "HEAD"); got != "feature/c1" {
		t.Fatalf("branch = %s", got)
	}
}
