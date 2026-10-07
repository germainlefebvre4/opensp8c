package pool

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/openspec"
)

// reviewFixture runs a hitl-review worker so that change ends in review, with
// a validation command counting its runs.
func reviewFixture(t *testing.T, change, tasks, validation string) (*Manager, string, string) {
	t.Helper()
	repo := newGoFixtureRepo(t, change, tasks)
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1},
		pipeAgent(writeInWorktree("a.txt"), okResult, nil))
	counter := filepath.Join(t.TempDir(), "count")
	setValidation(t, m, fmt.Sprintf("echo r >> %s; %s", counter, validation))
	runOnce(t, m, change)
	if has, _ := NewWorktreeController(repo, "ws1", m.worktreesRoot).HasReview(change); !has {
		t.Fatal("fixture: change should be in review")
	}
	return m, repo, counter
}

func inReview(t *testing.T, m *Manager, repo, change string) bool {
	t.Helper()
	has, err := NewWorktreeController(repo, "ws1", m.worktreesRoot).HasReview(change)
	if err != nil {
		t.Fatal(err)
	}
	return has
}

func assertUntouched(t *testing.T, m *Manager, repo, change string) {
	t.Helper()
	if !inReview(t, m, repo, change) || !branchExistsIn(t, repo, "feature/"+change) {
		t.Fatal("branch and review marker must be kept")
	}
	if fileIn(repo, "a.txt") {
		t.Fatal("nothing must be merged")
	}
}

func TestApprove_MergesWithPoolStopped(t *testing.T) {
	m, repo, counter := reviewFixture(t, "ok", "- [x] done\n", "true")
	res, err := m.ApproveReview(context.Background(), "ws1", repo, "ok")
	if err != nil {
		t.Fatal(err)
	}
	if res.Warning != nil {
		t.Fatalf("unexpected warning: %+v", res.Warning)
	}
	if target := res.Target; target == "" || !fileIn(repo, "a.txt") || branchExistsIn(t, repo, "feature/ok") {
		t.Fatalf("expected a merge into %q and the branch deleted", target)
	}
	if n := countRuns(t, counter); n != 1 {
		t.Fatalf("validation runs = %d, want 1 (worker only: target unchanged)", n)
	}
	if inReview(t, m, repo, "ok") {
		t.Fatal("marker must vanish with the branch")
	}
}

func TestApprove_AdvancedTargetIntegratesAndRevalidates(t *testing.T) {
	m, repo, counter := reviewFixture(t, "adv", "- [x] done\n", "true")
	userCommit(t, repo, "user.txt")("")
	if _, err := m.ApproveReview(context.Background(), "ws1", repo, "adv"); err != nil {
		t.Fatal(err)
	}
	if n := countRuns(t, counter); n != 2 {
		t.Fatalf("validation runs = %d, want 2", n)
	}
	if !fileIn(repo, "a.txt") || !fileIn(repo, "user.txt") {
		t.Fatal("both files expected on the target")
	}
}

func TestApprove_IntegrationConflictKeepsState(t *testing.T) {
	m, repo, _ := reviewFixture(t, "conf", "- [x] done\n", "true")
	writeFile(t, filepath.Join(repo, "a.txt"), "user version")
	gitIn(t, repo, "add", "a.txt")
	gitIn(t, repo, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "user a")

	_, err := m.ApproveReview(context.Background(), "ws1", repo, "conf")
	var ce *IntegrationConflictError
	if !errors.As(err, &ce) {
		t.Fatalf("err = %v, want IntegrationConflictError", err)
	}
	if !inReview(t, m, repo, "conf") || !branchExistsIn(t, repo, "feature/conf") {
		t.Fatal("branch and marker must be kept")
	}
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	if st := gitIn(t, wt.resolvePath("conf"), "status", "--porcelain"); st != "" {
		t.Fatalf("worktree must be clean after the aborted integration: %q", st)
	}
}

func TestApprove_ValidationFailureKeepsState(t *testing.T) {
	m, repo, _ := reviewFixture(t, "vf", "- [x] done\n", "[ ! -f user.txt ]")
	userCommit(t, repo, "user.txt")("")
	_, err := m.ApproveReview(context.Background(), "ws1", repo, "vf")
	var ve *ValidationFailedError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want ValidationFailedError", err)
	}
	if !inReview(t, m, repo, "vf") || !branchExistsIn(t, repo, "feature/vf") {
		t.Fatal("branch and marker must be kept")
	}
	if fileIn(repo, "a.txt") {
		t.Fatal("nothing must be merged")
	}
}

