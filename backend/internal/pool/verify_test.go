package pool

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/session"
)

// verifierStub scripts the verifier agent: it records how it was started and
// what it was asked, optionally modifies the worktree, then answers.
type verifierStub struct {
	mu      sync.Mutex
	starts  int
	models  []string
	prompts []string
	cwds    []string
	turns   []string

	answer   string // text of the result; "" with raw unused
	raw      string // when set, the exact final line
	before   func(ws string)
	gate     chan struct{} // when set, the agent waits for it before answering
	startErr error
}

func (s *verifierStub) start(ctx context.Context, ws string, agentCfg agents.AgentConfig, extra, _ string, _ bool, _ *conversation.SessionLog, _ map[string]string, _ bool, _ string) (*session.Subprocess, error) {
	s.mu.Lock()
	s.starts++
	s.models = append(s.models, agentCfg.Model)
	s.prompts = append(s.prompts, extra)
	s.cwds = append(s.cwds, ws)
	err := s.startErr
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	go func() {
		r := bufio.NewReader(inR)
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				_ = outW.Close()
				return
			}
			var turn struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}
			_ = json.Unmarshal([]byte(line), &turn)
			s.mu.Lock()
			s.turns = append(s.turns, turn.Message.Content)
			s.mu.Unlock()
			if s.gate != nil {
				select {
				case <-s.gate:
				case <-ctx.Done():
					_ = outW.Close()
					return
				}
			}
			if s.before != nil {
				s.before(ws)
			}
			final := s.raw
			if final == "" {
				final = verdictResult(s.answer)
			}
			if _, err := outW.Write([]byte(final + "\n")); err != nil {
				return
			}
		}
	}()
	return session.NewTestSubprocess(inW, outR, "claude"), nil
}

func (s *verifierStub) startCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.starts
}

// awaitingVerification implements change with a worker whose configuration
// enables the conformity verification, leaving it pending, then installs the
// verifier stub for the next subprocess starts.
func awaitingVerification(t *testing.T, change string, mode DelegationMode, v *verifierStub) (*Manager, string, *conversation.Store) {
	t.Helper()
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m, store, _ := newJournalManager(t, repo, mode, autoStub)
	m.prefs = newVerifyPrefs(t, true)
	m.activityStore = activity.NewStore(t.TempDir(), nil)
	runOnce(t, m, change)
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	if st, _ := wt.VerifyState(change); st != "pending" {
		t.Fatalf("setup: marker = %q", st)
	}
	prev := startSubprocessFn
	startSubprocessFn = v.start
	t.Cleanup(func() { startSubprocessFn = prev })
	return m, repo, store
}

func verifyStateOf(t *testing.T, m *Manager, repo, change string) string {
	t.Helper()
	st, ok := NewWorktreeController(repo, "ws1", m.worktreesRoot).VerifyState(change)
	if !ok {
		return ""
	}
	return st
}

// runVerifyNow runs the verification of change synchronously.
func runVerifyNow(m *Manager, change string) {
	m.mu.Lock()
	m.startVerification(change)
	m.mu.Unlock()
	m.waitWorkers()
}

func loadVerifyRun(t *testing.T, store *conversation.Store, change string) []journalLine {
	t.Helper()
	runs, err := store.List("ws1", change, "verify")
	if err != nil || len(runs) == 0 {
		t.Fatalf("expected a verify run, got %v (%v)", runs, err)
	}
	raw, err := store.Load("ws1", change, "verify", runs[len(runs)-1].Ts)
	if err != nil {
		t.Fatal(err)
	}
	var lines []journalLine
	for _, r := range raw {
		var l journalLine
		if err := json.Unmarshal(r, &l); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, l)
	}
	return lines
}

func verifyEnd(t *testing.T, lines []journalLine) map[string]any {
	t.Helper()
	last := lines[len(lines)-1]
	var end map[string]any
	if err := json.Unmarshal(last.Data, &end); err != nil || end["type"] != "verify_run_end" {
		t.Fatalf("last line should be verify_run_end, got %s", last.Data)
	}
	return end
}

