package pool

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/glefebvre/opensp8c/internal/openspec"
)

// worktreesEnv overrides the root directory of all worktrees.
const worktreesEnv = "OPENSP8C_WORKTREES_DIR"

// ErrMergeInProgress reports that the repository already has a merge in
// progress (typically started by the user), which the pool must never touch.
var ErrMergeInProgress = errors.New("un merge est déjà en cours dans le dépôt")

// BaseBranchMismatchError reports that the repository is not on the branch the
// change branch was created from, so merging would land the work elsewhere.
// Current is empty when HEAD is detached.
type BaseBranchMismatchError struct{ Base, Current string }

func (e *BaseBranchMismatchError) Error() string {
	if e.Current == "" {
		return fmt.Sprintf("le dépôt n'est sur aucune branche (HEAD détaché) alors que le changement est issu de « %s » : revenez sur « %s » puis reprenez le worker", e.Base, e.Base)
	}
	return fmt.Sprintf("le dépôt est sur la branche « %s » alors que le changement est issu de « %s » : revenez sur « %s » puis reprenez le worker", e.Current, e.Base, e.Base)
}

// ErrBaseBranchMismatch is matched with errors.Is against *BaseBranchMismatchError.
var ErrBaseBranchMismatch = errors.New("branche de base différente de la branche courante")

func (e *BaseBranchMismatchError) Is(target error) bool { return target == ErrBaseBranchMismatch }

// DefaultWorktreesRoot returns the root directory of the worktrees: the
// OPENSP8C_WORKTREES_DIR environment variable when set, else
// ~/.opensp8c/worktrees.
func DefaultWorktreesRoot() string {
	if dir := strings.TrimSpace(os.Getenv(worktreesEnv)); dir != "" {
		return dir
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".opensp8c", "worktrees")
}

// WorktreeController manages isolated git worktrees for workers.
type WorktreeController struct {
	repoRoot    string
	workspaceID string
	root        string
}

// NewWorktreeController builds a controller for repoRoot whose worktrees live
// under <worktreesRoot>/<workspaceID>/wt-<change>. An empty worktreesRoot
// selects DefaultWorktreesRoot.
func NewWorktreeController(repoRoot, workspaceID, worktreesRoot string) *WorktreeController {
	if worktreesRoot == "" {
		worktreesRoot = DefaultWorktreesRoot()
	}
	return &WorktreeController{
		repoRoot:    repoRoot,
		workspaceID: workspaceID,
		root:        worktreesRoot,
	}
}

// workspaceDir is the directory holding this workspace's worktrees.
func (wc *WorktreeController) workspaceDir() string {
	id := wc.workspaceID
	if id == "" {
		id = "default"
	}
	return filepath.Join(wc.root, id)
}

// worktreePath is the current location of the worktree of a change.
func (wc *WorktreeController) worktreePath(changeName string) string {
	return filepath.Join(wc.workspaceDir(), "wt-"+changeName)
}

// legacyWorktreePath is the location used before worktrees were isolated per
// workspace.
func (wc *WorktreeController) legacyWorktreePath(changeName string) string {
	return filepath.Join(wc.root, "wt-"+changeName)
}

// resolvePath returns where the worktree of a change lives: the legacy
// location when a worktree of this repository on the change's branch is
// registered there, the per-workspace location otherwise.
func (wc *WorktreeController) resolvePath(changeName string) string {
	branchName := "feature/" + changeName
	legacy := wc.legacyWorktreePath(changeName)
	current := wc.worktreePath(changeName)
	if wc.isWorktreeOnBranch(current, branchName) {
		return current
	}
	if wc.isWorktreeOnBranch(legacy, branchName) {
		if stat, err := os.Stat(legacy); err == nil && stat.IsDir() {
			return legacy
		}
	}
	return current
}

// Provision makes sure the branch and the git worktree of the given change
// exist and returns the absolute path of the worktree. It is idempotent: an
// existing branch and worktree (including one at the legacy, non-workspace
// location) are reused as-is, and the working tree is never reset or cleaned
// so uncommitted agent changes survive a resume.
func (wc *WorktreeController) Provision(changeName string) (string, error) {
	err := os.MkdirAll(wc.workspaceDir(), 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create worktrees directory: %w", err)
	}

	branchName := "feature/" + changeName
	worktreePath := wc.worktreePath(changeName)

	exists, err := wc.branchExists(branchName)
	if err != nil {
		return "", fmt.Errorf("failed to check branch %s: %w", branchName, err)
	}

	if !exists {
		// First time: create the branch together with its worktree, remembering
		// the branch it starts from (empty when HEAD is detached).
		base := wc.currentBranchName()
		if _, err := wc.runGit("worktree", "add", "-b", branchName, worktreePath); err != nil {
			return "", fmt.Errorf("failed to create worktree with new branch: %w", err)
		}
		wc.recordBase(changeName, base)
		return worktreePath, nil
	}

	if resolved := wc.resolvePath(changeName); wc.isWorktreeOnBranch(resolved, branchName) {
		if stat, statErr := os.Stat(resolved); statErr == nil && stat.IsDir() {
			return resolved, nil
		}
	}

	// Branch exists without a usable worktree: drop stale entries, then check
	// the existing branch out into a new worktree.
	_, _ = wc.runGit("worktree", "prune")
	if _, err := wc.runGit("worktree", "add", worktreePath, branchName); err != nil {
		return "", fmt.Errorf("failed to checkout existing branch to worktree: %w", err)
	}
	return worktreePath, nil
}

