package pool

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/verification"
)

// uiStepTimeout bounds the UI step once it holds the UI lock: application
// start, agent turn and task ticking. Variable so tests can shorten it.
var uiStepTimeout = 45 * time.Minute

// uiStep verifies a change in the running application: it starts the
// application, lets a verifier agent exercise the scenarios in a browser, then
// ticks the human-review tasks the agent declared verified.
type uiStep struct{}

func (uiStep) Name() string { return stepUI }

func (uiStep) Enabled(r verification.Resolved) bool { return r.UI }

func (uiStep) Run(ctx context.Context, r *verifyRun) stepResult {
	m, v := r.m, r.v
	fail := func(reason, report string) stepResult {
		return stepResult{Verdict: verdictError, Reason: reason, Report: report}
	}
	if r.resolved.UIStartCommand == "" {
		return fail("uiStartCommand non configurée", "")
	}
	if r.resolved.UIBaseURL == "" {
		return fail("uiBaseUrl non configurée", "")
	}

	// One UI step at a time on the machine; waiting does not use a pool slot.
	m.setVerifyWaiting(v, true)
	if err := uiLock.Acquire(ctx); err != nil {
		m.setVerifyWaiting(v, false)
		return fail("annulée", "")
	}
	defer uiLock.Release()
	m.setVerifyWaiting(v, false)

	stepCtx, cancel := context.WithTimeout(ctx, uiStepTimeout)
	defer cancel()
	res := r.runLocked(stepCtx)
	if ctx.Err() == nil && errors.Is(stepCtx.Err(), context.DeadlineExceeded) {
		res.Verdict = verdictError
		res.Reason = fmt.Sprintf("délai maximal de l'étape dépassé (%s)", uiStepTimeout)
		res.Verified, res.Ignored = nil, nil
	}
	return res
}

// runLocked is the UI step proper, run under the UI lock.
func (r *verifyRun) runLocked(ctx context.Context) stepResult {
	m, v, change := r.m, r.v, r.v.change
	fail := func(reason, report string) stepResult {
		return stepResult{Verdict: verdictError, Reason: reason, Report: report}
	}

	worktreePath, err := r.wt.Provision(change)
	if err != nil {
		return fail(fmt.Sprintf("Échec du provisionnement du worktree : %v", err), "")
	}

	app, err := startApp(ctx, worktreePath, r.resolved.UIStartCommand, r.resolved.UIBaseURL, v.workspacePath, func(line string) {
		m.logRunMarkerRef(r.ref, map[string]any{"type": "verify_app_output", "line": line})
	})
	if err != nil {
		return fail(err.Error(), "")
	}
	// Whatever the outcome, the application's process group ends with the step.
	defer app.Stop(teardownGrace)
	if err := app.WaitReady(ctx, uiStartTimeout); err != nil {
		return fail(err.Error(), "")
	}

	artifacts, cleanup, err := r.artifactsDir()
	if err != nil {
		return fail(fmt.Sprintf("Impossible de créer le dossier de preuves : %v", err), "")
	}
	defer cleanup()

	// The baseline is taken once the application is up: its start command may
	// legitimately install or build.
	before, err := gitStatusTracked(worktreePath)
	if err != nil {
		return fail(fmt.Sprintf("Impossible de lire l'état du worktree : %v", err), "")
	}

	var human []string
	if content, ok := r.wt.BranchTasks(change); ok {
		human = pendingHumanTasks(content)
	}
	turn := buildUITurn(app.URL(), artifacts, openspec.ChangeScenarios(worktreePath, change), human)
	text, reason := m.runVerifierTurn(ctx, r, worktreePath, uiVerifyDirective, map[string]string{artifactsEnv: artifacts}, turn)
	if reason != "" {
		return fail(reason, "")
	}

	after, err := gitStatusTracked(worktreePath)
	if err != nil {
		return fail(fmt.Sprintf("Impossible de lire l'état du worktree : %v", err), text)
	}
	if after != before {
		return fail("le vérificateur a modifié le worktree : "+changedFiles(before, after), text)
	}

	verdict, declared := parseUIVerdict(text)
	switch verdict {
	case "FAIL":
		return stepResult{Verdict: verdictFail, Reason: "la vérification UI a relevé au moins un échec", Report: text}
	case "SKIP":
		if content, ok := r.wt.BranchTasks(change); ok && len(pendingHumanTasks(content)) > 0 {
			return fail("tâches de validation humaine non vérifiées", text)
		}
		return stepResult{Verdict: verdictPass, Report: text}
	case "PASS":
		ticked, ignored, err := r.tickVerifiedTasks(worktreePath, declared)
		res := stepResult{Verdict: verdictPass, Report: text, Verified: ticked, Ignored: ignored}
		if err != nil {
			res.Verdict, res.Reason = verdictError, err.Error()
		}
		return res
	default:
		return fail("verdict absent", text)
	}
}

// artifactsDir creates the evidence directory of the run, next to its journal,
// and returns it with its cleanup (a no-op when it lives in the store).
func (r *verifyRun) artifactsDir() (dir string, cleanup func(), err error) {
	if r.m.convStore != nil && r.ts != "" {
		dir = r.m.convStore.RunDir(r.v.workspaceID, r.v.change, "verify", r.ts)
		return dir, func() {}, os.MkdirAll(dir, 0o755)
	}
	dir, err = os.MkdirTemp("", "opensp8c-verify-*")
	return dir, func() { _ = os.RemoveAll(dir) }, err
}

// tickVerifiedTasks checks off, one commit per task, the unchecked
// human-review tasks of the branch whose text equals a declared line. It
// returns the ticked task texts and the declared lines that matched nothing. On
// a commit failure it stops, keeping the earlier ticks.
func (r *verifyRun) tickVerifiedTasks(dir string, declared []string) (ticked, ignored []string, err error) {
	change := r.v.change
	lock := r.m.reviewLock(change)
	lock.Lock()
	defer lock.Unlock()

	data, rerr := os.ReadFile(filepath.Join(dir, "openspec", "changes", change, "tasks.md"))
	if rerr != nil {
		return nil, declared, fmt.Errorf("lecture de tasks.md : %w", rerr)
	}
	tasks := openspec.ParseTaskListContent(string(data))
	seen := map[string]bool{}
	for _, text := range declared {
		if seen[text] {
			continue
		}
		seen[text] = true
		idx := -1
		for i, t := range tasks {
			if !t.Done && t.HumanReview && t.Text == text {
				idx = i
				break
			}
		}
		if idx < 0 {
			ignored = append(ignored, text)
			continue
		}
		if _, _, terr := toggleTaskInWorktree(r.wt, dir, change, idx); terr != nil {
			return ticked, ignored, fmt.Errorf("coche de la tâche « %s » impossible : %w", text, terr)
		}
		tasks[idx].Done = true
		ticked = append(ticked, text)
	}
	if len(ticked) > 0 {
		r.m.publishChangeUpdated(r.v.workspaceID, change)
	}
	return ticked, ignored, nil
}

// gitStatusTracked is the porcelain status without untracked files: an
// application may legitimately create logs or a local database.
func gitStatusTracked(dir string) (string, error) {
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=no")
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimRight(string(out), "\n"), err
}
