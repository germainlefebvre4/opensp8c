package pool

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// runWorker coordinates the lifecycle of a worker on a specific change.
func (m *Manager) runWorker(ctx context.Context, w *Worker) {
	defer func() {
		m.mu.Lock()
		delete(m.activeWorkers, w.ID)
		m.mu.Unlock()
	}()

	wt := NewWorktreeController(m.workspacePath)

	// 1. Provision Environment
	worktreePath, err := wt.Provision(w.ActiveChange)
	if err != nil {
		log.Printf("[worker %d] failed to provision worktree: %v\n", w.ID, err)
		return
	}
	w.WorktreePath = worktreePath
	w.BranchName = "feature/" + w.ActiveChange

	// Note: We need a mechanism to read the change status to see if it's done.
	// For simulation of the execution loop:
	for {
		select {
		case <-ctx.Done():
			// Worker cancelled. We keep the worktree around or clean it depending on preference.
			// Let's just exit.
			return
		default:
		}

		// Update worker status
		w.Status = StatusWorking

		// 3.1 Invoke agent CLI (Simulated here because actual integration with Claude/Gemini CLI is complex to stub natively)
		err = m.invokeAgentApply(ctx, w)
		if err != nil {
			log.Printf("[worker %d] agent apply error: %v\n", w.ID, err)
			// Might be blocked. Pause.
			w.Status = StatusPaused
			return
		}

		// 3.2 Run local validation (Compilation + tests)
		w.Status = StatusTesting
		validationErr := m.runValidation(ctx, w)

		// 3.3 Healing loop
		attempts := 0
		for validationErr != nil && attempts < m.config.MaxAttempts {
			attempts++
			w.Status = StatusHealing
			log.Printf("[worker %d] Validation failed. Attempt %d/%d to heal.\n", w.ID, attempts, m.config.MaxAttempts)
			
			// Inject error back to agent
			err = m.invokeAgentHeal(ctx, w, validationErr)
			if err != nil {
				break
			}

			// Retest
			w.Status = StatusTesting
			validationErr = m.runValidation(ctx, w)
		}

		if validationErr != nil {
			log.Printf("[worker %d] Failed to heal after %d attempts. Pausing.\n", w.ID, m.config.MaxAttempts)
			w.Status = StatusPaused
			return
		}

		// 3.4 State Transitions based on Delegation Mode
		// Assuming tasks are fully complete here. In a real scenario, we'd check if tasks.md has remaining tasks.
		// For the sake of the architecture, we transition now.
		if m.config.DelegationMode == ModeFullAutonomy {
			// Merge and cleanup
			log.Printf("[worker %d] Full Autonomy: merging change %s\n", w.ID, w.ActiveChange)
			err = wt.MergeAndCleanup(w.ActiveChange)
			if err != nil {
				log.Printf("[worker %d] failed to merge: %v\n", w.ID, err)
			}
		} else {
			// HITL Review
			log.Printf("[worker %d] HITL Review: change %s ready for review\n", w.ID, w.ActiveChange)
			// State remains "to-review" implicitly as we leave the branch unmerged and worktree intact.
			// The UI will pick this up from the Kanban status.
		}

		// Job done
		return
	}
}

func (m *Manager) invokeAgentApply(ctx context.Context, w *Worker) error {
	// Stub for invoking openspec apply or the agent subprocess
	// cmd := exec.CommandContext(ctx, "openspec", "apply", "--change", w.ActiveChange)
	// cmd.Dir = w.WorktreePath
	// ...
	time.Sleep(2 * time.Second) // simulate work
	return nil
}

func (m *Manager) runValidation(ctx context.Context, w *Worker) error {
	// Stub for running project tests.
	// We could run `make test` or `go test ./...` based on project detection.
	cmd := exec.CommandContext(ctx, "go", "test", "./...")
	cmd.Dir = w.WorktreePath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tests failed: %v\nOutput:\n%s", err, string(out))
	}
	return nil
}

func (m *Manager) invokeAgentHeal(ctx context.Context, w *Worker, validationErr error) error {
	// Stub for prompting the agent with the validation error.
	time.Sleep(2 * time.Second) // simulate healing work
	// In a real implementation we would write the error out to the active subprocess's stdin.
	if strings.Contains(validationErr.Error(), "unfixable") {
		return fmt.Errorf("cannot fix")
	}
	return nil
}
