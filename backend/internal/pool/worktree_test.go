package pool

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// newTestRepo creates a git repo with one commit and points HOME to a temp dir
// so worktrees land outside the real home.
func newTestRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-q", "-m", "init")
	return repo
}

// newTestWC builds a controller whose worktrees live in a temp directory.
func newTestWC(t *testing.T, repo string) *WorktreeController {
	t.Helper()
	return NewWorktreeController(repo, "ws-a", t.TempDir())
}

func TestProvisionFirstTime(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, err := wc.Provision("add-auth")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(path, "README.md")); err != nil {
		t.Fatalf("worktree not checked out: %v", err)
	}
	if got := gitIn(t, path, "rev-parse", "--abbrev-ref", "HEAD"); got != "feature/add-auth" {
		t.Fatalf("branch = %s", got)
	}
}

func TestProvisionBranchExistsWithoutWorktree(t *testing.T) {
	repo := newTestRepo(t)
	gitIn(t, repo, "branch", "feature/add-auth")
	wc := newTestWC(t, repo)
	path, err := wc.Provision("add-auth")
	if err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, path, "rev-parse", "--abbrev-ref", "HEAD"); got != "feature/add-auth" {
		t.Fatalf("branch = %s", got)
	}
}

func TestProvisionStaleWorktreeEntry(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, err := wc.Provision("add-auth")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
	if _, err := wc.Provision("add-auth"); err != nil {
		t.Fatalf("re-provision after deleted dir: %v", err)
	}
}

func TestProvisionReusesExistingWorktreeKeepingUncommitted(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, err := wc.Provision("add-auth")
	if err != nil {
		t.Fatal(err)
	}
	dirty := filepath.Join(path, "wip.txt")
	if err := os.WriteFile(dirty, []byte("wip"), 0644); err != nil {
		t.Fatal(err)
	}
	again, err := wc.Provision("add-auth")
	if err != nil {
		t.Fatal(err)
	}
	if again != path {
		t.Fatalf("path changed: %s vs %s", again, path)
	}
	if b, err := os.ReadFile(dirty); err != nil || string(b) != "wip" {
		t.Fatalf("uncommitted file lost: %v %q", err, b)
	}
}

func TestProvisionGitFailureIsReported(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	notRepo := t.TempDir()
	if _, err := NewWorktreeController(notRepo, "ws", t.TempDir()).Provision("add-auth"); err == nil {
		t.Fatal("expected an error outside a git repository")
	}
}

func TestProvisionIsolatesWorkspaces(t *testing.T) {
	repoA, repoB := newTestRepo(t), newTestRepo(t)
	root := t.TempDir()
	pathA, err := NewWorktreeController(repoA, "ws-a", root).Provision("shared-change")
	if err != nil {
		t.Fatal(err)
	}
	pathB, err := NewWorktreeController(repoB, "ws-b", root).Provision("shared-change")
	if err != nil {
		t.Fatal(err)
	}
	if pathA == pathB {
		t.Fatalf("both workspaces share %s", pathA)
	}
	if want := filepath.Join(root, "ws-a", "wt-shared-change"); pathA != want {
		t.Fatalf("path = %s, want %s", pathA, want)
	}
	// Cleaning one workspace never touches the other.
	if err := NewWorktreeController(repoA, "ws-a", root).Remove("shared-change"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(pathB); err != nil {
		t.Fatalf("workspace B worktree affected: %v", err)
	}
}

func TestProvisionReusesLegacyWorktree(t *testing.T) {
	repo := newTestRepo(t)
	root := t.TempDir()
	legacy := filepath.Join(root, "wt-add-auth")
	gitIn(t, repo, "worktree", "add", "-b", "feature/add-auth", legacy)
	writeFile(t, filepath.Join(legacy, "wip.txt"), "wip")

	wc := NewWorktreeController(repo, "ws-a", root)
	got, err := wc.Provision("add-auth")
	if err != nil {
		t.Fatal(err)
	}
	if got != legacy {
		t.Fatalf("legacy worktree not reused: %s", got)
	}
	if b, err := os.ReadFile(filepath.Join(legacy, "wip.txt")); err != nil || string(b) != "wip" {
		t.Fatalf("uncommitted file lost: %v %q", err, b)
	}
	if _, err := os.Stat(filepath.Join(root, "ws-a", "wt-add-auth")); err == nil {
		t.Fatal("a second worktree was created")
	}
}

func TestRemoveRefusesDirtyWorktree(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, err := wc.Provision("add-auth")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(path, "wip.txt"), "wip")
	if err := wc.Remove("add-auth"); err == nil {
		t.Fatal("Remove must fail on a dirty worktree")
	}
	if _, err := os.Stat(filepath.Join(path, "wip.txt")); err != nil {
		t.Fatalf("uncommitted file lost: %v", err)
	}
}

