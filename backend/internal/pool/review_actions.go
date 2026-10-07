package pool

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/watcher"
)

// Refusals of the review actions (approve, request a correction).
var (
	// ErrNotInReview reports a change that carries no review marker.
	ErrNotInReview = errors.New("le changement n'est pas en revue")
	// ErrWorkerActive reports a worker (active or paused) holding the change.
	ErrWorkerActive = errors.New("un worker est assigné à ce changement")
	// ErrReviewBusy reports a review action already running on the change
	// (approval, correction, reset or another task toggle).
	ErrReviewBusy = errors.New("une action de revue est déjà en cours sur ce changement")
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

// Element of ApproveResult.Warning.Remaining besides the cleanup elements: the
// review marker could not be lifted.
const cleanupMarker = "marker"

// CleanupWarning reports what an approval that merged could not clean up.
// Remaining is an ordered subset of "worktree", "branch" and "marker".
type CleanupWarning struct {
	Message   string
	Remaining []string
}

// ApproveResult is the outcome of a successful approval: the target branch and,
// when the merge happened but the cleanup is incomplete, a warning.
type ApproveResult struct {
	Target  string
	Warning *CleanupWarning
}

// ApproveReview merges a change in review into the repository's current
// branch: it integrates and revalidates the target branch when it advanced
// (plain validation, no agent heal), merges under the workspace merge lock,
// then removes the worktree and the branch. It does not need the pool to run
// and runs detached from ctx: a merge that started is never interrupted.
// On refusal or failure before the merge nothing is lost: branch, worktree and
// review marker are kept. Once the merge happened the approval succeeds: the
// marker is lifted and what could not be removed is reported in the warning.
// A branch already merged (a previous approval whose marker could not be
// lifted) is only cleaned up: no integration, validation or merge.
func (m *Manager) ApproveReview(ctx context.Context, workspaceID, workspacePath, change string) (ApproveResult, error) {
	ctx = context.WithoutCancel(ctx)
	lock := m.reviewLock(change)
	lock.Lock()
	defer lock.Unlock()

	wt := NewWorktreeController(workspacePath, workspaceID, m.worktreesRoot)
	if err := m.checkReviewable(wt, change); err != nil {
		return ApproveResult{}, err
	}

	merged, err := wt.IsMerged(change)
	if err != nil {
		return ApproveResult{}, fmt.Errorf("impossible de vérifier la fusion de la branche : %w", err)
	}
	if merged {
		// Never delete a branch merged into something else than its base.
		if err := wt.CheckBase(change); err != nil {
			return ApproveResult{}, err
		}
		return m.finishMerged(wt, workspaceID, change, wt.CurrentBranch(), cleanupMerged(wt, change)), nil
	}

	// The integration needs a worktree: recreate it from the branch if it vanished.
	dir, err := wt.Provision(change)
	if err != nil {
		return ApproveResult{}, fmt.Errorf("impossible de préparer le worktree : %w", err)
	}

	validate := func() error {
		err := m.runValidationIn(ctx, workspaceID, dir)
		var envErr *ValidationEnvError
		if err == nil || errors.As(err, &envErr) {
			return err
		}
		return &ValidationFailedError{Err: err}
	}
	target, mergedNow, err := m.integrateAndMerge(ctx, wt, change, validate)
	if !mergedNow {
		return ApproveResult{}, err
	}
	var failed *cleanupFailure
	var ce *CleanupError
	if errors.As(err, &ce) {
		failed = &cleanupFailure{Element: ce.Element, Err: ce.Err}
	}
	return m.finishMerged(wt, workspaceID, change, target, failed), nil
}

// finishMerged completes an approval whose merge is done: it lifts the review
// marker (the change then derives its status from the merged tasks.md),
// publishes change_updated and builds the warning for what is left behind.
func (m *Manager) finishMerged(wt *WorktreeController, workspaceID, change, target string, failed *cleanupFailure) ApproveResult {
	var remaining []string
	if failed != nil {
		log.Printf("[review] %s merged into %s but %s could not be cleaned up: %v\n", change, target, failed.Element, failed.Err)
		remaining = append(remaining, failed.Element)
	}
	if err := wt.ClearReview(change); err != nil {
		log.Printf("[review] cannot lift the review marker of %s: %v\n", change, err)
		remaining = append(remaining, cleanupMarker)
	}
	m.publishChangeUpdated(workspaceID, change)

	res := ApproveResult{Target: target}
	if len(remaining) > 0 {
		res.Warning = &CleanupWarning{
			Message: fmt.Sprintf("Fusion réussie dans %s mais le nettoyage est incomplet : %s à traiter manuellement",
				target, strings.Join(remaining, ", ")),
			Remaining: remaining,
		}
	}
	return res
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
	if err := wt.CommitFile(change, rel, correctionCommitMessage(change, wt.changeScope(change))); err != nil {
		_ = os.WriteFile(tasksPath, original, 0o644)
		return fmt.Errorf("commit de la correction : %w", err)
	}
	if err := wt.ClearReview(change); err != nil {
		return fmt.Errorf("levée du marqueur de revue : %w", err)
	}
	m.publishChangeUpdated(workspaceID, change)
	return nil
}

// TryLockReview takes the review lock of change without waiting and returns
// its release function, or ErrReviewBusy when the lock is held.
func (m *Manager) TryLockReview(change string) (func(), error) {
	lock := m.reviewLock(change)
	if !lock.TryLock() {
		return nil, ErrReviewBusy
	}
	return lock.Unlock, nil
}

// HasWorkerFor reports whether a worker, active or paused, holds change.
func (m *Manager) HasWorkerFor(change string) bool { return m.hasWorkerFor(change) }

// ToggleBranchTask toggles the index-th task of the tasks.md carried by
// feature/<change> and commits that single file into the branch, one commit
// per tick, under the review lock. The worktree is recreated from the branch
// when it vanished. The review marker is left untouched. It refuses with
// ErrReviewBusy while another review action runs and with ErrWorkerActive when
// a worker holds the change. On a write or commit failure the file is
// restored.
func (m *Manager) ToggleBranchTask(ctx context.Context, workspaceID, workspacePath, change string, index int) (taskText string, done bool, err error) {
	release, err := m.TryLockReview(change)
	if err != nil {
		return "", false, err
	}
	defer release()
	if m.hasWorkerFor(change) {
		return "", false, ErrWorkerActive
	}

	wt := NewWorktreeController(workspacePath, workspaceID, m.worktreesRoot)
	dir, err := wt.Provision(change)
	if err != nil {
		return "", false, fmt.Errorf("impossible de préparer le worktree : %w", err)
	}
	rel := filepath.Join("openspec", "changes", change, "tasks.md")
	tasksPath := filepath.Join(dir, rel)
	original, err := os.ReadFile(tasksPath)
	if err != nil {
		return "", false, fmt.Errorf("lecture de tasks.md : %w", err)
	}
	taskText, done, err = openspec.ToggleTask(dir, change, index)
	if err != nil {
		return "", false, err
	}
	if err := wt.CommitFile(change, rel, taskCommitMessage(change, wt.changeScope(change), index, taskText, done)); err != nil {
		_ = os.WriteFile(tasksPath, original, 0o644)
		return "", false, fmt.Errorf("commit de la coche : %w", err)
	}
	m.publishChangeUpdated(workspaceID, change)
	return taskText, done, nil
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