// branchExists interprets the exit code of git show-ref: 0 exists, 1 absent,
// anything else is an error.
func (wc *WorktreeController) branchExists(branchName string) (bool, error) {
	_, code, err := wc.runGitCode("show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	switch {
	case err != nil:
		return false, err
	case code == 0:
		return true, nil
	case code == 1:
		return false, nil
	default:
		return false, fmt.Errorf("git show-ref exited with status %d", code)
	}
}

// isWorktreeOnBranch reports whether git lists path as a worktree on branch.
func (wc *WorktreeController) isWorktreeOnBranch(path, branchName string) bool {
	out, err := wc.runGit("worktree", "list", "--porcelain")
	if err != nil {
		return false
	}
	wantPath, wantBranch := filepath.Clean(path), "branch refs/heads/"+branchName
	current := ""
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "worktree "):
			current = filepath.Clean(strings.TrimPrefix(line, "worktree "))
		case line == wantBranch && sameFile(current, wantPath):
			return true
		}
	}
	return false
}

func sameFile(a, b string) bool {
	if a == b {
		return true
	}
	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)
	return errA == nil && errB == nil && ra == rb
}

// Remove removes the worktree of a change without ever forcing: a worktree
// holding uncommitted changes (or locked) is kept and the error is returned.
// Only a worktree whose directory already vanished is pruned.
func (wc *WorktreeController) Remove(changeName string) error {
	worktreePath := wc.resolvePath(changeName)

	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		if _, perr := wc.runGit("worktree", "prune"); perr != nil {
			return fmt.Errorf("failed to prune worktrees: %w", perr)
		}
		return nil
	}
	if _, err := wc.runGit("worktree", "remove", worktreePath); err != nil {
		return fmt.Errorf("failed to remove worktree safely: %w", err)
	}
	return nil
}

// Discard removes the worktree and the branch of a change, uncommitted work
// included. It is the only forced removal and is reserved for an explicit
// user cancellation; it refuses any path that is not a registered worktree of
// the repository.
func (wc *WorktreeController) Discard(changeName string) error {
	worktreePath := wc.resolvePath(changeName)
	if !wc.isRegisteredWorktree(worktreePath) {
		return fmt.Errorf("%s is not a registered worktree of %s: nothing removed", worktreePath, wc.repoRoot)
	}
	if _, err := wc.runGit("worktree", "remove", "--force", worktreePath); err != nil {
		return fmt.Errorf("failed to discard worktree: %w", err)
	}
	branchName := "feature/" + changeName
	if exists, err := wc.branchExists(branchName); err == nil && exists {
		if _, err := wc.runGit("branch", "-D", branchName); err != nil {
			return err
		}
	}
	return nil
}

// Cleanup removes everything a change left in the repository: its worktree
// (uncommitted work included), its branch and its review marker. Unlike
// Discard it tolerates a worktree that is already gone (pruned) and a branch
// that does not exist; it is reserved for an explicit user deletion.
func (wc *WorktreeController) Cleanup(changeName string) error {
	worktreePath := wc.resolvePath(changeName)
	if wc.isRegisteredWorktree(worktreePath) {
		if _, err := wc.runGit("worktree", "remove", "--force", worktreePath); err != nil {
			return fmt.Errorf("failed to remove worktree: %w", err)
		}
	}
	if _, err := wc.runGit("worktree", "prune"); err != nil {
		return fmt.Errorf("failed to prune worktrees: %w", err)
	}
	branchName := "feature/" + changeName
	if exists, err := wc.branchExists(branchName); err != nil {
		return err
	} else if exists {
		if _, err := wc.runGit("branch", "-D", branchName); err != nil {
			return err
		}
	}
	return wc.ClearReview(changeName)
}

