package preferences

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glefebvre/opensp8c/internal/agents"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	return NewService(filepath.Join(dir, "preferences.json"))
}

func TestAddAndListExplorations(t *testing.T) {
	svc := newTestService(t)

	if err := svc.AddExploration(ExplorationRecord{ID: "a1", WorkspaceID: "ws1", Name: "feat-one", SessionID: "s1"}); err != nil {
		t.Fatalf("AddExploration: %v", err)
	}
	if err := svc.AddExploration(ExplorationRecord{ID: "b2", WorkspaceID: "ws2", Name: "feat-two", SessionID: "s2"}); err != nil {
		t.Fatalf("AddExploration: %v", err)
	}

	list := svc.ListExplorations("ws1")
	if len(list) != 1 {
		t.Fatalf("expected 1 record for ws1, got %d", len(list))
	}
	if list[0].ID != "a1" || list[0].Name != "feat-one" {
		t.Errorf("unexpected record: %+v", list[0])
	}

	// Other workspace not leaked
	list2 := svc.ListExplorations("ws2")
	if len(list2) != 1 || list2[0].ID != "b2" {
		t.Errorf("unexpected ws2 list: %+v", list2)
	}

	// Unknown workspace returns empty
	empty := svc.ListExplorations("unknown")
	if len(empty) != 0 {
		t.Errorf("expected empty for unknown workspace, got %v", empty)
	}
}

func TestGetExploration(t *testing.T) {
	svc := newTestService(t)

	svc.AddExploration(ExplorationRecord{ID: "x1", WorkspaceID: "wsA", Name: "my-feature", SessionID: "sess"})

	rec := svc.GetExploration("x1", "wsA")
	if rec == nil {
		t.Fatal("expected record, got nil")
	}
	if rec.Name != "my-feature" {
		t.Errorf("expected name 'my-feature', got %q", rec.Name)
	}

	// Wrong workspace → nil
	if got := svc.GetExploration("x1", "wsB"); got != nil {
		t.Errorf("expected nil for wrong workspace, got %+v", got)
	}

	// Unknown ID → nil
	if got := svc.GetExploration("unknown", "wsA"); got != nil {
		t.Errorf("expected nil for unknown id, got %+v", got)
	}
}

func TestUpdateExplorationName(t *testing.T) {
	svc := newTestService(t)

	svc.AddExploration(ExplorationRecord{ID: "r1", WorkspaceID: "ws1", Name: "explore-a3f8bc", SessionID: "s"})

	if err := svc.UpdateExplorationName("r1", "drag-drop-workspaces"); err != nil {
		t.Fatalf("UpdateExplorationName: %v", err)
	}

	rec := svc.GetExploration("r1", "ws1")
	if rec == nil {
		t.Fatal("record not found after update")
	}
	if rec.Name != "drag-drop-workspaces" {
		t.Errorf("expected updated name, got %q", rec.Name)
	}

	// Update non-existent ID is a no-op (not an error)
	if err := svc.UpdateExplorationName("nope", "new"); err != nil {
		t.Errorf("update non-existent should be no-op, got error: %v", err)
	}
}

func TestDeleteExploration(t *testing.T) {
	svc := newTestService(t)

	svc.AddExploration(ExplorationRecord{ID: "d1", WorkspaceID: "ws1", Name: "to-delete", SessionID: "s1"})
	svc.AddExploration(ExplorationRecord{ID: "d2", WorkspaceID: "ws1", Name: "keep-me", SessionID: "s2"})

	if err := svc.DeleteExploration("d1"); err != nil {
		t.Fatalf("DeleteExploration: %v", err)
	}

	if rec := svc.GetExploration("d1", "ws1"); rec != nil {
		t.Errorf("deleted record still present: %+v", rec)
	}

	// The other record is untouched
	if rec := svc.GetExploration("d2", "ws1"); rec == nil {
		t.Error("sibling record should not be deleted")
	}

	// Delete non-existent is a no-op
	if err := svc.DeleteExploration("nope"); err != nil {
		t.Errorf("delete non-existent should be no-op, got error: %v", err)
	}
}

func TestCreatedAtAutoSet(t *testing.T) {
	svc := newTestService(t)

	svc.AddExploration(ExplorationRecord{ID: "ts1", WorkspaceID: "ws1", Name: "test", SessionID: "s"})

	rec := svc.GetExploration("ts1", "ws1")
	if rec == nil {
		t.Fatal("record not found")
	}
	if rec.CreatedAt == "" {
		t.Error("CreatedAt should be auto-set when empty")
	}
}

func TestPersistenceAcrossReloads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prefs.json")

	svc1 := NewService(path)
	svc1.AddExploration(ExplorationRecord{ID: "p1", WorkspaceID: "ws1", Name: "persistent", SessionID: "s"})

	// New service instance reads same file
	svc2 := NewService(path)
	list := svc2.ListExplorations("ws1")
	if len(list) != 1 || list[0].ID != "p1" {
		t.Errorf("expected persisted record on reload, got %+v", list)
	}
}

func TestMissingFileReturnsEmpty(t *testing.T) {
	svc := NewService(filepath.Join(t.TempDir(), "nonexistent.json"))

	list := svc.ListExplorations("ws1")
	if len(list) != 0 {
		t.Errorf("expected empty list for missing file, got %v", list)
	}

	if rec := svc.GetExploration("any", "ws1"); rec != nil {
		t.Errorf("expected nil for missing file, got %+v", rec)
	}
}

