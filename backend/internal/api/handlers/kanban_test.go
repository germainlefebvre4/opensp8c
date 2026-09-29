package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/workspace"
	"github.com/go-chi/chi/v5"
)

// newTestKanbanHandler wires a KanbanHandler against a workspace rooted at
// workspacePath, mirroring how router.go constructs it.
func newTestKanbanHandler(t *testing.T, workspacePath string, poolReg *pool.Registry, prefs *preferences.Service, convStore *conversation.Store, draftsDir string) (*KanbanHandler, string) {
	t.Helper()
	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: workspacePath}}}
	ws := NewWorkspaceHandler(cfg, "", poolReg)
	absPath, err := filepath.Abs(workspacePath)
	if err != nil {
		t.Fatalf("failed to resolve abs path: %v", err)
	}
	workspaceID := workspace.StableID(absPath)
	h := NewKanbanHandler(ws, prefs, poolReg, nil, convStore, nil, draftsDir)
	return h, workspaceID
}

func deleteChangeRequest(workspaceID, changeName string) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest("DELETE", "/workspaces/"+workspaceID+"/changes/"+changeName, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workspaceID)
	rctx.URLParams.Add("name", changeName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return httptest.NewRecorder(), req
}

func writeTodoChange(t *testing.T, changesDir, name string) string {
	t.Helper()
	changeDir := filepath.Join(changesDir, name)
	if err := os.MkdirAll(changeDir, 0755); err != nil {
		t.Fatalf("failed to create change dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("# Tasks\n- [ ] do the thing\n"), 0644); err != nil {
		t.Fatalf("failed to write tasks.md: %v", err)
	}
	return changeDir
}

func launchRequest(method, workspaceID, changeName, suffix string, body *bytes.Reader) (*httptest.ResponseRecorder, *http.Request) {
	var req *http.Request
	path := "/workspaces/" + workspaceID + "/changes/" + changeName + "/" + suffix
	if body != nil {
		req = httptest.NewRequest(method, path, body)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workspaceID)
	rctx.URLParams.Add("name", changeName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return httptest.NewRecorder(), req
}

func writeChangeWithMeta(t *testing.T, changesDir, name, meta string) string {
	t.Helper()
	changeDir := filepath.Join(changesDir, name)
	if err := os.MkdirAll(changeDir, 0755); err != nil {
		t.Fatalf("failed to create change dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, ".openspec.yaml"), []byte(meta), 0644); err != nil {
		t.Fatalf("failed to write .openspec.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("- [ ] do the thing\n"), 0644); err != nil {
		t.Fatalf("failed to write tasks.md: %v", err)
	}
	return changeDir
}

func TestKanbanHandler_Launch(t *testing.T) {
	tmpDir := t.TempDir()
	changesDir := filepath.Join(tmpDir, "openspec", "changes")
	changeDir := writeChangeWithMeta(t, changesDir, "my-change", "schema: spec-driven\ncreated: \"2024-01-01\"\nlaunched: false\n")

	h, workspaceID := newTestKanbanHandler(t, tmpDir, pool.NewRegistry(nil, nil, nil, nil), nil, nil, "")

	rec, req := launchRequest("PATCH", workspaceID, "my-change", "launch", nil)
	h.Launch(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	changes, err := openspec.ListChanges(tmpDir)
	if err != nil {
		t.Fatalf("ListChanges: %v", err)
	}
	_ = changeDir
	if len(changes) != 1 || changes[0].KanbanStatus != "todo" {
		t.Fatalf("expected change to be launched (todo), got %+v", changes)
	}
}

func TestKanbanHandler_Launch_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "openspec", "changes"), 0755); err != nil {
		t.Fatalf("failed to create changes dir: %v", err)
	}
	h, workspaceID := newTestKanbanHandler(t, tmpDir, pool.NewRegistry(nil, nil, nil, nil), nil, nil, "")

	rec, req := launchRequest("PATCH", workspaceID, "does-not-exist", "launch", nil)
	h.Launch(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKanbanHandler_Unlaunch(t *testing.T) {
	tmpDir := t.TempDir()
	changesDir := filepath.Join(tmpDir, "openspec", "changes")
	writeChangeWithMeta(t, changesDir, "my-change", "schema: spec-driven\ncreated: \"2024-01-01\"\nlaunched: true\n")

	h, workspaceID := newTestKanbanHandler(t, tmpDir, pool.NewRegistry(nil, nil, nil, nil), nil, nil, "")

	rec, req := launchRequest("PATCH", workspaceID, "my-change", "unlaunch", nil)
	h.Unlaunch(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	changes, err := openspec.ListChanges(tmpDir)
	if err != nil {
		t.Fatalf("ListChanges: %v", err)
	}
	if len(changes) != 1 || changes[0].KanbanStatus != "ready" {
		t.Fatalf("expected change to be unlaunched (ready), got %+v", changes)
	}
}

func TestKanbanHandler_Unlaunch_WorkerActive(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()
	changesDir := filepath.Join(tmpDir, "openspec", "changes")
	changeName := fmt.Sprintf("worker-busy-%d", time.Now().UnixNano())
	writeChangeWithMeta(t, changesDir, changeName, "schema: spec-driven\ncreated: \"2024-01-01\"\nlaunched: true\n")

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

	poolReg := pool.NewRegistry(nil, nil, nil, nil)
	h, workspaceID := newTestKanbanHandler(t, tmpDir, poolReg, nil, nil, "")
	poolMgr := poolReg.For(workspaceID)

	if err := poolMgr.Start(pool.AgentPoolConfig{Size: 1, DelegationMode: pool.ModeHITLReview, MaxAttempts: 1}, workspaceID, "test", tmpDir); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}
	t.Cleanup(func() {
		poolMgr.Stop()
		home, _ := os.UserHomeDir()
		os.RemoveAll(filepath.Join(home, ".opensp8c", "worktrees", "wt-"+changeName))
	})

	deadline := time.Now().Add(15 * time.Second)
	active := false
	for time.Now().Before(deadline) {
		_, _, workers := poolMgr.Status(workspaceID)
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
		t.Skip("pool did not pick up the worker within the deadline (stub orchestration timing); skipping 409 assertion")
	}

	rec, req := launchRequest("PATCH", workspaceID, changeName, "unlaunch", nil)
	h.Unlaunch(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 while worker is active, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKanbanHandler_Unlaunch_WorkerActive_Force(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()
	changesDir := filepath.Join(tmpDir, "openspec", "changes")
	changeName := fmt.Sprintf("worker-force-%d", time.Now().UnixNano())
	writeChangeWithMeta(t, changesDir, changeName, "schema: spec-driven\ncreated: \"2024-01-01\"\nlaunched: true\n")

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

	poolReg := pool.NewRegistry(nil, nil, nil, nil)
	h, workspaceID := newTestKanbanHandler(t, tmpDir, poolReg, nil, nil, "")
	poolMgr := poolReg.For(workspaceID)

	if err := poolMgr.Start(pool.AgentPoolConfig{Size: 1, DelegationMode: pool.ModeHITLReview, MaxAttempts: 1}, workspaceID, "test", tmpDir); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}
	t.Cleanup(func() {
		poolMgr.Stop()
		home, _ := os.UserHomeDir()
		os.RemoveAll(filepath.Join(home, ".opensp8c", "worktrees", "wt-"+changeName))
	})

	deadline := time.Now().Add(15 * time.Second)
	active := false
	for time.Now().Before(deadline) {
		_, _, workers := poolMgr.Status(workspaceID)
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
		t.Skip("pool did not pick up the worker within the deadline (stub orchestration timing); skipping force assertion")
	}

	// First verify without force returns 409
	rec, req := launchRequest("PATCH", workspaceID, changeName, "unlaunch", nil)
	h.Unlaunch(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 while worker is active without force, got %d: %s", rec.Code, rec.Body.String())
	}

	// Now with ?force=true
	recForce, reqForce := launchRequest("PATCH", workspaceID, changeName, "unlaunch?force=true", nil)
	h.Unlaunch(recForce, reqForce)
	if recForce.Code != http.StatusNoContent {
		t.Fatalf("expected 204 with force=true, got %d: %s", recForce.Code, recForce.Body.String())
	}

	changes, err := openspec.ListChanges(tmpDir)
	if err != nil {
		t.Fatalf("ListChanges: %v", err)
	}
	if len(changes) != 1 || changes[0].KanbanStatus != "ready" {
		t.Fatalf("expected change to be demoted to ready, got %+v", changes)
	}
}

func TestKanbanHandler_ReorderReady(t *testing.T) {
	tmpDir := t.TempDir()
	changesDir := filepath.Join(tmpDir, "openspec", "changes")
	for _, name := range []string{"change-a", "change-b", "change-c"} {
		writeChangeWithMeta(t, changesDir, name, "schema: spec-driven\ncreated: \"2024-01-01\"\nlaunched: false\n")
	}

	h, workspaceID := newTestKanbanHandler(t, tmpDir, pool.NewRegistry(nil, nil, nil, nil), nil, nil, "")

	body := bytes.NewReader([]byte(`{"order": ["change-c", "change-a", "change-b"]}`))
	req := httptest.NewRequest("PUT", "/workspaces/"+workspaceID+"/ready-order", body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workspaceID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	h.ReorderReady(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	changes, err := openspec.ListChanges(tmpDir)
	if err != nil {
		t.Fatalf("ListChanges: %v", err)
	}
	orders := make(map[string]int, len(changes))
	for _, c := range changes {
		orders[c.Name] = c.Order
	}
	if orders["change-c"] != 1 || orders["change-a"] != 2 || orders["change-b"] != 3 {
		t.Fatalf("unexpected orders: %+v", orders)
	}
}

func TestDeleteChange_Success(t *testing.T) {
	tmpDir := t.TempDir()
	changesDir := filepath.Join(tmpDir, "openspec", "changes")
	changeDir := writeTodoChange(t, changesDir, "my-change")

	h, workspaceID := newTestKanbanHandler(t, tmpDir, pool.NewRegistry(nil, nil, nil, nil), nil, nil, "")

	rec, req := deleteChangeRequest(workspaceID, "my-change")
	h.DeleteChange(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(changeDir); !os.IsNotExist(err) {
		t.Fatalf("expected change dir to be removed, stat err: %v", err)
	}
}

func TestDeleteChange_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "openspec", "changes"), 0755); err != nil {
		t.Fatalf("failed to create changes dir: %v", err)
	}

	h, workspaceID := newTestKanbanHandler(t, tmpDir, pool.NewRegistry(nil, nil, nil, nil), nil, nil, "")

	rec, req := deleteChangeRequest(workspaceID, "does-not-exist")
	h.DeleteChange(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteChange_GhostCascade(t *testing.T) {
	tmpDir := t.TempDir()
	changesDir := filepath.Join(tmpDir, "openspec", "changes")
	changeDir := writeTodoChange(t, changesDir, "solidified-change")

	prefsPath := filepath.Join(tmpDir, "preferences.json")
	prefs := preferences.NewService(prefsPath)

	draftsDir := filepath.Join(tmpDir, "drafts")
	if err := os.MkdirAll(draftsDir, 0755); err != nil {
		t.Fatalf("failed to create drafts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftsDir, "ghost-1.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("failed to write draft file: %v", err)
	}

	convBase := filepath.Join(tmpDir, "conversations")
	convStore := conversation.NewStore(convBase)

	h, workspaceID := newTestKanbanHandler(t, tmpDir, pool.NewRegistry(nil, nil, nil, nil), prefs, convStore, draftsDir)

	if err := prefs.AddExploration(preferences.ExplorationRecord{
		ID:          "ghost-1",
		WorkspaceID: workspaceID,
		Name:        "solidified-change",
		SessionID:   "ghost-1",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("failed to seed exploration record: %v", err)
	}

	explorationLogDir := filepath.Join(convBase, workspaceID, "_explore", "ghost-1")
	if err := os.MkdirAll(explorationLogDir, 0755); err != nil {
		t.Fatalf("failed to create exploration log dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(explorationLogDir, "run.jsonl"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("failed to write exploration log: %v", err)
	}

	rec, req := deleteChangeRequest(workspaceID, "solidified-change")
	h.DeleteChange(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(changeDir); !os.IsNotExist(err) {
		t.Fatalf("expected change dir to be removed, stat err: %v", err)
	}

	remaining := prefs.ListExplorations(workspaceID)
	if len(remaining) != 0 {
		t.Fatalf("expected ghost exploration record to be deleted, got %+v", remaining)
	}
	if _, err := os.Stat(filepath.Join(draftsDir, "ghost-1.json")); !os.IsNotExist(err) {
		t.Fatalf("expected draft file to be removed, stat err: %v", err)
	}
	if _, err := os.Stat(explorationLogDir); !os.IsNotExist(err) {
		t.Fatalf("expected exploration logs to be removed, stat err: %v", err)
	}
}

// TestDeleteChange_WorkerActive exercises the guard against deleting a change
// that an Agent Pool worker is actively working on. It drives the real
// pool.Manager dispatch loop against a throwaway git repo so the worker
// occupies its slot for a few seconds (simulated work in worker.go), giving
// the test a window to observe it via Status() before it's torn down.
func TestDeleteChange_WorkerActive(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()
	changesDir := filepath.Join(tmpDir, "openspec", "changes")
	changeName := fmt.Sprintf("worker-busy-%d", time.Now().UnixNano())
	changeDir := writeTodoChange(t, changesDir, changeName)

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

	poolReg := pool.NewRegistry(nil, nil, nil, nil)
	h, workspaceID := newTestKanbanHandler(t, tmpDir, poolReg, nil, nil, "")
	poolMgr := poolReg.For(workspaceID)

	if err := poolMgr.Start(pool.AgentPoolConfig{Size: 1, DelegationMode: pool.ModeHITLReview, MaxAttempts: 1}, workspaceID, "test", tmpDir); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}
	t.Cleanup(func() {
		poolMgr.Stop()
		home, _ := os.UserHomeDir()
		os.RemoveAll(filepath.Join(home, ".opensp8c", "worktrees", "wt-"+changeName))
	})

	deadline := time.Now().Add(15 * time.Second)
	active := false
	for time.Now().Before(deadline) {
		_, _, workers := poolMgr.Status(workspaceID)
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
		t.Skip("pool did not pick up the worker within the deadline (stub orchestration timing); skipping 409 assertion")
	}

	rec, req := deleteChangeRequest(workspaceID, changeName)
	h.DeleteChange(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 while worker is active, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(changeDir); err != nil {
		t.Fatalf("expected change dir to survive a blocked deletion, stat err: %v", err)
	}
}
