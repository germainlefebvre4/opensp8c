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

// TestRegistry_AllPools verifies that AllPools returns one PoolSummary per
// running pool across every known workspace, each carrying its configured
// size, delegation mode, and its own workers - covering zero, one, and
// multiple workspaces with active pools.
func TestRegistry_AllPools(t *testing.T) {
	reg := NewRegistry(nil, nil, nil, nil)

	if pools := reg.AllPools(); len(pools) != 0 {
		t.Fatalf("expected no pools with no registered workspaces, got %+v", pools)
	}

	tmpA := t.TempDir()
	mgrA := reg.For("workspace-a")
	if err := mgrA.Start(AgentPoolConfig{Size: 2, DelegationMode: ModeHITLReview, MaxAttempts: 1}, "workspace-a", "Workspace A", tmpA); err != nil {
		t.Fatalf("failed to start pool for workspace-a: %v", err)
	}
	t.Cleanup(mgrA.Stop)

	mgrA.mu.Lock()
	mgrA.startWorker("change-1")
	mgrA.startWorker("change-2")
	mgrA.mu.Unlock()

	pools := reg.AllPools()
	if len(pools) != 1 {
		t.Fatalf("expected 1 pool with only workspace-a active, got %d: %+v", len(pools), pools)
	}
	if pools[0].WorkspaceID != "workspace-a" || pools[0].WorkspaceName != "Workspace A" {
		t.Fatalf("expected pool tagged with workspace-a identity, got %+v", pools[0])
	}
	if pools[0].Size != 2 || pools[0].DelegationMode != ModeHITLReview {
		t.Fatalf("expected pool to carry its configured size and delegation mode, got %+v", pools[0])
	}
	if len(pools[0].Workers) != 2 {
		t.Fatalf("expected 2 workers in workspace-a's pool, got %d: %+v", len(pools[0].Workers), pools[0].Workers)
	}

	tmpB := t.TempDir()
	mgrB := reg.For("workspace-b")
	if err := mgrB.Start(AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1}, "workspace-b", "Workspace B", tmpB); err != nil {
		t.Fatalf("failed to start pool for workspace-b: %v", err)
	}
	t.Cleanup(mgrB.Stop)

	mgrB.mu.Lock()
	mgrB.startWorker("change-3")
	mgrB.mu.Unlock()

	pools = reg.AllPools()
	if len(pools) != 2 {
		t.Fatalf("expected 2 pools across workspace-a and workspace-b, got %d: %+v", len(pools), pools)
	}

	byWorkspace := map[string]PoolSummary{}
	for _, p := range pools {
		byWorkspace[p.WorkspaceID] = p
	}
	if len(byWorkspace["workspace-a"].Workers) != 2 || len(byWorkspace["workspace-b"].Workers) != 1 {
		t.Fatalf("expected 2 workers for workspace-a's pool and 1 for workspace-b's, got %+v", byWorkspace)
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
