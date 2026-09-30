package pool

import (
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

func TestProvisionFirstTime(t *testing.T) {
	repo := newTestRepo(t)
	wc := NewWorktreeController(repo)
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
	wc := NewWorktreeController(repo)
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
	wc := NewWorktreeController(repo)
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
	wc := NewWorktreeController(repo)
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
	if _, err := NewWorktreeController(notRepo).Provision("add-auth"); err == nil {
		t.Fatal("expected an error outside a git repository")
	}
}
