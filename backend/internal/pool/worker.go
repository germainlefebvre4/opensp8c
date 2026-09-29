package pool

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
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

// pauseWorker marks w as paused with a human-readable reason and snapshots it
// into m.pausedWorkers, so it stays visible via Status()/AllPools() even
// after runWorker's own deferred cleanup removes it from m.activeWorkers.
func (m *Manager) pauseWorker(w *Worker, reason string) {
	w.Status = StatusPaused
	w.BlockedReason = reason

	m.mu.Lock()
	snapshot := *w
	m.pausedWorkers[w.ID] = &snapshot
	m.mu.Unlock()

	m.notify()
}

// runWorker coordinates the lifecycle of a worker on a specific change.
func (m *Manager) runWorker(ctx context.Context, w *Worker) {
	defer func() {
		m.mu.Lock()
		delete(m.activeWorkers, w.ID)
		delete(m.lastWorkerStatus, w.ID)
		m.mu.Unlock()
		m.notify()
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

	select {
	case <-ctx.Done():
		return
	default:
	}

	w.Status = StatusWorking
	m.notify()

	// 2. Start a single agent subprocess for the whole attempt loop (the
	// initial apply turn and every heal turn share it, so the model sees its
	// own prior turns). It is terminated when this function returns, on any
	// exit path (success, paused, or cancellation).
	procCtx, procCancel := context.WithCancel(ctx)
	defer procCancel()

	var agentCfg agents.AgentConfig
	if m.sessionMgr != nil {
		agentCfg = m.sessionMgr.ResolveAgentConfig(w.WorkspaceID, w.ActiveChange)
	}

	var customEnv map[string]string
	if m.prefs != nil {
		if p, err := m.prefs.Load(); err == nil && p != nil {
			customEnv = p.EnvFor(agentCfg.ID)
		}
	}

	proc, err := startSubprocessFn(procCtx, w.WorktreePath, agentCfg, "", "", false, nil, customEnv, false)
	if err != nil {
		log.Printf("[worker %d] failed to start agent subprocess: %v\n", w.ID, err)
		m.pauseWorker(w, fmt.Sprintf("Échec du démarrage du subprocess de l'agent : %v", err))
		return
	}
	defer func() {
		_ = proc.CloseStdin()
		_ = proc.Wait()
	}()

	// 3. Invoke the agent CLI to implement the remaining tasks.
	if err := m.invokeAgentApply(w, proc); err != nil {
		log.Printf("[worker %d] agent apply error: %v\n", w.ID, err)
		m.pauseWorker(w, fmt.Sprintf("Échec de l'invocation de l'agent pour appliquer les tâches restantes : %v", err))
		return
	}

	// 4. Run local validation (compilation + tests).
	w.Status = StatusTesting
	m.notify()
	validationErr := m.runValidation(ctx, w)

	// 5. Self-healing loop: re-inject validation errors into the same
	// subprocess's context until it passes or attempts are exhausted.
	attempts := 0
	for validationErr != nil && attempts < m.config.MaxAttempts {
		attempts++
		w.Status = StatusHealing
		m.notify()
		log.Printf("[worker %d] Validation failed. Attempt %d/%d to heal.\n", w.ID, attempts, m.config.MaxAttempts)

		if err := m.invokeAgentHeal(w, proc, validationErr); err != nil {
			log.Printf("[worker %d] agent heal error: %v\n", w.ID, err)
			break
		}

		w.Status = StatusTesting
		m.notify()
		validationErr = m.runValidation(ctx, w)
	}

	if validationErr != nil {
		log.Printf("[worker %d] Failed to heal after %d attempts. Pausing.\n", w.ID, m.config.MaxAttempts)
		m.pauseWorker(w, fmt.Sprintf("Tentatives de réparation épuisées après %d essai(s) : %v", m.config.MaxAttempts, validationErr))
		return
	}

	// 6. Completion check: don't finalize (merge or transition) unless the
	// worktree's tasks.md has no task left unchecked, even though validation
	// passed - the agent may have declared victory without actually doing
	// the remaining work. The worktree mirrors the whole workspace (see
	// WorktreeController.Provision), so the change's tasks.md lives at its
	// normal nested path within it, not at the worktree root.
	tasksPath := filepath.Join(w.WorktreePath, "openspec", "changes", w.ActiveChange, "tasks.md")
	done, total := openspec.ParseTaskProgress(tasksPath)
	if done < total {
		log.Printf("[worker %d] validation passed but tasks.md incomplete (%d/%d done); pausing without finalizing\n", w.ID, done, total)
		m.pauseWorker(w, fmt.Sprintf("Validation réussie mais tâches restantes incomplètes (%d/%d) dans tasks.md", done, total))
		return
	}

	// 7. State Transitions based on Delegation Mode
	if m.config.DelegationMode == ModeFullAutonomy {
		// Merge and cleanup
		log.Printf("[worker %d] Full Autonomy: merging change %s\n", w.ID, w.ActiveChange)
		if err := wt.MergeAndCleanup(w.ActiveChange); err != nil {
			log.Printf("[worker %d] failed to merge: %v\n", w.ID, err)
		}
	} else {
		// HITL Review
		log.Printf("[worker %d] HITL Review: change %s ready for review\n", w.ID, w.ActiveChange)
		// State remains "to-review" implicitly as we leave the branch unmerged and worktree intact.
		// The UI will pick this up from the Kanban status.
	}
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
	if _, err := proc.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write turn to agent subprocess: %w", err)
	}

	scanner := bufio.NewScanner(proc.Stdout())
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var lastNotify time.Time
	for scanner.Scan() {
		line := scanner.Bytes()

		if isTurnCompleteLine(line) {
			return nil
		}

		if activity := extractActivity(line); activity != "" {
			w.Activity = activity
			if now := time.Now(); lastNotify.IsZero() || now.Sub(lastNotify) >= activityBroadcastInterval {
				m.notify()
				lastNotify = now
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("agent subprocess stdout error before completion: %w", err)
	}
	return fmt.Errorf("agent subprocess ended before returning a result")
}

// isTurnCompleteLine reports whether a stdout line signals the end of an
// agent turn: a native stream-json "result" event, the "message_complete"
// event that translateGeminiLine and translateAntigravityLine produce, or
// raw Antigravity event: "result".
func isTurnCompleteLine(line []byte) bool {
	var data struct {
		Type  string `json:"type"`
		Event string `json:"event"`
	}
	if err := json.Unmarshal(line, &data); err != nil {
		return false
	}
	return data.Type == "result" || data.Type == "message_complete" || data.Event == "result"
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