func activityTypes(t *testing.T, m *Manager, change string) []string {
	t.Helper()
	entries, err := m.activityStore.Read("ws1", change)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Type, "pool.verification_") {
			out = append(out, e.Type)
		}
	}
	return out
}

func TestConformityStep_Pass(t *testing.T) {
	v := &verifierStub{answer: "All good.\nVERDICT: PASS"}
	m, repo, store := awaitingVerification(t, "vc-pass", ModeHITLReview, v)

	runVerifyNow(m, "vc-pass")

	if st := verifyStateOf(t, m, repo, "vc-pass"); st != "passed" {
		t.Fatalf("marker = %q, want passed", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "vc-pass"))
	if end["verdict"] != "pass" || end["step"] != "conformity" || !strings.Contains(end["report"].(string), "All good.") {
		t.Errorf("end marker = %v", end)
	}
	if got := activityTypes(t, m, "vc-pass"); len(got) != 2 || got[0] != "pool.verification_started" || got[1] != "pool.verification_passed" {
		t.Errorf("activity = %v", got)
	}
	if v.turns[0] != "/opsx:verify vc-pass" {
		t.Errorf("turn = %q", v.turns[0])
	}
	if !strings.Contains(v.prompts[0], "VERDICT: PASS") || !strings.Contains(v.prompts[0], "NEVER create, modify") {
		t.Errorf("directive = %q", v.prompts[0])
	}
	if !strings.Contains(v.cwds[0], "vc-pass") || v.cwds[0] == repo {
		t.Errorf("the verifier must run in the change worktree, got %q", v.cwds[0])
	}
	if m.VerificationRunning("vc-pass") {
		t.Error("verification should be finished")
	}
}

func TestConformityStep_FailKeepsReport(t *testing.T) {
	v := &verifierStub{answer: "CRITICAL: missing requirement\nVERDICT: FAIL"}
	m, repo, store := awaitingVerification(t, "vc-fail", ModeHITLReview, v)

	runVerifyNow(m, "vc-fail")

	if st := verifyStateOf(t, m, repo, "vc-fail"); st != "failed" {
		t.Fatalf("marker = %q, want failed", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "vc-fail"))
	if end["verdict"] != "fail" || !strings.Contains(end["report"].(string), "CRITICAL: missing requirement") {
		t.Errorf("end marker = %v", end)
	}
	if got := activityTypes(t, m, "vc-fail"); len(got) != 2 || got[1] != "pool.verification_failed" {
		t.Errorf("activity = %v", got)
	}
	if v.startCount() != 1 || len(v.turns) != 1 {
		t.Errorf("no second turn expected: starts=%d turns=%d", v.startCount(), len(v.turns))
	}
}

func TestConformityStep_MissingVerdictFailsClosed(t *testing.T) {
	v := &verifierStub{answer: "I looked at things."}
	m, repo, store := awaitingVerification(t, "vc-none", ModeHITLReview, v)

	runVerifyNow(m, "vc-none")

	if st := verifyStateOf(t, m, repo, "vc-none"); st != "failed" {
		t.Fatalf("marker = %q", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "vc-none"))
	if end["verdict"] != "error" || end["reason"] != "verdict absent" || end["report"] != "I looked at things." {
		t.Errorf("end marker = %v", end)
	}
}

func TestConformityStep_LastVerdictLineWins(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: FAIL\nafter a second look\nverdict: pass  "}
	m, repo, _ := awaitingVerification(t, "vc-last", ModeHITLReview, v)
	runVerifyNow(m, "vc-last")
	if st := verifyStateOf(t, m, repo, "vc-last"); st != "passed" {
		t.Fatalf("marker = %q, want passed", st)
	}
}

