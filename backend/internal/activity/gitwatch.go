package activity

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/workspace"
)

type GitWatcher struct {
	store        *Store
	cfg          *config.Config
	worktreesDir string
	lastHEADs    map[string]string // key: wsID + "/" + changeName -> commit SHA
	mu           sync.Mutex
}

func NewGitWatcher(store *Store, cfg *config.Config, worktreesDir string) *GitWatcher {
	if worktreesDir == "" {
		if dir := os.Getenv("OPENSP8C_WORKTREES_DIR"); dir != "" {
			worktreesDir = dir
		} else {
			homeDir, _ := os.UserHomeDir()
			worktreesDir = filepath.Join(homeDir, ".opensp8c", "worktrees")
		}
	}
	return &GitWatcher{
		store:        store,
		cfg:          cfg,
		worktreesDir: worktreesDir,
		lastHEADs:    make(map[string]string),
	}
}

type gitCommitInfo struct {
	Hash    string
	Author  string
	Date    string
	Message string
}

func (w *GitWatcher) PollOnce(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cfg == nil {
		return
	}

	for _, ws := range w.cfg.Workspaces {
		absPath, err := filepath.Abs(ws.Path)
		if err != nil {
			continue
		}
		wsID := workspace.StableID(absPath)

		changes, err := openspec.ListChanges(absPath)
		if err != nil {
			continue
		}

		for _, ch := range changes {
			key := wsID + "/" + ch.Name
			// Per-workspace location first, then the legacy one.
			wtPath := filepath.Join(w.worktreesDir, wsID, "wt-"+ch.Name)
			stat, err := os.Stat(wtPath)
			if err != nil || !stat.IsDir() {
				wtPath = filepath.Join(w.worktreesDir, "wt-"+ch.Name)
				stat, err = os.Stat(wtPath)
			}
			if err != nil || !stat.IsDir() {
				// Worktree inactive or removed
				delete(w.lastHEADs, key)
				continue
			}

			head, err := gitRevParseHead(ctx, wtPath)
			if err != nil || head == "" {
				continue
			}

			last, exists := w.lastHEADs[key]
			if !exists {
				// Initial observation: set baseline without logging historical commits
				w.lastHEADs[key] = head
				continue
			}

			if last == head {
				continue
			}

			commits, err := gitLogRange(ctx, wtPath, last, head)
			if err != nil {
				continue
			}

			for _, c := range commits {
				ts := c.Date
				if ts == "" {
					ts = time.Now().UTC().Format(time.RFC3339Nano)
				}
				_ = w.store.Append(wsID, ch.Name, Entry{
					Ts:       ts,
					Type:     "git.commit",
					Category: "git",
					Summary:  c.Message,
					Meta: map[string]any{
						"hash":    c.Hash,
						"message": c.Message,
						"author":  c.Author,
					},
				})
			}

			w.lastHEADs[key] = head
		}
	}
}

func (w *GitWatcher) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	w.PollOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.PollOnce(ctx)
		}
	}
}

func StartGitWatcherLoop(ctx context.Context, store *Store, cfg *config.Config, worktreesDir string, interval time.Duration) *GitWatcher {
	w := NewGitWatcher(store, cfg, worktreesDir)
	go w.Start(ctx, interval)
	return w
}

func gitRevParseHead(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitLogRange(ctx context.Context, dir, from, to string) ([]gitCommitInfo, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "--reverse", "--format=%H%x1f%an%x1f%aI%x1f%s", from+".."+to)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var commits []gitCommitInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x1f")
		if len(parts) < 4 {
			continue
		}
		commits = append(commits, gitCommitInfo{
			Hash:    parts[0],
			Author:  parts[1],
			Date:    parts[2],
			Message: parts[3],
		})
	}
	return commits, nil
}
