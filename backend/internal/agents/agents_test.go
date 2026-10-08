package agents

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestBuildSubprocessArgs_Gemini(t *testing.T) {
	cfg := AgentConfig{
		ID: "gemini",
	}
	args := cfg.BuildSubprocessArgs("base prompt", "extra prompt")
	expected := []string{
		"--output-format", "stream-json",
		"--approval-mode", "auto_edit",
		"--skip-trust",
	}

	if !reflect.DeepEqual(args, expected) {
		t.Errorf("expected gemini args: %v, got: %v", expected, args)
	}
}

func TestBuildSubprocessArgs_Antigravity(t *testing.T) {
	cfg := AgentConfig{
		ID: "antigravity",
	}
	args := cfg.BuildSubprocessArgs("base prompt", "extra prompt")
	expected := []string{
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--dangerously-skip-permissions",
	}

	if !reflect.DeepEqual(args, expected) {
		t.Errorf("expected antigravity args: %v, got: %v", expected, args)
	}
}

func TestBuildSubprocessArgs_Other(t *testing.T) {
	cfg := AgentConfig{
		ID: "claude",
	}
	args := cfg.BuildSubprocessArgs("base prompt", "extra prompt")
	expected := []string{
		"--print",
		"--verbose",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--include-partial-messages",
		"--append-system-prompt", "base prompt",
		"--append-system-prompt", "extra prompt",
	}

	if !reflect.DeepEqual(args, expected) {
		t.Errorf("expected claude args: %v, got: %v", expected, args)
	}
}

func TestByID(t *testing.T) {
	gemini, ok := ByID("gemini")
	if !ok {
		t.Fatal("expected to find gemini")
	}
	if gemini.ID != "gemini" {
		t.Errorf("expected ID 'gemini', got: %s", gemini.ID)
	}

	_, ok = ByID("nonexistent")
	if ok {
		t.Fatal("expected nonexistent agent to not be found")
	}
}

func TestDetectAll_ExposesDocsURL(t *testing.T) {
	statuses := DetectAll()
	if len(statuses) != len(SupportedAgents) {
		t.Fatalf("expected %d statuses, got %d", len(SupportedAgents), len(statuses))
	}
	for i, s := range statuses {
		if s.DocsURL == "" || s.DocsURL != SupportedAgents[i].DocsURL {
			t.Errorf("agent %s: unexpected DocsURL %q", s.ID, s.DocsURL)
		}
	}
}

func TestSupportedAgents_Capabilities(t *testing.T) {
	cases := []struct {
		id, modelFlag, effortFlag string
		levels                    []string
	}{
		{"claude", "--model", "--effort", []string{"low", "medium", "high", "xhigh", "max"}},
		{"antigravity", "--model", "--effort", []string{"low", "medium", "high", "max"}},
		{"gemini", "-m", "", nil},
		{"codex", "-m", "", nil},
		{"copilot", "", "", nil},
	}
	for _, c := range cases {
		a, ok := ByID(c.id)
		if !ok {
			t.Fatalf("missing agent %s", c.id)
		}
		if a.ModelFlag != c.modelFlag || a.EffortFlag != c.effortFlag || !reflect.DeepEqual(a.EffortLevels, c.levels) {
			t.Errorf("%s: unexpected capabilities %q %q %v", c.id, a.ModelFlag, a.EffortFlag, a.EffortLevels)
		}
		if a.SupportsModel() != (c.modelFlag != "") || a.SupportsEffort() != (c.effortFlag != "") {
			t.Errorf("%s: unexpected Supports*", c.id)
		}
	}
	claude, _ := ByID("claude")
	ids := map[string]bool{}
	for _, m := range claude.SeedModels {
		ids[m.ID] = true
	}
	for _, want := range []string{"fable", "opus", "sonnet", "haiku"} {
		if !ids[want] {
			t.Errorf("claude seed missing %s", want)
		}
	}
	gemini, _ := ByID("gemini")
	if len(gemini.SeedModels) != 4 {
		t.Errorf("gemini seed: %v", gemini.SeedModels)
	}
	for _, a := range SupportedAgents {
		if len(a.SeedModels) > 0 && a.ModelFlag == "" {
			t.Errorf("%s has seed models without a model flag", a.ID)
		}
	}
}

func withModelEffort(id, model, effort string) AgentConfig {
	a, _ := ByID(id)
	a.Model, a.Effort = model, effort
	return a
}

func containsPair(args []string, k, v string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == k && args[i+1] == v {
			return true
		}
	}
	return false
}

