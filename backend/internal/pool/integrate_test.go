package pool

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/glefebvre/opensp8c/internal/preferences"
)

// setValidation configures the pool-wide validation command of m.
func setValidation(t *testing.T, m *Manager, command string) {
	t.Helper()
	svc := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	if err := svc.SetPoolDefaults(preferences.PoolPatch{ValidationCommand: preferences.StringPatch{Set: true, Value: command}}); err != nil {
		t.Fatal(err)
	}
	m.prefs = svc
}

func countRuns(t *testing.T, counter string) int {
	t.Helper()
	b, err := os.ReadFile(counter)
	if err != nil {
		return 0
	}
	return strings.Count(string(b), "r\n")
}

// userCommit makes the user commit a file on the repository's current branch,
// once, when the agent's first turn runs.
func userCommit(t *testing.T, repo, name string) func(string) {
	var once sync.Once
	return func(string) {
		once.Do(func() {
			writeFile(t, filepath.Join(repo, name), "user")
			gitIn(t, repo, "add", name)
			gitIn(t, repo, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "user "+name)
		})
	}
}

func chain(fns ...func(string)) func(string) {
	return func(ws string) {
		for _, f := range fns {
			f(ws)
		}
	}
}

func fileIn(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

func integrationFixture(t *testing.T, change string, mode DelegationMode, maxAttempts int, validation string, hook func(string)) (*Manager, string, string) {
	t.Helper()
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: mode, MaxAttempts: maxAttempts}, pipeAgent(hook, okResult, nil))
	counter := filepath.Join(t.TempDir(), "count")
	setValidation(t, m, fmt.Sprintf("echo r >> %s; %s", counter, validation))
	return m, repo, counter
}

func TestIntegrate_UnchangedTargetValidatesOnce(t *testing.T) {
	m, repo, counter := integrationFixture(t, "unchanged", ModeFullAutonomy, 1, "true", writeInWorktree("a.txt"))
	w := runOnce(t, m, "unchanged")
	if _, paused := pausedReason(m, w.ID); paused {
		t.Fatal("must not pause")
	}
	if n := countRuns(t, counter); n != 1 {
		t.Fatalf("validation runs = %d, want 1", n)
	}
	if !fileIn(repo, "a.txt") || branchExistsIn(t, repo, "feature/unchanged") {
		t.Fatal("expected a normal merge")
	}
}

func TestIntegrate_AdvancedTargetIsIntegratedAndRevalidated(t *testing.T) {
	change := "advanced"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1},
		pipeAgent(chain(writeInWorktree("a.txt"), userCommit(t, repo, "user.txt")), okResult, nil))
	counter := filepath.Join(t.TempDir(), "count")
	// Records whether the user's file was already in the validated tree.
	setValidation(t, m, fmt.Sprintf("echo r >> %s; if [ -f user.txt ]; then echo seen >> %s.seen; fi", counter, counter))

	w := runOnce(t, m, change)

	if _, paused := pausedReason(m, w.ID); paused {
		t.Fatal("must not pause")
	}
	if n := countRuns(t, counter); n != 2 {
		t.Fatalf("validation runs = %d, want 2", n)
	}
	if !fileIn(filepath.Dir(counter), "count.seen") {
		t.Fatal("the second validation must run on the integrated tree")
	}
	if !fileIn(repo, "a.txt") || !fileIn(repo, "user.txt") || branchExistsIn(t, repo, "feature/"+change) {
		t.Fatal("expected both files merged and the branch deleted")
	}
}

func TestIntegrate_FailureAfterIntegrationIsHealed(t *testing.T) {
	change := "heal-after"
	heal := func(ws string) {
		if fileIn(ws, "user.txt") {
			_ = os.WriteFile(filepath.Join(ws, "fixed.txt"), []byte("fix"), 0644)
		}
	}
	m, repo, counter := integrationFixture(t, change, ModeFullAutonomy, 2,
		"[ ! -f user.txt ] || [ -f fixed.txt ]", nil)
	repoUser := userCommit(t, repo, "user.txt")
	startSubprocessFn = pipeAgent(chain(writeInWorktree("a.txt"), repoUser, heal), okResult, nil)

	w := runOnce(t, m, change)

	if _, paused := pausedReason(m, w.ID); paused {
		t.Fatal("must not pause")
	}
	if n := countRuns(t, counter); n != 3 {
		t.Fatalf("validation runs = %d, want 3 (initial, failing integrated, healed)", n)
	}
	if !fileIn(repo, "fixed.txt") || !fileIn(repo, "user.txt") {
		t.Fatal("the correction must be committed and merged")
	}
}

