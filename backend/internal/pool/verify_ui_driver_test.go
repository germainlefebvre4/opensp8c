package pool

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
)

// driverSetup is awaitingUIVerification with a driver configured and the
// verifier resolved to the (mock) Claude agent.
func driverSetup(t *testing.T, change, driverPatch, tasks string, v *verifierStub, appExtra string) (*Manager, string, *conversation.Store) {
	t.Helper()
	m, repo, store := awaitingUIVerification(t, change, uiVerifyConfig{
		command: appCmd(appExtra), baseURL: uiTestURL, withSpec: true, tasks: tasks,
	}, v)
	installMockClaude(t)
	m.sessionMgr = session.NewManager(m.prefs, nil)
	if driverPatch != "" {
		if err := m.prefs.SetVerificationDefaults(vpatchJSON(t, driverPatch)); err != nil {
			t.Fatal(err)
		}
	}
	return m, repo, store
}

func vpatchJSON(t *testing.T, j string) preferences.VerificationPatch {
	t.Helper()
	var p preferences.VerificationPatch
	if err := json.Unmarshal([]byte(j), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func streamLines(t *testing.T, files ...string) []string {
	t.Helper()
	var out []string
	for _, f := range files {
		for _, l := range testdataLines(t, f) {
			out = append(out, string(l))
		}
	}
	return out
}

func flagValue(args []string, flag string) string {
	if i := indexOf(args, flag); i >= 0 && i+1 < len(args) {
		return args[i+1]
	}
	return ""
}

func TestUIStep_AutoUnchanged(t *testing.T) {
	shortUITimings(t)
	v := &verifierStub{answer: "OK.\nVERDICT: PASS"}
	m, repo, store := driverSetup(t, "drv-auto", "", "- [x] done\n", v, "")
	runVerifyNow(m, "drv-auto")

	if st := verifyStateOf(t, m, repo, "drv-auto"); st != "passed" {
		t.Fatalf("marker = %q", st)
	}
	args := v.cfgs[0].ExtraArgs
	if len(args) != 2 || args[0] != "--permission-prompts" || args[1] != "none" {
		t.Errorf("auto passes only --permission-prompts none, got %v", args)
	}
	if !strings.Contains(v.prompts[0], "Choose the way you drive the browser") {
		t.Errorf("auto lets the agent choose: %q", v.prompts[0])
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "drv-auto"))
	if end["driver"] != "auto" || end["allowed_tools"] != nil {
		t.Errorf("end = %v", end)
	}
}

func TestUIStep_PlaywrightPassWithToolUse(t *testing.T) {
	shortUITimings(t)
	withFakeNpx(t)
	v := &verifierStub{
		answer: "Navigation OK.\nTASK-VERIFIED: 4.2 Parcours\nVERDICT: PASS",
		events: streamLines(t, "claude_init_playwright_connected.jsonl", "claude_tool_use_playwright.jsonl"),
	}
	existedDuringRun := false
	v.before = func(string) {
		v.mu.Lock()
		defer v.mu.Unlock()
		_, err := os.Stat(flagValue(v.cfgs[0].ExtraArgs, "--mcp-config"))
		existedDuringRun = err == nil
	}
	m, repo, store := driverSetup(t, "drv-pw", `{"uiDriver":"playwright","uiGuidance":"Viser le desktop"}`,
		"- [x] 1 impl\n- [ ] 4.2 Parcours"+humanMarker+"\n", v, "")
	runVerifyNow(m, "drv-pw")

	if st := verifyStateOf(t, m, repo, "drv-pw"); st != "passed" {
		t.Fatalf("marker = %q", st)
	}
	if !strings.Contains(branchTasksOf(t, m, repo, "drv-pw"), "- [x] 4.2") {
		t.Error("the verified task must be ticked")
	}
	args := v.cfgs[0].ExtraArgs
	cfgPath := flagValue(args, "--mcp-config")
	if cfgPath == "" || indexOf(args, "--strict-mcp-config") < 0 || indexOf(args, "--allowedTools") < 0 {
		t.Fatalf("args = %v", args)
	}
	if !existedDuringRun {
		t.Error("the MCP file must exist while the agent runs")
	}
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Errorf("the MCP file must be removed after the step: %v", err)
	}
	if strings.HasPrefix(cfgPath, v.cwds[0]) {
		t.Errorf("the MCP file must live outside the worktree: %s", cfgPath)
	}
	if !strings.Contains(v.prompts[0], "Drive the browser only with the tools of Playwright") {
		t.Errorf("directive: %q", v.prompts[0])
	}
	if !strings.Contains(v.turns[0], "Viser le desktop") {
		t.Errorf("guidance missing from the turn: %q", v.turns[0])
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "drv-pw"))
	tools, _ := end["allowed_tools"].([]any)
	if end["driver"] != "playwright" || len(tools) != 4 || tools[0] != "mcp__playwright" {
		t.Errorf("end = %v", end)
	}
}

