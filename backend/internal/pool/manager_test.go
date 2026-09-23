package pool

import (
	"sync"
	"testing"

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
	m := NewManager(nil)
	tmpDir := t.TempDir()

	if err := m.Start(AgentPoolConfig{Size: 1, MaxAttempts: 1}, "workspace-a", tmpDir); err != nil {
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

// TestStartStop_BroadcastsPoolUpdated verifies that Start and Stop each emit
// a pool_updated event for the workspace the pool was started for.
func TestStartStop_BroadcastsPoolUpdated(t *testing.T) {
	bc := &mockBroadcaster{}
	m := NewManager(bc)
	tmpDir := t.TempDir()

	if err := m.Start(AgentPoolConfig{Size: 1, MaxAttempts: 1}, "workspace-a", tmpDir); err != nil {
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
			t.Errorf("expected event type pool_updated, got %q", call.ev.Type)
		}
	}
}