func TestApprove_BaseMismatch(t *testing.T) {
	m, repo, _ := reviewFixture(t, "base", "- [x] done\n", "true")
	gitIn(t, repo, "checkout", "-q", "-b", "other")
	_, err := m.ApproveReview(context.Background(), "ws1", repo, "base")
	if !errors.Is(err, ErrBaseBranchMismatch) {
		t.Fatalf("err = %v, want base mismatch", err)
	}
	assertUntouched(t, m, repo, "base")
}

func TestApprove_MergeInProgress(t *testing.T) {
	m, repo, _ := reviewFixture(t, "mip", "- [x] done\n", "true")
	head := gitIn(t, repo, "rev-parse", "HEAD")
	writeFile(t, filepath.Join(repo, ".git", "MERGE_HEAD"), head+"\n")
	_, err := m.ApproveReview(context.Background(), "ws1", repo, "mip")
	if !errors.Is(err, ErrMergeInProgress) {
		t.Fatalf("err = %v, want merge in progress", err)
	}
	if _, statErr := os.Stat(filepath.Join(repo, ".git", "MERGE_HEAD")); statErr != nil {
		t.Fatal("the merge in progress must not be aborted")
	}
	if !inReview(t, m, repo, "mip") || !branchExistsIn(t, repo, "feature/mip") {
		t.Fatal("branch and marker must be kept")
	}
}

func TestApprove_Refusals(t *testing.T) {
	m, repo, _ := reviewFixture(t, "ref", "- [x] done\n", "true")
	if _, err := m.ApproveReview(context.Background(), "ws1", repo, "unknown"); !errors.Is(err, ErrNotInReview) {
		t.Fatalf("err = %v, want not in review", err)
	}
	m.activeWorkers[7] = &Worker{ID: 7, ActiveChange: "ref"}
	if _, err := m.ApproveReview(context.Background(), "ws1", repo, "ref"); !errors.Is(err, ErrWorkerActive) {
		t.Fatalf("err = %v, want worker active", err)
	}
	assertUntouched(t, m, repo, "ref")
}

func TestApprove_ConcurrentApprovalsMergeOnce(t *testing.T) {
	m, repo, _ := reviewFixture(t, "conc", "- [x] done\n", "true")
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = m.ApproveReview(context.Background(), "ws1", repo, "conc")
		}()
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
		} else if !errors.Is(err, ErrNotInReview) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 {
		t.Fatalf("successful approvals = %d, want 1", ok)
	}
	if n := strings.Count(gitIn(t, repo, "log", "--merges", "--format=%B"), "feat: Conc\n\nChange: conc"); n != 1 {
		t.Fatalf("merge commits = %d, want 1", n)
	}
}

func TestApprove_WaitsForMergeLockHeldByWorker(t *testing.T) {
	m, repo, _ := reviewFixture(t, "wait", "- [x] done\n", "true")
	m.mergeMu.Lock()
	done := make(chan error, 1)
	go func() {
		_, err := m.ApproveReview(context.Background(), "ws1", repo, "wait")
		done <- err
	}()
	select {
	case <-done:
		t.Fatal("approval must wait for the merge lock")
	case <-time.After(300 * time.Millisecond):
	}
	m.mergeMu.Unlock()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("approval did not resume")
	}
	if !fileIn(repo, "a.txt") {
		t.Fatal("expected the merge")
	}
}

func TestApprove_SurvivesCancelledRequest(t *testing.T) {
	m, repo, _ := reviewFixture(t, "detach", "- [x] done\n", "true")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.ApproveReview(ctx, "ws1", repo, "detach"); err != nil {
		t.Fatal(err)
	}
	if !fileIn(repo, "a.txt") {
		t.Fatal("a merge is not interrupted by the request context")
	}
}