func TestConformityStep_ModifiedWorktreeFails(t *testing.T) {
	v := &verifierStub{
		answer: "VERDICT: PASS",
		before: func(ws string) { _ = os.WriteFile(filepath.Join(ws, "sneaky.txt"), []byte("x"), 0644) },
	}
	m, repo, store := awaitingVerification(t, "vc-dirty", ModeHITLReview, v)

	runVerifyNow(m, "vc-dirty")

	if st := verifyStateOf(t, m, repo, "vc-dirty"); st != "failed" {
		t.Fatalf("marker = %q", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "vc-dirty"))
	if !strings.Contains(end["reason"].(string), "le vérificateur a modifié le worktree") || !strings.Contains(end["reason"].(string), "sneaky.txt") {
		t.Errorf("reason = %v", end["reason"])
	}
	if _, err := os.Stat(filepath.Join(v.cwds[0], "sneaky.txt")); err != nil {
		t.Errorf("the modified file must be left in place: %v", err)
	}
}

func TestConformityStep_StartFailure(t *testing.T) {
	v := &verifierStub{startErr: io.ErrUnexpectedEOF}
	m, repo, store := awaitingVerification(t, "vc-start", ModeHITLReview, v)
	runVerifyNow(m, "vc-start")
	if st := verifyStateOf(t, m, repo, "vc-start"); st != "failed" {
		t.Fatalf("marker = %q", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "vc-start"))
	if end["verdict"] != "error" || !strings.Contains(end["reason"].(string), "démarrage") {
		t.Errorf("end marker = %v", end)
	}
}

func TestConformityStep_TurnError(t *testing.T) {
	v := &verifierStub{raw: `{"type":"result","subtype":"error_during_execution","is_error":true,"result":"boom"}`}
	m, repo, store := awaitingVerification(t, "vc-err", ModeHITLReview, v)
	runVerifyNow(m, "vc-err")
	if st := verifyStateOf(t, m, repo, "vc-err"); st != "failed" {
		t.Fatalf("marker = %q", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "vc-err"))
	if !strings.Contains(end["reason"].(string), "tour en erreur") {
		t.Errorf("reason = %v", end["reason"])
	}
}

func TestConformityStep_IdleAgentFails(t *testing.T) {
	setVar(t, &agentIdleTimeout, 150*time.Millisecond)
	v := &verifierStub{gate: make(chan struct{})}
	m, repo, store := awaitingVerification(t, "vc-idle", ModeHITLReview, v)
	runVerifyNow(m, "vc-idle")
	if st := verifyStateOf(t, m, repo, "vc-idle"); st != "failed" {
		t.Fatalf("marker = %q", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "vc-idle"))
	if !strings.Contains(end["reason"].(string), "inactif") {
		t.Errorf("reason = %v", end["reason"])
	}
}

func TestConformityStep_VerifierRoleModel(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, repo, _ := awaitingVerification(t, "vc-role", ModeHITLReview, v)
	installMockClaude(t)
	prefs := newRolePrefs(t, `{"roles":{"verifier":{"model":"haiku","effort":"low"}}}`)
	if err := prefs.SetVerificationDefaults(vpatchConformity(true)); err != nil {
		t.Fatal(err)
	}
	m.prefs = prefs
	m.sessionMgr = session.NewManager(prefs, nil)

	runVerifyNow(m, "vc-role")

	if st := verifyStateOf(t, m, repo, "vc-role"); st != "passed" {
		t.Fatalf("marker = %q", st)
	}
	if len(v.models) != 1 || v.models[0] != "haiku" {
		t.Errorf("verifier model = %v, want haiku", v.models)
	}
}

func TestRunVerification_NoStepEnabledPassesWithoutAgent(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, repo, _ := awaitingVerification(t, "vc-off", ModeHITLReview, v)
	m.prefs = newVerifyPrefs(t, false)

	runVerifyNow(m, "vc-off")

	if st := verifyStateOf(t, m, repo, "vc-off"); st != "passed" {
		t.Fatalf("marker = %q, want passed", st)
	}
	if v.startCount() != 0 {
		t.Error("no agent must be started")
	}
}

