package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/glefebvre/opensp8c/internal/watcher"
	"github.com/glefebvre/opensp8c/internal/workspace"
)

const (
	branchChange   = "c"
	branchTasks8of = "- [x] 1\n- [x] 2\n- [x] 3\n- [x] 4\n- [x] 5\n- [x] 6\n- [x] 7\n- [x] 8\n- [ ] 9 manual\n- [ ] 10\n"
	mainTasks0of10 = "- [ ] 1\n- [ ] 2\n- [ ] 3\n- [ ] 4\n- [ ] 5\n- [ ] 6\n- [ ] 7\n- [ ] 8\n- [ ] 9 manual\n- [ ] 10\n"
)

type branchFixture struct {
	repo        string
	workspaceID string
	reg         *pool.Registry
	kanban      *KanbanHandler
	task        *TaskHandler
	act         *activity.Store
	bc          *recordingBroadcaster
}

type recordingBroadcaster struct {
	mu     sync.Mutex
	events []watcher.Event
}

func (b *recordingBroadcaster) Broadcast(_ string, ev watcher.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, ev)
}

func bgit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// newBranchFixture builds a git workspace with change "c" whose main tasks.md
// is mainTasks, plus, when branchTasks is not empty, a branch feature/c that
// commits branchTasks. review lifts the review marker on that branch.
func newBranchFixture(t *testing.T, launched bool, mainTasks, branchTasks string, review bool) *branchFixture {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	dir := filepath.Join(repo, "openspec", "changes", branchChange)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := "schema: spec-driven\ncreated: \"2024-01-01\"\n"
	if !launched {
		meta += "launched: false\n"
	}
	for name, content := range map[string]string{".openspec.yaml": meta, "tasks.md": mainTasks} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	bgit(t, repo, "init", "-q", "-b", "main")
	bgit(t, repo, "config", "user.name", "t")
	bgit(t, repo, "config", "user.email", "t@t")
	bgit(t, repo, "add", "-A")
	bgit(t, repo, "commit", "-q", "-m", "init")
	if branchTasks != "" {
		bgit(t, repo, "checkout", "-q", "-b", "feature/"+branchChange)
		if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(branchTasks), 0o644); err != nil {
			t.Fatal(err)
		}
		bgit(t, repo, "commit", "-q", "-am", "work")
		bgit(t, repo, "checkout", "-q", "main")
		if review {
			bgit(t, repo, "config", openspec.ReviewKey(branchChange), "2026-01-01T00:00:00Z")
		}
	}

	bc := &recordingBroadcaster{}
	reg := pool.NewRegistry(bc, nil, nil, nil, nil)
	abs, _ := filepath.Abs(repo)
	id := workspace.StableID(abs)
	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: repo}}}
	ws := NewWorkspaceHandler(cfg, "", reg)
	act := activity.NewStore(filepath.Join(t.TempDir(), "activity"), nil)
	return &branchFixture{
		repo: repo, workspaceID: id, reg: reg, bc: bc, act: act,
		kanban: NewKanbanHandler(ws, nil, reg, nil, nil, nil, ""),
		task:   NewTaskHandler(ws, act, reg),
	}
}

func (f *branchFixture) detail(t *testing.T) openspec.ChangeDetail {
	t.Helper()
	rec, req := launchRequest("GET", f.workspaceID, branchChange, "", nil)
	f.kanban.GetChange(rec, req)
	var d openspec.ChangeDetail
	if err := json.NewDecoder(rec.Body).Decode(&d); err != nil {
		t.Fatal(err)
	}
	return d
}

func (f *branchFixture) toggle(index string) *http_ {
	rec, req := patchTaskRequest(f.workspaceID, branchChange, index)
	f.task.PatchTask(rec, req)
	return &http_{rec.Code, rec.Body.String()}
}

type http_ struct {
	code int
	body string
}

