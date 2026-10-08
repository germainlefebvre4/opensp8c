package pool

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/language"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/verification"
)

// Verdicts recorded in the verify_run_end marker.
const (
	verdictPass  = "pass"
	verdictFail  = "fail"
	verdictError = "error"
)

// stepConformity is the name of the conformity verification step.
const stepConformity = "conformity"

// verifyDirective is appended to the system prompt of the verifier agent.
const verifyDirective = `You are verifying a change, not implementing it. NEVER create, modify, move or delete any file, and never run a command that does. Conclude your final answer with exactly one last line: "VERDICT: PASS" when you found no critical issue (warnings and suggestions alone are acceptable), or "VERDICT: FAIL" as soon as you found at least one critical issue.`

// verdictRe matches a verdict line of the verifier's answer.
var verdictRe = regexp.MustCompile(`(?i)^VERDICT:\s*(PASS|FAIL)\s*$`)

// stepResult is the outcome of one verification step. Verdict is pass, fail
// (a critical issue was reported) or error (no usable verdict).
type stepResult struct {
	Verdict string
	Reason  string
	Report  string
}

// verifyStep is one ordered stage of the verification.
type verifyStep interface {
	Name() string
	Enabled(verification.Resolved) bool
	Run(ctx context.Context, v *verifyRun) stepResult
}

// verifySteps lists the steps, in execution order.
var verifySteps = []verifyStep{conformityStep{}}

// verifyJob is one verification in flight, owned by the Manager.
type verifyJob struct {
	workspaceID   string
	workspaceName string
	workspacePath string
	change        string
	cancel        context.CancelFunc
	step          string // current step name, guarded by Manager.mu
}

// verifyRun is what a step sees of its verification.
type verifyRun struct {
	m             *Manager
	v             *verifyJob
	wt            *WorktreeController
	ref           runRef
	worktreesRoot string
}

// resolveVerification returns the effective verification settings of change:
// the change's own override (read from the main repository), then the
// workspace, then the Configuration. It is the single adapter over the
// verification settings.
func (m *Manager) resolveVerification(workspaceID, change string) verification.Resolved {
	m.mu.Lock()
	path := m.workspacePath
	m.mu.Unlock()
	return m.resolveVerificationAt(workspaceID, path, change)
}

func (m *Manager) resolveVerificationAt(workspaceID, workspacePath, change string) verification.Resolved {
	var override *verification.Override
	if o, err := openspec.ReadVerification(workspacePath, change); err == nil {
		override = o
	}
	var p *preferences.Preferences // nil resolves to the built-in values
	if m.prefs != nil {
		if loaded, err := m.prefs.Load(); err == nil {
			p = loaded
		}
	}
	return p.ResolveVerification(workspaceID, override)
}

// verificationRequired reports whether at least one step is enabled for change.
func (m *Manager) verificationRequired(workspaceID, workspacePath, change string) bool {
	return anyStepEnabled(m.resolveVerificationAt(workspaceID, workspacePath, change))
}

func anyStepEnabled(r verification.Resolved) bool {
	for _, s := range verifySteps {
		if s.Enabled(r) {
			return true
		}
	}
	return false
}

// VerificationRunning reports whether a verification of change is in flight.
func (m *Manager) VerificationRunning(change string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.verifications[change]
	return ok
}

// VerificationStep returns the step a verification of change is running, or
// "" when none is.
func (m *Manager) VerificationStep(change string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := m.verifications[change]; ok {
		return v.step
	}
	return ""
}

// tickVerifications starts the queued verifications, up to the pool size and
// regardless of the worker slots, then hands the verified changes to a
// finalizing worker. Callers hold m.mu.
func (m *Manager) tickVerifications(changes []openspec.Change, held map[string]bool) {
	var queued, passed []string
	for _, c := range changes {
		if c.KanbanStatus != "verifying" {
			continue
		}
		switch c.VerificationState {
		case "queued":
			queued = append(queued, c.Name)
		case "passed":
			passed = append(passed, c.Name)
		}
	}
	sort.Strings(queued)
	sort.Strings(passed)

	for _, name := range queued {
		if len(m.verifications) >= m.config.Size {
			break
		}
		// A worker still holds the change while it winds down its agent.
		if _, running := m.verifications[name]; running || held[name] {
			continue
		}
		m.startVerification(name)
	}
	for _, name := range passed {
		if len(m.activeWorkers) >= m.config.Size {
			break
		}
		if held[name] {
			continue
		}
		m.startWorkerFor(name, true)
		held[name] = true
	}
}