func TestBuildSubprocessArgs_ModelAndEffort(t *testing.T) {
	args := withModelEffort("claude", "sonnet", "medium").BuildSubprocessArgs("b", "")
	if !containsPair(args, "--model", "sonnet") || !containsPair(args, "--effort", "medium") {
		t.Errorf("claude args: %v", args)
	}
	// model precedes effort
	if len(args) < 4 || args[len(args)-4] != "--model" {
		t.Errorf("model/effort should end the args: %v", args)
	}

	args = withModelEffort("codex", "gpt-5.5", "high").BuildSubprocessArgs("b", "")
	if !containsPair(args, "-m", "gpt-5.5") {
		t.Errorf("codex args: %v", args)
	}
	for _, a := range args {
		if a == "--effort" || a == "high" {
			t.Errorf("codex must not receive effort: %v", args)
		}
	}

	args = withModelEffort("claude", "", "").BuildSubprocessArgs("b", "")
	for _, a := range args {
		if a == "--model" || a == "--effort" {
			t.Errorf("empty values must not add flags: %v", args)
		}
	}

	args = withModelEffort("copilot", "x", "high").BuildSubprocessArgs("b", "")
	for _, a := range args {
		if a == "--model" || a == "-m" || a == "--effort" {
			t.Errorf("copilot must not receive flags: %v", args)
		}
	}

	args = withModelEffort("gemini", "flash", "high").BuildSubprocessArgs("b", "")
	if !containsPair(args, "-m", "flash") {
		t.Errorf("gemini args: %v", args)
	}
	args = withModelEffort("antigravity", "m1", "max").BuildSubprocessArgs("b", "")
	if !containsPair(args, "--model", "m1") || !containsPair(args, "--effort", "max") {
		t.Errorf("agy args: %v", args)
	}
}

func TestBuildSubprocessArgs_ExtraArgs(t *testing.T) {
	base := AgentConfig{ID: "claude"}.BuildSubprocessArgs("base", "")
	extra := []string{"--permission-prompts", "none", "--chrome"}
	got := AgentConfig{ID: "claude", ModelFlag: "--model", Model: "opus", EffortFlag: "--effort", Effort: "high", ExtraArgs: extra}.BuildSubprocessArgs("base", "")
	want := append(append([]string(nil), base...), "--model", "opus", "--effort", "high", "--permission-prompts", "none", "--chrome")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// Without ExtraArgs the arguments are those of before.
	if got := (AgentConfig{ID: "claude", ExtraArgs: nil}).BuildSubprocessArgs("base", ""); !reflect.DeepEqual(got, base) {
		t.Errorf("got %v, want %v", got, base)
	}
	// Other agents ignore ExtraArgs.
	for _, id := range []string{"codex", "gemini", "antigravity", "copilot"} {
		with := AgentConfig{ID: id, ExtraArgs: extra}.BuildSubprocessArgs("base", "")
		without := AgentConfig{ID: id}.BuildSubprocessArgs("base", "")
		if !reflect.DeepEqual(with, without) {
			t.Errorf("%s must ignore ExtraArgs: %v", id, with)
		}
	}
}

func TestSupportsDrivers(t *testing.T) {
	for id, want := range map[string]bool{"claude": true, "codex": false, "gemini": false, "antigravity": false, "copilot": false} {
		if got := (AgentConfig{ID: id}).SupportsDrivers(); got != want {
			t.Errorf("%s: got %v", id, got)
		}
	}
}

func TestClaudeSupportsFlag(t *testing.T) {
	old := claudeHelp
	defer func() { claudeHelp = old; claudeHelpCache = "" }()

	calls := 0
	claudeHelp = func(context.Context) (string, error) {
		calls++
		return "  --permission-prompts <target>  Who answers\n  --chrome", nil
	}
	claudeHelpCache = ""
	if !ClaudeSupportsFlag(context.Background(), "--permission-prompts") {
		t.Error("flag listed in the help must be detected")
	}
	if ClaudeSupportsFlag(context.Background(), "--nope") {
		t.Error("flag absent from the help must not be detected")
	}
	if calls != 1 {
		t.Errorf("help must be read once, got %d calls", calls)
	}

	claudeHelp = func(context.Context) (string, error) { return "", errors.New("not installed") }
	claudeHelpCache = ""
	if ClaudeSupportsFlag(context.Background(), "--permission-prompts") {
		t.Error("a failing CLI must answer false")
	}
}
