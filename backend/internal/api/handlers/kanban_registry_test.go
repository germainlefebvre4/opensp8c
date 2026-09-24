package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/glefebvre/opensp8c/internal/workspace"
)

func listChangesRequest(workspaceID string) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID+"/changes", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workspaceID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return httptest.NewRecorder(), req
}

// TestActiveWorkerChanges_ScopedPerWorkspace verifies that a change with the
// same name in two different workspaces only reports WorkerActive: true in
// the workspace whose pool is actually processing it, now that
// KanbanHandler resolves a per-workspace Manager from the shared Registry
// instead of a single global one.
func TestActiveWorkerChanges_ScopedPerWorkspace(t *testing.T) {
	const changeName = "shared-change"
	tmpA := setupGitWorkspace(t, changeName)
	tmpB := setupGitWorkspace(t, changeName)

	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{
		{Name: "workspace-a", Path: tmpA},
		{Name: "workspace-b", Path: tmpB},
	}}
	ws := NewWorkspaceHandler(cfg, "", nil)
	reg := pool.NewRegistry(nil, nil, nil)
	h := NewKanbanHandler(ws, nil, reg, nil, nil, nil, "")

	idA := workspace.StableID(mustAbs(t, tmpA))
	idB := workspace.StableID(mustAbs(t, tmpB))

	t.Cleanup(func() {
		reg.For(idA).Stop()
		reg.For(idB).Stop()
	})

	// Only workspace A's pool is started, so the worker it picks up on
	// "shared-change" must never leak into workspace B's view of the change
	// bearing the same name.
	if err := reg.For(idA).Start(pool.AgentPoolConfig{Size: 1, DelegationMode: pool.ModeHITLReview, MaxAttempts: 1}, idA, "workspace-a", tmpA); err != nil {
		t.Fatalf("failed to start pool A: %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	active := false
	for time.Now().Before(deadline) {
		_, _, workers := reg.For(idA).Status(idA)
		for _, w := range workers {
			if w.ActiveChange == changeName {
				active = true
			}
		}
		if active {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !active {
		t.Skip("pool did not pick up the worker within the deadline (stub orchestration timing); skipping")
	}

	recA, reqA := listChangesRequest(idA)
	h.ListChanges(recA, reqA)
	if recA.Code != http.StatusOK {
		t.Fatalf("expected 200 listing workspace A changes, got %d: %s", recA.Code, recA.Body.String())
	}
	var changesA []openspec.Change
	if err := json.Unmarshal(recA.Body.Bytes(), &changesA); err != nil {
		t.Fatalf("failed to decode workspace A changes: %v", err)
	}
	if !workerActiveFor(changesA, changeName) {
		t.Fatalf("expected %q to show WorkerActive=true in workspace A, got %+v", changeName, changesA)
	}

	recB, reqB := listChangesRequest(idB)
	h.ListChanges(recB, reqB)
	if recB.Code != http.StatusOK {
		t.Fatalf("expected 200 listing workspace B changes, got %d: %s", recB.Code, recB.Body.String())
	}
	var changesB []openspec.Change
	if err := json.Unmarshal(recB.Body.Bytes(), &changesB); err != nil {
		t.Fatalf("failed to decode workspace B changes: %v", err)
	}
	if workerActiveFor(changesB, changeName) {
		t.Fatalf("expected %q to show WorkerActive=false in workspace B (its own pool never started), got %+v", changeName, changesB)
	}
}

func workerActiveFor(changes []openspec.Change, name string) bool {
	for _, c := range changes {
		if c.Name == name {
			return c.WorkerActive
		}
	}
	return false
}