func TestAppendCorrection(t *testing.T) {
	count := func(s string) int {
		p := filepath.Join(t.TempDir(), "tasks.md")
		writeFile(t, p, s)
		_, total := openspec.ParseTaskProgress(p)
		return total
	}

	t.Run("section absente", func(t *testing.T) {
		got := AppendCorrection("# Tasks\n\n- [x] 1.1 a\n", "Le bouton est cassé")
		want := "# Tasks\n\n- [x] 1.1 a\n\n## Corrections\n\n- [ ] Correction : Le bouton est cassé\n"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})
	t.Run("section existante et deuxième demande", func(t *testing.T) {
		first := AppendCorrection("# Tasks\n\n- [x] a\n\n## Corrections\n\n- [x] Correction : un\n\n## Autre\n\n- [x] b\n", "deux")
		want := "# Tasks\n\n- [x] a\n\n## Corrections\n\n- [x] Correction : un\n- [ ] Correction : deux\n\n## Autre\n\n- [x] b\n"
		if first != want {
			t.Fatalf("got %q want %q", first, want)
		}
		second := AppendCorrection(first, "trois")
		if strings.Count(second, correctionsHeading) != 1 || count(second) != 5 {
			t.Fatalf("section duplicated or wrong count:\n%s", second)
		}
	})
	t.Run("section en fin de fichier", func(t *testing.T) {
		got := AppendCorrection("- [x] a\n\n## Corrections\n\n- [x] Correction : un\n", "deux")
		if got != "- [x] a\n\n## Corrections\n\n- [x] Correction : un\n- [ ] Correction : deux\n" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("retour multi-lignes", func(t *testing.T) {
		base := "- [x] a\n"
		got := AppendCorrection(base, "ligne un\n- [ ] faux\n\n- [x] autre\r\nfin")
		if count(got)-count(base) != 1 {
			t.Fatalf("exactly one task expected:\n%s", got)
		}
		for _, l := range []string{"  > - [ ] faux", "  >", "  > - [x] autre", "  > fin"} {
			if !strings.Contains(got, l+"\n") {
				t.Fatalf("missing line %q in:\n%s", l, got)
			}
		}
	})
}

func TestRequestCorrection_Succeeds(t *testing.T) {
	m, repo, _ := reviewFixture(t, "corr", "- [x] done\n", "true")
	bc := &mockBroadcaster{}
	m.broadcaster = bc
	if err := m.RequestCorrection(context.Background(), "ws1", repo, "corr", "Le bouton Annuler ne ferme pas\nle dialogue"); err != nil {
		t.Fatal(err)
	}
	if inReview(t, m, repo, "corr") {
		t.Fatal("marker must be lifted")
	}
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	tasks := filepath.Join(wt.resolvePath("corr"), "openspec", "changes", "corr", "tasks.md")
	done, total := openspec.ParseTaskProgress(tasks)
	if done != 1 || total != 2 {
		t.Fatalf("progress = %d/%d, want 1/2", done, total)
	}
	if st := gitIn(t, repo, "status", "--porcelain"); st != "" {
		t.Fatalf("repo must stay clean: %q", st)
	}
	if st := gitIn(t, wt.resolvePath("corr"), "status", "--porcelain"); st != "" {
		t.Fatalf("worktree must be clean (committed): %q", st)
	}
	if log := gitIn(t, repo, "log", "-1", "--format=%B", "feature/corr"); log != "chore: Add review correction\n\nChange: corr" {
		t.Fatalf("last commit = %q", log)
	}
	published := false
	for _, c := range bc.snapshot() {
		if c.ev.Type == "change_updated" && c.ev.Name == "corr" {
			published = true
		}
	}
	if !published {
		t.Fatal("change_updated expected")
	}
}

func TestRequestCorrection_RecreatesMissingWorktree(t *testing.T) {
	m, repo, _ := reviewFixture(t, "gone", "- [x] done\n", "true")
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	gitIn(t, repo, "worktree", "remove", "--force", wt.resolvePath("gone"))
	if err := m.RequestCorrection(context.Background(), "ws1", repo, "gone", "à refaire"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gitIn(t, repo, "show", "feature/gone:openspec/changes/gone/tasks.md"), "Correction : à refaire") {
		t.Fatal("correction missing from the branch")
	}
}

func TestRequestCorrection_Refusals(t *testing.T) {
	m, repo, _ := reviewFixture(t, "refc", "- [x] done\n", "true")
	if err := m.RequestCorrection(context.Background(), "ws1", repo, "refc", "  \n "); !errors.Is(err, ErrEmptyFeedback) {
		t.Fatalf("err = %v, want empty feedback", err)
	}
	if err := m.RequestCorrection(context.Background(), "ws1", repo, "other", "x"); !errors.Is(err, ErrNotInReview) {
		t.Fatalf("err = %v, want not in review", err)
	}
	m.pausedWorkers[3] = &Worker{ID: 3, ActiveChange: "refc"}
	if err := m.RequestCorrection(context.Background(), "ws1", repo, "refc", "x"); !errors.Is(err, ErrWorkerActive) {
		t.Fatalf("err = %v, want worker active", err)
	}
	if !inReview(t, m, repo, "refc") {
		t.Fatal("marker must be kept")
	}
	if strings.Contains(gitIn(t, repo, "show", "feature/refc:openspec/changes/refc/tasks.md"), "Corrections") {
		t.Fatal("nothing must be written")
	}
}

func TestRequestCorrection_CommitFailureKeepsMarker(t *testing.T) {
	m, repo, _ := reviewFixture(t, "cf", "- [x] done\n", "true")
	hook := filepath.Join(repo, ".git", "hooks", "pre-commit")
	writeFile(t, hook, "#!/bin/sh\nexit 1\n")
	if err := os.Chmod(hook, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := m.RequestCorrection(context.Background(), "ws1", repo, "cf", "x"); err == nil {
		t.Fatal("expected a commit error")
	}
	if !inReview(t, m, repo, "cf") {
		t.Fatal("marker must be kept")
	}
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	if st := gitIn(t, wt.resolvePath("cf"), "status", "--porcelain"); st != "" {
		t.Fatalf("tasks.md must be restored: %q", st)
	}
}

// correctionFlow runs a hitl-review worker on a launched change until it is in
// review, then requests a correction and lets the dispatcher pick it up again.
// agent receives the 1-based number of the agent turn (one per worker run, as
// validation never fails) and the worktree.
func correctionFlow(t *testing.T, change string, agent func(run int, ws string)) (*Manager, string) {
	t.Helper()
	repo := newGoFixtureRepo(t, change, "- [ ] do the thing\n")
	writeFile(t, filepath.Join(repo, "openspec", "changes", change, ".openspec.yaml"), "schema: spec-driven\nlaunched: true\n")
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "launch")

	run := 0
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1},
		pipeAgent(func(ws string) {
			run++
			agent(run, ws)
		}, okResult, nil))
	setValidation(t, m, "true")
	m.workspaceID = "ws1"

	// tick is the dispatcher's single dispatch point; calling it directly
	// avoids waiting for the 5 s loop.
	dispatch := func() {
		m.tick()
		m.waitWorkers()
	}
	dispatch()
	if !inReview(t, m, repo, change) {
		t.Fatal("first run should end in review")
	}
	if err := m.RequestCorrection(context.Background(), "ws1", repo, change, "à corriger"); err != nil {
		t.Fatal(err)
	}
	dispatch()
	return m, repo
}

