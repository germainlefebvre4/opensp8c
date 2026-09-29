package pool

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
)

// mockBroadcaster records every Broadcast call for assertions.
type mockBroadcaster struct {
	mu     sync.Mutex
	events []broadcastCall
}

type broadcastCall struct {
	workspaceID string
	ev          watcher.Event
}

func (b *mockBroadcaster) Broadcast(workspaceID string, ev watcher.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, broadcastCall{workspaceID: workspaceID, ev: ev})
}

func (b *mockBroadcaster) snapshot() []broadcastCall {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]broadcastCall, len(b.events))
	copy(out, b.events)
	return out
}

// TestStatus_ScopedToWorkspace verifies that a pool started on behalf of one
// workspace is reported as not running when queried for a different one,
// so the UI never shows another workspace's pool state as its own.
func TestStatus_ScopedToWorkspace(t *testing.T) {
	m := NewManager(nil, nil, nil, nil)
	tmpDir := t.TempDir()

	if err := m.Start(AgentPoolConfig{Size: 1, MaxAttempts: 1}, "workspace-a", "Workspace A", tmpDir); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}
	t.Cleanup(m.Stop)

	if _, running, workers := m.Status("workspace-b"); running || len(workers) != 0 {
		t.Fatalf("expected pool started for workspace-a to report not running for workspace-b, got running=%v workers=%v", running, workers)
	}

	if _, running, _ := m.Status("workspace-a"); !running {
		t.Fatalf("expected pool to report running for its own workspace (workspace-a)")
	}
}

// TestStartWorker_StampsWorkspaceIdentity verifies that a worker created by
// the orchestration loop carries the workspace identity (and start time)
// that was passed to Start, as Status() would report it. The assertion is
// made while still holding m.mu, so the real worker goroutine's async
// teardown (its provisioning will fail against a non-git tmpDir) cannot race
// the read out from under us.
func TestStartWorker_StampsWorkspaceIdentity(t *testing.T) {
	m := NewManager(nil, nil, nil, nil)
	tmpDir := t.TempDir()

	if err := m.Start(AgentPoolConfig{Size: 1, MaxAttempts: 1}, "workspace-a", "Workspace A", tmpDir); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}
	defer m.Stop()

	before := time.Now()
	m.mu.Lock()
	m.startWorker("some-change")
	w, ok := m.activeWorkers[1]
	var got Worker
	if ok {
		got = *w
	}
	m.mu.Unlock()

	if !ok {
		t.Fatalf("expected worker 1 to be registered")
	}
	if got.WorkspaceID != "workspace-a" {
		t.Errorf("expected WorkspaceID %q, got %q", "workspace-a", got.WorkspaceID)
	}
	if got.WorkspaceName != "Workspace A" {
		t.Errorf("expected WorkspaceName %q, got %q", "Workspace A", got.WorkspaceName)
	}
	if got.StartedAt.Before(before) {
		t.Errorf("expected StartedAt to be set at worker creation time, got %v (before %v)", got.StartedAt, before)
	}
}

// TestStartStop_BroadcastsPoolUpdated verifies that Start and Stop each emit
// a pool_updated event for the workspace the pool was started for.
func TestStartStop_BroadcastsPoolUpdated(t *testing.T) {
	bc := &mockBroadcaster{}
	m := NewManager(bc, nil, nil, nil)
	tmpDir := t.TempDir()

	if err := m.Start(AgentPoolConfig{Size: 1, MaxAttempts: 1}, "workspace-a", "Workspace A", tmpDir); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}
	m.Stop()

	events := bc.snapshot()
	if len(events) != 2 {
		t.Fatalf("expected 2 pool_updated events (start + stop), got %d: %+v", len(events), events)
	}
	for _, call := range events {
		if call.workspaceID != "workspace-a" {
			t.Errorf("expected event for workspace-a, got %q", call.workspaceID)
		}
		if call.ev.Type != "pool_updated" {
			t.Errorf("expected pool_updated event, got %q", call.ev.Type)
		}
	}
}