func (f *branchFixture) mainFile(t *testing.T) string {
	t.Helper()
	return readFileString(t, filepath.Join(f.repo, "openspec", "changes", branchChange, "tasks.md"))
}

func (f *branchFixture) noWorktree(t *testing.T) {
	t.Helper()
	if out := bgit(t, f.repo, "worktree", "list", "--porcelain"); strings.Count(out, "worktree ") != 1 {
		t.Fatalf("no worktree must be created:\n%s", out)
	}
}

func TestBranchTasks_ReviewReadsBranchKeepsColumn(t *testing.T) {
	f := newBranchFixture(t, true, mainTasks0of10, branchTasks8of, true)
	heads := bgit(t, f.repo, "rev-parse", "feature/c", "main")

	c := listedChange(t, f.kanban, f.workspaceID, branchChange)
	if c.KanbanStatus != "to-review" || c.TasksDone != 8 || c.TasksTotal != 10 || !c.HasBranch {
		t.Errorf("list: %s %d/%d branch=%v", c.KanbanStatus, c.TasksDone, c.TasksTotal, c.HasBranch)
	}
	d := f.detail(t)
	if d.TasksDone != 8 || len(d.Tasks) != 10 || !d.Tasks[7].Done || d.Tasks[8].Done || !d.HasBranch {
		t.Errorf("detail: %d/%d tasks=%d", d.TasksDone, d.TasksTotal, len(d.Tasks))
	}
	f.noWorktree(t)
	if got := bgit(t, f.repo, "rev-parse", "feature/c", "main"); got != heads {
		t.Errorf("GET must not commit: %s != %s", got, heads)
	}
}

func TestBranchTasks_BranchAloneStaysReady(t *testing.T) {
	f := newBranchFixture(t, false, mainTasks0of10, "- [x] a\n- [x] b\n", false)
	c := listedChange(t, f.kanban, f.workspaceID, branchChange)
	if c.KanbanStatus != "ready" || c.TasksDone != 2 || c.TasksTotal != 2 {
		t.Errorf("got %s %d/%d, want ready 2/2", c.KanbanStatus, c.TasksDone, c.TasksTotal)
	}
}

func TestBranchTasks_BranchWithoutTaskListKeepsMain(t *testing.T) {
	f := newBranchFixture(t, true, "- [x] a\n- [ ] b\n", "# nothing\n", true)
	c := listedChange(t, f.kanban, f.workspaceID, branchChange)
	if c.TasksDone != 1 || c.TasksTotal != 2 || !c.HasBranch {
		t.Errorf("got %d/%d branch=%v", c.TasksDone, c.TasksTotal, c.HasBranch)
	}
	if d := f.detail(t); len(d.Tasks) != 2 || d.TasksDone != 1 {
		t.Errorf("detail %d/%d", d.TasksDone, len(d.Tasks))
	}
}

func TestBranchTasks_NoBranchNoFlag(t *testing.T) {
	f := newBranchFixture(t, true, mainTasks0of10, "", false)
	if c := listedChange(t, f.kanban, f.workspaceID, branchChange); c.HasBranch || c.TasksDone != 0 {
		t.Errorf("got %+v", c)
	}
	if f.toggle("8").code != http.StatusOK {
		t.Fatal("toggle without branch must succeed")
	}
	if !strings.Contains(f.mainFile(t), "- [x] 9 manual") {
		t.Fatal("main repository must be toggled")
	}
}