func checkAll(ws, change string) {
	p := filepath.Join(ws, "openspec", "changes", change, "tasks.md")
	if b, err := os.ReadFile(p); err == nil {
		_ = os.WriteFile(p, []byte(strings.ReplaceAll(string(b), "- [ ]", "- [x]")), 0o644)
	}
}

func TestCorrectionFlow_WorkerReappliesAndReturnsToReview(t *testing.T) {
	m, repo := correctionFlow(t, "flow-ok", func(run int, ws string) {
		writeInWorktree(fmt.Sprintf("run%d.txt", run))(ws)
		checkAll(ws, "flow-ok")
	})
	if !inReview(t, m, repo, "flow-ok") {
		t.Fatal("change should be back in review after the correction was applied")
	}
	if got := gitIn(t, repo, "show", "feature/flow-ok:openspec/changes/flow-ok/tasks.md"); strings.Contains(got, "- [ ]") || !strings.Contains(got, "- [x] Correction : à corriger") {
		t.Fatalf("correction not checked in the branch:\n%s", got)
	}
	if got := gitIn(t, repo, "ls-tree", "--name-only", "feature/flow-ok"); !strings.Contains(got, "run1.txt") || !strings.Contains(got, "run2.txt") {
		t.Fatalf("the second run must reuse the branch of the first:\n%s", got)
	}
}

