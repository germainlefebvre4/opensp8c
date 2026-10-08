package pool

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/preferences"
)

func hasConfigKey(t *testing.T, repo, key string) bool {
	t.Helper()
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Dir = repo
	return cmd.Run() == nil
}

func TestRunWorker_EntersVerificationInHitlReview(t *testing.T) {
	change := "verify-hitl"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m, store, bc := newJournalManager(t, repo, ModeHITLReview, autoStub)
	m.prefs = newVerifyPrefs(t, true)

	w := runOnce(t, m, change)

	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	if st, ok := wt.VerifyState(change); !ok || st != "pending" {
		t.Fatalf("verify marker = %q %v, want pending", st, ok)
	}
	if hasConfigKey(t, repo, openspec.ReviewKey(change)) {
		t.Error("no review marker expected")
	}
	if !branchExistsIn(t, repo, "feature/"+change) {
		t.Error("branch must be kept")
	}
	if _, err := os.Stat(w.WorktreePath); err != nil {
		t.Errorf("worktree must be kept: %v", err)
	}
	m.mu.Lock()
	_, active := m.activeWorkers[w.ID]
	_, paused := m.pausedWorkers[w.ID]
	m.mu.Unlock()
	if active || paused {
		t.Errorf("slot should be freed without a pause (active=%v paused=%v)", active, paused)
	}
	_, lines := loadSingleRun(t, store, "ws1", change)
	assertEndMarker(t, lines, OutcomeAwaitingVerification)
	var published bool
	for _, e := range bc.snapshot() {
		if e.ev.Type == "change_updated" && e.ev.Name == change {
			published = true
		}
	}
	if !published {
		t.Error("change_updated should be published")
	}
	changes, _ := openspec.ListChanges(repo)
	if changes[0].KanbanStatus != "verifying" || changes[0].VerificationState != "queued" {
		t.Errorf("change = %+v", changes[0])
	}
}

func TestRunWorker_EntersVerificationInFullAutonomyWithoutMerging(t *testing.T) {
	change := "verify-auto"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m, _, _ := newJournalManager(t, repo, ModeFullAutonomy, autoStub)
	m.prefs = newVerifyPrefs(t, true)

	runOnce(t, m, change)

	if _, err := os.Stat(repo + "/agent-output.txt"); err == nil {
		t.Error("nothing must be merged into the current branch")
	}
	if !branchExistsIn(t, repo, "feature/"+change) {
		t.Error("branch must be kept")
	}
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	if st, _ := wt.VerifyState(change); st != "pending" {
		t.Errorf("marker = %q", st)
	}
}

func TestRunWorker_NoVerificationStepKeepsPreviousBehavior(t *testing.T) {
	t.Run("hitl-review", func(t *testing.T) {
		change := "no-verify-hitl"
		repo := newGoFixtureRepo(t, change, "- [x] done\n")
		m, store, _ := newJournalManager(t, repo, ModeHITLReview, autoStub)
		m.prefs = newVerifyPrefs(t, false)
		runOnce(t, m, change)
		wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
		if _, ok := wt.VerifyState(change); ok {
			t.Error("no verification marker expected")
		}
		if !hasConfigKey(t, repo, openspec.ReviewKey(change)) {
			t.Error("review marker expected")
		}
		_, lines := loadSingleRun(t, store, "ws1", change)
		assertEndMarker(t, lines, OutcomeAwaitingReview)
	})
	t.Run("full-autonomy", func(t *testing.T) {
		change := "no-verify-auto"
		repo := newGoFixtureRepo(t, change, "- [x] done\n")
		m, store, _ := newJournalManager(t, repo, ModeFullAutonomy, autoStub)
		runOnce(t, m, change)
		if branchExistsIn(t, repo, "feature/"+change) {
			t.Error("the change should be merged")
		}
		_, lines := loadSingleRun(t, store, "ws1", change)
		assertEndMarker(t, lines, OutcomeCompleted)
	})
}

func TestRunWorker_FinalizeOnlyNeverEntersVerification(t *testing.T) {
	change := "verify-finalize-only"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m, _, _ := newJournalManager(t, repo, ModeHITLReview, autoStub)
	m.prefs = newVerifyPrefs(t, true)

	runOnce(t, m, change) // implementation -> pending
	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	if err := wt.ClearVerify(change); err != nil {
		t.Fatal(err)
	}

	w := &Worker{ID: 2, WorkspaceID: "ws1", ActiveChange: change, finalizeOnly: true}
	m.activeWorkers[2] = w
	m.runWorker(context.Background(), w)

	if _, ok := wt.VerifyState(change); ok {
		t.Error("a finalizing worker must not set a verification marker")
	}
	if !hasConfigKey(t, repo, openspec.ReviewKey(change)) {
		t.Error("the change should reach To Review")
	}
}

func TestRunWorker_VerificationEnabledDuringImplementation(t *testing.T) {
	change := "verify-late"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	prefs := newVerifyPrefs(t, false)
	stub := func(ws string) {
		_ = prefs.SetVerificationDefaults(preferences.VerificationPatch{Conformity: boolPatch(true)})
		writeInWorktree("agent-output.txt")(ws)
	}
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1}, pipeAgent(stub, okResult, nil))
	m.prefs = prefs

	runOnce(t, m, change)

	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	if st, _ := wt.VerifyState(change); st != "pending" {
		t.Errorf("marker = %q, want pending (value resolved at the decision)", st)
	}
}
