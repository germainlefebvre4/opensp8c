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

// WorktreeController manages isolated git worktrees for workers.
type WorktreeController struct {
	repoRoot string
}

func NewWorktreeController(repoRoot string) *WorktreeController {
	return &WorktreeController{
		repoRoot: repoRoot,
	}
}

func (wc *WorktreeController) worktreesDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".opensp8c", "worktrees")
}

// Provision makes sure the branch and the git worktree of the given change
// exist and returns the absolute path of the worktree. It is idempotent: an
// existing branch and worktree are reused as-is, and the working tree is never
// reset or cleaned so uncommitted agent changes survive a resume.
func (wc *WorktreeController) Provision(changeName string) (string, error) {
	err := os.MkdirAll(wc.worktreesDir(), 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create worktrees directory: %w", err)
	}

	branchName := "feature/" + changeName
	worktreePath := filepath.Join(wc.worktreesDir(), "wt-"+changeName)

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

	if wc.isWorktreeOnBranch(worktreePath, branchName) {
		if stat, statErr := os.Stat(worktreePath); statErr == nil && stat.IsDir() {
			return worktreePath, nil
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

// Remove cleanly removes a worktree.
func (wc *WorktreeController) Remove(changeName string) error {
	worktreePath := filepath.Join(wc.worktreesDir(), "wt-"+changeName)

	// Force remove the worktree
	_, err := wc.runGit("worktree", "remove", "--force", worktreePath)
	if err != nil {
		// Fallback to manual removal and prune if `git worktree remove` fails
		os.RemoveAll(worktreePath)
		wc.runGit("worktree", "prune")
		return fmt.Errorf("failed to remove worktree safely: %w", err)
	}
	return nil
}

// MergeAndCleanup merges the change branch into the main branch and deletes the feature branch.
func (wc *WorktreeController) MergeAndCleanup(changeName string) error {
	branchName := "feature/" + changeName

	// 1. Ensure worktree is removed first, so we aren't blocked by checked out branch.
	wc.Remove(changeName)

	// 2. Merge branch into current (presumably main)
	// We use the main repo workspace for this
	_, err := wc.runGit("merge", "--no-ff", "-m", "Merge change "+changeName, branchName)
	if err != nil {
		// Merge conflict or failure
		// We could abort the merge
		wc.runGit("merge", "--abort")
		return fmt.Errorf("merge failed (aborted): %w", err)
	}

	// 3. Delete the feature branch
	_, err = wc.runGit("branch", "-D", branchName)
	if err != nil {
		return fmt.Errorf("failed to delete branch after merge: %w", err)
	}

	return nil
}

// CleanupDiscard removes the worktree and deletes the branch without merging.
func (wc *WorktreeController) CleanupDiscard(changeName string) error {
	branchName := "feature/" + changeName
	wc.Remove(changeName)
	_, err := wc.runGit("branch", "-D", branchName)
	return err
}

func (wc *WorktreeController) runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = wc.repoRoot
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