func TestRemoveCleanAndVanishedWorktrees(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, err := wc.Provision("clean")
	if err != nil {
		t.Fatal(err)
	}
	if err := wc.Remove("clean"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("worktree still there: %v", err)
	}
	path, err = wc.Provision("vanished")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
	if err := wc.Remove("vanished"); err != nil {
		t.Fatalf("vanished worktree should just be pruned: %v", err)
	}
}

func TestDiscard(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)

	// An unregistered directory at the expected path is never deleted.
	stray := wc.worktreePath("stray")
	writeFile(t, filepath.Join(stray, "precious.txt"), "x")
	if err := wc.Discard("stray"); err == nil {
		t.Fatal("Discard must refuse a path that is not a registered worktree")
	}
	if _, err := os.Stat(filepath.Join(stray, "precious.txt")); err != nil {
		t.Fatalf("stray directory was touched: %v", err)
	}

	// An explicit Discard removes a dirty worktree and its branch.
	path, err := wc.Provision("dirty")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(path, "wip.txt"), "wip")
	if err := wc.Discard("dirty"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("worktree still there: %v", err)
	}
	if exists, _ := wc.branchExists("feature/dirty"); exists {
		t.Fatal("branch still there")
	}
}

func TestCommitAllAndHasWork(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, err := wc.Provision("add-auth")
	if err != nil {
		t.Fatal(err)
	}
	if has, err := wc.HasWork("add-auth"); err != nil || has {
		t.Fatalf("HasWork on an empty branch = %v, %v", has, err)
	}
	if files, err := wc.CommitAll("add-auth"); err != nil || len(files) != 0 {
		t.Fatalf("CommitAll on a clean worktree = %v, %v", files, err)
	}

	writeFile(t, filepath.Join(path, "new.txt"), "n")
	if has, _ := wc.HasWork("add-auth"); !has {
		t.Fatal("dirty worktree counts as work")
	}
	files, err := wc.CommitAll("add-auth")
	if err != nil || len(files) != 1 || files[0] != "new.txt" {
		t.Fatalf("CommitAll = %v, %v", files, err)
	}
	if gitIn(t, path, "status", "--porcelain") != "" {
		t.Fatal("worktree still dirty after commit")
	}
	if msg := gitIn(t, path, "log", "-1", "--format=%B"); msg != "feat: Add auth\n\nChange: add-auth" {
		t.Fatalf("commit message = %q", msg)
	}
	if has, _ := wc.HasWork("add-auth"); !has {
		t.Fatal("a branch-only commit counts as work")
	}

	// An agent that already committed leaves nothing to commit: no empty commit.
	before := gitIn(t, path, "rev-parse", "HEAD")
	if files, err := wc.CommitAll("add-auth"); err != nil || len(files) != 0 {
		t.Fatalf("CommitAll = %v, %v", files, err)
	}
	if gitIn(t, path, "rev-parse", "HEAD") != before {
		t.Fatal("an empty commit was created")
	}
}

func TestCommitAllUsesComponentScope(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, _ := wc.Provision("add-auth")
	writeScopeFiles(t, filepath.Join(path, "openspec", "changes", "add-auth"), "tags:\n  components: [auth]\n")
	if _, err := wc.CommitAll("add-auth"); err != nil {
		t.Fatal(err)
	}
	if msg := gitIn(t, path, "log", "-1", "--format=%B"); msg != "feat(auth): Add auth\n\nChange: add-auth" {
		t.Fatalf("commit message = %q", msg)
	}
}

