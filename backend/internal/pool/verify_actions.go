package pool

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/glefebvre/opensp8c/internal/openspec"
)

// Refusals of the actions on a failed verification.
var (
	// ErrVerificationRunning reports a verification in flight for the change.
	ErrVerificationRunning = errors.New("une vérification est en cours sur ce changement")
	// ErrVerificationNotFailed reports a change whose verification marker is
	// not `failed`.
	ErrVerificationNotFailed = errors.New("la vérification du changement n'a pas échoué")
)

// verificationGuard runs an action on a change whose verification failed. It
// refuses while a verification runs, then when the marker is not `failed`,
// then when another action holds the change's lock; the marker is read again
// under the lock. action runs with the lock held.
func (m *Manager) verificationGuard(workspaceID, workspacePath, change string, action func(wt *WorktreeController) error) error {
	wt := NewWorktreeController(workspacePath, workspaceID, m.worktreesRoot)
	if err := m.checkFailed(wt, change); err != nil {
		return err
	}
	release, err := m.TryLockReview(change)
	if err != nil {
		return err
	}
	defer release()
	if err := m.checkFailed(wt, change); err != nil {
		return err
	}
	return action(wt)
}

func (m *Manager) checkFailed(wt *WorktreeController, change string) error {
	if m.VerificationRunning(change) {
		return ErrVerificationRunning
	}
	if st, ok := wt.VerifyState(change); !ok || st != openspec.VerifyFailed {
		return ErrVerificationNotFailed
	}
	if m.hasWorkerFor(change) {
		return ErrWorkerActive
	}
	return nil
}

// RerunVerification queues a failed verification again (marker `pending`); it
// runs at the next tick of a started pool.
func (m *Manager) RerunVerification(workspaceID, workspacePath, change string) error {
	return m.verificationGuard(workspaceID, workspacePath, change, func(wt *WorktreeController) error {
		if err := wt.SetVerify(change, openspec.VerifyPending); err != nil {
			return fmt.Errorf("enregistrement de l'état de vérification : %w", err)
		}
		m.publishChangeUpdated(workspaceID, change)
		return nil
	})
}

// FinalizeWithoutVerification accepts a failed change as verified (marker
// `passed`) so the next tick finalizes it. Every task of the branch's tasks.md
// must be checked, human review tasks included, else *ErrTasksIncomplete.
func (m *Manager) FinalizeWithoutVerification(workspaceID, workspacePath, change string) error {
	return m.verificationGuard(workspaceID, workspacePath, change, func(wt *WorktreeController) error {
		content, ok := wt.BranchTasks(change)
		done, total := 0, 0
		if ok {
			for _, t := range openspec.ParseTaskListContent(content) {
				total++
				if t.Done {
					done++
				}
			}
		}
		if total == 0 || done < total {
			return &ErrTasksIncomplete{Remaining: total - done, Total: total}
		}
		if err := wt.SetVerify(change, openspec.VerifyPassed); err != nil {
			return fmt.Errorf("enregistrement de l'état de vérification : %w", err)
		}
		m.publishChangeUpdated(workspaceID, change)
		return nil
	})
}

// RequestVerificationCorrection records feedback as a correction task of a
// failed change, committed into its branch, then lifts the verification marker
// so the change is eligible to a worker again.
func (m *Manager) RequestVerificationCorrection(ctx context.Context, workspaceID, workspacePath, change, feedback string) error {
	if strings.TrimSpace(feedback) == "" {
		return ErrEmptyFeedback
	}
	return m.verificationGuard(workspaceID, workspacePath, change, func(wt *WorktreeController) error {
		return m.applyCorrection(workspaceID, wt, change, feedback, CorrectionOptions{}, wt.ClearVerify, "levée du marqueur de vérification")
	})
}
