package pool

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/session"
)

func countChangeUpdated(b *mockBroadcaster, change string) int {
	n := 0
	for _, c := range b.snapshot() {
		if c.ev.Type == "change_updated" && c.ev.Name == change {
			n++
		}
	}
	return n
}

func newWatchedWorktree(t *testing.T, change string) (string, string) {
	t.Helper()
	wt := t.TempDir()
	dir := filepath.Join(wt, "openspec", "changes", change)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	return wt, filepath.Join(dir, "tasks.md")
}

func TestWatchWorktreeTasks_BurstYieldsOneEvent(t *testing.T) {
	wt, tasks := newWatchedWorktree(t, "c1")
	b := &mockBroadcaster{}
	m := NewManager(b, nil, nil, nil, nil)
	stop := m.watchWorktreeTasks("ws1", wt, "c1")
	defer stop()

	for i := 0; i < 5; i++ {
		if err := os.WriteFile(tasks, []byte("- [x] a\n"), 0644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)

	if got := countChangeUpdated(b, "c1"); got != 1 {
		t.Errorf("change_updated events = %d, want 1", got)
	}
	if ws := b.snapshot()[0].workspaceID; ws != "ws1" {
		t.Errorf("workspaceID = %q, want ws1", ws)
	}
}

func TestWatchWorktreeTasks_IgnoresOtherFiles(t *testing.T) {
	wt, tasks := newWatchedWorktree(t, "c1")
	b := &mockBroadcaster{}
	m := NewManager(b, nil, nil, nil, nil)
	stop := m.watchWorktreeTasks("ws1", wt, "c1")
	defer stop()

	if err := os.WriteFile(filepath.Join(filepath.Dir(tasks), "design.md"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond)
	if got := len(b.snapshot()); got != 0 {
		t.Errorf("events = %d, want 0", got)
	}
}

func TestWatchWorktreeTasks_NoEventAfterStop(t *testing.T) {
	wt, tasks := newWatchedWorktree(t, "c1")
	b := &mockBroadcaster{}
	m := NewManager(b, nil, nil, nil, nil)
	stop := m.watchWorktreeTasks("ws1", wt, "c1")
	stop()
	stop() // idempotent

	if err := os.WriteFile(tasks, []byte("- [x] a\n"), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond)
	if got := len(b.snapshot()); got != 0 {
		t.Errorf("events after stop = %d, want 0", got)
	}
}

func TestRunWorker_BroadcastsWorktreeTaskProgressUntilExit(t *testing.T) {
	const change = "live-progress-change"
	repoDir := newGoFixtureRepo(t, change, "- [x] one\n- [x] two\n")
	b := &mockBroadcaster{}

	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1},
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
			// The agent ticks a task in its worktree, then keeps running
			// long enough for the debounce to fire.
			tasks := filepath.Join(workspacePath, "openspec", "changes", change, "tasks.md")
			if err := os.WriteFile(tasks, []byte("- [x] one\n- [x] two\n"), 0644); err != nil {
				t.Errorf("write worktree tasks: %v", err)
			}
			time.Sleep(500 * time.Millisecond)
			return fakeAutoRespondingSubprocess(), nil
		})
	m.broadcaster = b
	m.workspaceID = "ws1"

	before := runtime.NumGoroutine()
	w := &Worker{ID: 1, WorkspaceID: "ws1", ActiveChange: change}
	m.activeWorkers[1] = w
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.runWorker(ctx, w)

	during := countChangeUpdated(b, change)
	if during < 1 {
		t.Fatalf("expected a change_updated during the run, got %d", during)
	}

	// After the worker exited, the watcher is gone.
	tasks := filepath.Join(w.WorktreePath, "openspec", "changes", change, "tasks.md")
	if err := os.WriteFile(tasks, []byte("- [ ] one\n"), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond)
	if got := countChangeUpdated(b, change); got != during {
		t.Errorf("events after exit: %d -> %d", during, got)
	}

	// No goroutine leak from the watcher (allow slack for unrelated ones).
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before+2 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > before+2 {
		t.Errorf("goroutines %d -> %d, possible leak", before, n)
	}
}
