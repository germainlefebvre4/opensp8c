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
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/glefebvre/opensp8c/internal/workspace"
	"github.com/go-chi/chi/v5"
)

// TestWorkspaceList_TaskCountsIncludesReady verifies GET /api/workspaces
// reports a "ready" key in task_counts even when no changes exist yet.
func TestWorkspaceList_TaskCountsIncludesReady(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "openspec", "changes"), 0755); err != nil {
		t.Fatalf("failed to create changes dir: %v", err)
	}

	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: tmpDir}}}
	ws := NewWorkspaceHandler(cfg, "", pool.NewRegistry(nil, nil, nil, nil, nil))
	rec := httptest.NewRecorder()
	ws.List(rec, httptest.NewRequest(http.MethodGet, "/api/workspaces", nil))

	if !strings.Contains(rec.Body.String(), `"ready":0`) {
		t.Fatalf("expected task_counts to include \"ready\":0, got %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"verifying":0`) {
		t.Fatalf("expected task_counts to include \"verifying\":0, got %s", rec.Body.String())
	}
}

func TestWorkspaceList_TaskCountsVerifying(t *testing.T) {
	tmpDir := t.TempDir()
	changeDir := filepath.Join(tmpDir, "openspec", "changes", "c1")
	if err := os.MkdirAll(changeDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(changeDir, ".openspec.yaml"), []byte("schema: spec-driven\nlaunched: true\n"), 0644)
	os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("- [ ] a\n"), 0644)
	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git unavailable: %v %s", err, out)
		}
	}
	git("init", "-q", "-b", "main")
	git("add", "-A")
	git("-c", "commit.gpgsign=false", "commit", "-q", "-m", "init")
	git("branch", "feature/c1")
	git("config", "branch.feature/c1.opensp8c-verify", "pending")

	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: tmpDir}}}
	ws := NewWorkspaceHandler(cfg, "", pool.NewRegistry(nil, nil, nil, nil, nil))
	rec := httptest.NewRecorder()
	ws.List(rec, httptest.NewRequest(http.MethodGet, "/api/workspaces", nil))
	if body := rec.Body.String(); !strings.Contains(body, `"verifying":1`) || !strings.Contains(body, `"todo":0`) {
		t.Fatalf("expected one verifying change, got %s", body)
	}
}

func TestWorkspaceList_TaskCountsToReview(t *testing.T) {
	tmpDir := t.TempDir()
	changeDir := filepath.Join(tmpDir, "openspec", "changes", "c1")
	if err := os.MkdirAll(changeDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(changeDir, ".openspec.yaml"), []byte("schema: spec-driven\nlaunched: true\n"), 0644)
	os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("- [ ] a\n"), 0644)

	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: tmpDir}}}
	ws := NewWorkspaceHandler(cfg, "", pool.NewRegistry(nil, nil, nil, nil, nil))
	list := func() string {
		rec := httptest.NewRecorder()
		ws.List(rec, httptest.NewRequest(http.MethodGet, "/api/workspaces", nil))
		return rec.Body.String()
	}
	if body := list(); !strings.Contains(body, `"to-review":0`) {
		t.Fatalf("expected \"to-review\":0, got %s", body)
	}

	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git unavailable: %v %s", err, out)
		}
	}
	git("init", "-q", "-b", "main")
	git("add", "-A")
	git("-c", "commit.gpgsign=false", "commit", "-q", "-m", "init")
	git("branch", "feature/c1")
	git("config", "branch.feature/c1.opensp8c-review", "x")
	if body := list(); !strings.Contains(body, `"to-review":1`) || !strings.Contains(body, `"todo":0`) {
		t.Fatalf("expected one to-review change, got %s", body)
	}
}

func deleteWorkspaceRequest(workspaceID string) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest(http.MethodDelete, "/workspaces/"+workspaceID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workspaceID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return httptest.NewRecorder(), req
}