func TestCorrectionFlow_UncheckedCorrectionPausesWorker(t *testing.T) {
	m, repo := correctionFlow(t, "flow-skip", func(run int, ws string) {
		if run == 1 {
			writeInWorktree("a.txt")(ws)
			checkAll(ws, "flow-skip")
		}
		// run 2: the agent does nothing and leaves the correction unchecked.
	})
	if inReview(t, m, repo, "flow-skip") {
		t.Fatal("an unchecked correction must not return to review")
	}
	var reason string
	m.mu.Lock()
	for _, w := range m.pausedWorkers {
		reason = w.BlockedReason
	}
	m.mu.Unlock()
	if !strings.Contains(reason, "tâches restantes incomplètes") {
		t.Fatalf("paused reason = %q", reason)
	}
}

// junkInWorktree leaves an untracked file in the worktree of change, which
// makes `git worktree remove` refuse the removal.
func junkInWorktree(t *testing.T, m *Manager, repo, change string) {
	t.Helper()
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	writeFile(t, filepath.Join(wt.resolvePath(change), "junk.txt"), "untracked")
}

func changeStatus(t *testing.T, repo, change string) string {
	t.Helper()
	changes, err := openspec.ListChanges(repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range changes {
		if c.Name == change {
			return c.KanbanStatus
		}
	}
	t.Fatalf("change %s not listed", change)
	return ""
}

func published(bc *mockBroadcaster, change string) bool {
	for _, c := range bc.snapshot() {
		if c.ev.Type == "change_updated" && c.ev.Name == change {
			return true
		}
	}
	return false
}

func TestApprove_WorktreeCleanupFailureIsAWarning(t *testing.T) {
	m, repo, _ := reviewFixture(t, "dirty", "- [x] done\n", "true")
	bc := &mockBroadcaster{}
	m.broadcaster = bc
	junkInWorktree(t, m, repo, "dirty")

	res, err := m.ApproveReview(context.Background(), "ws1", repo, "dirty")
	if err != nil {
		t.Fatalf("a merged change must not fail: %v", err)
	}
	if res.Target == "" || res.Warning == nil || !reflect.DeepEqual(res.Warning.Remaining, []string{"worktree"}) {
		t.Fatalf("result = %+v, want a warning remaining [worktree]", res)
	}
	if !fileIn(repo, "a.txt") {
		t.Fatal("the merge must be on the target")
	}
	if inReview(t, m, repo, "dirty") {
		t.Fatal("marker must be lifted")
	}
	if got := changeStatus(t, repo, "dirty"); got == "to-review" {
		t.Fatalf("status = %s, the change must leave to-review", got)
	}
	if !published(bc, "dirty") {
		t.Fatal("change_updated must be published")
	}
}

// mergedFixture is a change in review whose branch was merged by hand, leaving
// its marker (an approval whose marker could not be lifted).
func mergedFixture(t *testing.T, change string) (*Manager, string, string) {
	t.Helper()
	m, repo, counter := reviewFixture(t, change, "- [x] done\n", "true")
	setValidation(t, m, fmt.Sprintf("echo r >> %s; false", counter))
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	if _, err := wt.MergeInto(change); err != nil {
		t.Fatal(err)
	}
	return m, repo, counter
}

func TestApprove_AlreadyMergedOnlyCleansUp(t *testing.T) {
	m, repo, counter := mergedFixture(t, "again")
	before := countRuns(t, counter)
	bc := &mockBroadcaster{}
	m.broadcaster = bc

	res, err := m.ApproveReview(context.Background(), "ws1", repo, "again")
	if err != nil {
		t.Fatal(err)
	}
	if res.Warning != nil || res.Target != gitIn(t, repo, "symbolic-ref", "--short", "HEAD") {
		t.Fatalf("result = %+v", res)
	}
	if n := countRuns(t, counter); n != before {
		t.Fatalf("validation ran %d time(s) on an already merged branch", n-before)
	}
	if branchExistsIn(t, repo, "feature/again") || inReview(t, m, repo, "again") {
		t.Fatal("branch and marker must be gone")
	}
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	if _, err := os.Stat(wt.resolvePath("again")); !os.IsNotExist(err) {
		t.Fatal("worktree must be removed")
	}
	if !published(bc, "again") {
		t.Fatal("change_updated must be published")
	}
}

func TestApprove_AlreadyMergedStillChecksBase(t *testing.T) {
	m, repo, _ := mergedFixture(t, "other-base")
	gitIn(t, repo, "checkout", "-q", "-b", "other")

	_, err := m.ApproveReview(context.Background(), "ws1", repo, "other-base")
	if !errors.Is(err, ErrBaseBranchMismatch) {
		t.Fatalf("err = %v, want base mismatch", err)
	}
	if !inReview(t, m, repo, "other-base") || !branchExistsIn(t, repo, "feature/other-base") {
		t.Fatal("branch and marker must be kept")
	}
}

func TestCleanupMerged(t *testing.T) {
	cases := []struct {
		name   string
		break_ func(t *testing.T, repo string, wc *WorktreeController)
		want   string // "" when everything is removed
	}{
		{"all removed", func(*testing.T, string, *WorktreeController) {}, ""},
		{"untracked file in worktree", func(t *testing.T, _ string, wc *WorktreeController) {
			writeFile(t, filepath.Join(wc.resolvePath("c"), "junk.txt"), "x")
		}, cleanupWorktree},
		{"branch ref locked", func(t *testing.T, repo string, _ *WorktreeController) {
			writeFile(t, filepath.Join(repo, ".git", "refs", "heads", "feature", "c.lock"), "")
		}, cleanupBranch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newTestRepo(t)
			wc := newTestWC(t, repo)
			path, _ := wc.Provision("c")
			commitFile(t, path, "feat.txt", "f")
			if _, err := wc.MergeInto("c"); err != nil {
				t.Fatal(err)
			}
			tc.break_(t, repo, wc)

			failed := cleanupMerged(wc, "c")
			switch {
			case tc.want == "" && failed != nil:
				t.Fatalf("unexpected failure: %+v", failed)
			case tc.want != "" && (failed == nil || failed.Element != tc.want):
				t.Fatalf("failure = %+v, want %s", failed, tc.want)
			}
		})
	}
}

