package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/glefebvre/opensp8c/internal/workspace"
)

const nineOfTen = "- [x] 1\n- [x] 2\n- [x] 3\n- [x] 4\n- [x] 5\n- [x] 6\n- [x] 7\n- [x] 8\n- [x] 9\n- [ ] 10 manual\n"

// taskWorkerFixture is a workspace holding change "c" (9/10 in the main
// repository), with a running pool whose single worker, seeded with the given
// status, holds a worktree whose tasks.md is worktreeTasks ("" = none).
type taskWorkerFixture struct {
	task        *TaskHandler
	kanban      *KanbanHandler
	workspaceID string
	mainTasks   string
	wtTasks     string
}

func newTaskWorkerFixture(t *testing.T, status pool.WorkerStatus, worktreeTasks string, seed bool) *taskWorkerFixture {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	wsPath := t.TempDir()
	changeDir := filepath.Join(wsPath, "openspec", "changes", "c")
	if err := os.MkdirAll(changeDir, 0755); err != nil {
		t.Fatal(err)
	}
	mainTasks := filepath.Join(changeDir, "tasks.md")
	if err := os.WriteFile(mainTasks, []byte(nineOfTen), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, ".openspec.yaml"), []byte("schema: spec-driven\ncreated: \"2024-01-01\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	reg := pool.NewRegistry(nil, nil, nil, nil, nil)
	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: wsPath}}}
	ws := NewWorkspaceHandler(cfg, "", reg)
	abs, _ := filepath.Abs(wsPath)
	f := &taskWorkerFixture{
		task:        NewTaskHandler(ws, nil, reg),
		kanban:      NewKanbanHandler(ws, nil, reg, nil, nil, nil, ""),
		workspaceID: workspace.StableID(abs),
		mainTasks:   mainTasks,
	}

	if seed {
		wtDir := t.TempDir()
		f.wtTasks = filepath.Join(wtDir, "openspec", "changes", "c", "tasks.md")
		if worktreeTasks != "" {
			if err := os.MkdirAll(filepath.Dir(f.wtTasks), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(f.wtTasks, []byte(worktreeTasks), 0644); err != nil {
				t.Fatal(err)
			}
		}
		mgr := reg.For(f.workspaceID)
		// The change is not launched, so the dispatcher leaves it alone.
		if err := mgr.Start(pool.AgentPoolConfig{Size: 1, DelegationMode: pool.ModeHITLReview, MaxAttempts: 1}, f.workspaceID, "test", wsPath); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(mgr.Stop)
		pool.SeedWorkerForTest(mgr, pool.Worker{ID: 3, ActiveChange: "c", Status: status, WorktreePath: wtDir, BlockedReason: "Validation réussie mais tâches restantes incomplètes (9/10) dans tasks.md"})
	}
	return f
}

func (f *taskWorkerFixture) toggle(t *testing.T, index string) {
	t.Helper()
	rec, req := patchTaskRequest(f.workspaceID, "c", index)
	f.task.PatchTask(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("toggle: %d %s", rec.Code, rec.Body.String())
	}
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestPatchTask_TargetsWorkerWorktree(t *testing.T) {
	for _, status := range []pool.WorkerStatus{pool.StatusPaused, pool.StatusWorking} {
		t.Run(string(status), func(t *testing.T) {
			f := newTaskWorkerFixture(t, status, nineOfTen, true)
			f.toggle(t, "9")
			if got := readFileString(t, f.wtTasks); !strings.Contains(got, "- [x] 10 manual") {
				t.Fatalf("worktree not toggled:\n%s", got)
			}
			if got := readFileString(t, f.mainTasks); got != nineOfTen {
				t.Fatalf("main repository must be untouched:\n%s", got)
			}
		})
	}
}

func TestPatchTask_WithoutWorkerTargetsMainRepo(t *testing.T) {
	f := newTaskWorkerFixture(t, "", "", false)
	f.toggle(t, "9")
	if got := readFileString(t, f.mainTasks); !strings.Contains(got, "- [x] 10 manual") {
		t.Fatalf("main not toggled:\n%s", got)
	}
}

func TestPatchTask_WorktreeWithoutUsableListFallsBackToMain(t *testing.T) {
	for name, content := range map[string]string{"absent": "", "empty": "# Tasks\n"} {
		t.Run(name, func(t *testing.T) {
			f := newTaskWorkerFixture(t, pool.StatusPaused, content, true)
			f.toggle(t, "9")
			if got := readFileString(t, f.mainTasks); !strings.Contains(got, "- [x] 10 manual") {
				t.Fatalf("main not toggled:\n%s", got)
			}
		})
	}
}

// Backend end-to-end: the last task checked on a paused change reaches the
// detail read, with the main repository left alone.
func TestPatchTask_PausedLastTaskVisibleInDetail(t *testing.T) {
	f := newTaskWorkerFixture(t, pool.StatusPaused, nineOfTen, true)
	f.toggle(t, "9")

	rec, req := launchRequest("GET", f.workspaceID, "c", "", nil)
	f.kanban.GetChange(rec, req)
	var d openspec.ChangeDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	if d.TasksDone != 10 || d.TasksTotal != 10 || !d.WorkerPaused {
		t.Fatalf("detail: %+v", d.Change)
	}
	if got := readFileString(t, f.mainTasks); got != nineOfTen {
		t.Fatalf("main repository must be untouched:\n%s", got)
	}
}

func TestKanban_ExposesWorkerIDAndBlockedReason(t *testing.T) {
	// Paused: id on the list, id and reason on the detail.
	f := newTaskWorkerFixture(t, pool.StatusPaused, nineOfTen, true)
	rec, req := launchRequest("GET", f.workspaceID, "", "", nil)
	f.kanban.ListChanges(rec, req)
	var list []openspec.Change
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) != 1 || list[0].WorkerID == nil || *list[0].WorkerID != 3 {
		t.Fatalf("list: %v %+v", err, list)
	}
	rec, req = launchRequest("GET", f.workspaceID, "c", "", nil)
	f.kanban.GetChange(rec, req)
	var d openspec.ChangeDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	if d.WorkerID == nil || *d.WorkerID != 3 || !strings.Contains(d.WorkerBlockedReason, "9/10") {
		t.Fatalf("detail: %+v reason=%q", d.WorkerID, d.WorkerBlockedReason)
	}

	// Active: id only, no reason.
	f = newTaskWorkerFixture(t, pool.StatusWorking, nineOfTen, true)
	rec, req = launchRequest("GET", f.workspaceID, "c", "", nil)
	f.kanban.GetChange(rec, req)
	d = openspec.ChangeDetail{}
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	if d.WorkerID == nil || d.WorkerBlockedReason != "" || strings.Contains(rec.Body.String(), "worker_blocked_reason") {
		t.Fatalf("active detail: %s", rec.Body.String())
	}

	// No worker: both absent.
	f = newTaskWorkerFixture(t, "", "", false)
	rec, req = launchRequest("GET", f.workspaceID, "c", "", nil)
	f.kanban.GetChange(rec, req)
	if strings.Contains(rec.Body.String(), "worker_id") || strings.Contains(rec.Body.String(), "worker_blocked_reason") {
		t.Fatalf("no worker: %s", rec.Body.String())
	}
	rec, req = launchRequest("GET", f.workspaceID, "", "", nil)
	f.kanban.ListChanges(rec, req)
	if strings.Contains(rec.Body.String(), "worker_id") {
		t.Fatalf("no worker list: %s", rec.Body.String())
	}
}