func TestUIStep_PassWithoutDriverUseRefused(t *testing.T) {
	shortUITimings(t)
	withFakeNpx(t)
	v := &verifierStub{
		answer: "Je pense que tout va bien.\nTASK-VERIFIED: 4.2 Parcours\nVERDICT: PASS",
		events: streamLines(t, "claude_init_playwright_connected.jsonl"),
	}
	m, repo, store := driverSetup(t, "drv-nouse", `{"uiDriver":"playwright"}`,
		"- [x] 1 impl\n- [ ] 4.2 Parcours"+humanMarker+"\n", v, "")
	runVerifyNow(m, "drv-nouse")

	if st := verifyStateOf(t, m, repo, "drv-nouse"); st != "failed" {
		t.Fatalf("marker = %q", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "drv-nouse"))
	if end["reason"] != "aucun outil du pilote n'a été utilisé" || !strings.Contains(end["report"].(string), "Je pense que tout va bien.") {
		t.Errorf("end = %v", end)
	}
	if strings.Contains(branchTasksOf(t, m, repo, "drv-nouse"), "- [x] 4.2") {
		t.Error("no task may be ticked")
	}
}

func TestUIStep_SkipWithoutDriverUseAccepted(t *testing.T) {
	shortUITimings(t)
	withFakeNpx(t)
	v := &verifierStub{answer: "Rien d'observable.\nVERDICT: SKIP", events: streamLines(t, "claude_init_playwright_connected.jsonl")}
	m, repo, _ := driverSetup(t, "drv-skip", `{"uiDriver":"playwright"}`, "- [x] done\n", v, "")
	runVerifyNow(m, "drv-skip")
	if st := verifyStateOf(t, m, repo, "drv-skip"); st != "passed" {
		t.Fatalf("a SKIP needs no driver use, marker = %q", st)
	}
}

func TestUIStep_ServerNotConnectedStopsAgent(t *testing.T) {
	shortUITimings(t)
	withFakeNpx(t)
	v := &verifierStub{
		answer: "VERDICT: PASS", // never read: the step stops at the init event
		events: streamLines(t, "claude_init_playwright_failed.jsonl"),
	}
	m, repo, store := driverSetup(t, "drv-down", `{"uiDriver":"playwright"}`,
		"- [x] 1 impl\n- [ ] 4.2 Parcours"+humanMarker+"\n", v, "")
	runVerifyNow(m, "drv-down")

	if st := verifyStateOf(t, m, repo, "drv-down"); st != "failed" {
		t.Fatalf("marker = %q", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "drv-down"))
	if !strings.Contains(end["reason"].(string), "failed") || !strings.Contains(end["reason"].(string), "playwright") {
		t.Errorf("reason = %v", end["reason"])
	}
	if strings.Contains(branchTasksOf(t, m, repo, "drv-down"), "- [x] 4.2") {
		t.Error("no task may be ticked")
	}
	if p := flagValue(v.cfgs[0].ExtraArgs, "--mcp-config"); p != "" {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("the MCP file must be removed after a failure: %v", err)
		}
	}
}

func TestUIStep_PrecheckFailureStartsNothing(t *testing.T) {
	shortUITimings(t)
	cases := []struct {
		name, patch, want string
	}{
		{"custom without tools", `{"uiDriver":"custom","uiMcpConfig":"/nonexistent/mcp.json"}`, "/nonexistent/mcp.json"},
		{"custom without config", `{"uiDriver":"custom","uiAllowedTools":["mcp__x"]}`, "uiMcpConfig non configuré"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "app.started")
			v := &verifierStub{answer: "VERDICT: PASS"}
			change := "drv-pre-" + strings.ReplaceAll(c.name, " ", "-")
			m, repo, store := driverSetup(t, change, c.patch, "- [x] done\n", v, "touch "+marker+"; ")
			runVerifyNow(m, change)

			if st := verifyStateOf(t, m, repo, change); st != "failed" {
				t.Fatalf("marker = %q", st)
			}
			end := verifyEnd(t, loadVerifyRun(t, store, change))
			if !strings.Contains(end["reason"].(string), c.want) || end["driver"] != "custom" {
				t.Errorf("end = %v", end)
			}
			if v.startCount() != 0 {
				t.Error("the agent must not start")
			}
			if _, err := os.Stat(marker); err == nil {
				t.Error("the application must not start")
			}
		})
	}
}

