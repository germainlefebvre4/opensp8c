package openspec

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// newReviewRepo builds a git workspace with the given changes (tasks all open)
// and one commit.
func newReviewRepo(t *testing.T, launched bool, names ...string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	ws := t.TempDir()
	changesDir := filepath.Join(ws, "openspec", "changes")
	meta := "schema: spec-driven\n"
	if launched {
		meta += "launched: true\n"
	} else {
		meta += "launched: false\n"
	}
	for _, n := range names {
		writeChangeFixture(t, changesDir, n, meta, "- [ ] a\n- [ ] b\n")
	}
	gitRun(t, ws, "init", "-q", "-b", "main")
	gitRun(t, ws, "add", "-A")
	gitRun(t, ws, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "init")
	return ws
}

func markReview(t *testing.T, ws, change string) {
	t.Helper()
	gitRun(t, ws, "branch", "feature/"+change)
	gitRun(t, ws, "config", ReviewKey(change), "2026-01-01T00:00:00Z")
}

func statusOf(t *testing.T, ws, name string) string {
	t.Helper()
	changes, err := ListChanges(ws)
	if err != nil {
		t.Fatalf("ListChanges: %v", err)
	}
	for _, c := range changes {
		if c.Name == name {
			return c.KanbanStatus
		}
	}
	t.Fatalf("change %s not listed", name)
	return ""
}

func TestReviewMarkers(t *testing.T) {
	ws := newReviewRepo(t, true, "a", "b")
	if got := ReviewMarkers(ws); len(got) != 0 {
		t.Errorf("no marker expected, got %v", got)
	}
	markReview(t, ws, "a")
	got := ReviewMarkers(ws)
	if !got["a"] || got["b"] || len(got) != 1 {
		t.Errorf("unexpected markers: %v", got)
	}
	if got := ReviewMarkers(t.TempDir()); len(got) != 0 {
		t.Errorf("non-git folder: expected empty set, got %v", got)
	}
}

func TestListChanges_ReviewStatus(t *testing.T) {
	cases := []struct {
		name     string
		launched bool
		marker   bool
		branch   bool
		want     string
	}{
		{"launched, marked", true, true, true, "to-review"},
		{"not launched (ready), marked", false, true, true, "to-review"},
		{"launched, no marker", true, false, true, "todo"},
		{"not launched, no marker", false, false, true, "ready"},
		{"orphan marker (no branch)", true, true, false, "todo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ws := newReviewRepo(t, tc.launched, "c")
			if tc.branch {
				gitRun(t, ws, "branch", "feature/c")
			}
			if tc.marker {
				gitRun(t, ws, "config", ReviewKey("c"), "x")
			}
			if got := statusOf(t, ws, "c"); got != tc.want {
				t.Errorf("status = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestListChanges_ReviewWithProgressAndStale(t *testing.T) {
	ws := newReviewRepo(t, true, "c")
	tasks := filepath.Join(ws, "openspec", "changes", "c", "tasks.md")
	if err := os.WriteFile(tasks, []byte("- [x] a\n- [x] b\n"), 0644); err != nil {
		t.Fatal(err)
	}
	old := mustAge(t, tasks, 30)
	_ = old
	markReview(t, ws, "c")
	changes, _ := ListChanges(ws)
	if changes[0].KanbanStatus != "to-review" || changes[0].IsStale {
		t.Errorf("got %+v", changes[0])
	}
}

func mustAge(t *testing.T, path string, days int) bool {
	t.Helper()
	old := timeAgo(days)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	return true
}

func TestListChanges_ArchivedUnaffected(t *testing.T) {
	ws := newReviewRepo(t, true, "c")
	markReview(t, ws, "c")
	if _, err := GetChangeDetail(ws, "c", ""); err != nil {
		t.Fatal(err)
	}
	d, _ := GetChangeDetail(ws, "c", "")
	if d.KanbanStatus != "to-review" {
		t.Errorf("detail status = %s", d.KanbanStatus)
	}
	archive := filepath.Join(ws, "openspec", "changes", "archive")
	if err := os.MkdirAll(archive, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(ws, "openspec", "changes", "c"), filepath.Join(archive, "c")); err != nil {
		t.Fatal(err)
	}
	d, err := GetChangeDetail(ws, "c", "")
	if err != nil || d.KanbanStatus != "archived" {
		t.Errorf("archived change: %v %+v", err, d)
	}
}

func TestApplyWorktreeProgress_KeepsToReview(t *testing.T) {
	ws := t.TempDir()
	wtPath := t.TempDir()
	tasksDir := filepath.Join(wtPath, "openspec", "changes", "c")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tasksDir, "tasks.md"), []byte("- [x] a\n- [ ] b\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ch := Change{Name: "c", KanbanStatus: "to-review"}
	if !ApplyWorktreeProgress(&ch, ws, wtPath) {
		t.Fatal("overlay should apply")
	}
	if ch.KanbanStatus != "to-review" || ch.TasksDone != 1 || ch.TasksTotal != 2 || ch.IsStale {
		t.Errorf("got %+v", ch)
	}
}

func TestListChanges_NoGitBinary(t *testing.T) {
	ws := newReviewRepo(t, true, "c")
	markReview(t, ws, "c")
	t.Setenv("PATH", t.TempDir())
	changes, err := ListChanges(ws)
	if err != nil || len(changes) != 1 || changes[0].KanbanStatus != "todo" {
		t.Errorf("without git: %v %+v", err, changes)
	}
}

func TestListChanges_NonGitFolder(t *testing.T) {
	ws := t.TempDir()
	writeChangeFixture(t, filepath.Join(ws, "openspec", "changes"), "c", "schema: spec-driven\nlaunched: true\n", "- [ ] a\n")
	if got := statusOf(t, ws, "c"); got != "todo" {
		t.Errorf("status = %s", got)
	}
}

func timeAgo(days int) time.Time { return time.Now().Add(-time.Duration(days) * 24 * time.Hour) }