func TestFinishMerged_WarningRemaining(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	if _, err := wc.Provision("c"); err != nil {
		t.Fatal(err)
	}
	if err := wc.MarkReview("c"); err != nil {
		t.Fatal(err)
	}
	m := &Manager{}

	if res := m.finishMerged(wc, "ws1", "c", "main", nil); res.Warning != nil {
		t.Fatalf("no failure, warning = %+v", res.Warning)
	}

	// Worktree failure plus a marker that cannot be lifted (config locked).
	writeFile(t, filepath.Join(repo, ".git", "config.lock"), "")
	res := m.finishMerged(wc, "ws1", "c", "main", &cleanupFailure{Element: cleanupWorktree, Err: errors.New("boom")})
	if res.Warning == nil || !reflect.DeepEqual(res.Warning.Remaining, []string{"worktree", "marker"}) {
		t.Fatalf("warning = %+v, want remaining [worktree marker]", res.Warning)
	}
	if !strings.Contains(res.Warning.Message, "worktree, marker") {
		t.Fatalf("message = %q", res.Warning.Message)
	}
}

func TestToggleBranchTask_TickAndUntickInReview(t *testing.T) {
	m, repo, _ := reviewFixture(t, "tg", "- [x] done\n", "true")
	bc := &mockBroadcaster{}
	m.broadcaster = bc
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	before := gitIn(t, repo, "rev-parse", "feature/tg")

	text, done, err := m.ToggleBranchTask(context.Background(), "ws1", repo, "tg", 0)
	if err != nil || done || text != "done" {
		t.Fatalf("untick = %q, %v, %v", text, done, err)
	}
	if got := gitIn(t, repo, "log", "-1", "--format=%B", "feature/tg"); got != "chore: Reopen task 1\n\nChange: tg\nTask: done" {
		t.Fatalf("untick message = %q", got)
	}
	if files := gitIn(t, repo, "show", "--name-only", "--format=", "feature/tg"); files != "openspec/changes/tg/tasks.md" {
		t.Fatalf("commit files = %q", files)
	}
	if !strings.Contains(gitIn(t, repo, "show", "feature/tg:openspec/changes/tg/tasks.md"), "- [ ] done") {
		t.Fatal("branch must carry the unticked task")
	}

	if _, done, err = m.ToggleBranchTask(context.Background(), "ws1", repo, "tg", 0); err != nil || !done {
		t.Fatalf("tick = %v, %v", done, err)
	}
	if got := gitIn(t, repo, "log", "-1", "--format=%B", "feature/tg"); got != "chore: Validate task 1\n\nChange: tg\nTask: done" {
		t.Fatalf("tick message = %q", got)
	}
	if gitIn(t, repo, "rev-parse", "feature/tg~2") != before {
		t.Fatal("one commit per tick expected")
	}
	if !inReview(t, m, repo, "tg") {
		t.Fatal("review marker must be kept")
	}
	if st := gitIn(t, repo, "status", "--porcelain"); st != "" {
		t.Fatalf("main repo must stay clean: %q", st)
	}
	if st := gitIn(t, wt.resolvePath("tg"), "status", "--porcelain"); st != "" {
		t.Fatalf("worktree must be clean: %q", st)
	}
	if !published(bc, "tg") {
		t.Fatal("change_updated expected")
	}
}

