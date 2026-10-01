package pool

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/language"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/session"
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

	// 1. Provision Environment
	worktreePath, err := wt.Provision(w.ActiveChange)
	if err != nil {
		log.Printf("[worker %d] failed to provision worktree: %v\n", w.ID, err)
		pause(fmt.Sprintf("Échec du provisionnement du worktree : %v", err))
		return
	}
	m.setWorktree(w, worktreePath, "feature/"+w.ActiveChange)

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

	// The worktree starts from the last commit: an uncommitted change is not
	// in it, and no agent could apply it.
	tasksPath := filepath.Join(w.WorktreePath, "openspec", "changes", w.ActiveChange, "tasks.md")
	if !fileExists(tasksPath) {
		pause(fmt.Sprintf("Le changement « %s » doit être committé dans le dépôt avant d'être lancé (openspec/changes/%s/tasks.md est absent du worktree).", w.ActiveChange, w.ActiveChange))
		return
	}

	var stderrLog *conversation.SessionLog
	if runLog != nil {
		stderrLog = runLog.sess
	}
	proc, err := startSubprocessFn(procCtx, w.WorktreePath, agentCfg, "", "", false, stderrLog, customEnv, false, langDirective)
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

	// 4. Run local validation (compilation + tests).
	m.setStatus(w, StatusTesting)
	m.notify()
	validationErr := m.runValidation(ctx, w)
	var envErr *ValidationEnvError
	if errors.As(validationErr, &envErr) {
		// No heal turn can fix the environment: pause at once, no attempt used.
		pause(envErr.Reason)
		return
	}

	// 5. Self-healing loop: re-inject validation errors into the same
	// subprocess's context until it passes or attempts are exhausted.
	attempts := 0
	for validationErr != nil && attempts < cfg.MaxAttempts {
		attempts++
		m.setStatus(w, StatusHealing)
		m.notify()
		log.Printf("[worker %d] Validation failed. Attempt %d/%d to heal.\n", w.ID, attempts, cfg.MaxAttempts)

		if err := m.invokeAgentHeal(w, proc, validationErr); err != nil {
			log.Printf("[worker %d] agent heal error: %v\n", w.ID, err)
			if isAgentStop(err) {
				pause(agentTurnPauseReason(err, ""))
				return
			}
			break
		}

		m.setStatus(w, StatusTesting)
		m.notify()
		validationErr = m.runValidation(ctx, w)
		if errors.As(validationErr, &envErr) {
			pause(envErr.Reason)
			return
		}
	}

	if validationErr != nil {
		log.Printf("[worker %d] Failed to heal after %d attempts. Pausing.\n", w.ID, cfg.MaxAttempts)
		pause(fmt.Sprintf("Tentatives de réparation épuisées après %d essai(s) : %v", cfg.MaxAttempts, validationErr))
		return
	}

	// 6. Completion check: don't finalize (merge or transition) unless the
	// worktree's tasks.md lists at least one task and none is left unchecked,
	// even though validation passed - the agent may have declared victory
	// without actually doing the work. A missing or empty list is not "done".
	done, total := openspec.ParseTaskProgress(tasksPath)
	if total == 0 {
		log.Printf("[worker %d] validation passed but tasks.md is absent or empty; pausing without finalizing\n", w.ID)
		pause("Validation réussie mais la liste des tâches (tasks.md) est absente ou vide")
		return
	}
	if done < total {
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

	// 8. State transitions based on delegation mode.
	if cfg.DelegationMode == ModeFullAutonomy {
		m.mergeMu.Lock()
		// Point of no return: checked right after the lock, which may have
		// been waited on for a long time. A merge that has started is never
		// interrupted.
		if ctx.Err() != nil {
			m.mergeMu.Unlock()
			outcome = OutcomeStopped
			return
		}
		target, err := wt.MergeInto(w.ActiveChange)
		m.mergeMu.Unlock()
		if err != nil {
			log.Printf("[worker %d] failed to merge: %v\n", w.ID, err)
			if errors.Is(err, ErrMergeInProgress) {
				pause("Un merge est déjà en cours dans le dépôt : terminez-le ou annulez-le, puis reprenez le worker")
			} else {
				pause(fmt.Sprintf("Échec de la fusion dans %s (merge annulé, branche et worktree conservés) : %s", target, truncateReason(err)))
			}
			return
		}
		merged, mergeTarget = true, target
		log.Printf("[worker %d] Full Autonomy: merged %s into %s\n", w.ID, w.ActiveChange, target)
		if err := wt.Remove(w.ActiveChange); err != nil {
			pause(fmt.Sprintf("Fusion réussie dans %s mais le worktree n'a pas pu être supprimé : %s", target, truncateReason(err)))
			return
		}
		if err := wt.DeleteBranch(w.ActiveChange); err != nil {
			pause(fmt.Sprintf("Fusion réussie dans %s mais la branche n'a pas pu être supprimée : %s", target, truncateReason(err)))
			return
		}
		outcome = OutcomeCompleted
		return
	}

	// HITL review: branch and worktree stay in place; the dispatcher must not
	// hand the change out again while it awaits review.
	log.Printf("[worker %d] HITL Review: change %s ready for review\n", w.ID, w.ActiveChange)
	m.mu.Lock()
	m.reviewChanges[w.ActiveChange] = true
	m.mu.Unlock()
	outcome = OutcomeAwaitingReview
}

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
	turn := map[string]interface{}{
		"type": "user",
		"message": map[string]string{
			"role":    "user",
			"content": content,
		},
	}
	data, err := json.Marshal(turn)
	if err != nil {
		return fmt.Errorf("failed to encode turn: %w", err)
	}
	m.logRun(w, "in", data)
	if _, err := proc.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write turn to agent subprocess: %w", err)
	}

	scanner := bufio.NewScanner(proc.Stdout())
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	// Idle watchdog, re-armed on every line read: on expiry the agent's
	// process group is killed and stdout closed to unblock the scan.
	var idled atomic.Bool
	idle := time.AfterFunc(agentIdleTimeout, func() {
		idled.Store(true)
		if w.procCancel != nil {
			w.procCancel()
		}
		_ = proc.Stdout().Close()
	})
	defer idle.Stop()

	var lastNotify time.Time
	for scanner.Scan() {
		idle.Reset(agentIdleTimeout)
		line := scanner.Bytes()
		m.logRun(w, "out", line)

		switch kind, reason := classifyTurnLine(line); kind {
		case turnOK:
			return nil
		case turnError:
			return &agentTurnError{reason: reason}
		}

		if activity := extractActivity(line); activity != "" {
			m.setActivity(w, activity)
			if now := time.Now(); lastNotify.IsZero() || now.Sub(lastNotify) >= activityBroadcastInterval {
				m.notify()
				lastNotify = now
			}
		}
	}
	if idled.Load() {
		return &agentIdleError{after: agentIdleTimeout}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("agent subprocess stdout error before completion: %w", err)
	}
	return fmt.Errorf("agent subprocess ended before returning a result")
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