func TestMergeInto(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, _ := wc.Provision("ok-change")
	writeFile(t, filepath.Join(path, "ok.txt"), "ok")
	if _, err := wc.CommitAll("ok-change"); err != nil {
		t.Fatal(err)
	}
	target, err := wc.MergeInto("ok-change")
	if err != nil || target != "main" {
		t.Fatalf("MergeInto = %s, %v", target, err)
	}
	if _, err := os.Stat(filepath.Join(repo, "ok.txt")); err != nil {
		t.Fatalf("merged file missing: %v", err)
	}
	if msg := gitIn(t, repo, "log", "-1", "--format=%B"); msg != "feat: Ok change\n\nChange: ok-change" {
		t.Fatalf("merge message = %q", msg)
	}
	if err := wc.Remove("ok-change"); err != nil {
		t.Fatal(err)
	}
	if err := wc.DeleteBranch("ok-change"); err != nil {
		t.Fatal(err)
	}
}

func TestMergeIntoConflictIsAborted(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, _ := wc.Provision("conflict")
	writeFile(t, filepath.Join(path, "README.md"), "from branch")
	if _, err := wc.CommitAll("conflict"); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo, "README.md"), "from main")
	gitIn(t, repo, "commit", "-q", "-am", "main change")

	if _, err := wc.MergeInto("conflict"); err == nil {
		t.Fatal("expected a conflict")
	}
	if wc.mergeInProgress() {
		t.Fatal("the failed merge must be aborted")
	}
	if gitIn(t, repo, "status", "--porcelain") != "" {
		t.Fatal("main tree left dirty")
	}
	if exists, _ := wc.branchExists("feature/conflict"); !exists {
		t.Fatal("branch must be kept")
	}
}

func TestMergeIntoRefusesExistingMerge(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, _ := wc.Provision("blocked")
	writeFile(t, filepath.Join(path, "b.txt"), "b")
	if _, err := wc.CommitAll("blocked"); err != nil {
		t.Fatal(err)
	}
	// A merge started by the user.
	mergeHead := filepath.Join(repo, ".git", "MERGE_HEAD")
	writeFile(t, mergeHead, gitIn(t, repo, "rev-parse", "HEAD")+"\n")

	if _, err := wc.MergeInto("blocked"); !errors.Is(err, ErrMergeInProgress) {
		t.Fatalf("err = %v, want ErrMergeInProgress", err)
	}
	if _, err := os.Stat(mergeHead); err != nil {
		t.Fatalf("the user's merge state was touched: %v", err)
	}
}

func baseOf(wc *WorktreeController, change string) string {
	b, _ := wc.BaseBranch(change)
	return b
}

func TestProvisionRecordsBase(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	if _, err := wc.Provision("add-auth"); err != nil {
		t.Fatal(err)
	}
	if got := baseOf(wc, "add-auth"); got != "main" {
		t.Fatalf("base = %q, want main", got)
	}
}

func TestProvisionResumeKeepsBase(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	if _, err := wc.Provision("resume"); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "checkout", "-q", "-b", "other")
	if _, err := wc.Provision("resume"); err != nil {
		t.Fatal(err)
	}
	if got := baseOf(wc, "resume"); got != "main" {
		t.Fatalf("base = %q, want main", got)
	}
}

func TestProvisionDetachedHeadRecordsNoBase(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	gitIn(t, repo, "checkout", "-q", "--detach")
	if _, err := wc.Provision("detached"); err != nil {
		t.Fatal(err)
	}
	if got, ok := wc.BaseBranch("detached"); ok {
		t.Fatalf("unexpected base %q", got)
	}
}