// TestWorkerStatusTransition_BroadcastsAndAppendsActivity verifies that a
// worker status transition emits both pool_updated broadcast and appends a pool.worker_status
// entry to activity.jsonl for the assigned change.
func TestWorkerStatusTransition_BroadcastsAndAppendsActivity(t *testing.T) {
	tmpDir := t.TempDir()
	actStore := activity.NewStore(filepath.Join(tmpDir, "activity"), nil)
	bc := &mockBroadcaster{}
	m := NewManager(bc, nil, nil, actStore)

	wsID := "workspace-test"
	changeName := "feature-worker"

	if err := m.Start(AgentPoolConfig{Size: 1, MaxAttempts: 1}, wsID, "Workspace Test", tmpDir); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}
	defer m.Stop()

	// Initial start emitted 1 event
	bcEvents := bc.snapshot()
	if len(bcEvents) != 1 {
		t.Fatalf("expected 1 event on Start, got %d", len(bcEvents))
	}

	// Create worker
	m.mu.Lock()
	m.startWorker(changeName)
	worker := m.activeWorkers[1]
	m.mu.Unlock()

	// startWorker calls broadcastLocked(), so bc should now have 2 events
	bcEvents = bc.snapshot()
	if len(bcEvents) != 2 {
		t.Fatalf("expected 2 events after startWorker, got %d", len(bcEvents))
	}

	entries, err := actStore.Read(wsID, changeName)
	if err != nil {
		t.Fatalf("failed to read activity: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 activity entry, got %d", len(entries))
	}
	if entries[0].Type != "pool.worker_status" {
		t.Errorf("expected type pool.worker_status, got %s", entries[0].Type)
	}
	if entries[0].Category != "pool" {
		t.Errorf("expected category pool, got %s", entries[0].Category)
	}
	if entries[0].Meta["status"] != string(StatusWorking) {
		t.Errorf("expected status working, got %v", entries[0].Meta["status"])
	}

	// Now transition worker status to StatusTesting and notify
	m.mu.Lock()
	worker.Status = StatusTesting
	m.broadcastLocked()
	m.mu.Unlock()

	bcEvents = bc.snapshot()
	if len(bcEvents) != 3 {
		t.Fatalf("expected 3 events after status change, got %d", len(bcEvents))
	}

	entries, err = actStore.Read(wsID, changeName)
	if err != nil {
		t.Fatalf("failed to read activity: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 activity entries, got %d", len(entries))
	}
	if entries[1].Meta["status"] != string(StatusTesting) {
		t.Errorf("expected status testing, got %v", entries[1].Meta["status"])
	}
}

// TestStatus_IncludesPausedWorkers verifies task 1.4: once runWorker pauses a
// worker (via pauseWorker) and returns, its own deferred cleanup removes it
// from m.activeWorkers, but Status() still reports it, because it was
// snapshotted into m.pausedWorkers - unlike the pre-task-1 behavior where a
// paused worker vanished from Status() as soon as runWorker returned.
func TestStatus_IncludesPausedWorkers(t *testing.T) {
	changeName := "partial-tasks-status-change"
	repoDir := newGoFixtureRepo(t, changeName, "- [x] one\n- [ ] two\n")

	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1},
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool) (*session.Subprocess, error) {
			return fakeAutoRespondingSubprocess(), nil
		})
	m.workspaceID = "workspace-status-test"
	m.isRunning = true

	w := &Worker{ID: 1, WorkspaceID: m.workspaceID, ActiveChange: changeName}
	m.activeWorkers[1] = w
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.runWorker(ctx, w)

	m.mu.Lock()
	_, stillActive := m.activeWorkers[1]
	m.mu.Unlock()
	if stillActive {
		t.Fatalf("expected runWorker's own deferred cleanup to remove the worker from activeWorkers")
	}

	_, running, workers := m.Status(m.workspaceID)
	if !running {
		t.Fatalf("expected pool to still report running")
	}
	if len(workers) != 1 {
		t.Fatalf("expected the paused worker to still be reported by Status(), got %d workers: %+v", len(workers), workers)
	}
	if workers[0].Status != StatusPaused || workers[0].BlockedReason == "" {
		t.Errorf("expected the reported worker to be paused with its BlockedReason, got %+v", workers[0])
	}
}

