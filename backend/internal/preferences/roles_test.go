package preferences

import (
	"encoding/json"
	"errors"
	"github.com/glefebvre/opensp8c/internal/agents"
	"os"
	"reflect"
	"strings"
	"testing"
)

func rs(agent, model, effort string) RoleSetting {
	return RoleSetting{Agent: agent, Model: model, Effort: effort}
}

func TestLoadWithoutNewFieldsDoesNotRewrite(t *testing.T) {
	svc := newTestService(t)
	content := `{"defaultAgent":"claude","env":{"A":"1"}}`
	if err := os.WriteFile(svc.Path(), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	p, err := svc.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if p.AgentSettings != nil || p.PoolDefaults != nil || p.Workspaces != nil {
		t.Errorf("new fields should be nil: %+v", p)
	}
	after, _ := os.ReadFile(svc.Path())
	if string(after) != content {
		t.Errorf("file was rewritten: %s", after)
	}
	// Saving other data must not introduce the new keys.
	if err := svc.SetDefaultAgent("codex"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(svc.Path())
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	for _, k := range []string{"agentSettings", "poolDefaults", "workspaces"} {
		if _, ok := m[k]; ok {
			t.Errorf("key %s should be omitted", k)
		}
	}
}

func TestResolveRole(t *testing.T) {
	cases := []struct {
		name   string
		p      *Preferences
		ws     string
		role   Role
		locked string
		want   Resolved
	}{
		{"nil receiver uses claude preset", nil, "", RoleExplorer, "", Resolved{"claude", "opus", "high"}},
		{"documenter preset", &Preferences{DefaultAgent: "claude"}, "", RoleDocumenter, "", Resolved{"claude", "haiku", "low"}},
		{"fixer preset", &Preferences{}, "", RoleFixer, "", Resolved{"claude", "sonnet", "medium"}},
		{"no preset for gemini", &Preferences{DefaultAgent: "gemini"}, "", RoleImplementer, "", Resolved{"gemini", "", ""}},
		{"role before global", &Preferences{AgentSettings: &AgentSettings{
			Global: rs("claude", "sonnet", "medium"),
			Roles:  map[Role]RoleSetting{RoleDocumenter: rs("", "", "low")},
		}}, "", RoleDocumenter, "", Resolved{"claude", "sonnet", "low"}},
		{"global beats preset", &Preferences{AgentSettings: &AgentSettings{Global: rs("claude", "haiku", "")}},
			"", RoleExplorer, "", Resolved{"claude", "haiku", "high"}},
		{"role changes agent, model not inherited", &Preferences{AgentSettings: &AgentSettings{
			Global: rs("claude", "sonnet", "medium"),
			Roles:  map[Role]RoleSetting{RoleFF: rs("codex", "", "")},
		}}, "", RoleFF, "", Resolved{"codex", "", ""}},
		{"workspace beats configuration", &Preferences{
			AgentSettings: &AgentSettings{Roles: map[Role]RoleSetting{RoleImplementer: rs("claude", "sonnet", "")}},
			Workspaces: map[string]*WorkspacePrefs{"A": {AgentSettings: &AgentSettings{
				Roles: map[Role]RoleSetting{RoleImplementer: rs("", "opus", "")}}}},
		}, "A", RoleImplementer, "", Resolved{"claude", "opus", "medium"}},
		{"other workspace keeps configuration", &Preferences{
			AgentSettings: &AgentSettings{Roles: map[Role]RoleSetting{RoleImplementer: rs("claude", "sonnet", "")}},
			Workspaces: map[string]*WorkspacePrefs{"A": {AgentSettings: &AgentSettings{
				Roles: map[Role]RoleSetting{RoleImplementer: rs("", "opus", "")}}}},
		}, "B", RoleImplementer, "", Resolved{"claude", "sonnet", "medium"}},
		{"workspace global beats configuration role", &Preferences{
			AgentSettings: &AgentSettings{Roles: map[Role]RoleSetting{RoleExplorer: rs("", "opus", "")}},
			Workspaces:    map[string]*WorkspacePrefs{"A": {AgentSettings: &AgentSettings{Global: rs("", "haiku", "")}}},
		}, "A", RoleExplorer, "", Resolved{"claude", "haiku", "high"}},
		{"locked agent wins, settings for locked agent", &Preferences{AgentSettings: &AgentSettings{
			Roles: map[Role]RoleSetting{RoleExplorer: rs("codex", "gpt-x", "")},
		}}, "", RoleExplorer, "claude", Resolved{"claude", "opus", "high"}},
		{"locked agent keeps its own level model", &Preferences{AgentSettings: &AgentSettings{
			Global: rs("claude", "haiku", ""),
			Roles:  map[Role]RoleSetting{RoleExplorer: rs("codex", "gpt-x", "")},
		}}, "", RoleExplorer, "claude", Resolved{"claude", "haiku", "high"}},
		{"effort outside levels dropped", &Preferences{AgentSettings: &AgentSettings{
			Global: rs("antigravity", "", "xhigh"),
		}}, "", RoleFF, "", Resolved{"antigravity", "", ""}},
		{"effort dropped for agent without effort", &Preferences{AgentSettings: &AgentSettings{
			Global: rs("gemini", "flash", "high"),
		}}, "", RoleFF, "", Resolved{"gemini", "flash", ""}},
		{"model inherited when lower level agent unset but default matches", &Preferences{
			DefaultAgent:  "claude",
			AgentSettings: &AgentSettings{Global: rs("", "haiku", "")},
		}, "", RoleFF, "", Resolved{"claude", "haiku", "medium"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.p.ResolveRole(c.ws, c.role, c.locked)
			if got != c.want {
				t.Errorf("got %+v want %+v", got, c.want)
			}
		})
	}
}

func TestApplyRole(t *testing.T) {
	// covered through agents.BuildSubprocessArgs in handlers; here only fields
	cfg := ApplyRole(mustAgent(t, "claude"), Resolved{Agent: "claude", Model: "sonnet", Effort: "medium"})
	args := cfg.BuildSubprocessArgs("b", "")
	found := 0
	for i := range args {
		if (args[i] == "--model" && args[i+1] == "sonnet") || (args[i] == "--effort" && args[i+1] == "medium") {
			found++
		}
	}
	if found != 2 {
		t.Errorf("expected both flags: %v", args)
	}
}

func TestSetAgentSettingsValidation(t *testing.T) {
	svc := newTestService(t)
	if err := svc.SetDefaultAgent("claude"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(svc.Path())

	patch := func(j string) AgentSettingsPatch {
		var p AgentSettingsPatch
		if err := json.Unmarshal([]byte(j), &p); err != nil {
			t.Fatal(err)
		}
		return p
	}
	bad := []string{
		`{"global":{"agent":"nope"}}`,
		`{"roles":{"explorer":{"model":"--dangerously-skip-permissions"}}}`,
		`{"roles":{"explorer":{"model":"a b"}}}`,
		`{"roles":{"explorer":{"model":"   "}}}`,
		`{"roles":{"explorer":{"model":"a\u0007b"}}}`,
		`{"roles":{"nope":{"model":"x"}}}`,
		`{"global":{"agent":"antigravity","effort":"xhigh"}}`,
		`{"roles":{"ff":{"agent":"gemini","effort":"high"}}}`,
		`{"roles":{"ff":{"effort":"bogus"}}}`,
	}
	for _, j := range bad {
		err := svc.SetAgentSettings(patch(j))
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("%s: expected ValidationError, got %v", j, err)
		}
		after, _ := os.ReadFile(svc.Path())
		if string(after) != string(before) {
			t.Fatalf("%s: file changed on rejection", j)
		}
	}

	if err := svc.SetAgentSettings(patch(`{"roles":{"documenter":{"model":"haiku"}}}`)); err != nil {
		t.Fatal(err)
	}
	p, _ := svc.Load()
	if p.AgentSettings.Roles[RoleDocumenter].Model != "haiku" || len(p.AgentSettings.Roles) != 1 {
		t.Errorf("unexpected: %+v", p.AgentSettings)
	}
	// Reset via null and via null role
	if err := svc.SetAgentSettings(patch(`{"roles":{"documenter":{"model":null}}}`)); err != nil {
		t.Fatal(err)
	}
	p, _ = svc.Load()
	if p.AgentSettings != nil {
		t.Errorf("expected cleaned settings: %+v", p.AgentSettings)
	}
	_ = svc.SetAgentSettings(patch(`{"roles":{"ff":{"model":"x"}}}`))
	_ = svc.SetAgentSettings(patch(`{"roles":{"ff":null}}`))
	p, _ = svc.Load()
	if p.AgentSettings != nil {
		t.Errorf("expected role reset: %+v", p.AgentSettings)
	}
}

func TestResolvePool(t *testing.T) {
	var nilP *Preferences
	if got := nilP.ResolvePool("A"); got != (PoolSettings{Size: 3, DelegationMode: "hitl-review", MaxAttempts: 3}) {
		t.Errorf("builtin: %+v", got)
	}
	four, mode := 4, "full-autonomy"
	p := &Preferences{
		PoolDefaults: &PoolSettings{Size: 2, MaxAttempts: 5},
		Workspaces:   map[string]*WorkspacePrefs{"A": {Pool: &PoolOverride{Size: &four, DelegationMode: &mode}}},
	}
	if got := p.ResolvePool("A"); got != (PoolSettings{Size: 4, DelegationMode: "full-autonomy", MaxAttempts: 5}) {
		t.Errorf("A: %+v", got)
	}
	if got := p.ResolvePool("B"); got != (PoolSettings{Size: 2, DelegationMode: "hitl-review", MaxAttempts: 5}) {
		t.Errorf("B: %+v", got)
	}
}

func TestPoolPatchValidation(t *testing.T) {
	svc := newTestService(t)
	mk := func(j string) PoolPatch {
		var p PoolPatch
		if err := json.Unmarshal([]byte(j), &p); err != nil {
			t.Fatal(err)
		}
		return p
	}
	for _, j := range []string{`{"size":0}`, `{"size":9}`, `{"delegationMode":"x"}`, `{"maxAttempts":0}`} {
		var ve *ValidationError
		if err := svc.SetPoolDefaults(mk(j)); !errors.As(err, &ve) {
			t.Errorf("defaults %s: expected ValidationError, got %v", j, err)
		}
		if err := svc.PatchWorkspace("A", WorkspaceSettingsPatch{Pool: ptr(mk(j))}); !errors.As(err, &ve) {
			t.Errorf("workspace %s: expected ValidationError, got %v", j, err)
		}
	}
	if _, err := os.Stat(svc.Path()); err == nil {
		t.Error("file should not be created by rejected updates")
	}
	if err := svc.PatchWorkspace("A", WorkspaceSettingsPatch{Pool: ptr(mk(`{"size":4}`))}); err != nil {
		t.Fatal(err)
	}
	p, _ := svc.Load()
	if p.ResolvePool("A").Size != 4 || p.ResolvePool("B").Size != 3 {
		t.Errorf("isolation broken: %+v", p.Workspaces)
	}
	if err := svc.PatchWorkspace("A", WorkspaceSettingsPatch{Pool: ptr(mk(`{"size":null}`))}); err != nil {
		t.Fatal(err)
	}
	p, _ = svc.Load()
	if len(p.Workspaces) != 0 {
		t.Errorf("empty section should be removed: %+v", p.Workspaces)
	}
	if err := svc.SetPoolDefaults(mk(`{"size":2}`)); err != nil {
		t.Fatal(err)
	}
	p, _ = svc.Load()
	if p.ResolvePool("Z").Size != 2 {
		t.Errorf("defaults not applied: %+v", p.PoolDefaults)
	}
}

func TestEnvForWorkspaceOrder(t *testing.T) {
	p := &Preferences{
		Env:      map[string]string{"FOO": "global", "G": "g"},
		AgentEnv: map[string]map[string]string{"claude": {"FOO": "agent", "A": "a"}},
		Workspaces: map[string]*WorkspacePrefs{
			"A": {
				Env:      map[string]string{"FOO": "ws-global", "W": "w"},
				AgentEnv: map[string]map[string]string{"claude": {"FOO": "ws-claude"}},
			},
			"B": {Env: map[string]string{"FOO": "ws-b"}},
		},
	}
	if got := p.EnvForWorkspace("A", "claude"); got["FOO"] != "ws-claude" || got["W"] != "w" || got["A"] != "a" || got["G"] != "g" {
		t.Errorf("A/claude: %v", got)
	}
	if got := p.EnvForWorkspace("A", "gemini"); got["FOO"] != "ws-global" {
		t.Errorf("A/gemini: %v", got)
	}
	if got := p.EnvForWorkspace("C", "claude"); got["FOO"] != "agent" {
		t.Errorf("C/claude: %v", got)
	}
	if got := p.EnvForWorkspace("B", "claude"); got["FOO"] != "ws-b" {
		t.Errorf("B: %v", got)
	}
	if !reflect.DeepEqual(p.EnvFor("claude"), p.EnvForWorkspace("", "claude")) {
		t.Error("EnvFor alias mismatch")
	}
	var nilP *Preferences
	if got := nilP.EnvFor("claude"); len(got) != 0 {
		t.Errorf("nil: %v", got)
	}
}

func TestPatchWorkspaceEnvAndRemoval(t *testing.T) {
	svc := newTestService(t)
	if err := svc.PatchWorkspace("A", WorkspaceSettingsPatch{
		Env:      map[string]string{"FOO": "1"},
		AgentEnv: map[string]map[string]string{"claude": {"BAR": "2"}},
	}); err != nil {
		t.Fatal(err)
	}
	var ve *ValidationError
	if err := svc.PatchWorkspace("A", WorkspaceSettingsPatch{AgentEnv: map[string]map[string]string{"nope": {"X": "1"}}}); !errors.As(err, &ve) {
		t.Errorf("expected ValidationError: %v", err)
	}
	p, _ := svc.Load()
	if p.Workspaces["A"].Env["FOO"] != "1" || p.Workspaces["A"].AgentEnv["claude"]["BAR"] != "2" {
		t.Errorf("unexpected: %+v", p.Workspaces["A"])
	}
	if err := svc.PatchWorkspace("A", WorkspaceSettingsPatch{Env: map[string]string{}, AgentEnv: map[string]map[string]string{"claude": {}}}); err != nil {
		t.Fatal(err)
	}
	p, _ = svc.Load()
	if _, ok := p.Workspaces["A"]; ok {
		t.Errorf("section should be removed: %+v", p.Workspaces)
	}
}

func ptr[T any](v T) *T { return &v }

func mustAgent(t *testing.T, id string) agents.AgentConfig {
	t.Helper()
	a, ok := agents.ByID(id)
	if !ok {
		t.Fatalf("no agent %s", id)
	}
	return a
}

func TestValidationCommand(t *testing.T) {
	svc := newTestService(t)
	mk := func(j string) PoolPatch {
		var p PoolPatch
		if err := json.Unmarshal([]byte(j), &p); err != nil {
			t.Fatal(err)
		}
		return p
	}

	// Default: empty (auto-detect).
	p, _ := svc.Load()
	if got := p.ResolvePool("A").ValidationCommand; got != "" {
		t.Fatalf("default = %q", got)
	}

	// Global default, persisted in preferences.json.
	if err := svc.SetPoolDefaults(mk(`{"validationCommand":"  make test "}`)); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(svc.Path())
	if err != nil || !strings.Contains(string(raw), `"validationCommand": "make test"`) {
		t.Fatalf("not persisted (trimmed): %v %s", err, raw)
	}
	p, _ = svc.Load()
	if got := p.ResolvePool("A").ValidationCommand; got != "make test" {
		t.Fatalf("global = %q", got)
	}

	// Workspace override wins, only for that workspace.
	if err := svc.PatchWorkspace("A", WorkspaceSettingsPatch{Pool: ptr(mk(`{"validationCommand":"cd backend && go test ./..."}`))}); err != nil {
		t.Fatal(err)
	}
	p, _ = svc.Load()
	if got := p.ResolvePool("A").ValidationCommand; got != "cd backend && go test ./..." {
		t.Fatalf("workspace = %q", got)
	}
	if got := p.ResolvePool("B").ValidationCommand; got != "make test" {
		t.Fatalf("B = %q", got)
	}

	// Reset (null) returns to inheritance; blank resets too.
	if err := svc.PatchWorkspace("A", WorkspaceSettingsPatch{Pool: ptr(mk(`{"validationCommand":null}`))}); err != nil {
		t.Fatal(err)
	}
	p, _ = svc.Load()
	if got := p.ResolvePool("A").ValidationCommand; got != "make test" || len(p.Workspaces) != 0 {
		t.Fatalf("after reset: %q %+v", got, p.Workspaces)
	}
	if err := svc.PatchWorkspace("A", WorkspaceSettingsPatch{Pool: ptr(mk(`{"validationCommand":"x"}`))}); err != nil {
		t.Fatal(err)
	}
	if err := svc.PatchWorkspace("A", WorkspaceSettingsPatch{Pool: ptr(mk(`{"validationCommand":"   "}`))}); err != nil {
		t.Fatal(err)
	}
	p, _ = svc.Load()
	if len(p.Workspaces) != 0 {
		t.Fatalf("blank should reset: %+v", p.Workspaces)
	}

	// Clearing the global default.
	if err := svc.SetPoolDefaults(mk(`{"validationCommand":""}`)); err != nil {
		t.Fatal(err)
	}
	p, _ = svc.Load()
	if p.PoolDefaults != nil || p.ResolvePool("A").ValidationCommand != "" {
		t.Fatalf("global not cleared: %+v", p.PoolDefaults)
	}
}