func TestBaseDroppedWithBranch(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, _ := wc.Provision("gone")
	writeFile(t, filepath.Join(path, "g.txt"), "g")
	if _, err := wc.CommitAll("gone"); err != nil {
		t.Fatal(err)
	}
	if _, err := wc.MergeInto("gone"); err != nil {
		t.Fatal(err)
	}
	if err := wc.Remove("gone"); err != nil {
		t.Fatal(err)
	}
	if err := wc.DeleteBranch("gone"); err != nil {
		t.Fatal(err)
	}
	if got, ok := wc.BaseBranch("gone"); ok {
		t.Fatalf("base %q survived branch deletion", got)
	}
}

func TestMergeIntoRefusesOtherBranch(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, _ := wc.Provision("wrong-base")
	writeFile(t, filepath.Join(path, "w.txt"), "w")
	if _, err := wc.CommitAll("wrong-base"); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "checkout", "-q", "-b", "feature/other")
	head := gitIn(t, repo, "rev-parse", "HEAD")

	_, err := wc.MergeInto("wrong-base")
	var mm *BaseBranchMismatchError
	if !errors.Is(err, ErrBaseBranchMismatch) || !errors.As(err, &mm) || mm.Base != "main" || mm.Current != "feature/other" {
		t.Fatalf("err = %v", err)
	}
	if gitIn(t, repo, "rev-parse", "HEAD") != head || wc.mergeInProgress() {
		t.Fatal("the target must be untouched")
	}
	if !strings.Contains(err.Error(), "main") || !strings.Contains(err.Error(), "feature/other") {
		t.Fatalf("reason must name both branches: %v", err)
	}

	// Detached HEAD is refused too.
	gitIn(t, repo, "checkout", "-q", "--detach")
	if _, err := wc.MergeInto("wrong-base"); !errors.As(err, &mm) || mm.Current != "" {
		t.Fatalf("detached err = %v", err)
	}

	// Back on the base: the merge goes through.
	gitIn(t, repo, "checkout", "-q", "main")
	if _, err := wc.MergeInto("wrong-base"); err != nil {
		t.Fatal(err)
	}
}

func TestMergeIntoWithoutRecordedBase(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, _ := wc.Provision("legacy")
	gitIn(t, repo, "config", "--unset", baseConfigKey("legacy"))
	writeFile(t, filepath.Join(path, "l.txt"), "l")
	if _, err := wc.CommitAll("legacy"); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "checkout", "-q", "-b", "elsewhere")
	if target, err := wc.MergeInto("legacy"); err != nil || target != "elsewhere" {
		t.Fatalf("MergeInto = %s, %v", target, err)
	}
}

func commitFile(t *testing.T, dir, name, content string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, name), content)
	gitIn(t, dir, "add", name)
	gitIn(t, dir, "commit", "-q", "-m", "add "+name)
}

func TestTargetAheadAndIntegrate(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, _ := wc.Provision("integ")
	commitFile(t, path, "feat.txt", "f")

	if ahead, err := wc.TargetAhead("integ"); err != nil || ahead {
		t.Fatalf("unchanged target: ahead=%v err=%v", ahead, err)
	}

	commitFile(t, repo, "user.txt", "u")
	if ahead, err := wc.TargetAhead("integ"); err != nil || !ahead {
		t.Fatalf("advanced target: ahead=%v err=%v", ahead, err)
	}
	if err := wc.IntegrateTarget("integ", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(path, "user.txt")); err != nil {
		t.Fatalf("target commit not integrated: %v", err)
	}
	if n := gitIn(t, path, "rev-list", "--merges", "--count", "HEAD"); n != "1" {
		t.Fatalf("expected one merge commit in the worktree, got %s", n)
	}
	if ahead, _ := wc.TargetAhead("integ"); ahead {
		t.Fatal("target must no longer be ahead after integration")
	}
}

