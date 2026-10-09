package pool

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/session"
)

// countingNeverStart is a startSubprocessFn stub that fails the test when the
// worker tries to start an agent, and counts the attempts.
func countingNeverStart(calls *atomic.Int32) stubFn {
	return func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
		calls.Add(1)
		return nil, errors.New("subprocess must not start")
	}
}

// pausedWithWorktree registers a paused worker on change whose worktree
// carries the given tasks.md ("" means no file at all).
func pausedWithWorktree(t *testing.T, m *Manager, change, tasksMd string) string {
	t.Helper()
	wtDir := t.TempDir()
	if tasksMd != "" {
		dir := filepath.Join(wtDir, "openspec", "changes", change)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(tasksMd), 0644); err != nil {
			t.Fatal(err)
		}
	}
	m.workspaceID = "ws"
	m.isRunning = true
	m.mu.Lock()
	m.pausedWorkers[1] = &Worker{ID: 1, ActiveChange: change, Status: StatusPaused, WorktreePath: wtDir, BlockedReason: "r"}
	m.mu.Unlock()
	return wtDir
}

func TestResumeWorker_FinalizeOnly(t *testing.T) {
	t.Run("complete list lifts the pause and records the intention", func(t *testing.T) {
		m := NewManager(nil, nil, nil, nil, nil)
		pausedWithWorktree(t, m, "c", "- [x] a\n- [x] b\n")
		if err := m.ResumeWorker(1, true); err != nil {
			t.Fatal(err)
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		if len(m.pausedWorkers) != 0 || !m.finalizeChanges["c"] {
			t.Fatalf("paused=%v finalize=%v", m.pausedWorkers, m.finalizeChanges)
		}
	})

	t.Run("incomplete list is refused with the exact count and no side effect", func(t *testing.T) {
		m := NewManager(nil, nil, nil, nil, nil)
		pausedWithWorktree(t, m, "c", "- [x] a\n- [ ] b\n- [ ] c\n")
		err := m.ResumeWorker(1, true)
		var inc *ErrTasksIncomplete
		if !errors.As(err, &inc) || inc.Remaining != 2 || !strings.Contains(err.Error(), "2") {
			t.Fatalf("got %v", err)
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		if _, ok := m.pausedWorkers[1]; !ok || m.pausedWorkers[1].BlockedReason != "r" || len(m.finalizeChanges) != 0 {
			t.Fatalf("refusal must change nothing: %+v %v", m.pausedWorkers, m.finalizeChanges)
		}
	})

	t.Run("flagged remaining task still refuses the finalizing resume", func(t *testing.T) {
		m := NewManager(nil, nil, nil, nil, nil)
		pausedWithWorktree(t, m, "c", "- [x] a\n- [ ] b <!-- human review required -->\n")
		var inc *ErrTasksIncomplete
		if err := m.ResumeWorker(1, true); !errors.As(err, &inc) || inc.Remaining != 1 {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("absent or empty list is refused", func(t *testing.T) {
		for name, content := range map[string]string{"absent": "", "empty": "# Tasks\nnothing\n"} {
			m := NewManager(nil, nil, nil, nil, nil)
			pausedWithWorktree(t, m, "c", content)
			var inc *ErrTasksIncomplete
			if err := m.ResumeWorker(1, true); !errors.As(err, &inc) || inc.Total != 0 {
				t.Fatalf("%s: got %v", name, err)
			}
		}
	})

	t.Run("unknown worker and stopped pool", func(t *testing.T) {
		m := NewManager(nil, nil, nil, nil, nil)
		if err := m.ResumeWorker(1, true); !errors.Is(err, ErrPoolNotRunning) {
			t.Fatalf("stopped: %v", err)
		}
		pausedWithWorktree(t, m, "c", "- [x] a\n")
		if err := m.ResumeWorker(9, true); !errors.Is(err, ErrWorkerNotPaused) {
			t.Fatalf("unknown: %v", err)
		}
	})

	t.Run("ordinary resume records no intention", func(t *testing.T) {
		m := NewManager(nil, nil, nil, nil, nil)
		pausedWithWorktree(t, m, "c", "- [x] a\n")
		if err := m.ResumeWorker(1, false); err != nil {
			t.Fatal(err)
		}
		if len(m.finalizeChanges) != 0 {
			t.Fatalf("got %v", m.finalizeChanges)
		}
	})
}

func TestFinalizeIntention_Lifecycle(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	notRepo := t.TempDir() // provisioning fails fast: the worker pauses at once

	t.Run("consumed exactly once by startWorker", func(t *testing.T) {
		m := newWorkerTestManager(t, notRepo, AgentPoolConfig{Size: 2, MaxAttempts: 1}, neverStart(t))
		m.workspaceID = "ws"
		m.isRunning = true
		m.mu.Lock()
		m.finalizeChanges["c"] = true
		m.startWorker("c")
		first := m.activeWorkers[1]
		m.startWorker("c")
		second := m.activeWorkers[2]
		m.mu.Unlock()
		m.waitWorkers()
		if !first.finalizeOnly || second.finalizeOnly || len(m.finalizeChanges) != 0 {
			t.Fatalf("first=%v second=%v left=%v", first.finalizeOnly, second.finalizeOnly, m.finalizeChanges)
		}
	})

	t.Run("cleared by Stop", func(t *testing.T) {
		m := NewManager(nil, nil, nil, nil, nil)
		pausedWithWorktree(t, m, "c", "- [x] a\n")
		if err := m.ResumeWorker(1, true); err != nil {
			t.Fatal(err)
		}
		m.Stop()
		if len(m.finalizeChanges) != 0 {
			t.Fatalf("got %v", m.finalizeChanges)
		}
	})

	t.Run("cleared by demotion", func(t *testing.T) {
		m := NewManager(nil, nil, nil, nil, nil)
		pausedWithWorktree(t, m, "c", "- [x] a\n")
		if err := m.ResumeWorker(1, true); err != nil {
			t.Fatal(err)
		}
		m.ReleasePausedForChange("c")
		if len(m.finalizeChanges) != 0 {
			t.Fatalf("got %v", m.finalizeChanges)
		}
	})
}

// finalizeFixture builds a fixture repo whose worktree is provisioned and
// carries uncommitted work: tasks.md fully checked, as after a manual task.
func finalizeFixture(t *testing.T, change string, mode DelegationMode) (*Manager, *Worker, *atomic.Int32) {
	t.Helper()
	repo := newGoFixtureRepo(t, change, "- [x] one\n- [ ] two\n")
	calls := new(atomic.Int32)
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: mode, MaxAttempts: 3}, countingNeverStart(calls))
	wtPath, err := NewWorktreeController(repo, "", m.worktreesRoot).Provision(change)
	if err != nil {
		t.Fatal(err)
	}
	tasks := filepath.Join(wtPath, "openspec", "changes", change, "tasks.md")
	if err := os.WriteFile(tasks, []byte("- [x] one\n- [x] two\n"), 0644); err != nil {
		t.Fatal(err)
	}
	w := &Worker{ID: 1, ActiveChange: change, DelegationMode: mode, finalizeOnly: true}
	m.activeWorkers[1] = w
	return m, w, calls
}

func TestRunWorker_FinalizeOnly_HITLReview(t *testing.T) {
	m, w, calls := finalizeFixture(t, "fin-hitl", ModeHITLReview)
	m.runWorker(context.Background(), w)

	if calls.Load() != 0 {
		t.Fatalf("agent started %d time(s)", calls.Load())
	}
	if w.Status == StatusPaused {
		t.Fatalf("worker paused: %s", w.BlockedReason)
	}
	if w.result.Outcome != "" && w.result.Outcome != OutcomeAwaitingReview {
		t.Fatalf("outcome %q", w.result.Outcome)
	}
	if !openspec.ReviewMarkers(m.workspacePath)["fin-hitl"] {
		t.Fatal("change must be marked for review")
	}
}

func TestRunWorker_FinalizeOnly_FullAutonomy(t *testing.T) {
	m, w, calls := finalizeFixture(t, "fin-auto", ModeFullAutonomy)
	m.runWorker(context.Background(), w)

	if calls.Load() != 0 {
		t.Fatalf("agent started %d time(s)", calls.Load())
	}
	if w.Status == StatusPaused {
		t.Fatalf("worker paused: %s", w.BlockedReason)
	}
	out, err := exec.Command("git", "-C", m.workspacePath, "show", "HEAD:openspec/changes/fin-auto/tasks.md").CombinedOutput()
	if err != nil || strings.Contains(string(out), "- [ ]") {
		t.Fatalf("change must be merged with its checked tasks: %v\n%s", err, out)
	}
}

func TestRunWorker_FinalizeOnly_ValidationFailurePausesWithoutAttempt(t *testing.T) {
	m, w, calls := finalizeFixture(t, "fin-fail", ModeHITLReview)
	// WorktreePath is only set by runWorker: Provision returns the same worktree.
	wtPath, err := NewWorktreeController(m.workspacePath, "", m.worktreesRoot).Provision("fin-fail")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wtPath, "main.go"), []byte("package main\n\nfunc main( {\n"), 0644); err != nil {
		t.Fatal(err)
	}
	m.runWorker(context.Background(), w)

	if calls.Load() != 0 {
		t.Fatalf("agent started %d time(s)", calls.Load())
	}
	if w.Status != StatusPaused || !strings.Contains(w.BlockedReason, "Reprise en finalisant : la validation a échoué et aucun agent n'est lancé pour la corriger") {
		t.Fatalf("status=%s reason=%q", w.Status, w.BlockedReason)
	}
	if strings.Contains(w.BlockedReason, "Tentatives de réparation") {
		t.Fatalf("no attempt may be reported: %q", w.BlockedReason)
	}
	if _, err := os.Stat(filepath.Join(wtPath, "main.go")); err != nil {
		t.Fatalf("worktree work must be kept: %v", err)
	}
}
