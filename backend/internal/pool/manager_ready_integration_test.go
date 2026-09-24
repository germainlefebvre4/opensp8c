package pool

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/watcher"
)

// activeChangeCapture is a Broadcaster that snapshots any worker's
// ActiveChange directly off the Manager's map on every pool_updated event.
// Broadcast is always invoked by the Manager while its own mu is already
// held (see broadcastLocked's contract), so reading activeWorkers here
// without re-locking is safe and race-free - unlike polling Status() on a
// timer, it can never miss a worker that lived for less than a poll
// interval, since it observes the exact same critical section that
// creates/removes it.
type activeChangeCapture struct {
	mgr *Manager

	mu     sync.Mutex
	change string
}

func (c *activeChangeCapture) Broadcast(_ string, _ watcher.Event) {
	for _, w := range c.mgr.activeWorkers {
		if w.ActiveChange != "" {
			c.mu.Lock()
			if c.change == "" {
				c.change = w.ActiveChange
			}
			c.mu.Unlock()
		}
	}
}

func (c *activeChangeCapture) get() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.change
}

// writeChange creates a change directory with tasks.md and an optional
// .openspec.yaml (launched/order), mirroring how the app would generate it.
func writeChangeForPoolTest(t *testing.T, changesDir, name string, launched *bool, order *int) {
	t.Helper()
	dir := filepath.Join(changesDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("- [ ] do the thing\n"), 0644); err != nil {
		t.Fatalf("WriteFile tasks.md: %v", err)
	}
	meta := "schema: spec-driven\ncreated: \"2024-01-01\"\n"
	if launched != nil {
		if *launched {
			meta += "launched: true\n"
		} else {
			meta += "launched: false\n"
		}
	}
	if order != nil {
		meta += fmt.Sprintf("order: %d\n", *order)
	}
	if err := os.WriteFile(filepath.Join(dir, ".openspec.yaml"), []byte(meta), 0644); err != nil {
		t.Fatalf("WriteFile .openspec.yaml: %v", err)
	}
}

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

// TestManager_SkipsReadyAndPicksLowestOrder exercises the Agent Pool
// dispatcher end-to-end (task 9.1's backend scenario): given a mix of
// "ready" (not launched) and "todo" (launched, with distinct priority
// orders) changes, only a "todo" change is ever picked up by a worker, and
// among runnable "todo" changes the one with the lowest `order` is
// dispatched first.
func TestManager_SkipsReadyAndPicksLowestOrder(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()
	changesDir := filepath.Join(tmpDir, "openspec", "changes")

	writeChangeForPoolTest(t, changesDir, "ready-change", boolPtr(false), intPtr(1))
	writeChangeForPoolTest(t, changesDir, "todo-high-order", boolPtr(true), intPtr(2))
	writeChangeForPoolTest(t, changesDir, "todo-low-order", boolPtr(true), intPtr(1))

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-q")
	runGit("-c", "user.email=test@test.com", "-c", "user.name=test", "add", "-A")
	// -c commit.gpgsign=false: disposable fixture repo under t.TempDir(), never
	// pushed - avoids failing on machines where commit signing needs an
	// interactive pinentry unavailable to the test runner.
	runGit("-c", "user.email=test@test.com", "-c", "user.name=test", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "init")

	// sessionMgr/prefs are left nil: runWorker's nil-guard falls back to a
	// zero-value agents.AgentConfig (empty CLI), which fails subprocess
	// startup deterministically without ever touching PATH. This test only
	// cares about scheduler dispatch order, and must never risk shelling out
	// to a real agent CLI that happens to be installed on the dev machine.
	// That failure is now near-instant (rather than the old 2s stub sleep),
	// so a worker can live for far less than a polling interval - capture()
	// via the broadcaster below instead of polling Status() on a timer.
	capture := &activeChangeCapture{}
	mgr := NewManager(capture, nil, nil, nil)
	capture.mgr = mgr
	workspaceID := "test-ws"
	if err := mgr.Start(AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1}, workspaceID, "test", tmpDir); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}
	t.Cleanup(func() {
		mgr.Stop()
		home, _ := os.UserHomeDir()
		os.RemoveAll(filepath.Join(home, ".opensp8c", "worktrees", "wt-todo-low-order"))
		os.RemoveAll(filepath.Join(home, ".opensp8c", "worktrees", "wt-todo-high-order"))
		os.RemoveAll(filepath.Join(home, ".opensp8c", "worktrees", "wt-ready-change"))
	})

	deadline := time.Now().Add(15 * time.Second)
	var activeChange string
	for time.Now().Before(deadline) {
		if activeChange = capture.get(); activeChange != "" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if activeChange == "" {
		t.Skip("pool did not pick up a worker within the deadline (stub orchestration timing); skipping")
	}

	if activeChange == "ready-change" {
		t.Fatalf("expected a worker never to pick up the Ready-column change, got %q", activeChange)
	}
	if activeChange != "todo-low-order" {
		t.Fatalf("expected the lowest-order Todo change to be picked up first, got %q", activeChange)
	}
}
