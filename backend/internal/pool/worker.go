package pool

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/language"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
)

// activityBroadcastInterval throttles how often an in-flight turn's activity
// updates trigger a pool_updated broadcast, so a chatty agent doesn't flood
// clients with a broadcast per stdout line (see design.md decision 4).
const activityBroadcastInterval = time.Second

// maxActivityLen bounds the fallback raw-line activity text.
const maxActivityLen = 200

// startSubprocessFn is a seam over session.StartSubprocess so tests can stub
// subprocess creation without spawning a real agent CLI.
var startSubprocessFn = session.StartSubprocess

// HoldAgentStartsForTest makes every worker's agent start block until its
// context is cancelled, so cross-package tests get a worker that stays
// genuinely active (neither failing into a pause nor finishing). The returned
// function restores the real starter.
func HoldAgentStartsForTest() (restore func()) {
	orig := startSubprocessFn
	startSubprocessFn = func(ctx context.Context, _ string, _ agents.AgentConfig, _, _ string, _ bool, _ *conversation.SessionLog, _ map[string]string, _ bool, _ string) (*session.Subprocess, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return func() { startSubprocessFn = orig }
}

// SeedWorkerForTest registers w as a worker of m, paused or active according
// to w.Status, without running anything, so cross-package tests can observe a
// worker holding a change (with its worktree and pause reason) deterministically.
func SeedWorkerForTest(m *Manager, w Worker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if w.Status == StatusPaused {
		m.pausedWorkers[w.ID] = &w
	} else {
		m.activeWorkers[w.ID] = &w
	}
}

// beforeCommitHook is a test seam called once validation and the completion
// check have passed, right before the cancellation check that precedes the
// commit of the agent's work.
var beforeCommitHook func()

// pauseWorker marks w as paused with a human-readable reason and snapshots it
// into m.pausedWorkers, so it stays visible via Status()/AllPools() even
// after runWorker's own deferred cleanup removes it from m.activeWorkers.
// A worker whose ctx is already cancelled (pool stopped, change unlaunched) is
// not paused: it reports false and leaves no ghost pause behind. The check
// runs under m.mu, like the cancellation in Stop, so it cannot interleave.
func (m *Manager) pauseWorker(ctx context.Context, w *Worker, reason string) bool {
	m.mu.Lock()
	if ctx.Err() != nil {
		m.mu.Unlock()
		return false
	}
	w.Status = StatusPaused
	w.BlockedReason = reason
	snapshot := *w
	m.pausedWorkers[w.ID] = &snapshot
	m.mu.Unlock()

	m.notify()
	return true
}

// setStatus, setActivity and setRun mutate w under m.mu so that Status()
// snapshots (which copy the whole Worker under the same lock) never race.
func (m *Manager) setStatus(w *Worker, s WorkerStatus) {
	m.mu.Lock()
	w.Status = s
	m.mu.Unlock()
}

func (m *Manager) setActivity(w *Worker, a string) {
	m.mu.Lock()
	w.Activity = a
	m.mu.Unlock()
}

func (m *Manager) setRun(w *Worker, ts string, rl *poolRunLog) {
	m.mu.Lock()
	w.RunTS = ts
	w.runLog = rl
	m.mu.Unlock()
}

func (m *Manager) setWorktree(w *Worker, path, branch string) {
	m.mu.Lock()
	w.WorktreePath = path
	w.BranchName = branch
	m.mu.Unlock()
}

// runWorker coordinates the lifecycle of a worker on a specific change.
func (m *Manager) runWorker(ctx context.Context, w *Worker) {
	outcome, outcomeReason := "", ""
	merged, mergeTarget := false, ""

	// normalizeOutcome records a cancelled or outcome-less run as stopped,
	// except when the change was merged: its real outcome is then kept.
	normalizeOutcome := func() {
		if !merged && (outcome == "" || ctx.Err() != nil) {
			outcome = OutcomeStopped
		}
	}

	// First deferred, so last to run: removes the worker from activeWorkers,
	// notifies, then publishes the result and signals done.
	defer func() {
		normalizeOutcome()
		m.mu.Lock()
		// Only remove our own entry: after Stop/Start a newer worker may
		// have reused this ID.
		if m.activeWorkers[w.ID] == w {
			delete(m.activeWorkers, w.ID)
			delete(m.lastWorkerStatus, w.ID)
		}
		m.mu.Unlock()
		m.notify()
		if w.done != nil {
			w.result = WorkerResult{Outcome: outcome, Merged: merged, Target: mergeTarget}
			close(w.done)
		}
	}()

	pause := func(reason string) {
		if !m.pauseWorker(ctx, w, reason) && !merged {
			// Cancelled worker: recorded as stopped, never as a ghost pause.
			outcome, outcomeReason = OutcomeStopped, ""
			return
		}
		outcome, outcomeReason = OutcomePaused, reason
	}

	// Snapshot the pool settings: Stop/Start may rewrite them while this
	// worker is still winding down.
	m.mu.Lock()
	cfg, repoPath, worktreesRoot := m.config, m.workspacePath, m.worktreesRoot
	m.mu.Unlock()
	wt := NewWorktreeController(repoPath, w.WorkspaceID, worktreesRoot)

	notCommittedReason := fmt.Sprintf("Le changement « %s » doit être committé dans le dépôt avant d'être lancé (openspec/changes/%s/tasks.md est absent du worktree).", w.ActiveChange, w.ActiveChange)

	// 1. Provision Environment. A first launch needs the change committed in
	// HEAD: check before creating anything so a refusal leaves no orphan
	// branch or worktree behind.
	if exists, err := wt.branchExists("feature/" + w.ActiveChange); err != nil {
		pause(fmt.Sprintf("Échec du provisionnement du worktree : %v", err))
		return
	} else if !exists {
		committed, err := wt.ChangeCommitted(w.ActiveChange)
		if err != nil {
			pause(fmt.Sprintf("Échec du provisionnement du worktree : %v", err))
			return
		}
		if !committed {
			pause(notCommittedReason)
			return
		}
	}
	worktreePath, err := wt.Provision(w.ActiveChange)
	if err != nil {
		log.Printf("[worker %d] failed to provision worktree: %v\n", w.ID, err)
		pause(fmt.Sprintf("Échec du provisionnement du worktree : %v", err))
		return
	}
	m.setWorktree(w, worktreePath, "feature/"+w.ActiveChange)
	defer m.watchWorktreeTasks(w.WorkspaceID, worktreePath, w.ActiveChange)()

	select {
	case <-ctx.Done():
		return
	default:
	}

	m.setStatus(w, StatusWorking)
	m.notify()

	// 2. Start a single agent subprocess for the whole attempt loop (the
	// initial apply turn and every heal turn share it, so the model sees its
	// own prior turns). It is terminated when this function returns, on any
	// exit path (success, paused, or cancellation).
	procCtx, procCancel := context.WithCancel(ctx)
	defer procCancel()
	w.procCancel = procCancel

	var agentCfg agents.AgentConfig
	if m.sessionMgr != nil {
		agentCfg = m.sessionMgr.ResolveRoleConfig(w.WorkspaceID, w.role())
	}

	var customEnv map[string]string
	langDirective := language.Directive(language.Worker, language.Resolve(language.Levels{}, ""))
	if m.prefs != nil {
		if p, err := m.prefs.Load(); err == nil && p != nil {
			customEnv = p.EnvForWorkspace(w.WorkspaceID, agentCfg.ID)
			langDirective = p.LanguageDirective(language.Worker)
		}
	}

	// Journal the whole execution as a "pool" conversation run. The end
	// marker is written by a single defer, registered before any subprocess
	// cleanup so it runs after the subprocess has exited, and before the
	// worker is removed from activeWorkers.
	runLog, runTS := m.openRunLog(w)
	m.setRun(w, runTS, runLog)
	if runLog != nil {
		m.logRunMarker(w, map[string]any{
			"type":            "pool_run_start",
			"worker_id":       w.ID,
			"change":          w.ActiveChange,
			"delegation_mode": string(w.DelegationMode),
		})
		defer func() {
			normalizeOutcome()
			m.logRunMarker(w, map[string]any{
				"type":    "pool_run_end",
				"outcome": outcome,
				"reason":  outcomeReason,
			})
			m.finishRunEvents(w)
			_ = runLog.sess.Close()
		}()
	}

	if w.finalizeOnly {
		m.logRunMarker(w, map[string]any{"type": "pool_resume_finalize", "change": w.ActiveChange})
	}

	// The worktree starts from the last commit: an uncommitted change is not
	// in it, and no agent could apply it. A branch created before the change
	// was committed is recreated when it carries no work, else left intact.
	tasksPath := filepath.Join(w.WorktreePath, "openspec", "changes", w.ActiveChange, "tasks.md")
	if !fileExists(tasksPath) {
		committed, err := wt.ChangeCommitted(w.ActiveChange)
		if err != nil {
			pause(fmt.Sprintf("Échec du provisionnement du worktree : %v", err))
			return
		}
		if !committed {
			pause(notCommittedReason)
			return
		}
		hasWork, err := wt.HasWork(w.ActiveChange)
		if err != nil {
			pause(fmt.Sprintf("Échec du provisionnement du worktree : %v", err))
			return
		}
		if hasWork {
			pause(fmt.Sprintf("La branche « feature/%s » ne contient pas le changement (créée avant son commit) : y intégrer la branche courante ou la supprimer, puis reprendre le worker.", w.ActiveChange))
			return
		}
		newPath, err := wt.RecreateFromHead(w.ActiveChange)
		if err != nil {
			log.Printf("[worker %d] failed to recreate stale worktree: %v\n", w.ID, err)
			pause(fmt.Sprintf("Échec du provisionnement du worktree : %v", err))
			return
		}
		m.setWorktree(w, newPath, "feature/"+w.ActiveChange)
		if !fileExists(filepath.Join(w.WorktreePath, "openspec", "changes", w.ActiveChange, "tasks.md")) {
			pause(notCommittedReason)
			return
		}
	}

	var stderrLog *conversation.SessionLog
	if runLog != nil {
		stderrLog = runLog.sess
	}
	// A "resume and finalize" worker starts no agent and sends no turn: proc
	// stays nil and the rest of the flow (validation, completion check, commit,
	// finalization) runs unchanged.
	var proc *session.Subprocess
	if !w.finalizeOnly {
		var err error
		extraPrompt := ""
		if cfg.DelegationMode == ModeHITLReview {
			extraPrompt = humanReviewDirective
		}
		proc, err = startSubprocessFn(procCtx, w.WorktreePath, agentCfg, extraPrompt, "", false, stderrLog, customEnv, false, langDirective)
		if err != nil {
			log.Printf("[worker %d] failed to start agent subprocess: %v\n", w.ID, err)
			pause(fmt.Sprintf("Échec du démarrage du subprocess de l'agent : %v", err))
			return
		}
		// Ordered teardown: close stdin, give the agent a bounded time to exit,
		// then kill its whole process group and reap it.
		defer func() {
			_ = proc.CloseStdin()
			exited := make(chan struct{})
			go func() {
				_ = proc.Wait()
				close(exited)
			}()
			select {
			case <-exited:
			case <-time.After(teardownGrace):
				procCancel()
				<-exited
			}
			procCancel()
		}()

		// 3. Invoke the agent CLI to implement the remaining tasks.
		if err := m.invokeAgentApply(w, proc); err != nil {
			log.Printf("[worker %d] agent apply error: %v\n", w.ID, err)
			pause(agentTurnPauseReason(err, "Échec de l'invocation de l'agent pour appliquer les tâches restantes"))
			return
		}

		// 3b. Triage (hitl-review only): the remaining unflagged tasks are
		// either finished or flagged for the user, before validation so that
		// any work done here is validated too.
		if cfg.DelegationMode == ModeHITLReview {
			if !m.triageRemainingTasks(w, proc, tasksPath, pause) {
				return
			}
		}
	}

	// 4-5. Local validation (compilation + tests) and self-healing loop:
	// re-inject validation errors into the same subprocess's context until it
	// passes or attempts are exhausted. The attempts budget is shared by every
	// call (initial validation, then the revalidation after an integration).
	attempts := 0
	validateAndHeal := func() bool {
		m.setStatus(w, StatusTesting)
		m.notify()
		validationErr := m.runValidation(ctx, w)
		var envErr *ValidationEnvError
		if errors.As(validationErr, &envErr) {
			// No heal turn can fix the environment: pause at once, no attempt used.
			pause(envErr.Reason)
			return false
		}

		for validationErr != nil && proc != nil && attempts < cfg.MaxAttempts {
			attempts++
			m.setStatus(w, StatusHealing)
			m.notify()
			log.Printf("[worker %d] Validation failed. Attempt %d/%d to heal.\n", w.ID, attempts, cfg.MaxAttempts)

			if err := m.invokeAgentHeal(w, proc, validationErr); err != nil {
				log.Printf("[worker %d] agent heal error: %v\n", w.ID, err)
				if isAgentStop(err) {
					pause(agentTurnPauseReason(err, ""))
					return false
				}
				break
			}

			m.setStatus(w, StatusTesting)
			m.notify()
			validationErr = m.runValidation(ctx, w)
			if errors.As(validationErr, &envErr) {
				pause(envErr.Reason)
				return false
			}
		}

		if validationErr != nil && proc == nil {
			log.Printf("[worker %d] validation failed while finalizing without agent. Pausing.\n", w.ID)
			pause(fmt.Sprintf("Reprise en finalisant : la validation a échoué et aucun agent n'est lancé pour la corriger : %v", validationErr))
			return false
		}
		if validationErr != nil {
			log.Printf("[worker %d] Failed to heal after %d attempts. Pausing.\n", w.ID, cfg.MaxAttempts)
			pause(fmt.Sprintf("Tentatives de réparation épuisées après %d essai(s) : %v", cfg.MaxAttempts, validationErr))
			return false
		}
		return true
	}
	if !validateAndHeal() {
		return
	}

	// 6. Completion check: don't finalize (merge or transition) unless the
	// worktree's tasks.md lists at least one task and none is left unchecked,
	// even though validation passed - the agent may have declared victory
	// without actually doing the work. A missing or empty list is not "done".
	stats := openspec.ParseTaskStats(tasksPath)
	done, total := stats.Done, stats.Total
	if total == 0 {
		log.Printf("[worker %d] validation passed but tasks.md is absent or empty; pausing without finalizing\n", w.ID)
		pause("Validation réussie mais la liste des tâches (tasks.md) est absente ou vide")
		return
	}
	if tasksBlockCompletion(cfg.DelegationMode, stats) {
		log.Printf("[worker %d] validation passed but tasks.md incomplete (%d/%d done); pausing without finalizing\n", w.ID, done, total)
		pause(fmt.Sprintf("Validation réussie mais tâches restantes incomplètes (%d/%d) dans tasks.md", done, total))
		return
	}

	// 7. Commit the agent's work into feature/<change> before anything that
	// could discard the worktree. A cancelled worker commits nothing.
	if beforeCommitHook != nil {
		beforeCommitHook()
	}
	if ctx.Err() != nil {
		outcome = OutcomeStopped
		return
	}
	files, err := wt.CommitAll(w.ActiveChange)
	if err != nil {
		pause(fmt.Sprintf("Impossible de committer le travail de l'agent : %s", truncateReason(err)))
		return
	}
	if len(files) > 0 {
		log.Printf("[worker %d] committed %d file(s) for %s\n", w.ID, len(files), w.ActiveChange)
		m.logRunMarker(w, map[string]any{"type": "pool_commit", "change": w.ActiveChange, "files": files})
	}
	hasWork, err := wt.HasWork(w.ActiveChange)
	if err != nil {
		pause(fmt.Sprintf("Impossible de vérifier le travail produit : %s", truncateReason(err)))
		return
	}
	if !hasWork {
		pause("Aucun travail produit : la branche ne contient ni modification ni commit par rapport à la branche courante")
		return
	}

	// 7b. Verification stage: with at least one verification step enabled, the
	// change is not finalized here. The persistent marker queues it for the
	// verification executor and the worker leaves with its slot freed; the
	// worktree and branch stay. A worker that only finalizes (resume, or after
	// a successful verification) never comes back through here.
	if !w.finalizeOnly && m.verificationRequired(w.WorkspaceID, repoPath, w.ActiveChange) {
		log.Printf("[worker %d] change %s awaits verification\n", w.ID, w.ActiveChange)
		if err := wt.SetVerify(w.ActiveChange, openspec.VerifyPending); err != nil {
			log.Printf("[worker %d] failed to record verification state: %v\n", w.ID, err)
			pause(fmt.Sprintf("L'état de vérification n'a pas pu être enregistré : %s", truncateReason(err)))
			return
		}
		// The marker touches no OpenSpec file: publish the update explicitly.
		m.publishChangeUpdated(w.WorkspaceID, w.ActiveChange)
		outcome = OutcomeAwaitingVerification
		return
	}

	// 8. State transitions based on delegation mode.
	if cfg.DelegationMode == ModeFullAutonomy {
		// Integrate what landed on the target meanwhile and revalidate that
		// result (with self-healing) before merging; see integrateAndMerge.
		validate := func() error {
			if !validateAndHeal() {
				return errValidationHandled
			}
			if _, err := wt.CommitAll(w.ActiveChange); err != nil {
				pause(fmt.Sprintf("Impossible de committer les corrections après intégration : %s", truncateReason(err)))
				return errValidationHandled
			}
			return nil
		}
		target, ok, err := m.integrateAndMerge(ctx, wt, w.ActiveChange, validate)
		if ok {
			merged, mergeTarget = true, target
			log.Printf("[worker %d] Full Autonomy: merged %s into %s\n", w.ID, w.ActiveChange, target)
		}
		if err != nil {
			log.Printf("[worker %d] finalization failed: %v\n", w.ID, err)
			var envErr *ValidationEnvError
			var movingErr *TargetMovingError
			switch {
			case errors.Is(err, errValidationHandled):
				// The validate callback already paused the worker.
			case ctx.Err() != nil && !ok:
				outcome = OutcomeStopped
			case errors.Is(err, ErrMergeInProgress):
				pause("Un merge est déjà en cours dans le dépôt : terminez-le ou annulez-le, puis reprenez le worker")
			case errors.Is(err, ErrBaseBranchMismatch):
				pause("Fusion refusée : " + err.Error())
			case errors.As(err, &movingErr):
				pause(movingErr.Error() + " : reprenez le worker pour réessayer")
			case errors.As(err, &envErr):
				pause(envErr.Reason)
			default:
				pause(err.Error())
			}
			return
		}
		outcome = OutcomeCompleted
		return
	}

	// HITL review: branch and worktree stay in place. The persistent marker
	// makes the change to-review, which keeps the dispatcher from handing it
	// out again, across pool and backend restarts.
	log.Printf("[worker %d] HITL Review: change %s ready for review\n", w.ID, w.ActiveChange)
	if err := wt.MarkReview(w.ActiveChange); err != nil {
		log.Printf("[worker %d] failed to record review state: %v\n", w.ID, err)
		pause(fmt.Sprintf("L'état de revue n'a pas pu être enregistré : %s", truncateReason(err)))
		return
	}
	// Setting the marker touches no OpenSpec file, so the watcher stays silent:
	// publish the change update explicitly.
	if m.broadcaster != nil && w.WorkspaceID != "" {
		m.broadcaster.Broadcast(w.WorkspaceID, watcher.Event{Type: "change_updated", Name: w.ActiveChange})
	}
	outcome = OutcomeAwaitingReview
}

// maxIntegrationRounds bounds the integrate-and-revalidate rounds before a
// full-autonomy merge when the target branch keeps advancing.
const maxIntegrationRounds = 3

// maxReasonLen bounds the git/command output embedded in a pause reason.
const maxReasonLen = 400

func truncateReason(err error) string {
	msg := strings.TrimSpace(err.Error())
	if len(msg) > maxReasonLen {
		msg = msg[:maxReasonLen] + "…"
	}
	return msg
}

// teardownGrace is how long an agent gets to exit after its stdin is closed
// before its process group is killed.
var teardownGrace = 5 * time.Second

// agentIdleTimeout bounds a turn without any agent output.
var agentIdleTimeout = 30 * time.Minute

// agentTurnError is an agent turn that ended with an error result.
type agentTurnError struct{ reason string }

func (e *agentTurnError) Error() string { return "agent turn failed: " + e.reason }

// agentIdleError is an agent turn cut after agentIdleTimeout without output.
type agentIdleError struct{ after time.Duration }

func (e *agentIdleError) Error() string {
	return fmt.Sprintf("agent inactif depuis %s", e.after)
}

// humanReviewDirective is appended to the system prompt of hitl-review workers.
const humanReviewDirective = `Some tasks in tasks.md can only be validated by the user (manual walkthrough in the application, visual check, command to run outside your environment). Such a task carries the HTML comment ` + openspec.HumanReviewMarker + ` at the end of its line. NEVER check (- [x]) a task carrying that marker. When a task requires the user's intervention, do not check it as if you had done it: leave it unchecked and add ` + openspec.HumanReviewMarker + ` at the end of its line instead.`

// tasksBlockCompletion reports whether unchecked tasks must stop the worker
// from finalizing: in full-autonomy any unchecked task does, in hitl-review
// only those the user is not expected to validate.
func tasksBlockCompletion(mode DelegationMode, st openspec.TaskStats) bool {
	if mode == ModeHITLReview {
		return st.PendingOther > 0
	}
	return st.Done < st.Total
}

// flaggedTasks returns the texts of the human review tasks of a tasks.md.
func flaggedTasks(tasksPath string) map[string]bool {
	out := map[string]bool{}
	data, err := os.ReadFile(tasksPath)
	if err != nil {
		return out
	}
	for _, t := range openspec.ParseTaskListContent(string(data)) {
		if t.HumanReview {
			out[t.Text] = true
		}
	}
	return out
}

// triageRemainingTasks sends the single triage turn when unchecked tasks
// without the human review marker remain, then records the tasks the agent
// flagged. It returns false when the worker was paused.
func (m *Manager) triageRemainingTasks(w *Worker, proc *session.Subprocess, tasksPath string, pause func(string)) bool {
	var remaining []string
	for _, t := range openspec.ParseTaskListContent(readFileContent(tasksPath)) {
		if !t.Done && !t.HumanReview {
			remaining = append(remaining, t.Text)
		}
	}
	if len(remaining) == 0 {
		return true
	}
	before := flaggedTasks(tasksPath)
	if err := m.runTurn(w, proc, triagePrompt(remaining)); err != nil {
		log.Printf("[worker %d] agent triage error: %v\n", w.ID, err)
		pause(agentTurnPauseReason(err, "Échec de l'invocation de l'agent pour trier les tâches restantes"))
		return false
	}
	if m.activityStore != nil {
		for _, t := range openspec.ParseTaskListContent(readFileContent(tasksPath)) {
			if t.HumanReview && !before[t.Text] {
				_ = m.activityStore.Append(w.WorkspaceID, w.ActiveChange, activity.Entry{
					Type:     "pool.task_flagged",
					Category: "pool",
					Summary:  "Task flagged for human review: " + t.Text,
					Meta: map[string]any{
						"worker_id": w.ID,
						"change":    w.ActiveChange,
					},
				})
			}
		}
	}
	return true
}

func triagePrompt(tasks []string) string {
	var b strings.Builder
	b.WriteString("The following tasks of tasks.md are still unchecked:\n")
	for _, t := range tasks {
		b.WriteString("- " + t + "\n")
	}
	b.WriteString("For each of them, either complete it and check it (- [x]), or, if it requires the user's intervention (manual walkthrough, visual check, action outside your environment), leave it unchecked and add " + openspec.HumanReviewMarker + " at the end of its line. Do not modify any other file just to flag a task.")
	return b.String()
}

func readFileContent(path string) string {
	data, _ := os.ReadFile(path)
	return string(data)
}

func isAgentStop(err error) bool {
	var te *agentTurnError
	var ie *agentIdleError
	return errors.As(err, &te) || errors.As(err, &ie)
}

// agentTurnPauseReason turns a turn error into a pause reason.
func agentTurnPauseReason(err error, fallback string) string {
	var te *agentTurnError
	var ie *agentIdleError
	switch {
	case errors.As(err, &te):
		return fmt.Sprintf("L'agent a terminé son tour en erreur : %s", te.reason)
	case errors.As(err, &ie):
		return fmt.Sprintf("Agent inactif depuis %s : processus arrêté", ie.after)
	}
	return fmt.Sprintf("%s : %v", fallback, err)
}

// invokeAgentApply writes the initial turn instructing the agent to work
// through the change's remaining tasks, and waits for it to complete.
func (m *Manager) invokeAgentApply(w *Worker, proc *session.Subprocess) error {
	return m.runTurn(w, proc, "/opsx:apply "+w.ActiveChange)
}

// invokeAgentHeal writes a follow-up turn on the same subprocess, injecting
// the validation error's text into the agent's context, and waits for it to
// complete. It never starts a new subprocess.
func (m *Manager) invokeAgentHeal(w *Worker, proc *session.Subprocess, validationErr error) error {
	return m.runTurn(w, proc, validationErr.Error())
}

// runTurn writes a single stream-json user turn to proc's stdin, then reads
// its stdout until a line signals the turn is complete (a native "result"
// event, or its Gemini-translated "message_complete" form). While the turn is
// in flight, it best-effort extracts human-readable activity text from each
// line into w.Activity and broadcasts pool_updated at most once per second
// (never once per line). It returns an error if the subprocess ends before
// signaling completion.
func (m *Manager) runTurn(w *Worker, proc *session.Subprocess, content string) error {
	_, err := m.runTurnText(m.workerTurnTarget(w), proc, content)
	return err
}

// turnTarget is what a turn needs of its owner: the run journal, the activity
// display and the cancellation of the agent's process group. A worker and a
// verification both provide one.
type turnTarget struct {
	ref         runRef
	setActivity func(string)
	notify      func()
	procCancel  func() // nil when there is nothing to cancel
}

func (m *Manager) workerTurnTarget(w *Worker) turnTarget {
	return turnTarget{
		ref:         w.ref(),
		setActivity: func(a string) { m.setActivity(w, a) },
		notify:      m.notify,
		procCancel: func() {
			if w.procCancel != nil {
				w.procCancel()
			}
		},
	}
}

// runTurnText is runTurn returning the text of the turn: the "result" field of
// the final event or, failing that, the text deltas accumulated during the turn.
func (m *Manager) runTurnText(t turnTarget, proc *session.Subprocess, content string) (string, error) {
	turn := map[string]interface{}{
		"type": "user",
		"message": map[string]string{
			"role":    "user",
			"content": content,
		},
	}
	data, err := json.Marshal(turn)
	if err != nil {
		return "", fmt.Errorf("failed to encode turn: %w", err)
	}
	m.logRunRef(t.ref, "in", data)
	if _, err := proc.Write(append(data, '\n')); err != nil {
		return "", fmt.Errorf("failed to write turn to agent subprocess: %w", err)
	}

	scanner := bufio.NewScanner(proc.Stdout())
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	// Idle watchdog, re-armed on every line read: on expiry the agent's
	// process group is killed and stdout closed to unblock the scan.
	var idled atomic.Bool
	idle := time.AfterFunc(agentIdleTimeout, func() {
		idled.Store(true)
		if t.procCancel != nil {
			t.procCancel()
		}
		_ = proc.Stdout().Close()
	})
	defer idle.Stop()

	var deltas strings.Builder
	var lastNotify time.Time
	for scanner.Scan() {
		idle.Reset(agentIdleTimeout)
		line := scanner.Bytes()
		m.logRunRef(t.ref, "out", line)

		switch kind, reason := classifyTurnLine(line); kind {
		case turnOK:
			if text := turnResultText(line); text != "" {
				return text, nil
			}
			return deltas.String(), nil
		case turnError:
			return "", &agentTurnError{reason: reason}
		}

		deltas.WriteString(extractTextDelta(line))
		if activity := extractActivity(line); activity != "" {
			t.setActivity(activity)
			if now := time.Now(); lastNotify.IsZero() || now.Sub(lastNotify) >= activityBroadcastInterval {
				t.notify()
				lastNotify = now
			}
		}
	}
	if idled.Load() {
		return "", &agentIdleError{after: agentIdleTimeout}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("agent subprocess stdout error before completion: %w", err)
	}
	return "", fmt.Errorf("agent subprocess ended before returning a result")
}

// turnResultText returns the "result" text of an end-of-turn line, or "".
func turnResultText(line []byte) string {
	var data struct {
		Result json.RawMessage `json:"result"`
	}
	if json.Unmarshal(line, &data) != nil {
		return ""
	}
	var text string
	if json.Unmarshal(data.Result, &text) != nil {
		return ""
	}
	return strings.TrimSpace(text)
}

// extractTextDelta returns the assistant text carried by a streaming delta
// line (thinking and tool events excluded), or "".
func extractTextDelta(line []byte) string {
	var data struct {
		Type  string `json:"type"`
		Delta struct {
			Text string `json:"text"`
		} `json:"delta"`
		Event json.RawMessage `json:"event"`
	}
	if json.Unmarshal(line, &data) != nil {
		return ""
	}
	if data.Type == "content_block_delta" {
		return data.Delta.Text
	}
	if data.Type == "stream_event" && len(data.Event) > 0 {
		var evt struct {
			Type  string `json:"type"`
			Delta struct {
				Text string `json:"text"`
			} `json:"delta"`
		}
		if json.Unmarshal(data.Event, &evt) == nil && evt.Type == "content_block_delta" {
			return evt.Delta.Text
		}
	}
	return ""
}

// turnKind classifies a stdout line with respect to the end of a turn.
type turnKind int

const (
	turnNone  turnKind = iota // not an end-of-turn line
	turnOK                    // the turn ended successfully
	turnError                 // the turn ended with an error result
)

// classifyTurnLine reports whether a stdout line ends an agent turn and how:
// a native stream-json "result" event (an error when is_error is set or its
// subtype starts with "error"), the "message_complete" event that
// translateGeminiLine and translateAntigravityLine produce, or raw
// Antigravity event: "result". For an error it also returns the reason.
func classifyTurnLine(line []byte) (turnKind, string) {
	var data struct {
		Type    string          `json:"type"`
		Event   string          `json:"event"`
		Subtype string          `json:"subtype"`
		IsError bool            `json:"is_error"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(line, &data); err != nil {
		return turnNone, ""
	}
	if data.Type == "result" {
		if data.IsError || strings.HasPrefix(data.Subtype, "error") {
			reason := data.Subtype
			var text string
			if json.Unmarshal(data.Result, &text) == nil && strings.TrimSpace(text) != "" {
				if reason != "" {
					reason += " : "
				}
				reason += strings.TrimSpace(text)
			}
			if reason == "" {
				reason = "résultat en erreur"
			}
			return turnError, reason
		}
		return turnOK, ""
	}
	if data.Type == "message_complete" || data.Event == "result" {
		return turnOK, ""
	}
	return turnNone, ""
}

// isTurnCompleteLine reports whether a stdout line ends a turn, successfully or not.
func isTurnCompleteLine(line []byte) bool {
	kind, _ := classifyTurnLine(line)
	return kind != turnNone
}

// extractActivity best-effort extracts human-readable text from a single
// stdout line for display as the worker's current activity, otherwise a
// truncated version of the raw line. Fails soft on invalid JSON or an empty
// line rather than blocking the apply/heal loop.
func extractActivity(line []byte) string {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" {
		return ""
	}

	type delta struct {
		Text     string `json:"text"`
		Thinking string `json:"thinking"`
	}
	var data struct {
		Type       string          `json:"type"`
		Delta      delta           `json:"delta"`
		Event      json.RawMessage `json:"event"`
		StepUpdate struct {
			StepType  string `json:"step_type"`
			TextDelta string `json:"text_delta"`
			ToolName  string `json:"tool_name"`
		} `json:"step_update"`
		ContentBlock struct {
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"content_block"`
	}
	if err := json.Unmarshal(line, &data); err == nil {
		if data.Type == "stream_event" && len(data.Event) > 0 {
			var evt struct {
				Type  string `json:"type"`
				Delta delta  `json:"delta"`
			}
			if err := json.Unmarshal(data.Event, &evt); err == nil && evt.Type == "content_block_delta" {
				if evt.Delta.Text != "" {
					return evt.Delta.Text
				}
				return evt.Delta.Thinking
			}
		}
		if data.Type == "content_block_delta" {
			if data.Delta.Text != "" {
				return data.Delta.Text
			}
			return data.Delta.Thinking
		}
		if data.Type == "content_block_start" && data.ContentBlock.Type == "tool_use" && data.ContentBlock.Name != "" {
			return data.ContentBlock.Name
		}
		if data.StepUpdate.TextDelta != "" {
			return data.StepUpdate.TextDelta
		}
		if data.StepUpdate.ToolName != "" {
			return data.StepUpdate.ToolName
		}
	}

	if len(trimmed) > maxActivityLen {
		return trimmed[:maxActivityLen]
	}
	return trimmed
}