func TestUIStep_NpxMissingStartsNothing(t *testing.T) {
	shortUITimings(t)
	marker := filepath.Join(t.TempDir(), "app.started")
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, repo, store := driverSetup(t, "drv-nonpx", `{"uiDriver":"playwright"}`, "- [x] done\n", v, "touch "+marker+"; ")
	// An empty directory would also hide git; keep the PATH minus any npx.
	dir := t.TempDir()
	for _, tool := range []string{"git", "sh", "touch"} {
		if p, err := exec.LookPath(tool); err == nil {
			_ = os.Symlink(p, filepath.Join(dir, tool))
		}
	}
	t.Setenv("PATH", dir)
	runVerifyNow(m, "drv-nonpx")

	if st := verifyStateOf(t, m, repo, "drv-nonpx"); st != "failed" {
		t.Fatalf("marker = %q", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "drv-nonpx"))
	if !strings.Contains(end["reason"].(string), "npx") {
		t.Errorf("reason = %v", end["reason"])
	}
	if _, err := os.Stat(marker); err == nil || v.startCount() != 0 {
		t.Error("neither the application nor the agent may start")
	}
}

func TestUIStep_ChromeFromWorkspace(t *testing.T) {
	shortUITimings(t)
	chromeUse := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"mcp__claude-in-chrome__navigate","input":{}}]}}`
	v := &verifierStub{
		answer: "OK.\nVERDICT: PASS",
		events: append(streamLines(t, "claude_init_chrome.jsonl"), chromeUse),
	}
	m, repo, store := driverSetup(t, "drv-chrome", "", "- [x] done\n", v, "")
	if err := m.prefs.PatchWorkspace("ws1", preferences.WorkspaceSettingsPatch{Verification: ptrTo(vpatchJSON(t, `{"uiDriver":"chrome"}`))}); err != nil {
		t.Fatal(err)
	}
	runVerifyNow(m, "drv-chrome")

	if st := verifyStateOf(t, m, repo, "drv-chrome"); st != "passed" {
		t.Fatalf("marker = %q", st)
	}
	args := v.cfgs[0].ExtraArgs
	if indexOf(args, "--chrome") < 0 || indexOf(args, "--mcp-config") >= 0 {
		t.Errorf("args = %v", args)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "drv-chrome"))
	if end["driver"] != "chrome" {
		t.Errorf("end = %v", end)
	}
	if !strings.Contains(v.prompts[0], "your own tabs") {
		t.Errorf("directive: %q", v.prompts[0])
	}
}

func TestUIStep_ChromeUnavailableStopsAgent(t *testing.T) {
	shortUITimings(t)
	v := &verifierStub{answer: "VERDICT: PASS", events: streamLines(t, "claude_init_no_mcp.jsonl")}
	m, repo, store := driverSetup(t, "drv-chrome-off", "", "- [x] done\n", v, "")
	if err := m.prefs.PatchWorkspace("ws1", preferences.WorkspaceSettingsPatch{Verification: ptrTo(vpatchJSON(t, `{"uiDriver":"chrome"}`))}); err != nil {
		t.Fatal(err)
	}
	runVerifyNow(m, "drv-chrome-off")
	if st := verifyStateOf(t, m, repo, "drv-chrome-off"); st != "failed" {
		t.Fatalf("marker = %q", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "drv-chrome-off"))
	if !strings.Contains(end["reason"].(string), "Chrome est ouvert avec l'extension") {
		t.Errorf("reason = %v", end["reason"])
	}
}

func TestUIStep_DriverRefusedForOtherAgent(t *testing.T) {
	shortUITimings(t)
	withFakeNpx(t)
	v := &verifierStub{answer: "VERDICT: PASS"}
	m, repo, store := driverSetup(t, "drv-codex", `{"uiDriver":"playwright"}`, "- [x] done\n", v, "")
	// No session manager: the verifier agent is unknown, hence not Claude.
	m.sessionMgr = nil
	runVerifyNow(m, "drv-codex")
	if st := verifyStateOf(t, m, repo, "drv-codex"); st != "failed" {
		t.Fatalf("marker = %q", st)
	}
	end := verifyEnd(t, loadVerifyRun(t, store, "drv-codex"))
	if !strings.Contains(end["reason"].(string), "Claude") || v.startCount() != 0 {
		t.Errorf("end = %v, starts = %d", end, v.startCount())
	}
}

func ptrTo[T any](v T) *T { return &v }