func TestBranchTasks_ToggleEndToEnd(t *testing.T) {
	f := newBranchFixture(t, true, mainTasks0of10, branchTasks8of, true)
	if res := f.toggle("8"); res.code != http.StatusOK {
		t.Fatalf("toggle: %d %s", res.code, res.body)
	}

	if d := f.detail(t); d.TasksDone != 9 || !d.Tasks[8].Done || d.KanbanStatus != "to-review" {
		t.Errorf("detail after toggle: %d/%d %s", d.TasksDone, d.TasksTotal, d.KanbanStatus)
	}
	if got := bgit(t, f.repo, "log", "-1", "--format=%s", "feature/c"); got != "chore: Validate task 9" {
		t.Errorf("last commit = %q", got)
	}
	if !strings.Contains(bgit(t, f.repo, "show", "feature/c:openspec/changes/c/tasks.md"), "- [x] 9 manual") {
		t.Error("branch lacks the tick")
	}
	if st := bgit(t, f.repo, "status", "--porcelain"); st != "" {
		t.Errorf("main repository must be clean: %q", st)
	}
	if f.mainFile(t) != mainTasks0of10 {
		t.Error("main tasks.md must be untouched")
	}
	published := false
	for _, ev := range f.bc.events {
		published = published || (ev.Type == "change_updated" && ev.Name == branchChange)
	}
	if !published {
		t.Error("change_updated expected")
	}
	entries, err := f.act.Read(f.workspaceID, branchChange)
	if err != nil || len(entries) != 1 || entries[0].Type != "kanban.task_toggled" {
		t.Errorf("activity = %v, %v", entries, err)
	}
}

func TestBranchTasks_ToggleRefusals(t *testing.T) {
	f := newBranchFixture(t, true, mainTasks0of10, branchTasks8of, true)

	release, err := f.reg.For(f.workspaceID).TryLockReview(branchChange)
	if err != nil {
		t.Fatal(err)
	}
	res := f.toggle("8")
	release()
	if res.code != http.StatusConflict || !strings.Contains(res.body, `"review_busy"`) {
		t.Errorf("busy: %d %s", res.code, res.body)
	}

	if res := f.toggle("99"); res.code != http.StatusNotFound {
		t.Errorf("unknown task: %d %s", res.code, res.body)
	}
}

func TestBranchTasks_BranchWithoutTaskListTogglesMain(t *testing.T) {
	f := newBranchFixture(t, true, mainTasks0of10, "# nothing\n", true)
	before := bgit(t, f.repo, "rev-parse", "feature/c")
	if f.toggle("8").code != http.StatusOK {
		t.Fatal("toggle failed")
	}
	if !strings.Contains(f.mainFile(t), "- [x] 9 manual") {
		t.Error("main repository must be toggled")
	}
	if bgit(t, f.repo, "rev-parse", "feature/c") != before {
		t.Error("branch must be untouched")
	}
}

func (f *branchFixture) reset(t *testing.T) (int, string) {
	t.Helper()
	ws := NewWorkspaceHandler(&config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: f.repo}}}, "", f.reg)
	h := NewFFHandler(ws, nil, nil, f.act, nil, f.reg)
	rec, req := resetTasksRequest(f.workspaceID, branchChange)
	h.ResetTasks(rec, req)
	return rec.Code, rec.Body.String()
}

func (f *branchFixture) startPool(t *testing.T) *pool.Manager {
	t.Helper()
	mgr := f.reg.For(f.workspaceID)
	if err := mgr.Start(pool.AgentPoolConfig{Size: 1, DelegationMode: pool.ModeHITLReview, MaxAttempts: 1}, f.workspaceID, "test", f.repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mgr.Stop)
	return mgr
}

func branchExistsIn(t *testing.T, repo string) bool {
	t.Helper()
	return strings.Contains(bgit(t, repo, "branch", "--list", "feature/c"), "feature/c")
}