func TestIsMerged(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)

	if _, err := wc.IsMerged("absent"); err == nil {
		t.Fatal("a missing branch must be an error")
	}

	path, _ := wc.Provision("done")
	commitFile(t, path, "feat.txt", "f")
	if merged, err := wc.IsMerged("done"); err != nil || merged {
		t.Fatalf("unmerged branch: merged=%v err=%v", merged, err)
	}
	if _, err := wc.MergeInto("done"); err != nil {
		t.Fatal(err)
	}
	if merged, err := wc.IsMerged("done"); err != nil || !merged {
		t.Fatalf("merged branch: merged=%v err=%v", merged, err)
	}

	// A commit added to the branch after the merge is not merged.
	commitFile(t, path, "more.txt", "m")
	if merged, err := wc.IsMerged("done"); err != nil || merged {
		t.Fatalf("branch with a new commit: merged=%v err=%v", merged, err)
	}

	// Detached HEAD: compared against HEAD.
	gitIn(t, repo, "checkout", "-q", "--detach")
	if merged, err := wc.IsMerged("done"); err != nil || merged {
		t.Fatalf("detached HEAD: merged=%v err=%v", merged, err)
	}
}

func TestTargetAheadFalseAfterManualIntegration(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, _ := wc.Provision("manual")
	commitFile(t, path, "feat.txt", "f")
	commitFile(t, repo, "user.txt", "u")
	gitIn(t, path, "merge", "--no-edit", "main")
	if ahead, err := wc.TargetAhead("manual"); err != nil || ahead {
		t.Fatalf("ahead=%v err=%v", ahead, err)
	}
}

func TestIntegrateTargetConflictIsAborted(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, _ := wc.Provision("clash")
	commitFile(t, path, "README.md", "from branch")
	commitFile(t, repo, "README.md", "from main")
	head := gitIn(t, path, "rev-parse", "HEAD")

	if err := wc.IntegrateTarget("clash", "main"); err == nil {
		t.Fatal("expected a conflict")
	}
	if _, ok := wc.runGitInCode(path, "rev-parse", "-q", "--verify", "MERGE_HEAD"); ok {
		t.Fatal("MERGE_HEAD left behind")
	}
	if gitIn(t, path, "rev-parse", "HEAD") != head || gitIn(t, path, "status", "--porcelain") != "" {
		t.Fatal("worktree not restored")
	}
	if b, _ := os.ReadFile(filepath.Join(path, "README.md")); string(b) != "from branch" {
		t.Fatalf("README = %q", b)
	}
}

func TestReviewMarker(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	if _, err := wc.Provision("add-auth"); err != nil {
		t.Fatal(err)
	}
	if has, err := wc.HasReview("add-auth"); err != nil || has {
		t.Fatalf("HasReview before mark = %v, %v", has, err)
	}
	if err := wc.MarkReview("add-auth"); err != nil {
		t.Fatal(err)
	}
	if has, err := wc.HasReview("add-auth"); err != nil || !has {
		t.Fatalf("HasReview after mark = %v, %v", has, err)
	}
	if got := gitIn(t, repo, "config", "--get", "branch.feature/add-auth.opensp8c-review"); got == "" {
		t.Error("marker value should be a timestamp")
	}
	if out := gitIn(t, repo, "status", "--porcelain"); out != "" {
		t.Errorf("marking must not touch tracked files, status: %q", out)
	}
	if err := wc.ClearReview("add-auth"); err != nil {
		t.Fatal(err)
	}
	if err := wc.ClearReview("add-auth"); err != nil {
		t.Errorf("clearing an absent marker must not fail: %v", err)
	}
	if has, _ := wc.HasReview("add-auth"); has {
		t.Error("marker should be cleared")
	}
}

func TestReviewMarkerDisappearsWithBranch(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	if _, err := wc.Provision("add-auth"); err != nil {
		t.Fatal(err)
	}
	if err := wc.MarkReview("add-auth"); err != nil {
		t.Fatal(err)
	}
	if err := wc.Discard("add-auth"); err != nil {
		t.Fatal(err)
	}
	if has, err := wc.HasReview("add-auth"); err != nil || has {
		t.Errorf("marker should vanish with the branch: %v, %v", has, err)
	}
}