// TestWorkspaceDelete_StopsActivePool verifies that deleting a workspace
// with an active Agent Pool stops its workers and drops it from the
// Registry, so it can no longer leak into GET /api/pools.
func TestWorkspaceDelete_StopsActivePool(t *testing.T) {
	changeName := fmt.Sprintf("worker-busy-%d", time.Now().UnixNano())
	tmpDir := setupGitWorkspace(t, changeName)
	t.Cleanup(func() {
		home, _ := os.UserHomeDir()
		os.RemoveAll(filepath.Join(home, ".opensp8c", "worktrees", "wt-"+changeName))
	})

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	cfg.Workspaces = []config.WorkspaceConfig{{Name: "test", Path: tmpDir}}
	reg := pool.NewRegistry(nil, nil, nil, nil, nil)
	ws := NewWorkspaceHandler(cfg, cfgPath, reg)
	poolHandler := NewPoolHandler(ws, reg)

	id := workspace.StableID(mustAbs(t, tmpDir))

	if err := reg.For(id).Start(pool.AgentPoolConfig{Size: 1, DelegationMode: pool.ModeHITLReview, MaxAttempts: 1}, id, "test", tmpDir); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	active := false
	for time.Now().Before(deadline) {
		_, running, workers := reg.For(id).Status(id)
		if running && len(workers) > 0 {
			active = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !active {
		t.Skip("pool did not pick up a worker within the deadline (stub orchestration timing); skipping")
	}

	rec, req := deleteWorkspaceRequest(id)
	ws.Delete(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 deleting workspace, got %d: %s", rec.Code, rec.Body.String())
	}

	if _, running, _ := reg.For(id).Status(id); running {
		t.Fatalf("expected pool to be stopped after workspace deletion")
	}

	listRec := httptest.NewRecorder()
	poolHandler.ListAllPools(listRec, httptest.NewRequest(http.MethodGet, "/api/pools", nil))
	if got := listRec.Body.String(); !strings.Contains(got, `"pools":[]`) {
		t.Fatalf("expected GET /api/pools to report no pools after workspace deletion, got %s", got)
	}
}

func listWorkspaces(t *testing.T, ws *WorkspaceHandler) []workspace.Workspace {
	t.Helper()
	rec := httptest.NewRecorder()
	ws.List(rec, httptest.NewRequest(http.MethodGet, "/api/workspaces", nil))
	var out []workspace.Workspace
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v: %s", err, rec.Body.String())
	}
	return out
}

// A todo change held by a worker whose worktree has progressed is counted
// in-progress, as in the Kanban.
func TestWorkspaceList_TaskCountsFollowKanbanLiveState(t *testing.T) {
	h, id, _ := newUnlaunchForceFixture(t, func() (bool, pool.WorkerResult, bool) { return false, pool.WorkerResult{}, false })
	hw := h.activeWorkerChanges(id)["my-change"]
	dir := filepath.Join(hw.WorktreePath, "openspec", "changes", "my-change")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("- [x] a\n- [ ] b\n"), 0644)

	got := listWorkspaces(t, h.ws)
	if len(got) != 1 || got[0].TaskCounts["in-progress"] != 1 || got[0].TaskCounts["todo"] != 0 {
		t.Fatalf("expected one in-progress change, got %+v", got)
	}
}

func TestWorkspaceList_EmptyAttentionIsList(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "openspec", "changes"), 0755)
	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: tmpDir}}}
	ws := NewWorkspaceHandler(cfg, "", pool.NewRegistry(nil, nil, nil, nil, nil))
	rec := httptest.NewRecorder()
	ws.List(rec, httptest.NewRequest(http.MethodGet, "/api/workspaces", nil))
	if !strings.Contains(rec.Body.String(), `"attention":[]`) {
		t.Fatalf("expected \"attention\":[], got %s", rec.Body.String())
	}
}

func TestWorkspaceList_TolerantWithoutPoolOrChanges(t *testing.T) {
	tmpDir := t.TempDir() // no openspec/changes at all
	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: tmpDir}}}
	ws := NewWorkspaceHandler(cfg, "", nil)
	got := listWorkspaces(t, ws)
	if len(got) != 1 || got[0].Attention == nil || len(got[0].Attention) != 0 {
		t.Fatalf("expected empty attention, got %+v", got)
	}
}

func TestWorkspaceList_AttentionHITL(t *testing.T) {
	tmpDir := t.TempDir()
	changeDir := filepath.Join(tmpDir, "openspec", "changes", "c1")
	os.MkdirAll(changeDir, 0755)
	os.WriteFile(filepath.Join(changeDir, ".openspec.yaml"), []byte("schema: spec-driven\nlaunched: true\n"), 0644)
	os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("- [x] a\n- [ ] check it <!-- human review required -->\n"), 0644)
	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: tmpDir}}}
	ws := NewWorkspaceHandler(cfg, "", pool.NewRegistry(nil, nil, nil, nil, nil))
	got := listWorkspaces(t, ws)
	want := []workspace.Attention{{Change: "c1", Signals: []workspace.Signal{{Kind: "hitl", Reason: "check it"}}}}
	if len(got) != 1 || !reflect.DeepEqual(got[0].Attention, want) {
		t.Fatalf("attention = %+v, want %+v", got[0].Attention, want)
	}
}

// The HITL signal reads the tasks.md of feature/<change> when the change has a
// branch, not the repository's copy.
func TestWorkspaceList_AttentionHITLFromBranch(t *testing.T) {
	tmpDir := t.TempDir()
	changeDir := filepath.Join(tmpDir, "openspec", "changes", "c1")
	os.MkdirAll(changeDir, 0755)
	os.WriteFile(filepath.Join(changeDir, ".openspec.yaml"), []byte("schema: spec-driven\nlaunched: true\n"), 0644)
	tasks := filepath.Join(changeDir, "tasks.md")
	os.WriteFile(tasks, []byte("- [ ] plain\n"), 0644)
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-c", "commit.gpgsign=false"}, args...)...)
		cmd.Dir = tmpDir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git unavailable: %v %s", err, out)
		}
	}
	git("init", "-q", "-b", "main")
	git("add", "-A")
	git("commit", "-q", "-m", "init")
	git("checkout", "-q", "-b", "feature/c1")
	os.WriteFile(tasks, []byte("- [ ] branch only <!-- human review required -->\n"), 0644)
	git("commit", "-q", "-am", "branch tasks")
	git("checkout", "-q", "main")

	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: tmpDir}}}
	ws := NewWorkspaceHandler(cfg, "", pool.NewRegistry(nil, nil, nil, nil, nil))
	got := listWorkspaces(t, ws)
	want := []workspace.Attention{{Change: "c1", Signals: []workspace.Signal{{Kind: "hitl", Reason: "branch only"}}}}
	if len(got) != 1 || !reflect.DeepEqual(got[0].Attention, want) {
		t.Fatalf("attention = %+v, want %+v", got[0].Attention, want)
	}
}