func TestToggleBranchTask_ScopeFromComponents(t *testing.T) {
	m, repo, _ := reviewFixture(t, "sc", "- [x] done\n", "true")
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	writeScopeFiles(t, filepath.Join(wt.resolvePath("sc"), "openspec", "changes", "sc"), "tags:\n  components: [kanban]\n")
	if _, err := wt.CommitAll("sc"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.ToggleBranchTask(context.Background(), "ws1", repo, "sc", 0); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, repo, "log", "-1", "--format=%s", "feature/sc"); got != "chore(kanban): Reopen task 1" {
		t.Fatalf("subject = %q", got)
	}
}

func TestToggleBranchTask_RecreatesMissingWorktree(t *testing.T) {
	m, repo, _ := reviewFixture(t, "gn", "- [x] done\n", "true")
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	gitIn(t, repo, "worktree", "remove", "--force", wt.resolvePath("gn"))
	if _, _, err := m.ToggleBranchTask(context.Background(), "ws1", repo, "gn", 0); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gitIn(t, repo, "show", "feature/gn:openspec/changes/gn/tasks.md"), "- [ ] done") {
		t.Fatal("tick missing from the branch")
	}
}

func TestToggleBranchTask_Refusals(t *testing.T) {
	m, repo, _ := reviewFixture(t, "rf", "- [x] done\n", "true")
	before := gitIn(t, repo, "rev-parse", "feature/rf")

	m.pausedWorkers[3] = &Worker{ID: 3, ActiveChange: "rf"}
	if _, _, err := m.ToggleBranchTask(context.Background(), "ws1", repo, "rf", 0); !errors.Is(err, ErrWorkerActive) {
		t.Fatalf("err = %v, want worker active", err)
	}
	delete(m.pausedWorkers, 3)

	lock := m.reviewLock("rf")
	lock.Lock()
	_, _, err := m.ToggleBranchTask(context.Background(), "ws1", repo, "rf", 0)
	lock.Unlock()
	if !errors.Is(err, ErrReviewBusy) {
		t.Fatalf("err = %v, want review busy", err)
	}
	if gitIn(t, repo, "rev-parse", "feature/rf") != before {
		t.Fatal("branch must be unchanged")
	}
}

func TestToggleBranchTask_CommitFailureRestoresFile(t *testing.T) {
	m, repo, _ := reviewFixture(t, "cfl", "- [x] done\n", "true")
	before := gitIn(t, repo, "rev-parse", "feature/cfl")
	hook := filepath.Join(repo, ".git", "hooks", "pre-commit")
	writeFile(t, hook, "#!/bin/sh\nexit 1\n")
	if err := os.Chmod(hook, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.ToggleBranchTask(context.Background(), "ws1", repo, "cfl", 0); err == nil {
		t.Fatal("expected a commit error")
	}
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	if st := gitIn(t, wt.resolvePath("cfl"), "status", "--porcelain"); st != "" {
		t.Fatalf("tasks.md must be restored: %q", st)
	}
	if gitIn(t, repo, "rev-parse", "feature/cfl") != before || !inReview(t, m, repo, "cfl") {
		t.Fatal("branch and marker must be unchanged")
	}
}
