package openspec

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
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

func TestDeriveStatus_ReadyWhenNotLaunched(t *testing.T) {
	if got := deriveStatus(0, 3, false); got != "ready" {
		t.Errorf("expected \"ready\", got %q", got)
	}
}

func TestDeriveStatus_TodoWhenLaunched(t *testing.T) {
	if got := deriveStatus(0, 3, true); got != "todo" {
		t.Errorf("expected \"todo\", got %q", got)
	}
}

func TestListChangesLaunchedAbsentIsBackwardCompatible(t *testing.T) {
	workspacePath := t.TempDir()
	changesDir := filepath.Join(workspacePath, "openspec", "changes")

	writeChangeFixture(t, changesDir, "legacy-no-launched", `
schema: spec-driven
created: "2024-01-01"
`, "- [ ] todo item\n")

	changes, err := ListChanges(workspacePath)
	if err != nil {
		t.Fatalf("ListChanges: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].KanbanStatus != "todo" {
		t.Errorf("expected kanban_status \"todo\" for change without launched field, got %q", changes[0].KanbanStatus)
	}
}

func TestSetLaunchedRoundTrip(t *testing.T) {
	workspacePath := t.TempDir()
	changesDir := filepath.Join(workspacePath, "openspec", "changes")
	writeChangeFixture(t, changesDir, "my-change", `
schema: spec-driven
created: "2024-01-01"
`, "- [ ] item\n")
	changeRoot := filepath.Join(changesDir, "my-change")

	if err := SetLaunched(changeRoot, true); err != nil {
		t.Fatalf("SetLaunched: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(changeRoot, ".openspec.yaml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var meta openspecMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if meta.Launched == nil || !*meta.Launched {
		t.Fatalf("expected launched=true, got %+v", meta.Launched)
	}

	if err := SetLaunched(changeRoot, false); err != nil {
		t.Fatalf("SetLaunched: %v", err)
	}
	data, err = os.ReadFile(filepath.Join(changeRoot, ".openspec.yaml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	meta = openspecMeta{}
	if err := yaml.Unmarshal(data, &meta); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if meta.Launched == nil || *meta.Launched {
		t.Fatalf("expected launched=false, got %+v", meta.Launched)
	}
}

func TestClearKanbanState(t *testing.T) {
	workspacePath := t.TempDir()
	changesDir := filepath.Join(workspacePath, "openspec", "changes")
	writeChangeFixture(t, changesDir, "my-change", `
schema: spec-driven
created: "2024-01-01"
launched: false
order: 3
`, "- [ ] item\n")
	changeRoot := filepath.Join(changesDir, "my-change")

	if err := ClearKanbanState(changeRoot); err != nil {
		t.Fatalf("ClearKanbanState: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(changeRoot, ".openspec.yaml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var meta openspecMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if meta.Launched != nil {
		t.Errorf("expected launched to be cleared, got %+v", meta.Launched)
	}
	if meta.Order != nil {
		t.Errorf("expected order to be cleared, got %+v", meta.Order)
	}
}

func TestReorderReady(t *testing.T) {
	workspacePath := t.TempDir()
	changesDir := filepath.Join(workspacePath, "openspec", "changes")
	for _, name := range []string{"change-a", "change-b", "change-c"} {
		writeChangeFixture(t, changesDir, name, `
schema: spec-driven
created: "2024-01-01"
launched: false
`, "- [ ] item\n")
	}

	if err := ReorderReady(changesDir, []string{"change-c", "change-a", "change-b"}); err != nil {
		t.Fatalf("ReorderReady: %v", err)
	}

	changes, err := ListChanges(workspacePath)
	if err != nil {
		t.Fatalf("ListChanges: %v", err)
	}
	orders := make(map[string]int, len(changes))
	for _, c := range changes {
		orders[c.Name] = c.Order
	}
	if orders["change-c"] != 1 || orders["change-a"] != 2 || orders["change-b"] != 3 {
		t.Errorf("unexpected orders: %+v", orders)
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

// TestParseTaskProgress verifies that ParseTaskProgress counts checked vs.
// total checklist items from an arbitrary tasks.md path, since pool.Manager
// reuses it to read a worktree's tasks.md rather than the original change's.
func TestParseTaskProgress(t *testing.T) {
	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.md")
	content := `# Tasks

- [x] done task
- [ ] pending task
- [X] also done (uppercase)
Not a task line, ignored.
- [ ] another pending task
`
	if err := os.WriteFile(tasksPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	done, total := ParseTaskProgress(tasksPath)
	if done != 2 {
		t.Errorf("expected 2 done tasks, got %d", done)
	}
	if total != 4 {
		t.Errorf("expected 4 total tasks, got %d", total)
	}
}

// TestParseTaskProgress_MissingFile verifies the fail-soft zero/zero result
// for a path that doesn't exist (e.g. a worktree that failed to provision).
func TestParseTaskProgress_MissingFile(t *testing.T) {
	done, total := ParseTaskProgress(filepath.Join(t.TempDir(), "missing", "tasks.md"))
	if done != 0 || total != 0 {
		t.Errorf("expected (0, 0) for a missing file, got (%d, %d)", done, total)
	}
}
