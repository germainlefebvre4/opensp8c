package pool

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/verification"
)

var claudeAgent = agents.AgentConfig{ID: "claude"}

// withFakeNpx puts an executable npx first on PATH (the rest stays usable).
func withFakeNpx(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "npx"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func resolvedDriver(driver string) verification.Resolved {
	return verification.Resolved{LaunchParams: verification.LaunchParams{UIDriver: driver}}
}

func writeMCPFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "mcp.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPlanDriverAuto(t *testing.T) {
	plan, err := planDriver(context.Background(), resolvedDriver("auto"), claudeAgent, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer plan.cleanup()
	if !reflect.DeepEqual(plan.Args, []string{"--permission-prompts", "none"}) {
		t.Errorf("args: %v", plan.Args)
	}
	for _, a := range plan.Args {
		if a == "--mcp-config" || a == "--allowedTools" || a == "--chrome" {
			t.Errorf("auto must not pass %s", a)
		}
	}
	if len(plan.AllowedTools) != 0 || len(plan.Servers) != 0 || plan.NeedChromeTools || len(plan.ToolPrefixes) != 0 {
		t.Errorf("auto plans no tool control: %+v", plan)
	}
	// An empty driver is auto; auto works with any agent.
	if plan, err := planDriver(context.Background(), resolvedDriver(""), agents.AgentConfig{ID: "codex"}, ""); err != nil || plan.Driver != "auto" {
		t.Errorf("empty driver: %+v, %v", plan, err)
	}
}

func TestPlanDriverOmitsPermissionPromptsWhenUnsupported(t *testing.T) {
	old := permissionPromptsNone
	permissionPromptsNone = func(context.Context) bool { return false }
	defer func() { permissionPromptsNone = old }()

	plan, err := planDriver(context.Background(), resolvedDriver("auto"), claudeAgent, "")
	if err != nil || len(plan.Args) != 0 {
		t.Errorf("auto: %v, %v", plan.Args, err)
	}
	withFakeNpx(t)
	plan, err = planDriver(context.Background(), resolvedDriver("playwright"), claudeAgent, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer plan.cleanup()
	for _, a := range plan.Args {
		if a == "--permission-prompts" {
			t.Errorf("flag must be omitted: %v", plan.Args)
		}
	}
}

func TestPlanDriverPlaywright(t *testing.T) {
	withFakeNpx(t)
	artifacts := t.TempDir()
	plan, err := planDriver(context.Background(), resolvedDriver("playwright"), claudeAgent, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	cfgPath := plan.Args[indexOf(plan.Args, "--mcp-config")+1]
	want := []string{
		"--permission-prompts", "none",
		"--mcp-config", cfgPath, "--strict-mcp-config",
		"--allowedTools", "mcp__playwright", "Read", "Grep", "Glob",
	}
	if !reflect.DeepEqual(plan.Args, want) {
		t.Fatalf("args: %v\nwant: %v", plan.Args, want)
	}
	if !reflect.DeepEqual(plan.AllowedTools, []string{"mcp__playwright", "Read", "Grep", "Glob"}) ||
		!reflect.DeepEqual(plan.Servers, []string{"playwright"}) ||
		!reflect.DeepEqual(plan.ToolPrefixes, []string{"mcp__playwright__"}) || plan.NeedChromeTools {
		t.Errorf("plan: %+v", plan)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		McpServers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	pw, ok := cfg.McpServers["playwright"]
	if len(cfg.McpServers) != 1 || !ok || pw.Command != "npx" {
		t.Fatalf("config: %s", data)
	}
	wantArgs := []string{"-y", "@playwright/mcp@" + playwrightMCPVersion, "--headless", "--isolated", "--output-dir", artifacts}
	if !reflect.DeepEqual(pw.Args, wantArgs) {
		t.Errorf("playwright args: %v", pw.Args)
	}
	if strings.Contains(playwrightMCPVersion, "latest") || playwrightMCPVersion == "" {
		t.Error("the version must be pinned")
	}
	if strings.HasPrefix(cfgPath, artifacts) {
		t.Error("the configuration must not live in the evidence directory")
	}

	plan.cleanup()
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Errorf("cleanup must remove the file: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(cfgPath)); !os.IsNotExist(err) {
		t.Errorf("cleanup must remove its directory: %v", err)
	}
}

func TestPlanDriverChrome(t *testing.T) {
	plan, err := planDriver(context.Background(), resolvedDriver("chrome"), claudeAgent, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer plan.cleanup()
	want := []string{"--permission-prompts", "none", "--chrome", "--allowedTools", "mcp__claude-in-chrome", "Read", "Grep", "Glob"}
	if !reflect.DeepEqual(plan.Args, want) {
		t.Errorf("args: %v", plan.Args)
	}
	if !plan.NeedChromeTools || !reflect.DeepEqual(plan.ToolPrefixes, []string{"mcp__claude-in-chrome__"}) || len(plan.Servers) != 0 {
		t.Errorf("plan: %+v", plan)
	}
	for _, a := range plan.Args {
		if a == "--mcp-config" {
			t.Error("chrome passes no MCP file")
		}
	}
	if !strings.Contains(plan.Directive, "your own tabs") {
		t.Errorf("directive: %q", plan.Directive)
	}
}

func TestPlanDriverCustom(t *testing.T) {
	cfg := writeMCPFile(t, `{"mcpServers":{"cypress":{"command":"x"},"my db":{"command":"y"}}}`)
	res := resolvedDriver("custom")
	res.UIMcpConfig = cfg
	res.UIAllowedTools = []string{"mcp__cypress"}
	plan, err := planDriver(context.Background(), res, claudeAgent, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer plan.cleanup()
	want := []string{"--permission-prompts", "none", "--mcp-config", cfg, "--strict-mcp-config", "--allowedTools", "mcp__cypress", "Read", "Grep", "Glob"}
	if !reflect.DeepEqual(plan.Args, want) {
		t.Errorf("args: %v", plan.Args)
	}
	if !reflect.DeepEqual(plan.Servers, []string{"cypress", "my db"}) ||
		!reflect.DeepEqual(plan.ToolPrefixes, []string{"mcp__cypress__", "mcp__my_db__"}) {
		t.Errorf("plan: %+v", plan)
	}
	if _, err := os.Stat(cfg); err != nil {
		t.Errorf("the user's file must survive cleanup: %v", err)
	}
	plan.cleanup()
	if _, err := os.Stat(cfg); err != nil {
		t.Errorf("the user's file must survive cleanup: %v", err)
	}
}

func TestPlanDriverPrecheckFailures(t *testing.T) {
	good := writeMCPFile(t, `{"mcpServers":{"x":{}}}`)
	custom := func(cfg string, tools ...string) verification.Resolved {
		r := resolvedDriver("custom")
		r.UIMcpConfig, r.UIAllowedTools = cfg, tools
		return r
	}
	cases := []struct {
		name  string
		res   verification.Resolved
		agent agents.AgentConfig
		path  string // PATH override; "" keeps a fake npx
		want  string
	}{
		{"npx missing", resolvedDriver("playwright"), claudeAgent, "none", "npx"},
		{"agent not claude", resolvedDriver("playwright"), agents.AgentConfig{ID: "codex"}, "", "Claude"},
		{"agent not claude, chrome", resolvedDriver("chrome"), agents.AgentConfig{ID: "gemini"}, "", "Claude"},
		{"custom without config", custom("", "mcp__x"), claudeAgent, "", "uiMcpConfig non configuré"},
		{"custom missing file", custom("/nonexistent/mcp.json", "mcp__x"), claudeAgent, "", "/nonexistent/mcp.json"},
		{"custom invalid JSON", custom(writeMCPFile(t, "{"), "mcp__x"), claudeAgent, "", "JSON invalide"},
		{"custom without mcpServers", custom(writeMCPFile(t, `{"other":1}`), "mcp__x"), claudeAgent, "", "mcpServers"},
		{"custom empty mcpServers", custom(writeMCPFile(t, `{"mcpServers":{}}`), "mcp__x"), claudeAgent, "", "mcpServers"},
		{"custom without tools", custom(good), claudeAgent, "", "uiAllowedTools non configuré"},
		{"unknown driver", resolvedDriver("selenium"), claudeAgent, "", "selenium"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.path == "" {
				withFakeNpx(t)
			} else {
				t.Setenv("PATH", t.TempDir())
			}
			plan, err := planDriver(context.Background(), c.res, c.agent, t.TempDir())
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want an error containing %q, got %v", c.want, err)
			}
			plan.cleanup() // must always be callable
		})
	}
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

// testdataLines reads the captured claude events of testdata/<name>.
func testdataLines(t *testing.T, name string) [][]byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var out [][]byte
	for _, l := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		out = append(out, []byte(l))
	}
	return out
}

func observeAll(o *driverObserver, lines [][]byte) error {
	for _, l := range lines {
		if err := o.Observe(l); err != nil {
			return err
		}
	}
	return nil
}

func playwrightPlan() driverPlan {
	return driverPlan{Driver: "playwright", Servers: []string{"playwright"}, ToolPrefixes: []string{"mcp__playwright__"}}
}

func chromePlan() driverPlan {
	return driverPlan{Driver: "chrome", NeedChromeTools: true, ToolPrefixes: []string{"mcp__claude-in-chrome__"}}
}

func TestDriverObserverInit(t *testing.T) {
	cases := []struct {
		name string
		plan driverPlan
		file string
		want string // "" = accepted
	}{
		{"connected server", playwrightPlan(), "claude_init_playwright_connected.jsonl", ""},
		{"failed server", playwrightPlan(), "claude_init_playwright_failed.jsonl", "failed"},
		{"server absent", playwrightPlan(), "claude_init_no_mcp.jsonl", "absent"},
		{"chrome tools present", chromePlan(), "claude_init_chrome.jsonl", ""},
		{"chrome tools absent", chromePlan(), "claude_init_no_mcp.jsonl", "Chrome est ouvert"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o := newDriverObserver(c.plan)
			err := observeAll(o, testdataLines(t, c.file))
			if c.want == "" {
				if err != nil {
					t.Fatalf("unexpected: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want an error containing %q, got %v", c.want, err)
			}
			if !o.InitSeen() {
				t.Error("the init event must be recorded")
			}
		})
	}
}

func TestDriverObserverUnreadableInit(t *testing.T) {
	for name, plan := range map[string]driverPlan{"playwright": playwrightPlan(), "chrome": chromePlan()} {
		o := newDriverObserver(plan)
		err := o.Observe([]byte(`{"type":"system","subtype":"init","model":"x"}`))
		if err == nil || err.Error() != "événement d'initialisation illisible" {
			t.Errorf("%s: got %v", name, err)
		}
	}
	// Other events that merely contain the word are not the init event.
	o := newDriverObserver(playwrightPlan())
	if err := o.Observe([]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"init"}]}}`)); err != nil || o.InitSeen() {
		t.Errorf("got %v, seen %v", err, o.InitSeen())
	}
}

func TestDriverObserverAutoIsNotChecked(t *testing.T) {
	o := newDriverObserver(driverPlan{Driver: "auto"})
	if err := observeAll(o, testdataLines(t, "claude_init_no_mcp.jsonl")); err != nil || o.InitSeen() {
		t.Errorf("auto must not be checked: %v", err)
	}
}

func TestDriverObserverCountsOnlyDriverTools(t *testing.T) {
	o := newDriverObserver(playwrightPlan())
	if err := observeAll(o, testdataLines(t, "claude_tool_use_playwright.jsonl")); err != nil {
		t.Fatal(err)
	}
	if o.Calls() != 2 {
		t.Errorf("two playwright calls expected, got %d", o.Calls())
	}
	chrome := newDriverObserver(chromePlan())
	_ = observeAll(chrome, testdataLines(t, "claude_tool_use_playwright.jsonl"))
	if chrome.Calls() != 0 {
		t.Errorf("playwright calls must not count for chrome, got %d", chrome.Calls())
	}
	// A ToolSearch call or a built-in tool is not a driver call.
	other := newDriverObserver(playwrightPlan())
	_ = other.Observe([]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"ToolSearch","input":{}},{"type":"tool_use","name":"Read","input":{}}]}}`))
	if other.Calls() != 0 {
		t.Errorf("got %d", other.Calls())
	}
}

func TestRunTurnTextStopsOnObserverError(t *testing.T) {
	proc, stdinR, stdoutW := newPipeSubprocess()
	go func() { _, _ = io.Copy(io.Discard, stdinR) }()
	go func() {
		for _, l := range testdataLines(t, "claude_init_playwright_failed.jsonl") {
			_, _ = stdoutW.Write(append(l, '\n'))
		}
		_, _ = stdoutW.Write([]byte(`{"type":"result","subtype":"success","result":"VERDICT: PASS"}` + "\n"))
		_ = stdoutW.Close()
	}()
	cancelled := false
	tt, _ := textTarget()
	tt.procCancel = func() { cancelled = true }
	tt.observe = newDriverObserver(playwrightPlan()).Observe
	text, err := (&Manager{}).runTurnText(tt, proc, "go")
	var de *driverCheckError
	if !errors.As(err, &de) || text != "" {
		t.Fatalf("got %q, %v", text, err)
	}
	if !strings.Contains(agentTurnPauseReason(err, "x"), "failed") || !cancelled {
		t.Errorf("reason %q, cancelled %v", agentTurnPauseReason(err, "x"), cancelled)
	}
}
