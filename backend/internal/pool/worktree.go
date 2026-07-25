package pool

import (
	"bytes"
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

// Provision creates a new branch (if needed) and a git worktree for the given change.
// It returns the absolute path to the provisioned worktree.
func (wc *WorktreeController) Provision(changeName string) (string, error) {
	err := os.MkdirAll(wc.worktreesDir(), 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create worktrees directory: %w", err)
	}

	branchName := "feature/" + changeName
	worktreePath := filepath.Join(wc.worktreesDir(), "wt-"+changeName)

	// Clean up any existing dirty worktree/branch state if it exists locally,
	// but for simplicity, let's just attempt to create or reuse.
	// 1. Check if branch exists
	branchExists, err := wc.runGit("show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	if err != nil && !strings.Contains(err.Error(), "exit status 1") { // usually exit 1 means doesn't exist
		// Some other error
	}

	if branchExists == "" {
		// Branch doesn't exist, create worktree with new branch (-b)
		_, err = wc.runGit("worktree", "add", "-b", branchName, worktreePath)
		if err != nil {
			return "", fmt.Errorf("failed to create worktree with new branch: %w", err)
		}
	} else {
		// Branch exists, just add worktree for existing branch
		_, err = wc.runGit("worktree", "add", worktreePath, branchName)
		if err != nil {
			// If worktree already exists, it might fail. Check if the directory is already a worktree.
			if stat, statErr := os.Stat(worktreePath); statErr == nil && stat.IsDir() {
				// We assume it's valid.
			} else {
				return "", fmt.Errorf("failed to checkout existing branch to worktree: %w", err)
			}
		}
	}

	return worktreePath, nil
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