// startVerification registers and launches a verification. Callers hold m.mu.
func (m *Manager) startVerification(change string) {
	ctx, cancel := context.WithCancel(context.Background())
	v := &verifyJob{
		workspaceID:   m.workspaceID,
		workspaceName: m.workspaceName,
		workspacePath: m.workspacePath,
		change:        change,
		cancel:        cancel,
	}
	if m.verifications == nil {
		m.verifications = make(map[string]*verifyJob)
	}
	m.verifications[change] = v
	worktreesRoot := m.worktreesRoot

	m.workers.Add(1)
	go func() {
		defer m.workers.Done()
		m.runVerification(ctx, v, worktreesRoot)
	}()
	m.broadcastLocked()
	m.publishChangeUpdated(v.workspaceID, change)
}

func (m *Manager) verifyActivity(v *verifyJob, typ, summary string, meta map[string]any) {
	if m.activityStore == nil {
		return
	}
	if meta == nil {
		meta = map[string]any{}
	}
	meta["change"] = v.change
	_ = m.activityStore.Append(v.workspaceID, v.change, activity.Entry{
		Type:     typ,
		Category: "pool",
		Summary:  summary,
		Meta:     meta,
	})
}

// runVerification runs the enabled steps in order and records the result in
// the verification marker: passed when all succeed (or none is enabled), failed
// at the first step that does not. A cancellation leaves the marker untouched.
func (m *Manager) runVerification(ctx context.Context, v *verifyJob, worktreesRoot string) {
	defer func() {
		m.mu.Lock()
		if m.verifications[v.change] == v {
			delete(m.verifications, v.change)
		}
		m.broadcastLocked()
		m.mu.Unlock()
		m.publishChangeUpdated(v.workspaceID, v.change)
	}()
	defer v.cancel()

	wt := NewWorktreeController(v.workspacePath, v.workspaceID, worktreesRoot)
	resolved := m.resolveVerificationAt(v.workspaceID, v.workspacePath, v.change)
	var steps []verifyStep
	for _, s := range verifySteps {
		if s.Enabled(resolved) {
			steps = append(steps, s)
		}
	}
	if len(steps) == 0 {
		if err := wt.SetVerify(v.change, openspec.VerifyPassed); err != nil {
			log.Printf("[verify %s] cannot record the verification state: %v\n", v.change, err)
		}
		return
	}

	ref := runRef{workspaceID: v.workspaceID, change: v.change}
	runLog, runTS := m.openRunLogFor(ref, "verify")
	ref.log = runLog
	if runLog != nil {
		defer func() {
			m.finishRunRef(ref)
			_ = runLog.sess.Close()
		}()
	}
	_ = runTS // the run is identified by its timestamp in the store, not here
	m.logRunMarkerRef(ref, map[string]any{"type": "verify_run_start", "change": v.change})
	m.verifyActivity(v, "pool.verification_started", "Verification started", nil)

	run := &verifyRun{m: m, v: v, wt: wt, ref: ref, worktreesRoot: worktreesRoot}
	var last stepResult
	lastStep := ""
	for _, s := range steps {
		m.mu.Lock()
		v.step = s.Name()
		m.mu.Unlock()
		m.publishChangeUpdated(v.workspaceID, v.change)

		lastStep = s.Name()
		last = s.Run(ctx, run)
		if ctx.Err() != nil {
			m.logRunMarkerRef(ref, map[string]any{"type": "verify_run_end", "step": lastStep, "verdict": verdictError, "reason": "annulée", "report": last.Report})
			return
		}
		if last.Verdict != verdictPass {
			break
		}
	}

	state, typ, summary := openspec.VerifyPassed, "pool.verification_passed", "Verification passed"
	if last.Verdict != verdictPass {
		state, typ = openspec.VerifyFailed, "pool.verification_failed"
		summary = "Verification failed: " + last.Reason
	}
	if err := wt.SetVerify(v.change, state); err != nil {
		log.Printf("[verify %s] cannot record the verification state: %v\n", v.change, err)
	}
	m.logRunMarkerRef(ref, map[string]any{"type": "verify_run_end", "step": lastStep, "verdict": last.Verdict, "reason": last.Reason, "report": last.Report})
	m.verifyActivity(v, typ, summary, map[string]any{"step": lastStep, "reason": last.Reason})
}