func TestIntegrate_FailureAfterIntegrationExhaustsAttempts(t *testing.T) {
	change := "no-heal"
	m, repo, _ := integrationFixture(t, change, ModeFullAutonomy, 1, "[ ! -f user.txt ]", nil)
	startSubprocessFn = pipeAgent(chain(writeInWorktree("a.txt"), userCommit(t, repo, "user.txt")), okResult, nil)
	w := runOnce(t, m, change)

	reason, paused := pausedReason(m, w.ID)
	if !paused || !strings.Contains(reason, "réparation épuisées") {
		t.Fatalf("paused=%v reason=%q", paused, reason)
	}
	if fileIn(repo, "a.txt") {
		t.Fatal("nothing may be merged")
	}
	if !branchExistsIn(t, repo, "feature/"+change) {
		t.Fatal("branch must be kept")
	}
}

func TestIntegrate_ConflictPausesAndRestores(t *testing.T) {
	change := "clash"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	var once sync.Once
	hook := func(ws string) {
		_ = os.WriteFile(filepath.Join(ws, "main.go"), []byte("package main\n\n// branch\nfunc main() {}\n"), 0644)
		once.Do(func() {
			writeFile(t, filepath.Join(repo, "main.go"), "package main\n\n// main\nfunc main() {}\n")
			gitIn(t, repo, "-c", "commit.gpgsign=false", "commit", "-q", "-am", "main side")
		})
	}
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1}, pipeAgent(hook, okResult, nil))

	w := runOnce(t, m, change)

	reason, paused := pausedReason(m, w.ID)
	if !paused || !strings.Contains(reason, "Intégration") || !strings.Contains(reason, "conflit") {
		t.Fatalf("paused=%v reason=%q", paused, reason)
	}
	if _, ok := (&WorktreeController{}).runGitInCode(w.WorktreePath, "rev-parse", "-q", "--verify", "MERGE_HEAD"); ok {
		t.Fatal("integration must be aborted")
	}
	if gitIn(t, w.WorktreePath, "status", "--porcelain") != "" {
		t.Fatal("worktree must be clean and committed")
	}
	if !branchExistsIn(t, repo, "feature/"+change) {
		t.Fatal("branch must be kept")
	}
}

// A commit lands on the target during the revalidation: the lock is released,
// a new round integrates it, and the merge succeeds.
func TestIntegrate_TargetAdvancedDuringWindowStartsNewRound(t *testing.T) {
	change := "window"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1},
		pipeAgent(chain(writeInWorktree("a.txt"), userCommit(t, repo, "user.txt")), okResult, nil))
	counter := filepath.Join(t.TempDir(), "count")
	// Second validation (after the first integration) injects a late commit.
	setValidation(t, m, fmt.Sprintf(`echo r >> %[1]s; if [ "$(wc -l < %[1]s)" -eq 2 ]; then git -C %[2]s -c user.name=t -c user.email=t@t -c commit.gpgsign=false commit --allow-empty -q -m late; fi`, counter, repo))

	w := runOnce(t, m, change)

	if reason, paused := pausedReason(m, w.ID); paused {
		t.Fatalf("must not pause: %q", reason)
	}
	if n := countRuns(t, counter); n != 3 {
		t.Fatalf("validation runs = %d, want 3", n)
	}
	if !fileIn(repo, "a.txt") || branchExistsIn(t, repo, "feature/"+change) {
		t.Fatal("expected the merge to complete")
	}
	if !m.mergeMu.TryLock() {
		t.Fatal("merge lock must be free")
	}
	m.mergeMu.Unlock()
}

