package pool

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/glefebvre/opensp8c/internal/watcher"
)

// Refusals of the review actions (approve, request a correction).
var (
	// ErrNotInReview reports a change that carries no review marker.
	ErrNotInReview = errors.New("le changement n'est pas en revue")
	// ErrWorkerActive reports a worker (active or paused) holding the change.
	ErrWorkerActive = errors.New("un worker est assigné à ce changement")
	// ErrEmptyFeedback reports a correction request without text.
	ErrEmptyFeedback = errors.New("le retour de correction est vide")
)

// correctionsHeading is the tasks.md section collecting the corrections.
const correctionsHeading = "## Corrections"

// reviewLock serializes the review actions of one change.
func (m *Manager) reviewLock(change string) *sync.Mutex {
	l, _ := m.reviewOps.LoadOrStore(change, &sync.Mutex{})
	return l.(*sync.Mutex)
}

// hasWorkerFor reports whether a worker, active or paused, holds change.
func (m *Manager) hasWorkerFor(change string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, w := range m.activeWorkers {
		if w.ActiveChange == change {
			return true
		}
	}
	for _, w := range m.pausedWorkers {
		if w.ActiveChange == change {
			return true
		}
	}
	return false
}

// checkReviewable verifies, under the change's review lock, that the change is
// in review and held by no worker.
func (m *Manager) checkReviewable(wt *WorktreeController, change string) error {
	has, err := wt.HasReview(change)
	if err != nil {
		return err
	}
	if !has {
		return ErrNotInReview
	}
	if m.hasWorkerFor(change) {
		return ErrWorkerActive
	}
	return nil
}

func (m *Manager) publishChangeUpdated(workspaceID, change string) {
	if m.broadcaster != nil && workspaceID != "" {
		m.broadcaster.Broadcast(workspaceID, watcher.Event{Type: "change_updated", Name: change})
	}
}

// ApproveReview merges a change in review into the repository's current
// branch: it integrates and revalidates the target branch when it advanced
// (plain validation, no agent heal), merges under the workspace merge lock,
// then removes the worktree and the branch. It does not need the pool to run
// and runs detached from ctx: a merge that started is never interrupted.
// It returns the target branch. On refusal or failure nothing is lost: branch,
// worktree and review marker are kept.
func (m *Manager) ApproveReview(ctx context.Context, workspaceID, workspacePath, change string) (string, error) {
	ctx = context.WithoutCancel(ctx)
	lock := m.reviewLock(change)
	lock.Lock()
	defer lock.Unlock()

	wt := NewWorktreeController(workspacePath, workspaceID, m.worktreesRoot)
	if err := m.checkReviewable(wt, change); err != nil {
		return "", err
	}
	// The integration needs a worktree: recreate it from the branch if it vanished.
	dir, err := wt.Provision(change)
	if err != nil {
		return "", fmt.Errorf("impossible de préparer le worktree : %w", err)
	}

	validate := func() error {
		err := m.runValidationIn(ctx, workspaceID, dir)
		var envErr *ValidationEnvError
		if err == nil || errors.As(err, &envErr) {
			return err
		}
		return &ValidationFailedError{Err: err}
	}
	target, merged, err := m.integrateAndMerge(ctx, wt, change, validate)
	if merged {
		m.publishChangeUpdated(workspaceID, change)
	}
	return target, err
}

// RequestCorrection records a user correction for a change in review: one
// unchecked task appended to the "## Corrections" section of the worktree's
// tasks.md and committed into feature/<change>, then the review marker is
// lifted so the change becomes eligible again. The marker is kept when
// writing or committing fails.
func (m *Manager) RequestCorrection(ctx context.Context, workspaceID, workspacePath, change, feedback string) error {
	if strings.TrimSpace(feedback) == "" {
		return ErrEmptyFeedback
	}
	lock := m.reviewLock(change)
	lock.Lock()
	defer lock.Unlock()

	wt := NewWorktreeController(workspacePath, workspaceID, m.worktreesRoot)
	if err := m.checkReviewable(wt, change); err != nil {
		return err
	}
	dir, err := wt.Provision(change)
	if err != nil {
		return fmt.Errorf("impossible de préparer le worktree : %w", err)
	}

	rel := filepath.Join("openspec", "changes", change, "tasks.md")
	tasksPath := filepath.Join(dir, rel)
	original, err := os.ReadFile(tasksPath)
	if err != nil {
		return fmt.Errorf("lecture de tasks.md : %w", err)
	}
	updated := AppendCorrection(string(original), feedback)
	if err := os.WriteFile(tasksPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("écriture de tasks.md : %w", err)
	}
	if err := wt.CommitFile(change, rel, "chore("+change+"): add review correction"); err != nil {
		_ = os.WriteFile(tasksPath, original, 0o644)
		return fmt.Errorf("commit de la correction : %w", err)
	}
	if err := wt.ClearReview(change); err != nil {
		return fmt.Errorf("levée du marqueur de revue : %w", err)
	}
	m.publishChangeUpdated(workspaceID, change)
	return nil
}

// AppendCorrection adds one unchecked task carrying feedback at the end of the
// "## Corrections" section of a tasks.md content, creating the section when
// missing. Only the first line sits on the task line; the following ones are
// block-quoted so that none of them can be counted as a task, whatever the
// text.
func AppendCorrection(content, feedback string) string {
	feedback = strings.ReplaceAll(strings.TrimSpace(feedback), "\r\n", "\n")
	lines := strings.Split(feedback, "\n")
	task := []string{"- [ ] Correction : " + strings.TrimSpace(lines[0])}
	for _, l := range lines[1:] {
		if l = strings.TrimRight(l, " \t\r"); l == "" {
			task = append(task, "  >")
		} else {
			task = append(task, "  > "+l)
		}
	}

	all := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if strings.TrimSpace(content) == "" {
		all = nil
	}
	start := -1
	for i, l := range all {
		if strings.TrimSpace(l) == correctionsHeading {
			start = i
			break
		}
	}
	if start < 0 {
		out := all
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, correctionsHeading, "")
		out = append(out, task...)
		return strings.Join(out, "\n") + "\n"
	}

	// End of section: next heading, else EOF; insert after its last non-blank line.
	end := len(all)
	for i := start + 1; i < len(all); i++ {
		if strings.HasPrefix(all[i], "# ") || strings.HasPrefix(all[i], "## ") {
			end = i
			break
		}
	}
	insert := end
	for insert > start+1 && strings.TrimSpace(all[insert-1]) == "" {
		insert--
	}
	if insert == start+1 { // empty section: keep the blank line after the heading
		all = append(all[:insert], append([]string{""}, all[insert:]...)...)
		insert++
	}
	out := append([]string{}, all[:insert]...)
	out = append(out, task...)
	if insert < len(all) && strings.TrimSpace(all[insert]) != "" {
		out = append(out, "")
	}
	out = append(out, all[insert:]...)
	return strings.Join(out, "\n") + "\n"
}