// conformityStep runs /opsx:verify in a read-only verifier agent.
type conformityStep struct{}

func (conformityStep) Name() string { return stepConformity }

func (conformityStep) Enabled(r verification.Resolved) bool { return r.Conformity }

func (conformityStep) Run(ctx context.Context, r *verifyRun) stepResult {
	m, change := r.m, r.v.change
	fail := func(reason, report string) stepResult {
		return stepResult{Verdict: verdictError, Reason: reason, Report: report}
	}

	worktreePath, err := r.wt.Provision(change)
	if err != nil {
		return fail(fmt.Sprintf("Échec du provisionnement du worktree : %v", err), "")
	}
	before, err := gitStatusPorcelain(worktreePath)
	if err != nil {
		return fail(fmt.Sprintf("Impossible de lire l'état du worktree : %v", err), "")
	}

	procCtx, procCancel := context.WithCancel(ctx)
	defer procCancel()

	var agentCfg agents.AgentConfig
	if m.sessionMgr != nil {
		agentCfg = m.sessionMgr.ResolveRoleConfig(r.v.workspaceID, preferences.RoleVerifier)
	}
	var customEnv map[string]string
	langDirective := language.Directive(language.Worker, language.Resolve(language.Levels{}, ""))
	if m.prefs != nil {
		if p, err := m.prefs.Load(); err == nil && p != nil {
			customEnv = p.EnvForWorkspace(r.v.workspaceID, agentCfg.ID)
			langDirective = p.LanguageDirective(language.Worker)
		}
	}

	var stderrLog *conversation.SessionLog
	if r.ref.log != nil {
		stderrLog = r.ref.log.sess
	}
	proc, err := startSubprocessFn(procCtx, worktreePath, agentCfg, verifyDirective, "", false, stderrLog, customEnv, false, langDirective)
	if err != nil {
		return fail(fmt.Sprintf("Échec du démarrage du subprocess de l'agent : %v", err), "")
	}
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

	target := turnTarget{
		ref:         r.ref,
		setActivity: func(string) {},
		notify:      func() {},
		procCancel:  procCancel,
	}
	text, err := m.runTurnText(target, proc, "/opsx:verify "+change)
	if err != nil {
		return fail(agentTurnPauseReason(err, "Échec du tour de vérification"), "")
	}

	after, err := gitStatusPorcelain(worktreePath)
	if err != nil {
		return fail(fmt.Sprintf("Impossible de lire l'état du worktree : %v", err), text)
	}
	if after != before {
		return fail("le vérificateur a modifié le worktree : "+changedFiles(before, after), text)
	}

	switch parseVerdict(text) {
	case "PASS":
		return stepResult{Verdict: verdictPass, Report: text}
	case "FAIL":
		return stepResult{Verdict: verdictFail, Reason: "la vérification a relevé au moins un point critique", Report: text}
	default:
		return fail("verdict absent", text)
	}
}

// parseVerdict returns "PASS" or "FAIL" from the last VERDICT line of text,
// or "" when there is none.
func parseVerdict(text string) string {
	verdict := ""
	for _, line := range strings.Split(text, "\n") {
		if m := verdictRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			verdict = strings.ToUpper(m[1])
		}
	}
	return verdict
}

func gitStatusPorcelain(dir string) (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}

// changedFiles lists the porcelain lines present in after but not in before.
func changedFiles(before, after string) string {
	seen := map[string]bool{}
	for _, l := range strings.Split(before, "\n") {
		seen[l] = true
	}
	var files []string
	for _, l := range strings.Split(after, "\n") {
		if l != "" && !seen[l] {
			files = append(files, strings.TrimSpace(l))
		}
	}
	if len(files) == 0 {
		return "contenu de fichiers déjà modifiés"
	}
	return strings.Join(files, ", ")
}

// SeedVerificationForTest registers a verification of change as in flight
// (running step), without running anything, so cross-package tests can observe
// the running state. The returned function drops it.
func SeedVerificationForTest(m *Manager, change, step string) (release func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v := &verifyJob{change: change, step: step, cancel: func() {}}
	if m.verifications == nil {
		m.verifications = make(map[string]*verifyJob)
	}
	m.verifications[change] = v
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		delete(m.verifications, change)
	}
}