// TestStop_ClearsPausedWorkersToo verifies task 1.6: Stop() empties
// m.pausedWorkers in addition to m.activeWorkers, so a previously paused
// worker no longer appears in Status() once the pool has been stopped.
func TestStop_ClearsPausedWorkersToo(t *testing.T) {
	m := NewManager(nil, nil, nil, nil)
	tmpDir := t.TempDir()

	if err := m.Start(AgentPoolConfig{Size: 1, MaxAttempts: 1}, "workspace-a", "Workspace A", tmpDir); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}

	m.mu.Lock()
	m.pausedWorkers[1] = &Worker{ID: 1, ActiveChange: "some-change", Status: StatusPaused, BlockedReason: "some reason"}
	m.mu.Unlock()

	if _, running, workers := m.Status("workspace-a"); !running || len(workers) != 1 {
		t.Fatalf("expected the paused worker to be reported before Stop(), got running=%v workers=%+v", running, workers)
	}

	m.Stop()

	if _, running, workers := m.Status("workspace-a"); running || len(workers) != 0 {
		t.Fatalf("expected Stop() to clear paused workers too, got running=%v workers=%+v", running, workers)
	}
}

// TestTick_PausedWorkersDoNotBlockDispatch verifies task 1.5: tick() decides
// dispatch capacity solely from len(m.activeWorkers), so a worker already
// moved into m.pausedWorkers (occupying its own former slot conceptually,
// but no longer counted for capacity) does not prevent a new Todo change
// from being picked up - the "libère le worker pour d'autres tâches
// indépendantes" scenario of agent-pool-orchestrator.
func TestTick_PausedWorkersDoNotBlockDispatch(t *testing.T) {
	changeName := "todo-after-pause-change"
	repoDir := newGoFixtureRepo(t, changeName, "- [ ] do the thing\n")

	block := make(chan struct{})
	defer close(block)

	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, MaxAttempts: 1},
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool) (*session.Subprocess, error) {
			<-block
			return nil, context.Canceled
		})

	// Simulate a worker from a previous change that already paused: freed
	// from m.activeWorkers, tracked only in m.pausedWorkers (as pauseWorker
	// leaves things once runWorker's defer runs).
	m.mu.Lock()
	m.pausedWorkers[1] = &Worker{ID: 1, ActiveChange: "already-paused-change", Status: StatusPaused, BlockedReason: "some reason"}
	m.mu.Unlock()

	m.tick()

	deadline := time.Now().Add(5 * time.Second)
	for {
		m.mu.Lock()
		activeCount := len(m.activeWorkers)
		_, gotNewWorker := m.activeWorkers[2]
		m.mu.Unlock()
		if activeCount == 1 && gotNewWorker {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected tick() to dispatch a new worker (id 2) for the runnable Todo change despite a paused worker occupying id 1, got %d active worker(s)", activeCount)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestCancelWorkerForChange(t *testing.T) {
	m := NewManager(nil, nil, nil, nil)

	var canceledA, canceledB bool
	cancelA := func() { canceledA = true }
	cancelB := func() { canceledB = true }

	m.mu.Lock()
	m.activeWorkers[1] = &Worker{
		ID:           1,
		ActiveChange: "change-a",
		CancelFunc:   cancelA,
	}
	m.activeWorkers[2] = &Worker{
		ID:           2,
		ActiveChange: "change-b",
		CancelFunc:   cancelB,
	}
	m.mu.Unlock()

	// Cancel non-existent worker
	if got := m.CancelWorkerForChange("non-existent"); got {
		t.Errorf("expected CancelWorkerForChange to return false for non-existent change, got true")
	}
	if canceledA || canceledB {
		t.Errorf("expected neither worker to be canceled")
	}

	// Cancel change-a
	if got := m.CancelWorkerForChange("change-a"); !got {
		t.Errorf("expected CancelWorkerForChange to return true for change-a, got false")
	}
	if !canceledA {
		t.Errorf("expected worker for change-a to be canceled")
	}
	if canceledB {
		t.Errorf("expected worker for change-b to remain active and un-canceled")
	}
}

