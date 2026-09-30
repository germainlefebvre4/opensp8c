package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/preferences"
)

// installMockAgentCLI makes every supported agent look installed by pointing
// its CLI at a script that answers --version.
func installMockAgentCLI(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	cli := filepath.Join(dir, "mock-cli")
	if err := os.WriteFile(cli, []byte("#!/bin/sh\necho 'mock 1.0.0'\n"), 0755); err != nil {
		t.Fatal(err)
	}
	origs := make([]string, len(agents.SupportedAgents))
	for i := range agents.SupportedAgents {
		origs[i] = agents.SupportedAgents[i].CLI
		agents.SupportedAgents[i].CLI = cli
	}
	t.Cleanup(func() {
		for i := range agents.SupportedAgents {
			agents.SupportedAgents[i].CLI = origs[i]
		}
	})
}

func patchSettings(t *testing.T, svc *preferences.Service, ws, j string) {
	t.Helper()
	var p preferences.AgentSettingsPatch
	if err := jsonUnmarshal(j, &p); err != nil {
		t.Fatal(err)
	}
	var err error
	if ws == "" {
		err = svc.SetAgentSettings(p)
	} else {
		err = svc.PatchWorkspace(ws, preferences.WorkspaceSettingsPatch{AgentSettings: &p})
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestResolveRole_LockTakesPrecedenceAndAppliesModelEffort(t *testing.T) {
	installMockAgentCLI(t)
	svc := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	patchSettings(t, svc, "", `{"roles":{"explorer":{"agent":"codex","model":"gpt-x"}}}`)
	m := NewManager(svc, nil)

	// Existing session locked on claude: keeps claude, with claude's preset.
	r := m.resolveRole("ws", preferences.RoleExplorer, "claude")
	if r.config.ID != "claude" || r.config.Model != "opus" || r.config.Effort != "high" {
		t.Errorf("locked: %s %s %s", r.config.ID, r.config.Model, r.config.Effort)
	}

	// New session without lock: role agent wins.
	r = m.resolveRole("ws", preferences.RoleExplorer, "")
	if r.config.ID != "codex" || r.config.Model != "gpt-x" || r.config.Effort != "" {
		t.Errorf("unlocked: %s %s %s", r.config.ID, r.config.Model, r.config.Effort)
	}
}

func TestResolveRole_SettingChangeDoesNotAffectEarlierResolution(t *testing.T) {
	installMockAgentCLI(t)
	svc := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	m := NewManager(svc, nil)
	first := m.resolveRole("ws", preferences.RoleExplorer, "").config
	patchSettings(t, svc, "", `{"roles":{"explorer":{"model":"haiku"}}}`)
	second := m.resolveRole("ws", preferences.RoleExplorer, "").config
	if first.Model != "opus" || second.Model != "haiku" {
		t.Errorf("first %q second %q", first.Model, second.Model)
	}
}

func TestResolveRole_WorkspaceIsolation(t *testing.T) {
	installMockAgentCLI(t)
	svc := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	patchSettings(t, svc, "A", `{"roles":{"implementer":{"model":"opus"}}}`)
	m := NewManager(svc, nil)
	if got := m.ResolveRoleConfig("A", preferences.RoleImplementer).Model; got != "opus" {
		t.Errorf("A: %q", got)
	}
	if got := m.ResolveRoleConfig("B", preferences.RoleImplementer).Model; got != "sonnet" {
		t.Errorf("B: %q", got)
	}
}

func jsonUnmarshal(s string, v any) error { return json.Unmarshal([]byte(s), v) }
