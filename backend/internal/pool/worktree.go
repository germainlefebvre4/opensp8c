package pool

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// worktreesEnv overrides the root directory of all worktrees.
const worktreesEnv = "OPENSP8C_WORKTREES_DIR"

// ErrMergeInProgress reports that the repository already has a merge in
// progress (typically started by the user), which the pool must never touch.
var ErrMergeInProgress = errors.New("un merge est déjà en cours dans le dépôt")

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
		// First time: create the branch together with its worktree.
		if _, err := wc.runGit("worktree", "add", "-b", branchName, worktreePath); err != nil {
			return "", fmt.Errorf("failed to create worktree with new branch: %w", err)
		}
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
	_, err := wc.runGit("merge", "--no-ff", "-m", "Merge change "+changeName, "feature/"+changeName)
	if err != nil {
		if wc.mergeInProgress() {
			_, _ = wc.runGit("merge", "--abort")
		}
		return target, fmt.Errorf("merge failed (aborted): %w", err)
	}
	return target, nil
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