func TestIntegrate_TargetKeepsAdvancingPausesAfterThreeRounds(t *testing.T) {
	change := "restless"
	m, repo, counter := integrationFixture(t, change, ModeFullAutonomy, 1, "", nil)
	startSubprocessFn = pipeAgent(writeInWorktree("a.txt"), okResult, nil)
	setValidation(t, m, fmt.Sprintf("echo r >> %s; git -C %s -c user.name=t -c user.email=t@t -c commit.gpgsign=false commit --allow-empty -q -m late", counter, repo))

	w := runOnce(t, m, change)

	reason, paused := pausedReason(m, w.ID)
	if !paused || !strings.Contains(reason, "continué d'avancer") {
		t.Fatalf("paused=%v reason=%q", paused, reason)
	}
	if n := countRuns(t, counter); n != 1+maxIntegrationRounds {
		t.Fatalf("validation runs = %d, want %d", n, 1+maxIntegrationRounds)
	}
	if fileIn(repo, "a.txt") || !branchExistsIn(t, repo, "feature/"+change) {
		t.Fatal("nothing may be merged, branch kept")
	}
	if !m.mergeMu.TryLock() {
		t.Fatal("merge lock must be free")
	}
	m.mergeMu.Unlock()
}

func TestIntegrate_HITLReviewDoesNotIntegrate(t *testing.T) {
	change := "hitl"
	m, repo, counter := integrationFixture(t, change, ModeHITLReview, 1, "true", nil)
	startSubprocessFn = pipeAgent(chain(writeInWorktree("a.txt"), userCommit(t, repo, "user.txt")), okResult, nil)

	w := runOnce(t, m, change)

	if _, paused := pausedReason(m, w.ID); paused {
		t.Fatal("must not pause")
	}
	if n := countRuns(t, counter); n != 1 {
		t.Fatalf("validation runs = %d, want 1", n)
	}
	if fileIn(w.WorktreePath, "user.txt") || !branchExistsIn(t, repo, "feature/"+change) {
		t.Fatal("hitl-review must not integrate or merge")
	}
}

func switchBranch(t *testing.T, repo string, args ...string) func(string) {
	var once sync.Once
	return func(string) {
		once.Do(func() { gitIn(t, repo, args...) })
	}
}

func TestBase_OtherBranchPausesThenResumes(t *testing.T) {
	change := "base-moved"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	base := gitIn(t, repo, "rev-parse", "--abbrev-ref", "HEAD")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1},
		pipeAgent(chain(writeInWorktree("a.txt"), switchBranch(t, repo, "checkout", "-q", "-b", "feature/other")), okResult, nil))
	setValidation(t, m, "true")

	w := runOnce(t, m, change)

	reason, paused := pausedReason(m, w.ID)
	if !paused || !strings.Contains(reason, base) || !strings.Contains(reason, "feature/other") {
		t.Fatalf("paused=%v reason=%q", paused, reason)
	}
	if fileIn(repo, "a.txt") || !branchExistsIn(t, repo, "feature/"+change) {
		t.Fatal("nothing merged, branch kept")
	}
	if _, err := os.Stat(w.WorktreePath); err != nil {
		t.Fatalf("worktree must be kept: %v", err)
	}

	gitIn(t, repo, "checkout", "-q", base)
	runOnce(t, m, change)
	if !fileIn(repo, "a.txt") || branchExistsIn(t, repo, "feature/"+change) {
		t.Fatal("after returning to the base the worker must merge")
	}
}

func TestBase_DetachedHeadPauses(t *testing.T) {
	change := "detached"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1},
		pipeAgent(chain(writeInWorktree("a.txt"), switchBranch(t, repo, "checkout", "-q", "--detach")), okResult, nil))
	setValidation(t, m, "true")

	w := runOnce(t, m, change)

	reason, paused := pausedReason(m, w.ID)
	if !paused || !strings.Contains(reason, "aucune branche") {
		t.Fatalf("paused=%v reason=%q", paused, reason)
	}
	if fileIn(repo, "a.txt") {
		t.Fatal("nothing may be merged")
	}
}