// isRegisteredWorktree reports whether git lists path as a worktree of the repository.
func (wc *WorktreeController) isRegisteredWorktree(path string) bool {
	out, err := wc.runGit("worktree", "list", "--porcelain")
	if err != nil {
		return false
	}
	want := filepath.Clean(path)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") && sameFile(filepath.Clean(strings.TrimPrefix(line, "worktree ")), want) {
			return true
		}
	}
	return false
}

// CommitAll commits every uncommitted change of the worktree of changeName
// into its branch. It only commits when the worktree is dirty and returns the
// committed files (nil when there was nothing to commit).
func (wc *WorktreeController) CommitAll(changeName string) ([]string, error) {
	path := wc.resolvePath(changeName)
	status, err := wc.runGitIn(path, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	if status == "" {
		return nil, nil
	}
	if _, err := wc.runGitIn(path, "add", "-A"); err != nil {
		return nil, err
	}
	files, err := wc.runGitIn(path, "diff", "--cached", "--name-only")
	if err != nil {
		return nil, err
	}
	if _, err := wc.runGitIn(path, "commit", "-q", "-m", "feat("+changeName+"): apply OpenSpec change"); err != nil {
		return nil, err
	}
	return strings.Split(files, "\n"), nil
}

// CommitFile commits only the given file (relative to the worktree) of changeName
// into its branch, leaving any other uncommitted change alone.
func (wc *WorktreeController) CommitFile(changeName, relPath, message string) error {
	path := wc.resolvePath(changeName)
	if _, err := wc.runGitIn(path, "add", "--", relPath); err != nil {
		return err
	}
	if _, err := wc.runGitIn(path, "commit", "-q", "-m", message, "--only", "--", relPath); err != nil {
		_, _ = wc.runGitIn(path, "reset", "-q", "--", relPath)
		return err
	}
	return nil
}

// HasWork reports whether the change produced anything: a dirty worktree or
// commits on its branch that the repository's current branch does not have.
func (wc *WorktreeController) HasWork(changeName string) (bool, error) {
	status, err := wc.runGitIn(wc.resolvePath(changeName), "status", "--porcelain")
	if err != nil {
		return false, err
	}
	if status != "" {
		return true, nil
	}
	count, err := wc.runGit("rev-list", "--count", "HEAD..feature/"+changeName)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(count) != "0", nil
}

func baseConfigKey(changeName string) string {
	return "branch.feature/" + changeName + ".opensp8c-base"
}

// MarkReview records that the change awaits review, in the repository's git
// configuration (value: RFC 3339 timestamp). Git drops it with the branch and
// it touches no tracked file. Unlike recordBase, a failure is returned.
func (wc *WorktreeController) MarkReview(changeName string) error {
	_, err := wc.runGit("config", openspec.ReviewKey(changeName), time.Now().UTC().Format(time.RFC3339))
	return err
}

// ClearReview lifts the review marker of a change; lifting an absent marker is
// not an error.
func (wc *WorktreeController) ClearReview(changeName string) error {
	_, code, err := wc.runGitCode("config", "--unset", openspec.ReviewKey(changeName))
	switch {
	case err != nil:
		return err
	case code == 0, code == 5:
		return nil
	default:
		return fmt.Errorf("git config --unset exited with status %d", code)
	}
}

// HasReview reports whether the change carries a review marker.
func (wc *WorktreeController) HasReview(changeName string) (bool, error) {
	_, code, err := wc.runGitCode("config", "--get", openspec.ReviewKey(changeName))
	switch {
	case err != nil:
		return false, err
	case code == 0:
		return true, nil
	case code == 1:
		return false, nil
	default:
		return false, fmt.Errorf("git config --get exited with status %d", code)
	}
}

// currentBranchName returns the checked-out branch, or "" when HEAD is detached.
func (wc *WorktreeController) currentBranchName() string {
	out, err := wc.runGit("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return out
}

// recordBase stores the base branch of a change in the repository's git
// configuration (it is dropped by git together with the branch). Best effort:
// a failure leaves the branch without base, as before.
func (wc *WorktreeController) recordBase(changeName, base string) {
	if base == "" {
		return
	}
	if _, err := wc.runGit("config", baseConfigKey(changeName), base); err != nil {
		log.Printf("[worktree] cannot record base branch of %s: %v\n", changeName, err)
	}
}

// BaseBranch returns the recorded base branch of a change; ok is false when
// none was recorded.
func (wc *WorktreeController) BaseBranch(changeName string) (string, bool) {
	out, code, err := wc.runGitCode("config", "--get", baseConfigKey(changeName))
	if err != nil || code != 0 || out == "" {
		return "", false
	}
	return out, true
}

// CheckBase returns a *BaseBranchMismatchError when a base is recorded and the
// repository is not on it (or HEAD is detached); nil otherwise.
func (wc *WorktreeController) CheckBase(changeName string) error {
	base, ok := wc.BaseBranch(changeName)
	if !ok {
		return nil
	}
	if current := wc.currentBranchName(); current != base {
		return &BaseBranchMismatchError{Base: base, Current: current}
	}
	return nil
}

// CurrentBranch returns the branch currently checked out in the repository.
func (wc *WorktreeController) CurrentBranch() string {
	out, err := wc.runGit("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "HEAD"
	}
	return out
}

// MergeInto merges the change branch into the repository's current branch and
// returns that target branch. It refuses to run while a merge is already in
// progress and only aborts a merge that it started itself.
func (wc *WorktreeController) MergeInto(changeName string) (string, error) {
	target := wc.CurrentBranch()
	if wc.mergeInProgress() {
		return target, ErrMergeInProgress
	}
	if err := wc.CheckBase(changeName); err != nil {
		return target, err
	}
	_, err := wc.runGit("merge", "--no-ff", "-m", "Merge change "+changeName, "feature/"+changeName)
	if err != nil {
		if wc.mergeInProgress() {
			_, _ = wc.runGit("merge", "--abort")
		}
		return target, fmt.Errorf("merge failed (aborted): %w", err)
	}
	return target, nil
}

// TargetAhead reports whether the repository's current branch (the merge
// target) holds a commit that feature/<change> does not have. It needs no
// recorded state, so it stays true-to-date after a manual integration.
func (wc *WorktreeController) TargetAhead(changeName string) (bool, error) {
	_, code, err := wc.runGitCode("merge-base", "--is-ancestor", wc.CurrentBranch(), "feature/"+changeName)
	switch {
	case err != nil:
		return false, err
	case code == 0:
		return false, nil
	case code == 1:
		return true, nil
	default:
		return false, fmt.Errorf("git merge-base --is-ancestor exited with status %d", code)
	}
}

// IsMerged reports whether feature/<change> is already contained in the
// repository's current branch. A missing branch is an error.
func (wc *WorktreeController) IsMerged(changeName string) (bool, error) {
	_, code, err := wc.runGitCode("merge-base", "--is-ancestor", "feature/"+changeName, wc.CurrentBranch())
	switch {
	case err != nil:
		return false, err
	case code == 0:
		return true, nil
	case code == 1:
		return false, nil
	default:
		return false, fmt.Errorf("git merge-base --is-ancestor exited with status %d", code)
	}
}

// IntegrateTarget merges the target branch into the change branch, inside the
// worktree (which must be clean). A failed merge is aborted so the branch and
// the worktree are left as they were.
func (wc *WorktreeController) IntegrateTarget(changeName, target string) error {
	path := wc.resolvePath(changeName)
	if _, err := wc.runGitIn(path, "merge", "--no-edit", target); err != nil {
		if _, ok := wc.runGitInCode(path, "rev-parse", "-q", "--verify", "MERGE_HEAD"); ok {
			_, _ = wc.runGitIn(path, "merge", "--abort")
		}
		return fmt.Errorf("conflit lors de l'intégration de %s : %w", target, err)
	}
	return nil
}

// runGitInCode reports whether a git command in dir succeeded.
func (wc *WorktreeController) runGitInCode(dir string, args ...string) (string, bool) {
	out, err := wc.runGitIn(dir, args...)
	return out, err == nil
}

func (wc *WorktreeController) mergeInProgress() bool {
	_, code, err := wc.runGitCode("rev-parse", "-q", "--verify", "MERGE_HEAD")
	return err == nil && code == 0
}

// DeleteBranch deletes the merged change branch; git refuses when it is not
// fully merged.
func (wc *WorktreeController) DeleteBranch(changeName string) error {
	_, err := wc.runGit("branch", "-d", "feature/"+changeName)
	return err
}

func (wc *WorktreeController) runGit(args ...string) (string, error) {
	return wc.runGitIn(wc.repoRoot, args...)
}

// runGitIn runs git in dir (the repository or one of its worktrees).
func (wc *WorktreeController) runGitIn(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %v failed: %w, stderr: %s", args, err, errOut.String())
	}
	return strings.TrimSpace(out.String()), nil
}

// runGitCode runs git and returns its exit code. err is non-nil only when git
// could not be run at all; a non-zero exit is reported through the code.
func (wc *WorktreeController) runGitCode(args ...string) (string, int, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = wc.repoRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return strings.TrimSpace(out.String()), exitErr.ExitCode(), nil
		}
		return "", -1, fmt.Errorf("git %v failed: %w", args, err)
	}
	return strings.TrimSpace(out.String()), 0, nil
}
