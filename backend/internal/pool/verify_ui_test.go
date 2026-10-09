package pool

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
)

const (
	uiTestURL   = "http://127.0.0.1:{port}"
	humanMarker = " <!-- human review required -->"
)

const uiTestSpec = `## ADDED Requirements

### Requirement: Navigation
Texte.

#### Scenario: Ouvrir le menu
- **WHEN** l'utilisateur ouvre le menu
- **THEN** la liste s'affiche
`

// uiVerifyConfig is what a UI verification test configures.
type uiVerifyConfig struct {
	conformity bool
	command    string // uiStartCommand; "" = unset
	baseURL    string // uiBaseUrl; "" = unset
	tasks      string // tasks.md of the branch
	withSpec   bool
}

func uiPrefs(t *testing.T, c uiVerifyConfig) *preferences.Service {
	t.Helper()
	svc := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	patch := preferences.VerificationPatch{
		Conformity:     boolPatch(c.conformity),
		UI:             boolPatch(true),
		UIStartCommand: preferences.StringPatch{Set: true, Value: c.command},
		UIBaseURL:      preferences.StringPatch{Set: true, Value: c.baseURL},
	}
	if err := svc.SetVerificationDefaults(patch); err != nil {
		t.Fatal(err)
	}
	return svc
}

