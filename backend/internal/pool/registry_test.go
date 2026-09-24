package pool

import "testing"

// TestRegistry_For_CachesPerWorkspace verifies that For returns the same
// Manager instance for repeated calls with the same workspace ID, and a
// distinct instance for a different ID.
func TestRegistry_For_CachesPerWorkspace(t *testing.T) {
	reg := NewRegistry(nil, nil, nil, nil)

	a1 := reg.For("workspace-a")
	a2 := reg.For("workspace-a")
	if a1 != a2 {
		t.Fatalf("expected repeated For(\"workspace-a\") calls to return the same Manager instance")
	}

	b := reg.For("workspace-b")
	if a1 == b {
		t.Fatalf("expected For(\"workspace-b\") to return a different Manager instance than workspace-a")
	}
}

// TestRegistry_AllWorkers verifies that AllWorkers flattens the active
// workers of every known workspace's Manager, covering zero, one, and
// multiple workspaces with active workers.
func TestRegistry_AllWorkers(t *testing.T) {
	reg := NewRegistry(nil, nil, nil, nil)

	if workers := reg.AllWorkers(); len(workers) != 0 {
		t.Fatalf("expected no workers with no registered workspaces, got %+v", workers)
	}

	tmpA := t.TempDir()
	mgrA := reg.For("workspace-a")
	if err := mgrA.Start(AgentPoolConfig{Size: 2, MaxAttempts: 1}, "workspace-a", "Workspace A", tmpA); err != nil {
		t.Fatalf("failed to start pool for workspace-a: %v", err)
	}
	t.Cleanup(mgrA.Stop)

	mgrA.mu.Lock()
	mgrA.startWorker("change-1")
	mgrA.startWorker("change-2")
	mgrA.mu.Unlock()

	if workers := reg.AllWorkers(); len(workers) != 2 {
		t.Fatalf("expected 2 workers with only workspace-a active, got %d: %+v", len(workers), workers)
	}

	tmpB := t.TempDir()
	mgrB := reg.For("workspace-b")
	if err := mgrB.Start(AgentPoolConfig{Size: 1, MaxAttempts: 1}, "workspace-b", "Workspace B", tmpB); err != nil {
		t.Fatalf("failed to start pool for workspace-b: %v", err)
	}
	t.Cleanup(mgrB.Stop)

	mgrB.mu.Lock()
	mgrB.startWorker("change-3")
	mgrB.mu.Unlock()

	workers := reg.AllWorkers()
	if len(workers) != 3 {
		t.Fatalf("expected 3 workers across workspace-a and workspace-b, got %d: %+v", len(workers), workers)
	}

	byWorkspace := map[string]int{}
	for _, w := range workers {
		byWorkspace[w.WorkspaceID]++
	}
	if byWorkspace["workspace-a"] != 2 || byWorkspace["workspace-b"] != 1 {
		t.Fatalf("expected 2 workers for workspace-a and 1 for workspace-b, got %+v", byWorkspace)
	}
}

// TestRegistry_Remove verifies that Remove stops the workspace's pool and
// drops it from the registry, so a later For call creates a fresh Manager.
func TestRegistry_Remove(t *testing.T) {
	reg := NewRegistry(nil, nil, nil, nil)
	tmpDir := t.TempDir()

	mgr := reg.For("workspace-a")
	if err := mgr.Start(AgentPoolConfig{Size: 1, MaxAttempts: 1}, "workspace-a", "Workspace A", tmpDir); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}

	reg.Remove("workspace-a")

	if _, running, _ := mgr.Status("workspace-a"); running {
		t.Fatalf("expected pool to be stopped after Remove")
	}

	if fresh := reg.For("workspace-a"); fresh == mgr {
		t.Fatalf("expected For after Remove to create a fresh Manager instance")
	}
}