func TestRunVerification_CancellationKeepsPending(t *testing.T) {
	v := &verifierStub{gate: make(chan struct{})}
	m, repo, _ := awaitingVerification(t, "vc-stop", ModeHITLReview, v)
	m.isRunning = true

	m.mu.Lock()
	m.startVerification("vc-stop")
	m.mu.Unlock()
	waitFor(t, "verifier running", func() bool { return v.startCount() == 1 && m.VerificationStep("vc-stop") == "conformity" })
	if !m.VerificationRunning("vc-stop") {
		t.Fatal("verification should be running")
	}

	m.Stop()
	m.waitWorkers()

	if st := verifyStateOf(t, m, repo, "vc-stop"); st != "pending" {
		t.Errorf("marker = %q, want pending", st)
	}
	if m.VerificationRunning("vc-stop") {
		t.Error("verification should be dropped")
	}
	if got := activityTypes(t, m, "vc-stop"); len(got) != 1 || got[0] != "pool.verification_started" {
		t.Errorf("activity = %v", got)
	}
}

func listFor(t *testing.T, repo string) []openspec.Change {
	t.Helper()
	changes, err := openspec.ListChanges(repo)
	if err != nil {
		t.Fatal(err)
	}
	return changes
}

// multiChangeRepo returns a fixture repository holding every named change; the
// first one comes from newGoFixtureRepo, the others are committed afterwards.
func multiChangeRepo(t *testing.T, names ...string) string {
	t.Helper()
	repo := newGoFixtureRepo(t, names[0], "- [x] done\n")
	for _, n := range names[1:] {
		dir := filepath.Join(repo, "openspec", "changes", n)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("- [x] done\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if len(names) > 1 {
		commitAll(t, repo, "more changes")
	}
	return repo
}

func commitAll(t *testing.T, repo, msg string) {
	t.Helper()
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "-c", "user.email=t@t", "-c", "user.name=t", "-c", "commit.gpgsign=false", "commit", "-q", "-m", msg)
}

// addLaunchedChange commits a new To Do change in repo.
func addLaunchedChange(t *testing.T, repo, name string) {
	t.Helper()
	dir := filepath.Join(repo, "openspec", "changes", name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("- [ ] todo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	writeChangeMeta(t, repo, name, "schema: spec-driven\nlaunched: true\n")
	commitAll(t, repo, "add "+name)
}

// pendingChanges implements every change through a worker with the conformity
// verification enabled, leaving them all pending, and installs the verifier.
func pendingChanges(t *testing.T, mode DelegationMode, size int, v *verifierStub, names ...string) (*Manager, string, *conversation.Store) {
	t.Helper()
	repo := multiChangeRepo(t, names...)
	m, store, _ := newJournalManager(t, repo, mode, autoStub)
	m.prefs = newVerifyPrefs(t, true)
	m.activityStore = activity.NewStore(t.TempDir(), nil)
	for _, n := range names {
		runOnce(t, m, n)
	}
	m.config.Size = size
	prev := startSubprocessFn
	startSubprocessFn = v.start
	t.Cleanup(func() { startSubprocessFn = prev })
	return m, repo, store
}

func runningVerifications(m *Manager) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.verifications)
}

func TestTick_StartsQueuedVerifications(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, repo, _ := pendingChanges(t, ModeHITLReview, 1, v, "tk-a")

	m.tick()
	m.waitWorkers()

	if st := verifyStateOf(t, m, repo, "tk-a"); st != "passed" {
		t.Fatalf("marker = %q, want passed", st)
	}
}