func TestResetTasks_RemovesBranchWorktreeAndMarker(t *testing.T) {
	f := newBranchFixture(t, false, mainTasks0of10, branchTasks8of, true)
	// Provision a worktree through a tick, as a review would.
	if res := f.toggle("8"); res.code != http.StatusOK {
		t.Fatalf("toggle: %d %s", res.code, res.body)
	}

	if code, body := f.reset(t); code != http.StatusNoContent {
		t.Fatalf("reset: %d %s", code, body)
	}
	if branchExistsIn(t, f.repo) {
		t.Error("branch must be deleted")
	}
	f.noWorktree(t)
	if openspec.ReviewMarkers(f.repo)[branchChange] {
		t.Error("review marker must be lifted")
	}
	if f.mainFile(t) != "" {
		t.Error("tasks.md must be emptied")
	}
	c := listedChange(t, f.kanban, f.workspaceID, branchChange)
	if c.HasBranch || c.KanbanStatus != "to-explore" || c.TasksTotal != 0 {
		t.Errorf("after reset: %+v", c)
	}
	if entries, _ := f.act.Read(f.workspaceID, branchChange); len(entries) == 0 || entries[len(entries)-1].Type != "kanban.tasks_reset" {
		t.Errorf("activity = %v", entries)
	}
}

func TestResetTasks_ActiveWorkerRefused(t *testing.T) {
	f := newBranchFixture(t, true, mainTasks0of10, branchTasks8of, false)
	mgr := f.startPool(t)
	pool.SeedWorkerForTest(mgr, pool.Worker{ID: 1, ActiveChange: branchChange, Status: pool.StatusWorking, WorktreePath: t.TempDir()})

	code, body := f.reset(t)
	if code != http.StatusConflict || !strings.Contains(body, `"worker_active"`) {
		t.Fatalf("reset: %d %s", code, body)
	}
	if !branchExistsIn(t, f.repo) || f.mainFile(t) != mainTasks0of10 {
		t.Error("nothing may be modified")
	}
}

func TestResetTasks_PausedWorkerReleased(t *testing.T) {
	f := newBranchFixture(t, true, mainTasks0of10, branchTasks8of, false)
	mgr := f.startPool(t)
	pool.SeedWorkerForTest(mgr, pool.Worker{ID: 1, ActiveChange: branchChange, Status: pool.StatusPaused, WorktreePath: t.TempDir()})

	if code, body := f.reset(t); code != http.StatusNoContent {
		t.Fatalf("reset: %d %s", code, body)
	}
	if mgr.HasWorkerFor(branchChange) {
		t.Error("paused worker must be released")
	}
	if branchExistsIn(t, f.repo) {
		t.Error("branch must be deleted")
	}
}

func TestResetTasks_ReviewBusyKeepsBranch(t *testing.T) {
	f := newBranchFixture(t, true, mainTasks0of10, branchTasks8of, true)
	release, err := f.reg.For(f.workspaceID).TryLockReview(branchChange)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	code, body := f.reset(t)
	if code != http.StatusConflict || !strings.Contains(body, `"review_busy"`) {
		t.Fatalf("reset: %d %s", code, body)
	}
	if !branchExistsIn(t, f.repo) || !openspec.ReviewMarkers(f.repo)[branchChange] || f.mainFile(t) != mainTasks0of10 {
		t.Error("branch, marker and tasks must be intact")
	}
}

func TestResetTasks_CleanupFailureKeepsTasks(t *testing.T) {
	f := newBranchFixture(t, true, mainTasks0of10, branchTasks8of, true)
	// A branch checked out in the main repository cannot be deleted.
	bgit(t, f.repo, "checkout", "-q", "feature/c")
	defer bgit(t, f.repo, "checkout", "-q", "main")

	if code, body := f.reset(t); code != http.StatusInternalServerError {
		t.Fatalf("reset: %d %s", code, body)
	}
	if !branchExistsIn(t, f.repo) {
		t.Error("branch must still exist")
	}
	if !strings.Contains(f.mainFile(t), "- [x] 8") {
		t.Error("tasks.md must be intact")
	}
}

func TestResetTasks_WithoutBranchUnchanged(t *testing.T) {
	f := newBranchFixture(t, true, "- [x] a\n", "", false)
	if code, body := f.reset(t); code != http.StatusNoContent {
		t.Fatalf("reset: %d %s", code, body)
	}
	if f.mainFile(t) != "" {
		t.Error("tasks.md must be emptied")
	}
}