// awaitingUIVerification is awaitingVerification with the UI step configured
// by c and the branch rewritten (tasks, delta spec) and committed.
func awaitingUIVerification(t *testing.T, change string, c uiVerifyConfig, v *verifierStub) (*Manager, string, *conversation.Store) {
	t.Helper()
	m, repo, store := awaitingVerification(t, change, ModeHITLReview, v)
	m.prefs = uiPrefs(t, c)

	wt := NewWorktreeController(repo, "ws1", m.worktreesRoot)
	dir, err := wt.Provision(change)
	if err != nil {
		t.Fatal(err)
	}
	changeDir := filepath.Join(dir, "openspec", "changes", change)
	if c.tasks != "" {
		if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte(c.tasks), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if c.withSpec {
		specDir := filepath.Join(changeDir, "specs", "nav")
		if err := os.MkdirAll(specDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte(uiTestSpec), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := wt.CommitAll(change); err != nil {
		t.Fatal(err)
	}
	return m, repo, store
}

// branchTasksOf reads the tasks.md carried by the branch of change.
func branchTasksOf(t *testing.T, m *Manager, repo, change string) string {
	t.Helper()
	content, ok := NewWorktreeController(repo, "ws1", m.worktreesRoot).BranchTasks(change)
	if !ok {
		t.Fatal("no branch tasks")
	}
	return content
}

func gitLog(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "log", "--format=%s", "-n", "10")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func appCmd(extra string) string { return helperAppCommand(200, extra) }

func TestUIStep_Pass(t *testing.T) {
	shortUITimings(t)
	v := &verifierStub{answer: "Navigation OK.\nVERDICT: PASS"}
	m, repo, store := awaitingUIVerification(t, "ui-pass", uiVerifyConfig{
		command: appCmd(""), baseURL: uiTestURL, tasks: "- [x] done\n", withSpec: true,
	}, v)

	runVerifyNow(m, "ui-pass")

	if st := verifyStateOf(t, m, repo, "ui-pass"); st != "passed" {
		t.Fatalf("marker = %q, want passed", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "ui-pass"))
	if end["verdict"] != "pass" || end["step"] != "ui" || !strings.Contains(end["report"].(string), "Navigation OK.") {
		t.Errorf("end marker = %v", end)
	}
	if v.startCount() != 1 || len(v.turns) != 1 {
		t.Fatalf("starts=%d turns=%d", v.startCount(), len(v.turns))
	}
	for _, want := range []string{"http://127.0.0.1:", "Ouvrir le menu", "l'utilisateur ouvre le menu", "OPENSP8C_VERIFY_ARTIFACTS"} {
		if !strings.Contains(v.turns[0]+v.prompts[0], want) {
			t.Errorf("%q missing from the agent input", want)
		}
	}
	if strings.Contains(v.turns[0], ":{port}") {
		t.Errorf("the port must be substituted in the turn: %q", v.turns[0])
	}
	if !strings.Contains(v.prompts[0], "VERDICT: SKIP") || !strings.Contains(v.prompts[0], "NEVER create, modify") {
		t.Errorf("directive = %q", v.prompts[0])
	}
	dir := v.envs[0][artifactsEnv]
	if dir == "" || !strings.Contains(v.turns[0], dir) {
		t.Errorf("the evidence directory must be exported and in the turn: %q", dir)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("evidence directory should exist next to the journal: %v", err)
	}
	if !strings.Contains(v.cwds[0], "ui-pass") || v.cwds[0] == repo {
		t.Errorf("the verifier must run in the change worktree, got %q", v.cwds[0])
	}
	if m.VerificationRunning("ui-pass") {
		t.Error("verification should be finished")
	}
}

func TestUIStep_FailKeepsReport(t *testing.T) {
	shortUITimings(t)
	v := &verifierStub{answer: "Le menu ne s'ouvre pas.\nTASK-VERIFIED: 4.2 Parcours\nVERDICT: FAIL"}
	m, repo, store := awaitingUIVerification(t, "ui-fail", uiVerifyConfig{
		command: appCmd(""), baseURL: uiTestURL, withSpec: true,
		tasks: "- [x] 1 impl\n- [ ] 4.2 Parcours" + humanMarker + "\n",
	}, v)

	runVerifyNow(m, "ui-fail")

	if st := verifyStateOf(t, m, repo, "ui-fail"); st != "failed" {
		t.Fatalf("marker = %q", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "ui-fail"))
	if end["verdict"] != "fail" || !strings.Contains(end["report"].(string), "Le menu ne s'ouvre pas.") {
		t.Errorf("end marker = %v", end)
	}
	if strings.Contains(branchTasksOf(t, m, repo, "ui-fail"), "- [x] 4.2") {
		t.Error("no task may be ticked on FAIL")
	}
}

func TestUIStep_Skip(t *testing.T) {
	shortUITimings(t)
	t.Run("accepted", func(t *testing.T) {
		v := &verifierStub{answer: "Backend only.\nVERDICT: SKIP"}
		m, repo, store := awaitingUIVerification(t, "ui-skip", uiVerifyConfig{command: appCmd(""), baseURL: uiTestURL, tasks: "- [x] done\n"}, v)
		runVerifyNow(m, "ui-skip")
		if st := verifyStateOf(t, m, repo, "ui-skip"); st != "passed" {
			t.Fatalf("marker = %q", st)
		}
		if end := verifyEnd(t, loadVerifyRun(t, store, "ui-skip")); end["verdict"] != "pass" {
			t.Errorf("end = %v", end)
		}
		if !strings.Contains(v.turns[0], "VERDICT: SKIP") {
			t.Errorf("a change without scenario must invite to SKIP: %q", v.turns[0])
		}
	})
	t.Run("refused with a pending marked task", func(t *testing.T) {
		v := &verifierStub{answer: "VERDICT: SKIP"}
		m, repo, store := awaitingUIVerification(t, "ui-skip2", uiVerifyConfig{
			command: appCmd(""), baseURL: uiTestURL,
			tasks: "- [x] 1 impl\n- [ ] 4.2 Parcours" + humanMarker + "\n",
		}, v)
		runVerifyNow(m, "ui-skip2")
		if st := verifyStateOf(t, m, repo, "ui-skip2"); st != "failed" {
			t.Fatalf("marker = %q", st)
		}
		end := verifyEnd(t, loadVerifyRun(t, store, "ui-skip2"))
		if end["reason"] != "tâches de validation humaine non vérifiées" {
			t.Errorf("end = %v", end)
		}
	})
}

func TestUIStep_MissingVerdict(t *testing.T) {
	shortUITimings(t)
	v := &verifierStub{answer: "I clicked around."}
	m, repo, store := awaitingUIVerification(t, "ui-none", uiVerifyConfig{command: appCmd(""), baseURL: uiTestURL}, v)
	runVerifyNow(m, "ui-none")
	if st := verifyStateOf(t, m, repo, "ui-none"); st != "failed" {
		t.Fatalf("marker = %q", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "ui-none"))
	if end["reason"] != "verdict absent" || end["report"] != "I clicked around." {
		t.Errorf("end = %v", end)
	}
}

func TestUIStep_TrackedFileModifiedFailsUntrackedAllowed(t *testing.T) {
	shortUITimings(t)
	t.Run("tracked file modified", func(t *testing.T) {
		v := &verifierStub{answer: "VERDICT: PASS", before: func(ws string) {
			_ = os.WriteFile(filepath.Join(ws, "main.go"), []byte("package main\n// edited\n"), 0o644)
		}}
		m, repo, store := awaitingUIVerification(t, "ui-dirty", uiVerifyConfig{command: appCmd(""), baseURL: uiTestURL}, v)
		runVerifyNow(m, "ui-dirty")
		if st := verifyStateOf(t, m, repo, "ui-dirty"); st != "failed" {
			t.Fatalf("marker = %q", st)
		}
		end := verifyEnd(t, loadVerifyRun(t, store, "ui-dirty"))
		if !strings.Contains(end["reason"].(string), "le vérificateur a modifié le worktree") || !strings.Contains(end["reason"].(string), "main.go") {
			t.Errorf("reason = %v", end["reason"])
		}
		if b, _ := os.ReadFile(filepath.Join(v.cwds[0], "main.go")); !strings.Contains(string(b), "edited") {
			t.Error("the modified file must be left in place")
		}
	})
	t.Run("untracked file created", func(t *testing.T) {
		v := &verifierStub{answer: "VERDICT: PASS", before: func(ws string) {
			_ = os.WriteFile(filepath.Join(ws, "app.log"), []byte("log"), 0o644)
		}}
		m, repo, _ := awaitingUIVerification(t, "ui-untracked", uiVerifyConfig{command: appCmd(""), baseURL: uiTestURL}, v)
		runVerifyNow(m, "ui-untracked")
		if st := verifyStateOf(t, m, repo, "ui-untracked"); st != "passed" {
			t.Fatalf("marker = %q", st)
		}
	})
}

func TestUIStep_MissingParameters(t *testing.T) {
	for _, tc := range []struct {
		name, command, url, reason string
	}{
		{"command", "", uiTestURL, "uiStartCommand non configurée"},
		{"url", appCmd(""), "", "uiBaseUrl non configurée"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pidFile := filepath.Join(t.TempDir(), "app.pid")
			v := &verifierStub{answer: "VERDICT: PASS"}
			cmd := tc.command
			if cmd != "" {
				cmd = appCmd("echo $$ > " + pidFile + "; ")
			}
			m, repo, store := awaitingUIVerification(t, "ui-miss-"+tc.name, uiVerifyConfig{command: cmd, baseURL: tc.url}, v)
			runVerifyNow(m, "ui-miss-"+tc.name)
			if st := verifyStateOf(t, m, repo, "ui-miss-"+tc.name); st != "failed" {
				t.Fatalf("marker = %q", st)
			}
			end := verifyEnd(t, loadVerifyRun(t, store, "ui-miss-"+tc.name))
			if end["reason"] != tc.reason {
				t.Errorf("reason = %v", end["reason"])
			}
			if v.startCount() != 0 {
				t.Error("no agent may start")
			}
			if _, err := os.Stat(pidFile); err == nil {
				t.Error("no application process may start")
			}
		})
	}
}

func TestUIStep_AppThatDoesNotStartNeverRunsTheAgent(t *testing.T) {
	shortUITimings(t)
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, repo, store := awaitingUIVerification(t, "ui-down", uiVerifyConfig{command: "echo cannot start; exit 2", baseURL: uiTestURL}, v)
	runVerifyNow(m, "ui-down")
	if st := verifyStateOf(t, m, repo, "ui-down"); st != "failed" {
		t.Fatalf("marker = %q", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "ui-down"))
	if !strings.Contains(end["reason"].(string), "code 2") || !strings.Contains(end["reason"].(string), "cannot start") {
		t.Errorf("reason = %v", end["reason"])
	}
	if v.startCount() != 0 {
		t.Error("the agent must never start")
	}
}

func TestUIStep_CancelStopsAppAndReleasesLock(t *testing.T) {
	shortUITimings(t)
	pidFile := filepath.Join(t.TempDir(), "app.pid")
	v := &verifierStub{answer: "VERDICT: PASS", gate: make(chan struct{})}
	m, repo, _ := awaitingUIVerification(t, "ui-cancel", uiVerifyConfig{command: appCmd("echo $$ > " + pidFile + "; "), baseURL: uiTestURL}, v)

	m.mu.Lock()
	m.startVerification("ui-cancel")
	cancel := m.verifications["ui-cancel"].cancel
	m.mu.Unlock()

	deadline := time.Now().Add(10 * time.Second)
	for v.startCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if v.startCount() == 0 {
		t.Fatal("the agent never started")
	}
	cancel()
	m.waitWorkers()

	waitPidGone(t, pidFile)
	if st := verifyStateOf(t, m, repo, "ui-cancel"); st != "pending" {
		t.Errorf("a cancelled verification leaves the marker pending, got %q", st)
	}
	ctx, cancelCtx := context.WithTimeout(context.Background(), time.Second)
	defer cancelCtx()
	if err := uiLock.Acquire(ctx); err != nil {
		t.Fatalf("the UI lock must be released: %v", err)
	}
	uiLock.Release()
}

func TestUIStep_StepTimeout(t *testing.T) {
	shortUITimings(t)
	setVar(t, &uiStepTimeout, 300*time.Millisecond)
	pidFile := filepath.Join(t.TempDir(), "app.pid")
	v := &verifierStub{answer: "VERDICT: PASS", gate: make(chan struct{})}
	m, repo, store := awaitingUIVerification(t, "ui-slow", uiVerifyConfig{command: appCmd("echo $$ > " + pidFile + "; "), baseURL: uiTestURL}, v)
	runVerifyNow(m, "ui-slow")
	if st := verifyStateOf(t, m, repo, "ui-slow"); st != "failed" {
		t.Fatalf("marker = %q", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "ui-slow"))
	if !strings.Contains(end["reason"].(string), "délai maximal") {
		t.Errorf("reason = %v", end["reason"])
	}
	waitPidGone(t, pidFile)
}

func TestUIStep_VerifierRoleModel(t *testing.T) {
	shortUITimings(t)
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, repo, _ := awaitingUIVerification(t, "ui-role", uiVerifyConfig{command: appCmd(""), baseURL: uiTestURL}, v)
	installMockClaude(t)
	prefs := newRolePrefs(t, `{"roles":{"verifier":{"model":"haiku","effort":"low"}}}`)
	if err := prefs.SetVerificationDefaults(preferences.VerificationPatch{
		UI:             boolPatch(true),
		UIStartCommand: preferences.StringPatch{Set: true, Value: appCmd("")},
		UIBaseURL:      preferences.StringPatch{Set: true, Value: uiTestURL},
	}); err != nil {
		t.Fatal(err)
	}
	m.prefs = prefs
	m.sessionMgr = session.NewManager(prefs, nil)

	runVerifyNow(m, "ui-role")

	if st := verifyStateOf(t, m, repo, "ui-role"); st != "passed" {
		t.Fatalf("marker = %q", st)
	}
	if len(v.models) != 1 || v.models[0] != "haiku" {
		t.Errorf("the verifier role model must be passed, got %v", v.models)
	}
}

// ---- ticking of the verified human-review tasks ----

const threeHumanTasks = "- [x] 1 impl\n" +
	"- [ ] 4.1 Premier" + humanMarker + "\n" +
	"- [ ] 4.2 Second" + humanMarker + "\n" +
	"- [ ] 4.3 Troisième" + humanMarker + "\n"

func TestUIStep_TicksVerifiedTasks(t *testing.T) {
	shortUITimings(t)
	t.Run("one task", func(t *testing.T) {
		v := &verifierStub{answer: "TASK-VERIFIED: 4.2 Parcours\nVERDICT: PASS"}
		m, repo, store := awaitingUIVerification(t, "ui-tick1", uiVerifyConfig{
			command: appCmd(""), baseURL: uiTestURL,
			tasks: "- [x] 1 impl\n- [ ] 4.2 Parcours" + humanMarker + "\n",
		}, v)
		runVerifyNow(m, "ui-tick1")

		if st := verifyStateOf(t, m, repo, "ui-tick1"); st != "passed" {
			t.Fatalf("marker = %q", st)
		}
		got := branchTasksOf(t, m, repo, "ui-tick1")
		if !strings.Contains(got, "- [x] 4.2 Parcours"+humanMarker) {
			t.Errorf("the task must be ticked with its marker kept:\n%s", got)
		}
		wtDir := NewWorktreeController(repo, "ws1", m.worktreesRoot).resolvePath("ui-tick1")
		if log := gitLog(t, wtDir); !strings.Contains(log, "Validate task 2") {
			t.Errorf("a dedicated commit is expected:\n%s", log)
		}
		end := verifyEnd(t, loadVerifyRun(t, store, "ui-tick1"))
		if vf, _ := end["verified"].([]any); len(vf) != 1 || vf[0] != "4.2 Parcours" {
			t.Errorf("end = %v", end)
		}
	})
	t.Run("task reopened by a correction", func(t *testing.T) {
		// A human task ticked by the user, then reopened by a correction with
		// reopen_human_tasks, is verified and ticked like any marked task.
		reopened, n := openspec.ReopenHumanTasks("- [x] 1 impl\n- [x] 4.2 Parcours" + humanMarker + "\n")
		if n != 1 {
			t.Fatalf("reopened = %d", n)
		}
		v := &verifierStub{answer: "TASK-VERIFIED: 4.2 Parcours\nVERDICT: PASS"}
		m, repo, _ := awaitingUIVerification(t, "ui-tick-reo", uiVerifyConfig{command: appCmd(""), baseURL: uiTestURL, tasks: reopened}, v)
		runVerifyNow(m, "ui-tick-reo")
		if got := branchTasksOf(t, m, repo, "ui-tick-reo"); !strings.Contains(got, "- [x] 4.2 Parcours"+humanMarker) {
			t.Errorf("the reopened task must be ticked with its marker kept:\n%s", got)
		}
	})
	t.Run("two of three", func(t *testing.T) {
		v := &verifierStub{answer: "TASK-VERIFIED: 4.1 Premier\nTASK-VERIFIED: 4.3 Troisième\nVERDICT: PASS"}
		m, repo, _ := awaitingUIVerification(t, "ui-tick2", uiVerifyConfig{command: appCmd(""), baseURL: uiTestURL, tasks: threeHumanTasks}, v)
		runVerifyNow(m, "ui-tick2")
		got := branchTasksOf(t, m, repo, "ui-tick2")
		if !strings.Contains(got, "- [x] 4.1 Premier") || !strings.Contains(got, "- [x] 4.3 Troisième") || !strings.Contains(got, "- [ ] 4.2 Second") {
			t.Errorf("two ticks expected, 4.2 left:\n%s", got)
		}
		wtDir := NewWorktreeController(repo, "ws1", m.worktreesRoot).resolvePath("ui-tick2")
		log := gitLog(t, wtDir)
		if strings.Count(log, "Validate task") != 2 {
			t.Errorf("one commit per tick expected:\n%s", log)
		}
	})
	t.Run("never ticked on FAIL or SKIP", func(t *testing.T) {
		for _, verdict := range []string{"FAIL", "SKIP"} {
			v := &verifierStub{answer: "TASK-VERIFIED: 4.1 Premier\nVERDICT: " + verdict}
			change := "ui-notick-" + strings.ToLower(verdict)
			m, repo, _ := awaitingUIVerification(t, change, uiVerifyConfig{command: appCmd(""), baseURL: uiTestURL, tasks: threeHumanTasks}, v)
			runVerifyNow(m, change)
			if strings.Contains(branchTasksOf(t, m, repo, change), "- [x] 4.1") {
				t.Errorf("%s: no task may be ticked", verdict)
			}
		}
	})
	t.Run("unknown and unmarked lines are ignored and reported", func(t *testing.T) {
		v := &verifierStub{answer: "TASK-VERIFIED: tâche inventée\nTASK-VERIFIED: 1 impl\nTASK-VERIFIED: 4.1 Premier\nVERDICT: PASS"}
		m, repo, store := awaitingUIVerification(t, "ui-ignored", uiVerifyConfig{
			command: appCmd(""), baseURL: uiTestURL,
			tasks: "- [ ] 1 impl\n- [ ] 4.1 Premier" + humanMarker + "\n",
		}, v)
		runVerifyNow(m, "ui-ignored")
		got := branchTasksOf(t, m, repo, "ui-ignored")
		if !strings.Contains(got, "- [ ] 1 impl") || !strings.Contains(got, "- [x] 4.1 Premier") {
			t.Errorf("only the marked task is ticked:\n%s", got)
		}
		end := verifyEnd(t, loadVerifyRun(t, store, "ui-ignored"))
		ig, _ := end["ignored"].([]any)
		if len(ig) != 2 || ig[0] != "tâche inventée" || ig[1] != "1 impl" {
			t.Errorf("ignored = %v", end["ignored"])
		}
	})
	t.Run("a failing commit restores the file and keeps the earlier ticks", func(t *testing.T) {
		v := &verifierStub{answer: "TASK-VERIFIED: 4.1 Premier\nTASK-VERIFIED: 4.2 Second\nVERDICT: PASS"}
		m, repo, store := awaitingUIVerification(t, "ui-commitfail", uiVerifyConfig{command: appCmd(""), baseURL: uiTestURL, tasks: threeHumanTasks}, v)
		// The second commit is refused by a hook.
		counter := filepath.Join(t.TempDir(), "count")
		hook := "#!/bin/sh\nn=$(cat " + counter + " 2>/dev/null || echo 0)\nn=$((n+1))\necho $n > " + counter + "\n[ $n -le 1 ]\n"
		hookPath := filepath.Join(repo, ".git", "hooks", "pre-commit")
		if err := os.WriteFile(hookPath, []byte(hook), 0o755); err != nil {
			t.Fatal(err)
		}
		runVerifyNow(m, "ui-commitfail")

		if st := verifyStateOf(t, m, repo, "ui-commitfail"); st != "failed" {
			t.Fatalf("marker = %q", st)
		}
		got := branchTasksOf(t, m, repo, "ui-commitfail")
		if !strings.Contains(got, "- [x] 4.1 Premier") || !strings.Contains(got, "- [ ] 4.2 Second") {
			t.Errorf("4.1 kept, 4.2 restored:\n%s", got)
		}
		end := verifyEnd(t, loadVerifyRun(t, store, "ui-commitfail"))
		if !strings.Contains(end["reason"].(string), "commit de la coche") {
			t.Errorf("reason = %v", end["reason"])
		}
	})
}

// ---- UI lock, waiting state and per-step runs ----

func TestUILock_WaitingDoesNotUsePoolSlots(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, _, _ := awaitingVerification(t, "slot-c3", ModeHITLReview, v)
	m.config.Size = 1
	// A runs its UI step, B waits for the UI lock; C is ready for conformity.
	defer SeedVerificationForTest(m, "slot-a", stepUI)()
	defer SeedWaitingVerificationForTest(m, "slot-b")()

	m.mu.Lock()
	m.tickVerifications([]openspec.Change{{Name: "slot-c3", KanbanStatus: "verifying", VerificationState: "queued"}}, map[string]bool{})
	started := m.verifications["slot-c3"] != nil
	m.mu.Unlock()
	m.waitWorkers()
	if !started || v.startCount() != 1 {
		t.Fatalf("conformity must start without waiting for the UI step (started=%v, agent starts=%d)", started, v.startCount())
	}
}

func TestUILock_ConformityStillBoundedByPoolSize(t *testing.T) {
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, _, _ := awaitingVerification(t, "slot-d", ModeHITLReview, v)
	m.config.Size = 1
	defer SeedVerificationForTest(m, "slot-run", stepConformity)()

	m.mu.Lock()
	m.tickVerifications([]openspec.Change{{Name: "slot-d", KanbanStatus: "verifying", VerificationState: "queued"}}, map[string]bool{})
	started := m.verifications["slot-d"] != nil
	m.mu.Unlock()
	if started {
		t.Fatal("a running conformity step still occupies the only slot")
	}
}

func TestUILock_PoolStopWhileWaiting(t *testing.T) {
	shortUITimings(t)
	pidFile := filepath.Join(t.TempDir(), "app.pid")
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, repo, _ := awaitingUIVerification(t, "wait-stop", uiVerifyConfig{command: appCmd("echo $$ > " + pidFile + "; "), baseURL: uiTestURL}, v)

	if err := uiLock.Acquire(context.Background()); err != nil { // another change holds the lock
		t.Fatal(err)
	}
	m.mu.Lock()
	m.startVerification("wait-stop")
	cancel := m.verifications["wait-stop"].cancel
	m.mu.Unlock()

	deadline := time.Now().Add(5 * time.Second)
	for !m.VerificationWaiting("wait-stop") && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !m.VerificationWaiting("wait-stop") {
		t.Fatal("the verification should wait for the UI lock")
	}
	if m.VerificationStep("wait-stop") != stepUI {
		t.Errorf("step = %q", m.VerificationStep("wait-stop"))
	}

	cancel()
	m.waitWorkers()
	uiLock.Release()

	if m.VerificationRunning("wait-stop") {
		t.Error("the entry must leave the in-flight verifications")
	}
	if st := verifyStateOf(t, m, repo, "wait-stop"); st != "pending" {
		t.Errorf("marker = %q, want pending", st)
	}
	if v.startCount() != 0 {
		t.Error("nothing may run while waiting")
	}
	// The lock is free again, nobody queues on it.
	ctx, cancelCtx := context.WithTimeout(context.Background(), time.Second)
	defer cancelCtx()
	if err := uiLock.Acquire(ctx); err != nil {
		t.Fatalf("lock lost: %v", err)
	}
	uiLock.Release()
}

func verifyRunEnds(t *testing.T, store *conversation.Store, change string) []map[string]any {
	t.Helper()
	runs, err := store.List("ws1", change, "verify")
	if err != nil {
		t.Fatal(err)
	}
	var ends []map[string]any
	for i := len(runs) - 1; i >= 0; i-- { // oldest first
		raw, err := store.Load("ws1", change, "verify", runs[i].Ts)
		if err != nil {
			t.Fatal(err)
		}
		var lines []journalLine
		for _, r := range raw {
			var l journalLine
			_ = json.Unmarshal(r, &l)
			lines = append(lines, l)
		}
		ends = append(ends, verifyEnd(t, lines))
	}
	return ends
}

func TestVerification_OneRunPerStep(t *testing.T) {
	shortUITimings(t)
	t.Run("conformity then failing UI", func(t *testing.T) {
		v := &verifierStub{answer: "VERDICT: PASS"}
		m, repo, store := awaitingUIVerification(t, "steps-fail", uiVerifyConfig{conformity: true, command: "", baseURL: uiTestURL}, v)
		runVerifyNow(m, "steps-fail")
		if st := verifyStateOf(t, m, repo, "steps-fail"); st != "failed" {
			t.Fatalf("marker = %q", st)
		}
		ends := verifyRunEnds(t, store, "steps-fail")
		if len(ends) != 2 || ends[0]["step"] != "conformity" || ends[0]["verdict"] != "pass" || ends[1]["step"] != "ui" || ends[1]["verdict"] != "error" {
			t.Fatalf("ends = %v", ends)
		}
	})
	t.Run("both pass", func(t *testing.T) {
		v := &verifierStub{answer: "VERDICT: PASS"}
		m, repo, store := awaitingUIVerification(t, "steps-pass", uiVerifyConfig{conformity: true, command: appCmd(""), baseURL: uiTestURL}, v)
		runVerifyNow(m, "steps-pass")
		if st := verifyStateOf(t, m, repo, "steps-pass"); st != "passed" {
			t.Fatalf("marker = %q", st)
		}
		ends := verifyRunEnds(t, store, "steps-pass")
		if len(ends) != 2 || ends[0]["step"] != "conformity" || ends[1]["step"] != "ui" {
			t.Fatalf("ends = %v", ends)
		}
		runs, _ := store.List("ws1", "steps-pass", "verify")
		if len(runs) != 2 {
			t.Errorf("one listed run per step expected, got %d", len(runs))
		}
	})
	t.Run("UI disabled skips the step", func(t *testing.T) {
		v := &verifierStub{answer: "VERDICT: PASS"}
		m, repo, store := awaitingUIVerification(t, "steps-noui", uiVerifyConfig{conformity: true, command: appCmd(""), baseURL: uiTestURL}, v)
		m.prefs = newVerifyPrefs(t, true) // conformity only
		runVerifyNow(m, "steps-noui")
		if st := verifyStateOf(t, m, repo, "steps-noui"); st != "passed" {
			t.Fatalf("marker = %q", st)
		}
		if ends := verifyRunEnds(t, store, "steps-noui"); len(ends) != 1 || ends[0]["step"] != "conformity" {
			t.Fatalf("ends = %v", ends)
		}
	})
	t.Run("a failing conformity stops before the UI", func(t *testing.T) {
		v := &verifierStub{answer: "VERDICT: FAIL"}
		m, repo, store := awaitingUIVerification(t, "steps-cfail", uiVerifyConfig{conformity: true, command: appCmd(""), baseURL: uiTestURL}, v)
		runVerifyNow(m, "steps-cfail")
		if st := verifyStateOf(t, m, repo, "steps-cfail"); st != "failed" {
			t.Fatalf("marker = %q", st)
		}
		if ends := verifyRunEnds(t, store, "steps-cfail"); len(ends) != 1 || v.startCount() != 1 {
			t.Fatalf("ends = %v, agent starts = %d", ends, v.startCount())
		}
	})
}