func TestCleanup(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)

	// Worktree present (dirty), branch and review marker: everything goes.
	path, err := wc.Provision("present")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(path, "wip.txt"), "wip")
	if err := wc.MarkReview("present"); err != nil {
		t.Fatal(err)
	}
	if err := wc.Cleanup("present"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("worktree still there: %v", err)
	}
	if ok, _ := wc.branchExists("feature/present"); ok {
		t.Fatal("branch still there")
	}
	if has, _ := wc.HasReview("present"); has {
		t.Fatal("review marker still there")
	}

	// Worktree deleted by hand: pruned, branch deleted, marker lifted.
	path, err = wc.Provision("vanished")
	if err != nil {
		t.Fatal(err)
	}
	if err := wc.MarkReview("vanished"); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
	if err := wc.Cleanup("vanished"); err != nil {
		t.Fatal(err)
	}
	if ok, _ := wc.branchExists("feature/vanished"); ok {
		t.Fatal("branch still there")
	}
	if out := gitIn(t, repo, "worktree", "list", "--porcelain"); strings.Contains(out, "vanished") {
		t.Fatalf("stale worktree entry kept:\n%s", out)
	}

	// Nothing left at all: not an error.
	if err := wc.Cleanup("never-existed"); err != nil {
		t.Fatalf("Cleanup of an absent change: %v", err)
	}
}

func commitChange(t *testing.T, repo, change string) {
	t.Helper()
	dir := filepath.Join(repo, "openspec", "changes", change)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("- [ ] 1.1 x\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestChangeCommitted(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	if ok, err := wc.ChangeCommitted("add-auth"); err != nil || ok {
		t.Fatalf("absent: ok=%v err=%v", ok, err)
	}
	commitChange(t, repo, "add-auth")
	if ok, err := wc.ChangeCommitted("add-auth"); err != nil || ok {
		t.Fatalf("untracked: ok=%v err=%v", ok, err)
	}
	gitIn(t, repo, "add", ".")
	if ok, err := wc.ChangeCommitted("add-auth"); err != nil || ok {
		t.Fatalf("staged only: ok=%v err=%v", ok, err)
	}
	gitIn(t, repo, "commit", "-q", "-m", "change")
	if ok, err := wc.ChangeCommitted("add-auth"); err != nil || !ok {
		t.Fatalf("committed: ok=%v err=%v", ok, err)
	}
	notRepo := NewWorktreeController(t.TempDir(), "ws", t.TempDir())
	if _, err := notRepo.ChangeCommitted("add-auth"); err == nil {
		t.Fatal("expected an error outside a repository")
	}
}

func TestRecreateFromHeadStaleBranchWithoutWork(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	if _, err := wc.Provision("add-auth"); err != nil {
		t.Fatal(err)
	}
	commitChange(t, repo, "add-auth")
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-q", "-m", "change")

	path, err := wc.RecreateFromHead("add-auth")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(path, "openspec", "changes", "add-auth", "tasks.md")); err != nil {
		t.Fatalf("tasks.md missing from recreated worktree: %v", err)
	}
	if base, ok := wc.BaseBranch("add-auth"); !ok || base != "main" {
		t.Fatalf("base = %q ok=%v", base, ok)
	}
}

func TestRecreateFromHeadKeepsBranchWithWork(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, err := wc.Provision("add-auth")
	if err != nil {
		t.Fatal(err)
	}

	// Dirty worktree.
	if err := os.WriteFile(filepath.Join(path, "wip.txt"), []byte("w"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := wc.RecreateFromHead("add-auth"); err == nil {
		t.Fatal("expected an error for a dirty worktree")
	}
	if _, err := os.Stat(filepath.Join(path, "wip.txt")); err != nil {
		t.Fatalf("dirty worktree was touched: %v", err)
	}

	// Own commit.
	gitIn(t, path, "add", ".")
	gitIn(t, path, "commit", "-q", "-m", "work")
	if _, err := wc.RecreateFromHead("add-auth"); err == nil {
		t.Fatal("expected an error for a branch with its own commit")
	}
	if exists, _ := wc.branchExists("feature/add-auth"); !exists {
		t.Fatal("branch was deleted")
	}
}