func TestTick_VerificationConcurrencyLimitedByPoolSize(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: PASS", gate: make(chan struct{})}
	m, repo, _ := pendingChanges(t, ModeHITLReview, 2, v, "tk-1", "tk-2", "tk-3")

	m.tick()
	waitFor(t, "two verifications", func() bool { return v.startCount() == 2 })
	if n := runningVerifications(m); n != 2 {
		t.Fatalf("running = %d, want 2", n)
	}
	// Alphabetical order: the third is queued behind the first two.
	if m.VerificationRunning("tk-3") {
		t.Error("tk-3 should be queued")
	}
	for _, c := range listFor(t, repo) {
		if c.Name == "tk-3" && (c.KanbanStatus != "verifying" || c.VerificationState != "queued") {
			t.Errorf("tk-3 = %+v", c)
		}
	}
	m.tick()
	if n := runningVerifications(m); n != 2 {
		t.Errorf("a second tick must not exceed the limit, running = %d", n)
	}
	close(v.gate)
	m.waitWorkers()
}

func TestTick_VerificationIndependentOfWorkerSlots(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: PASS", gate: make(chan struct{})}
	m, _, _ := pendingChanges(t, ModeHITLReview, 1, v, "tk-ind")
	// Every worker slot is taken.
	m.activeWorkers[9] = &Worker{ID: 9, ActiveChange: "someone-else"}

	m.tick()
	waitFor(t, "verification started", func() bool { return v.startCount() == 1 })
	close(v.gate)
	m.waitWorkers()
}

func TestTick_WorkersIndependentOfVerifications(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: PASS", gate: make(chan struct{})}
	m, repo, _ := pendingChanges(t, ModeHITLReview, 1, v, "tk-busy")
	addLaunchedChange(t, repo, "tk-todo")

	m.tick()
	waitFor(t, "verification running", func() bool { return m.VerificationRunning("tk-busy") })
	m.mu.Lock()
	workers := len(m.activeWorkers)
	m.mu.Unlock()
	if workers != 1 {
		t.Fatalf("a running verification must not block a worker slot, workers = %d", workers)
	}
	close(v.gate)
	m.waitWorkers()
}

func TestTick_FailedVerificationIsNeverRerun(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, repo, _ := pendingChanges(t, ModeHITLReview, 1, v, "tk-failed")
	if err := NewWorktreeController(repo, "ws1", m.worktreesRoot).SetVerify("tk-failed", "failed"); err != nil {
		t.Fatal(err)
	}

	m.tick()
	m.tick()
	m.waitWorkers()

	if v.startCount() != 0 {
		t.Errorf("a failed verification must not restart, starts = %d", v.startCount())
	}
	if st := verifyStateOf(t, m, repo, "tk-failed"); st != "failed" {
		t.Errorf("marker = %q", st)
	}
}

func TestTick_PassedChangeIsFinalizedWithoutAgent_HitlReview(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, repo, store := pendingChanges(t, ModeHITLReview, 1, v, "tk-fin")

	m.tick() // verification
	m.waitWorkers()
	if v.startCount() != 1 {
		t.Fatalf("verifier starts = %d", v.startCount())
	}
	m.tick() // finalization
	m.waitWorkers()

	if v.startCount() != 1 {
		t.Errorf("the finalizing worker must start no agent, starts = %d", v.startCount())
	}
	if _, ok := NewWorktreeController(repo, "ws1", m.worktreesRoot).VerifyState("tk-fin"); ok {
		t.Error("the marker should be lifted")
	}
	if !hasConfigKey(t, repo, openspec.ReviewKey("tk-fin")) {
		t.Error("the change should reach To Review")
	}
	changes := listFor(t, repo)
	if changes[0].KanbanStatus != "to-review" {
		t.Errorf("status = %s", changes[0].KanbanStatus)
	}
	// No new verification after the finalization.
	m.tick()
	m.waitWorkers()
	if v.startCount() != 1 {
		t.Errorf("no new verification expected, starts = %d", v.startCount())
	}
	runs, _ := store.List("ws1", "tk-fin", "verify")
	if len(runs) != 1 {
		t.Errorf("verify runs = %d", len(runs))
	}
}