func TestDefaultAgentPreservedWithExplorations(t *testing.T) {
	svc := newTestService(t)

	svc.SetDefaultAgent("cursor")
	svc.AddExploration(ExplorationRecord{ID: "e1", WorkspaceID: "ws1", Name: "test", SessionID: "s"})

	if agent := svc.GetDefaultAgent(); agent != "cursor" {
		t.Errorf("default agent should not be overwritten by exploration ops, got %q", agent)
	}
}

func TestSetCustomAgentSpecializationsPersistsAcrossReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prefs.json")

	svc1 := NewService(path)
	if err := svc1.SetCustomAgentSpecializations([]string{"ml-ops", "embedded"}); err != nil {
		t.Fatalf("SetCustomAgentSpecializations: %v", err)
	}

	svc2 := NewService(path)
	p, err := svc2.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(p.CustomAgentSpecializations) != 2 || p.CustomAgentSpecializations[0] != "ml-ops" || p.CustomAgentSpecializations[1] != "embedded" {
		t.Errorf("expected persisted custom specializations, got %v", p.CustomAgentSpecializations)
	}
}

func TestFileCreatedWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "prefs.json")

	svc := NewService(path)
	if err := svc.AddExploration(ExplorationRecord{ID: "f1", WorkspaceID: "ws1", Name: "test", SessionID: "s"}); err != nil {
		t.Fatalf("AddExploration should create parent dirs: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should have been created: %v", err)
	}
}

func TestLoad_EnsuresAgentEnvEntryPerSupportedAgent(t *testing.T) {
	// Fresh file
	svc := newTestService(t)
	p, err := svc.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, a := range agents.SupportedAgents {
		if e, ok := p.AgentEnv[a.ID]; !ok || e == nil || len(e) != 0 {
			t.Errorf("fresh: expected empty entry for %s, got %v (ok=%v)", a.ID, e, ok)
		}
	}

	// Existing file without agentEnv
	svc2 := newTestService(t)
	if err := os.WriteFile(svc2.Path(), []byte(`{"defaultAgent":"claude"}`), 0644); err != nil {
		t.Fatal(err)
	}
	p2, err := svc2.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, a := range agents.SupportedAgents {
		if e, ok := p2.AgentEnv[a.ID]; !ok || len(e) != 0 {
			t.Errorf("existing: expected empty entry for %s, got %v (ok=%v)", a.ID, e, ok)
		}
	}
}

func TestLoad_MigratesGeminiEnvOnce(t *testing.T) {
	svc := newTestService(t)
	legacy := `{"defaultAgent":"claude","env":{"GOOGLE_CLOUD_PROJECT":"proj","GEMINI_MODEL":"m","GEMINI_SANDBOX":"true","OTHER":"x"}}`
	if err := os.WriteFile(svc.Path(), []byte(legacy), 0644); err != nil {
		t.Fatal(err)
	}
	p, err := svc.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	g := p.AgentEnv["gemini"]
	if g["GOOGLE_CLOUD_PROJECT"] != "proj" || g["GEMINI_MODEL"] != "m" || g["GEMINI_SANDBOX"] != "true" {
		t.Errorf("gemini keys not migrated: %v", g)
	}
	for _, k := range []string{"GOOGLE_CLOUD_PROJECT", "GEMINI_MODEL", "GEMINI_SANDBOX"} {
		if _, ok := p.Env[k]; ok {
			t.Errorf("%s still in global env", k)
		}
	}
	if p.Env["OTHER"] != "x" {
		t.Errorf("unrelated key lost: %v", p.Env)
	}

	before, _ := os.ReadFile(svc.Path())
	p2, err := svc.Load()
	if err != nil {
		t.Fatalf("Load 2: %v", err)
	}
	after, _ := os.ReadFile(svc.Path())
	if string(before) != string(after) {
		t.Errorf("second Load modified the file")
	}
	if p2.AgentEnv["gemini"]["GEMINI_MODEL"] != "m" || len(p2.Env) != 1 {
		t.Errorf("state changed on second load: env=%v agentEnv=%v", p2.Env, p2.AgentEnv["gemini"])
	}
}

func TestEnvFor(t *testing.T) {
	p := &Preferences{
		Env: map[string]string{"FOO": "global", "BAR": "b"},
		AgentEnv: map[string]map[string]string{
			"gemini": {"FOO": "gemini", "G": "1"},
			"codex":  {"C": "1"},
		},
	}
	claude := p.EnvFor("claude")
	if len(claude) != 2 || claude["FOO"] != "global" || claude["BAR"] != "b" {
		t.Errorf("claude env should equal global env, got %v", claude)
	}
	g := p.EnvFor("gemini")
	if g["FOO"] != "gemini" || g["BAR"] != "b" || g["G"] != "1" {
		t.Errorf("unexpected gemini env: %v", g)
	}
	if _, ok := g["C"]; ok {
		t.Errorf("codex key leaked into gemini: %v", g)
	}
	if p.Env["FOO"] != "global" {
		t.Errorf("EnvFor mutated global env")
	}
}

func TestSetAgentEnv_OnlyTouchesGivenAgent(t *testing.T) {
	svc := newTestService(t)
	if err := svc.SetEnv(map[string]string{"G": "global"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetAgentEnv(map[string]map[string]string{"claude": {"A": "1"}}); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetAgentEnv(map[string]map[string]string{"gemini": {"B": "2"}}); err != nil {
		t.Fatal(err)
	}
	p, err := svc.Load()
	if err != nil {
		t.Fatal(err)
	}
	if p.Env["G"] != "global" || len(p.Env) != 1 {
		t.Errorf("global env modified: %v", p.Env)
	}
	if p.AgentEnv["claude"]["A"] != "1" || p.AgentEnv["gemini"]["B"] != "2" {
		t.Errorf("unexpected agentEnv: %v", p.AgentEnv)
	}
}