func TestBase_BranchWithoutBaseIsNotConstrained(t *testing.T) {
	change := "legacy"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	hook := chain(writeInWorktree("a.txt"),
		switchBranch(t, repo, "config", "--unset", baseConfigKey(change)),
		switchBranch(t, repo, "checkout", "-q", "-b", "elsewhere"))
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1}, pipeAgent(hook, okResult, nil))
	setValidation(t, m, "true")

	w := runOnce(t, m, change)

	if reason, paused := pausedReason(m, w.ID); paused {
		t.Fatalf("must not pause: %q", reason)
	}
	if !fileIn(repo, "a.txt") {
		t.Fatal("the work must be merged into the current branch")
	}
}

// The second worker only starts working after the first one has merged: it
// integrates that result, revalidates and merges without pausing.
func TestIntegrate_SecondWorkerIntegratesFirstResult(t *testing.T) {
	repo := newGoFixtureRepo(t, "change-one", "- [x] done\n")
	writeFile(t, filepath.Join(repo, "openspec", "changes", "change-two", "tasks.md"), "- [x] done\n")
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "second change")

	var m *Manager
	var first sync.Once
	hook := func(ws string) {
		writeInWorktree(filepath.Base(ws) + ".txt")(ws)
		if filepath.Base(ws) != "wt-change-two" {
			return
		}
		first.Do(func() {
			w1 := &Worker{ID: 2, WorkspaceID: "ws1", ActiveChange: "change-one"}
			m.mu.Lock()
			m.activeWorkers[2] = w1
			m.mu.Unlock()
			m.runWorker(context.Background(), w1)
		})
	}
	m = newWorkerTestManager(t, repo, AgentPoolConfig{Size: 2, DelegationMode: ModeFullAutonomy, MaxAttempts: 1}, pipeAgent(hook, okResult, nil))
	counter := filepath.Join(t.TempDir(), "count")
	setValidation(t, m, fmt.Sprintf("echo r >> %s", counter))

	w := runOnce(t, m, "change-two")

	if reason, paused := pausedReason(m, w.ID); paused {
		t.Fatalf("must not pause: %q", reason)
	}
	if len(m.pausedWorkers) != 0 {
		t.Fatalf("no worker may pause: %+v", m.pausedWorkers)
	}
	if !fileIn(repo, "wt-change-one.txt") || !fileIn(repo, "wt-change-two.txt") {
		t.Fatal("both changes must be merged")
	}
	if n := countRuns(t, counter); n != 3 {
		t.Fatalf("validation runs = %d, want 3 (first worker once, second worker twice)", n)
	}
}

func TestIntegrateTargetReportsConflictFiles(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	path, _ := wc.Provision("two")
	commitFile(t, path, "b.txt", "branch b")
	commitFile(t, path, "a.txt", "branch a")
	commitFile(t, repo, "b.txt", "main b")
	commitFile(t, repo, "a.txt", "main a")

	files, err := wc.IntegrateTarget("two", "main")
	if err == nil {
		t.Fatal("expected a conflict")
	}
	if len(files) != 2 || files[0] != "a.txt" || files[1] != "b.txt" {
		t.Fatalf("files = %v, want [a.txt b.txt]", files)
	}
	if _, ok := wc.runGitInCode(path, "rev-parse", "-q", "--verify", "MERGE_HEAD"); ok {
		t.Fatal("MERGE_HEAD left behind")
	}
	if st := gitIn(t, path, "status", "--porcelain"); st != "" {
		t.Fatalf("worktree not clean: %q", st)
	}
}

func TestIntegrateTargetConflictWithFailingFileRead(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	wc.unmergedFiles = func(string) ([]string, error) { return nil, errors.New("boom") }
	path, _ := wc.Provision("blind")
	commitFile(t, path, "a.txt", "branch a")
	commitFile(t, repo, "a.txt", "main a")

	files, err := wc.IntegrateTarget("blind", "main")
	if err == nil {
		t.Fatal("expected a conflict")
	}
	if len(files) != 0 {
		t.Fatalf("files = %v, want none", files)
	}
	if _, ok := wc.runGitInCode(path, "rev-parse", "-q", "--verify", "MERGE_HEAD"); ok {
		t.Fatal("the merge must still be aborted")
	}
}
