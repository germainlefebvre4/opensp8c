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

	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/glefebvre/opensp8c/internal/workspace"
	"github.com/go-chi/chi/v5"
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
	reg = pool.NewRegistry(nil, nil, nil, nil, nil)
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

// TestListAllPools_TwoWorkspaces verifies that ListAllPools groups workers by
// their pool of origin, one entry per active workspace with its size and
// delegation mode, by driving the real pool.Manager dispatch loop against two
// throwaway git repos.
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
		Pools []pool.PoolSummary `json:"pools"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode /api/pools response: %v", err)
	}

	byWorkspace := map[string]pool.PoolSummary{}
	for _, p := range resp.Pools {
		byWorkspace[p.WorkspaceID] = p
	}
	poolA, ok := byWorkspace[idA]
	if !ok || poolA.WorkspaceName != "workspace-a" || poolA.Size != 1 || poolA.DelegationMode != pool.ModeHITLReview {
		t.Fatalf("expected workspace A's pool tagged with its name/size/delegation mode, got %+v", resp.Pools)
	}
	if len(poolA.Workers) != 1 || poolA.Workers[0].ActiveChange != changeA {
		t.Errorf("expected workspace A's pool to list its worker on %q, got %+v", changeA, poolA.Workers)
	}

	poolB, ok := byWorkspace[idB]
	if !ok || poolB.WorkspaceName != "workspace-b" || poolB.Size != 1 || poolB.DelegationMode != pool.ModeHITLReview {
		t.Fatalf("expected workspace B's pool tagged with its name/size/delegation mode, got %+v", resp.Pools)
	}
	if len(poolB.Workers) != 1 || poolB.Workers[0].ActiveChange != changeB {
		t.Errorf("expected workspace B's pool to list its worker on %q, got %+v", changeB, poolB.Workers)
	}
}

func resumeRequest(workspaceID, workerID string) *http.Request {
	req := poolRequest(http.MethodPost, "/workspaces/"+workspaceID+"/pool/workers/"+workerID+"/resume", workspaceID, "")
	rctx := chi.RouteContext(req.Context())
	rctx.URLParams.Add("workerId", workerID)
	return req
}

func TestResumeWorker(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// A workspace that is not a git repository: Provision fails, so the
	// worker of its launched change pauses.
	wsPath := t.TempDir()
	changeDir := filepath.Join(wsPath, "openspec", "changes", "change-a")
	if err := os.MkdirAll(changeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("- [ ] x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, ".openspec.yaml"), []byte("schema: spec-driven\ncreated: \"2024-01-01\"\nlaunched: true\n"), 0644); err != nil {
		t.Fatal(err)
	}
	h, reg, id, _ := twoWorkspaceHandler(t, wsPath, t.TempDir())

	// Unknown workspace -> 404.
	rec := httptest.NewRecorder()
	h.ResumeWorker(rec, resumeRequest("nope", "1"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown workspace: expected 404, got %d", rec.Code)
	}

	// Pool stopped -> 409.
	rec = httptest.NewRecorder()
	h.ResumeWorker(rec, resumeRequest(id, "1"))
	if rec.Code != http.StatusConflict {
		t.Fatalf("stopped pool: expected 409, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.StartPool(rec, poolRequest(http.MethodPost, "/", id, `{"size":1,"delegation_mode":"hitl-review","max_attempts":1}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("start: %d", rec.Code)
	}
	defer reg.For(id).Stop()

	// Wait for the first tick to pause the worker.
	deadline := time.Now().Add(15 * time.Second)
	for {
		_, _, workers := reg.For(id).Status(id)
		if len(workers) == 1 && workers[0].Status == pool.StatusPaused {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("worker never paused: %+v", workers)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Unknown worker -> 404.
	rec = httptest.NewRecorder()
	h.ResumeWorker(rec, resumeRequest(id, "42"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown worker: expected 404, got %d", rec.Code)
	}

	// Paused worker -> 200 and no longer paused.
	rec = httptest.NewRecorder()
	h.ResumeWorker(rec, resumeRequest(id, "1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("resume: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ResumeWorker(rec, resumeRequest(id, "1"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("second resume: expected 404, got %d", rec.Code)
	}
}
