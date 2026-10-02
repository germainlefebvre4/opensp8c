package pool

import (
	"context"
	"errors"
	"fmt"
	"log"
)

// Typed failures of integrateAndMerge. The full-autonomy worker turns them into
// pause reasons, the review approval into HTTP codes.

// IntegrationConflictError reports that integrating the target branch into
// the change branch failed (the integration was aborted).
type IntegrationConflictError struct {
	Target string
	Err    error
}

func (e *IntegrationConflictError) Error() string {
	return fmt.Sprintf("Intégration de %s impossible (annulée, branche et worktree conservés) : %s", e.Target, truncateReason(e.Err))
}
func (e *IntegrationConflictError) Unwrap() error { return e.Err }

// TargetMovingError reports that the target branch kept advancing during
// maxIntegrationRounds successive integrations.
type TargetMovingError struct{ Rounds int }

func (e *TargetMovingError) Error() string {
	return fmt.Sprintf("La branche cible a continué d'avancer pendant la finalisation (%d intégrations successives)", e.Rounds)
}

// ValidationFailedError reports a failing validation of the integrated result
// (no heal attempted).
type ValidationFailedError struct{ Err error }

func (e *ValidationFailedError) Error() string { return e.Err.Error() }
func (e *ValidationFailedError) Unwrap() error { return e.Err }

// CompareError reports that the target branch could not be compared with the
// change branch.
type CompareError struct{ Err error }

func (e *CompareError) Error() string {
	return fmt.Sprintf("Impossible de comparer la branche cible au changement : %s", truncateReason(e.Err))
}
func (e *CompareError) Unwrap() error { return e.Err }

// MergeFailedError reports a merge that git refused or aborted.
type MergeFailedError struct {
	Target string
	Err    error
}

func (e *MergeFailedError) Error() string {
	return fmt.Sprintf("Échec de la fusion dans %s (merge annulé, branche et worktree conservés) : %s", e.Target, truncateReason(e.Err))
}
func (e *MergeFailedError) Unwrap() error { return e.Err }

// CleanupError reports a failure after a successful merge: the worktree or the
// branch could not be removed.
type CleanupError struct {
	Target string
	What   string // "le worktree n'a pas pu être supprimé" | "la branche n'a pas pu être supprimée"
	Err    error
}

func (e *CleanupError) Error() string {
	return fmt.Sprintf("Fusion réussie dans %s mais %s : %s", e.Target, e.What, truncateReason(e.Err))
}
func (e *CleanupError) Unwrap() error { return e.Err }

// errValidationHandled is returned by a validate callback that already took
// care of the failure itself (the worker paused); integrateAndMerge passes it
// through.
var errValidationHandled = errors.New("validation failure already handled")

// integrateAndMerge is the finalization shared by the full-autonomy worker and
// the review approval: base check, bounded integration rounds (integrate the
// target branch, then validate the result), merge under the workspace merge
// lock, then removal of the worktree and the branch. validate runs on the
// integrated worktree and returns nil, a *ValidationEnvError, a
// *ValidationFailedError, or errValidationHandled. merged is true as soon as
// the merge happened, even if the cleanup then failed.
func (m *Manager) integrateAndMerge(ctx context.Context, wt *WorktreeController, change string, validate func() error) (target string, merged bool, err error) {
	for round := 1; ; round++ {
		// Fail fast when the repository left the base branch: no point in
		// integrating and revalidating for a merge that will be refused.
		if err := wt.CheckBase(change); err != nil {
			return "", false, err
		}
		ahead, err := wt.TargetAhead(change)
		if err != nil {
			return "", false, &CompareError{Err: err}
		}
		if ahead {
			if round > maxIntegrationRounds {
				return "", false, &TargetMovingError{Rounds: maxIntegrationRounds}
			}
			integrationTarget := wt.CurrentBranch()
			log.Printf("[merge] %s advanced: integrating it into %s before merging\n", integrationTarget, change)
			if err := wt.IntegrateTarget(change, integrationTarget); err != nil {
				return "", false, &IntegrationConflictError{Target: integrationTarget, Err: err}
			}
			if err := validate(); err != nil {
				return "", false, err
			}
		}

		m.mergeMu.Lock()
		// Point of no return: checked right after the lock, which may have
		// been waited on for a long time. A merge that has started is never
		// interrupted.
		if ctx.Err() != nil {
			m.mergeMu.Unlock()
			return "", false, ctx.Err()
		}
		// Revalidating under the lock would block the other merges: if the
		// target moved since the integration, start another round instead.
		ahead, err = wt.TargetAhead(change)
		if err != nil {
			m.mergeMu.Unlock()
			return "", false, &CompareError{Err: err}
		}
		if !ahead {
			break
		}
		m.mergeMu.Unlock()
	}
	target, err = wt.MergeInto(change)
	m.mergeMu.Unlock()
	if err != nil {
		if errors.Is(err, ErrMergeInProgress) || errors.Is(err, ErrBaseBranchMismatch) {
			return target, false, err
		}
		return target, false, &MergeFailedError{Target: target, Err: err}
	}
	if err := wt.Remove(change); err != nil {
		return target, true, &CleanupError{Target: target, What: "le worktree n'a pas pu être supprimé", Err: err}
	}
	if err := wt.DeleteBranch(change); err != nil {
		return target, true, &CleanupError{Target: target, What: "la branche n'a pas pu être supprimée", Err: err}
	}
	return target, true, nil
}
