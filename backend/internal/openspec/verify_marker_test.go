package openspec

import (
	"os"
	"path/filepath"
	"testing"
)

func markVerify(t *testing.T, ws, change, value string) {
	t.Helper()
	gitRun(t, ws, "branch", "feature/"+change)
	gitRun(t, ws, "config", VerifyKey(change), value)
}

func TestVerifyMarkers(t *testing.T) {
	ws := newReviewRepo(t, true, "a", "b", "c")
	if got := VerifyMarkers(ws); len(got) != 0 {
		t.Errorf("no marker expected, got %v", got)
	}
	markVerify(t, ws, "a", "pending")
	markVerify(t, ws, "b", "passed")
	markVerify(t, ws, "c", "weird")
	got := VerifyMarkers(ws)
	if got["a"] != "pending" || got["b"] != "passed" || got["c"] != "failed" || len(got) != 3 {
		t.Errorf("unexpected markers: %v", got)
	}
	if got := VerifyMarkers(t.TempDir()); len(got) != 0 {
		t.Errorf("non-git folder: expected empty map, got %v", got)
	}
}

func TestListChanges_VerifyingStatus(t *testing.T) {
	cases := []struct {
		name   string
		marker string
		branch bool
		review bool
		want   string
		state  string
	}{
		{"pending", "pending", true, false, "verifying", "queued"},
		{"failed", "failed", true, false, "verifying", "failed"},
		{"passed", "passed", true, false, "verifying", "passed"},
		{"unknown value", "bogus", true, false, "verifying", "failed"},
		{"review wins", "pending", true, true, "to-review", ""},
		{"no branch", "pending", false, false, "todo", ""},
		{"no marker", "", true, false, "todo", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ws := newReviewRepo(t, true, "c")
			if tc.branch {
				gitRun(t, ws, "branch", "feature/c")
			}
			if tc.marker != "" {
				gitRun(t, ws, "config", VerifyKey("c"), tc.marker)
			}
			if tc.review {
				gitRun(t, ws, "config", ReviewKey("c"), "x")
			}
			changes, err := ListChanges(ws)
			if err != nil {
				t.Fatal(err)
			}
			ch := changes[0]
			if ch.KanbanStatus != tc.want || ch.VerificationState != tc.state {
				t.Errorf("status=%s state=%q, want %s/%q", ch.KanbanStatus, ch.VerificationState, tc.want, tc.state)
			}
			if tc.want == "verifying" && ch.IsStale {
				t.Error("a verifying change must not be stale")
			}
			detail, err := GetChangeDetail(ws, "c", "")
			if err != nil {
				t.Fatal(err)
			}
			if detail.KanbanStatus != tc.want || detail.VerificationState != tc.state {
				t.Errorf("detail status=%s state=%q", detail.KanbanStatus, detail.VerificationState)
			}
		})
	}
}

func TestListChanges_VerifyingStaleAndBranchProgress(t *testing.T) {
	ws := newReviewRepo(t, true, "c")
	tasks := filepath.Join(ws, "openspec", "changes", "c", "tasks.md")
	if err := os.WriteFile(tasks, []byte("- [ ] a\n- [ ] b\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustAge(t, tasks, 30)
	markVerify(t, ws, "c", "pending")
	changes, _ := ListChanges(ws)
	if changes[0].KanbanStatus != "verifying" || changes[0].IsStale {
		t.Errorf("got %+v", changes[0])
	}
	// The branch progress overlay keeps the verifying status.
	if !ApplyBranchProgress(&changes[0], "- [x] a\n- [x] b\n") || changes[0].TasksDone != 2 || changes[0].KanbanStatus != "verifying" {
		t.Errorf("overlay: %+v", changes[0])
	}
}