func TestTick_PassedChangeIsMergedInFullAutonomy(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, repo, _ := pendingChanges(t, ModeFullAutonomy, 1, v, "tk-merge")

	m.tick()
	m.waitWorkers()
	m.tick()
	m.waitWorkers()

	if v.startCount() != 1 {
		t.Errorf("starts = %d, want 1 (the verifier only)", v.startCount())
	}
	if branchExistsIn(t, repo, "feature/tk-merge") {
		t.Error("the change should be merged")
	}
	if _, err := os.Stat(filepath.Join(repo, "agent-output.txt")); err != nil {
		t.Errorf("the agent's work should be merged: %v", err)
	}
}

func TestTick_PassedChangeWaitsForAFreeWorkerSlot(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, repo, _ := pendingChanges(t, ModeHITLReview, 1, v, "tk-wait")
	m.tick()
	m.waitWorkers()

	m.activeWorkers[9] = &Worker{ID: 9, ActiveChange: "someone-else"}
	m.tick()
	m.waitWorkers()
	if st := verifyStateOf(t, m, repo, "tk-wait"); st != "passed" {
		t.Fatalf("marker = %q, want passed while no slot is free", st)
	}

	delete(m.activeWorkers, 9)
	m.tick()
	m.waitWorkers()
	if _, ok := NewWorktreeController(repo, "ws1", m.worktreesRoot).VerifyState("tk-wait"); ok {
		t.Error("marker should be lifted once a slot is free")
	}
}

func TestTick_PassedChangesRespectWorkerLimitAndHolds(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, repo, _ := pendingChanges(t, ModeHITLReview, 1, v, "tk-p1", "tk-p2")
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	_ = wt.SetVerify("tk-p1", "passed")
	_ = wt.SetVerify("tk-p2", "passed")

	// A worker already holds tk-p1: no second worker for it, and the slot is full.
	m.activeWorkers[9] = &Worker{ID: 9, ActiveChange: "tk-p1"}
	m.tick()
	m.waitWorkers()
	if st := verifyStateOf(t, m, repo, "tk-p2"); st != "passed" {
		t.Fatalf("tk-p2 must wait (pool size 1), marker = %q", st)
	}
	delete(m.activeWorkers, 9)

	m.tick() // one slot: only the first change (name order) is dispatched
	m.waitWorkers()
	if _, ok := wt.VerifyState("tk-p1"); ok {
		t.Error("tk-p1 should have been dispatched")
	}
	if st := verifyStateOf(t, m, repo, "tk-p2"); st != "passed" {
		t.Errorf("tk-p2 should still wait, marker = %q", st)
	}
}

func TestTick_RestartBetweenPassedAndFinalization(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, repo, _ := pendingChanges(t, ModeHITLReview, 1, v, "tk-restart")
	m.tick()
	m.waitWorkers()
	if st := verifyStateOf(t, m, repo, "tk-restart"); st != "passed" {
		t.Fatalf("marker = %q", st)
	}

	// A fresh manager (backend restart) picks the intention up from the marker.
	m2 := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1}, v.start)
	m2.workspaceID = "ws1"
	m2.prefs = m.prefs
	m2.tick()
	m2.waitWorkers()
	if !hasConfigKey(t, repo, openspec.ReviewKey("tk-restart")) {
		t.Error("the restarted pool should finalize the change")
	}
}

func TestEndToEnd_FailStopsWithoutSecondTurn(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: FAIL"}
	m, repo, _ := pendingChanges(t, ModeHITLReview, 1, v, "e2e-fail")

	m.tick()
	m.waitWorkers()
	m.tick()
	m.waitWorkers()

	if st := verifyStateOf(t, m, repo, "e2e-fail"); st != "failed" {
		t.Fatalf("marker = %q, want failed", st)
	}
	if v.startCount() != 1 || len(v.turns) != 1 {
		t.Errorf("no second turn expected: starts=%d turns=%d", v.startCount(), len(v.turns))
	}
	if hasConfigKey(t, repo, openspec.ReviewKey("e2e-fail")) {
		t.Error("a failed change must not reach To Review")
	}
}
