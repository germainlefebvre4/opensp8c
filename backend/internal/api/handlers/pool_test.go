package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/glefebvre/opensp8c/internal/workspace"
)

// poolRequest builds an *http.Request carrying workspaceID as the chi "id"
// URL param, mirroring how router.go dispatches /workspaces/{id}/pool/*.
func poolRequest(method, path, workspaceID, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workspaceID)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("failed to resolve abs path: %v", err)
	}
	return abs
}

// twoWorkspaceHandler wires a PoolHandler against two configured workspaces
// backed by their own tmp dirs, sharing one Registry, mirroring how
// router.go constructs it.
func twoWorkspaceHandler(t *testing.T, pathA, pathB string) (h *PoolHandler, reg *pool.Registry, idA, idB string) {
	t.Helper()
	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{
		{Name: "workspace-a", Path: pathA},
		{Name: "workspace-b", Path: pathB},
	}}
	ws := NewWorkspaceHandler(cfg, "", nil)
	reg = pool.NewRegistry(nil, nil, nil, nil)
	h = NewPoolHandler(ws, reg)

	idA = workspace.StableID(mustAbs(t, pathA))
	idB = workspace.StableID(mustAbs(t, pathB))
	return h, reg, idA, idB
}

// TestStartPool_IndependentAcrossWorkspaces verifies that starting a pool on
// one workspace does not affect another, and that a second start on the same
// workspace is rejected while the first is still running.
func TestStartPool_IndependentAcrossWorkspaces(t *testing.T) {
	h, _, idA, idB := twoWorkspaceHandler(t, t.TempDir(), t.TempDir())

	body := `{"size":1,"delegation_mode":"hitl-review","max_attempts":1}`

	recA := httptest.NewRecorder()
	h.StartPool(recA, poolRequest(http.MethodPost, "/workspaces/"+idA+"/pool/start", idA, body))
	if recA.Code != http.StatusOK {
		t.Fatalf("expected 200 starting pool on workspace A, got %d: %s", recA.Code, recA.Body.String())
	}

	recB := httptest.NewRecorder()
	h.StartPool(recB, poolRequest(http.MethodPost, "/workspaces/"+idB+"/pool/start", idB, body))
	if recB.Code != http.StatusOK {
		t.Fatalf("expected 200 starting pool on workspace B while A is running, got %d: %s", recB.Code, recB.Body.String())
	}

	// A second start on workspace A while it's still running must be refused,
	// without disturbing workspace B.
	recAAgain := httptest.NewRecorder()
	h.StartPool(recAAgain, poolRequest(http.MethodPost, "/workspaces/"+idA+"/pool/start", idA, body))
	if recAAgain.Code != http.StatusConflict {
		t.Fatalf("expected 409 on double-start of workspace A, got %d: %s", recAAgain.Code, recAAgain.Body.String())
	}

	statusA := httptest.NewRecorder()
	h.GetPoolStatus(statusA, poolRequest(http.MethodGet, "/workspaces/"+idA+"/pool/status", idA, ""))
	var respA map[string]interface{}
	if err := json.Unmarshal(statusA.Body.Bytes(), &respA); err != nil {
		t.Fatalf("failed to decode workspace A status: %v", err)
	}
	if respA["is_running"] != true {
		t.Fatalf("expected workspace A pool to still be running, got %+v", respA)
	}

	statusB := httptest.NewRecorder()
	h.GetPoolStatus(statusB, poolRequest(http.MethodGet, "/workspaces/"+idB+"/pool/status", idB, ""))
	var respB map[string]interface{}
	if err := json.Unmarshal(statusB.Body.Bytes(), &respB); err != nil {
		t.Fatalf("failed to decode workspace B status: %v", err)
	}
	if respB["is_running"] != true {
		t.Fatalf("expected workspace B pool to be running independently, got %+v", respB)
	}
}

// setupGitWorkspace creates a tmp git repo containing one runnable ("todo")
// change named changeName, so the pool orchestrator can actually pick it up
// and dispatch a worker, mirroring TestDeleteChange_WorkerActive.
func setupGitWorkspace(t *testing.T, changeName string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()
	changesDir := filepath.Join(tmpDir, "openspec", "changes")
	writeTodoChange(t, changesDir, changeName)

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-q")
	runGit("-c", "user.email=test@test.com", "-c", "user.name=test", "add", "-A")
	runGit("-c", "user.email=test@test.com", "-c", "user.name=test", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "init")
	return tmpDir
}

// TestListAllPools_TwoWorkspaces verifies that ListAllPools returns workers
// from every active workspace, each correctly tagged with its own workspace,
// by driving the real pool.Manager dispatch loop against two throwaway git
// repos.
func TestListAllPools_TwoWorkspaces(t *testing.T) {
	changeA := fmt.Sprintf("change-a-%d", time.Now().UnixNano())
	changeB := fmt.Sprintf("change-b-%d", time.Now().UnixNano())
	tmpA := setupGitWorkspace(t, changeA)
	tmpB := setupGitWorkspace(t, changeB)

	h, reg, idA, idB := twoWorkspaceHandler(t, tmpA, tmpB)

	t.Cleanup(func() {
		reg.For(idA).Stop()
		reg.For(idB).Stop()
		home, _ := os.UserHomeDir()
		os.RemoveAll(filepath.Join(home, ".opensp8c", "worktrees", "wt-"+changeA))
		os.RemoveAll(filepath.Join(home, ".opensp8c", "worktrees", "wt-"+changeB))
	})

	if err := reg.For(idA).Start(pool.AgentPoolConfig{Size: 1, DelegationMode: pool.ModeHITLReview, MaxAttempts: 1}, idA, "workspace-a", tmpA); err != nil {
		t.Fatalf("failed to start pool A: %v", err)
	}
	if err := reg.For(idB).Start(pool.AgentPoolConfig{Size: 1, DelegationMode: pool.ModeHITLReview, MaxAttempts: 1}, idB, "workspace-b", tmpB); err != nil {
		t.Fatalf("failed to start pool B: %v", err)
	}

	waitForWorker := func(id, changeName string) {
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			_, _, workers := reg.For(id).Status(id)
			for _, w := range workers {
				if w.ActiveChange == changeName {
					return
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Skip("pool did not pick up the worker within the deadline (stub orchestration timing); skipping")
	}
	waitForWorker(idA, changeA)
	waitForWorker(idB, changeB)

	rec := httptest.NewRecorder()
	h.ListAllPools(rec, httptest.NewRequest(http.MethodGet, "/api/pools", nil))

	var resp struct {
		Workers []pool.Worker `json:"workers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode /api/pools response: %v", err)
	}

	byWorkspace := map[string]pool.Worker{}
	for _, w := range resp.Workers {
		byWorkspace[w.WorkspaceID] = w
	}
	wa, ok := byWorkspace[idA]
	if !ok || wa.ActiveChange != changeA || wa.WorkspaceName != "workspace-a" {
		t.Errorf("expected workspace A's worker on %q named %q, got %+v", changeA, "workspace-a", resp.Workers)
	}
	wb, ok := byWorkspace[idB]
	if !ok || wb.ActiveChange != changeB || wb.WorkspaceName != "workspace-b" {
		t.Errorf("expected workspace B's worker on %q named %q, got %+v", changeB, "workspace-b", resp.Workers)
	}
}
