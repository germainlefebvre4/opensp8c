package activity

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/workspace"
)

func runGitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test Author",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test Author",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed in %s: %v\nOutput: %s", args, dir, err, string(out))
	}
	return string(out)
}

func TestGitWatcher_PollsNewCommits(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	// 1. Initialize main git repo
	runGitCmd(t, wsDir, "init")
	runGitCmd(t, wsDir, "config", "user.name", "Test Author")
	runGitCmd(t, wsDir, "config", "user.email", "test@example.com")
	runGitCmd(t, wsDir, "config", "commit.gpgsign", "false")

	changeName := "feature-1"
	changeDir := filepath.Join(wsDir, "openspec", "changes", changeName)
	if err := os.MkdirAll(changeDir, 0755); err != nil {
		t.Fatalf("failed to create change dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, ".openspec.yaml"), []byte("schema: spec-driven\ncreated: \"2024-01-01\"\n"), 0644); err != nil {
		t.Fatalf("failed to write meta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("- [ ] Task 1\n"), 0644); err != nil {
		t.Fatalf("failed to write tasks.md: %v", err)
	}

	runGitCmd(t, wsDir, "add", "-A")
	runGitCmd(t, wsDir, "commit", "-m", "initial commit")

	// 2. Create worktree
	worktreesDir := filepath.Join(tmpDir, "worktrees")
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		t.Fatalf("failed to create worktrees dir: %v", err)
	}
	wtPath := filepath.Join(worktreesDir, "wt-"+changeName)
	runGitCmd(t, wsDir, "worktree", "add", "-b", "feature/"+changeName, wtPath)

	absWsPath, err := filepath.Abs(wsDir)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	wsID := workspace.StableID(absWsPath)
	cfg := &config.Config{
		Workspaces: []config.WorkspaceConfig{
			{Name: "test", Path: wsDir},
		},
	}

	actDir := filepath.Join(tmpDir, "activity")
	actStore := NewStore(actDir, nil)
	watcher := NewGitWatcher(actStore, cfg, worktreesDir)

	ctx := context.Background()

	// Initial poll sets baseline HEAD
	watcher.PollOnce(ctx)

	entries, err := actStore.Read(wsID, changeName)
	if err != nil {
		t.Fatalf("failed to read activity: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries on baseline poll, got %d", len(entries))
	}

	// 3. Make a commit in the worktree
	dummyFile := filepath.Join(wtPath, "feature.go")
	if err := os.WriteFile(dummyFile, []byte("package feature\n"), 0644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}
	runGitCmd(t, wtPath, "add", "feature.go")
	runGitCmd(t, wtPath, "commit", "-m", "feat: implement feature")

	// 4. Poll again
	watcher.PollOnce(ctx)

	entries, err = actStore.Read(wsID, changeName)
	if err != nil {
		t.Fatalf("failed to read activity after commit: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 git.commit entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Type != "git.commit" {
		t.Errorf("expected type git.commit, got %s", e.Type)
	}
	if e.Category != "git" {
		t.Errorf("expected category git, got %s", e.Category)
	}
	if e.Summary != "feat: implement feature" {
		t.Errorf("expected summary 'feat: implement feature', got %q", e.Summary)
	}
	if msg, ok := e.Meta["message"].(string); !ok || msg != "feat: implement feature" {
		t.Errorf("expected meta.message 'feat: implement feature', got %v", e.Meta["message"])
	}
	if author, ok := e.Meta["author"].(string); !ok || author != "Test Author" {
		t.Errorf("expected meta.author 'Test Author', got %v", e.Meta["author"])
	}
	if hash, ok := e.Meta["hash"].(string); !ok || len(hash) != 40 {
		t.Errorf("expected valid 40-char SHA in meta.hash, got %v", e.Meta["hash"])
	}

	// 5. Polling once more without new commits should not duplicate entries
	watcher.PollOnce(ctx)
	entries, _ = actStore.Read(wsID, changeName)
	if len(entries) != 1 {
		t.Fatalf("expected still 1 entry, got %d", len(entries))
	}
}
