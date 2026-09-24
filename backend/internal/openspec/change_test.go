package openspec

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeChangeFixture(t *testing.T, changesDir, name, openspecYAML, tasksMd string) {
	t.Helper()
	dir := filepath.Join(changesDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".openspec.yaml"), []byte(openspecYAML), 0644); err != nil {
		t.Fatalf("WriteFile .openspec.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(tasksMd), 0644); err != nil {
		t.Fatalf("WriteFile tasks.md: %v", err)
	}
}

func TestListChangesParsesTagsWithoutAgentSpecialization(t *testing.T) {
	workspacePath := t.TempDir()
	changesDir := filepath.Join(workspacePath, "openspec", "changes")

	writeChangeFixture(t, changesDir, "legacy-change", `
schema: spec-driven
created: "2024-01-01"
tags:
  type:
    - backend
  complexity: 2
  components:
    - kanban
  _auto: true
  _tagged_at: "2024-01-02"
`, "- [x] done\n")

	changes, err := ListChanges(workspacePath)
	if err != nil {
		t.Fatalf("ListChanges: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	tags := changes[0].Tags
	if tags == nil {
		t.Fatal("expected tags to be parsed")
	}
	if !reflect.DeepEqual(tags.Type, []string{"backend"}) || tags.Complexity != 2 {
		t.Errorf("unexpected tags: %+v", tags)
	}
	if tags.AgentSpecialization == nil || len(tags.AgentSpecialization) != 0 {
		t.Errorf("expected agent_specialization to be an empty slice, got %v", tags.AgentSpecialization)
	}
}

func TestListChangesParsesTagsWithAgentSpecialization(t *testing.T) {
	workspacePath := t.TempDir()
	changesDir := filepath.Join(workspacePath, "openspec", "changes")

	writeChangeFixture(t, changesDir, "new-change", `
schema: spec-driven
created: "2024-01-01"
tags:
  type:
    - backend
  complexity: 2
  components:
    - kanban
  agent_specialization:
    - backend
    - database
  _auto: true
  _tagged_at: "2024-01-02"
`, "- [x] done\n")

	changes, err := ListChanges(workspacePath)
	if err != nil {
		t.Fatalf("ListChanges: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	tags := changes[0].Tags
	if tags == nil {
		t.Fatal("expected tags to be parsed")
	}
	if len(tags.AgentSpecialization) != 2 || tags.AgentSpecialization[0] != "backend" || tags.AgentSpecialization[1] != "database" {
		t.Errorf("unexpected agent_specialization: %v", tags.AgentSpecialization)
	}
}
