package openspec

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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

func TestListChangesNormalizesMissingTagsType(t *testing.T) {
	cases := map[string]string{
		"missing-type": `
schema: spec-driven
tags:
  complexity: 2
  components:
    - kanban
`,
		"empty-type": `
schema: spec-driven
tags:
  type:
  complexity: 2
  components:
    - kanban
`,
	}

	for name, meta := range cases {
		t.Run(name, func(t *testing.T) {
			workspacePath := t.TempDir()
			changesDir := filepath.Join(workspacePath, "openspec", "changes")
			writeChangeFixture(t, changesDir, name, meta, "- [x] done\n")

			changes, err := ListChanges(workspacePath)
			if err != nil {
				t.Fatalf("ListChanges: %v", err)
			}
			if len(changes) != 1 || changes[0].Tags == nil {
				t.Fatalf("expected 1 change with tags, got %+v", changes)
			}
			if got := changes[0].Tags.Type; got == nil || len(got) != 0 {
				t.Errorf("expected type to be an empty slice, got %#v", got)
			}
		})
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

func TestSetLaunched_CreatesFileIfMissing(t *testing.T) {
	workspacePath := t.TempDir()
	changesDir := filepath.Join(workspacePath, "openspec", "changes")
	changeRoot := filepath.Join(changesDir, "legacy-no-yaml")
	if err := os.MkdirAll(changeRoot, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Verify .openspec.yaml does not exist
	if _, err := os.Stat(filepath.Join(changeRoot, ".openspec.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected .openspec.yaml to not exist")
	}

	// Call SetLaunched(changeRoot, false)
	if err := SetLaunched(changeRoot, false); err != nil {
		t.Fatalf("SetLaunched failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(changeRoot, ".openspec.yaml"))
	if err != nil {
		t.Fatalf("expected .openspec.yaml to be created: %v", err)
	}

	var meta openspecMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if meta.Schema != "spec-driven" {
		t.Errorf("expected schema to default to 'spec-driven', got %q", meta.Schema)
	}
	if meta.Launched == nil || *meta.Launched {
		t.Errorf("expected launched=false, got %+v", meta.Launched)
	}

	// Call SetLaunched(changeRoot, true) to ensure subsequent update works
	if err := SetLaunched(changeRoot, true); err != nil {
		t.Fatalf("SetLaunched failed: %v", err)
	}
	data, err = os.ReadFile(filepath.Join(changeRoot, ".openspec.yaml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	meta = openspecMeta{}
	if err := yaml.Unmarshal(data, &meta); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if meta.Launched == nil || !*meta.Launched {
		t.Errorf("expected launched=true, got %+v", meta.Launched)
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

// writeWorktreeTasks writes a worktree tasks.md for change and returns the
// worktree root.
func writeWorktreeTasks(t *testing.T, change, tasksMd string) string {
	t.Helper()
	wt := t.TempDir()
	dir := filepath.Join(wt, "openspec", "changes", change)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if tasksMd != "" {
		if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(tasksMd), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	return wt
}

func TestApplyWorktreeProgress(t *testing.T) {
	const mainTasks = "- [ ] a\n- [ ] b\n- [ ] c\n- [ ] d\n- [ ] e\n"
	cases := []struct {
		name       string
		worktree   string // tasks.md content; "-" = no worktree path
		wantStatus string
		wantDone   int
		wantTotal  int
		applied    bool
	}{
		{"partial", "- [x] a\n- [x] b\n- [ ] c\n- [ ] d\n- [ ] e\n", "in-progress", 2, 5, true},
		{"fully checked capped", "- [x] a\n- [x] b\n", "in-progress", 2, 2, true},
		{"none checked stays todo", "- [ ] a\n- [ ] b\n", "todo", 0, 2, true},
		{"missing file", "", "todo", 0, 5, false},
		{"empty file", "# nothing\n", "todo", 0, 5, false},
		{"no worktree path", "-", "todo", 0, 5, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workspacePath := t.TempDir()
			changesDir := filepath.Join(workspacePath, "openspec", "changes")
			writeChangeFixture(t, changesDir, "c1", "schema: spec-driven\n", mainTasks)
			wtPath := ""
			if tc.worktree != "-" {
				wtPath = writeWorktreeTasks(t, "c1", tc.worktree)
			}
			changes, err := ListChanges(workspacePath)
			if err != nil || len(changes) != 1 {
				t.Fatalf("ListChanges: %v %v", changes, err)
			}
			ch := changes[0]
			if got := ApplyWorktreeProgress(&ch, workspacePath, wtPath); got != tc.applied {
				t.Errorf("applied = %v, want %v", got, tc.applied)
			}
			if ch.KanbanStatus != tc.wantStatus || ch.TasksDone != tc.wantDone || ch.TasksTotal != tc.wantTotal {
				t.Errorf("got %s %d/%d, want %s %d/%d", ch.KanbanStatus, ch.TasksDone, ch.TasksTotal, tc.wantStatus, tc.wantDone, tc.wantTotal)
			}
		})
	}
}

func TestApplyWorktreeProgress_StalenessFromWorktree(t *testing.T) {
	workspacePath := t.TempDir()
	changesDir := filepath.Join(workspacePath, "openspec", "changes")
	writeChangeFixture(t, changesDir, "c1", "schema: spec-driven\n", "- [x] a\n- [ ] b\n")
	old := time.Now().Add(-10 * 24 * time.Hour)
	mainTasks := filepath.Join(changesDir, "c1", "tasks.md")
	if err := os.Chtimes(mainTasks, old, old); err != nil {
		t.Fatal(err)
	}
	wt := writeWorktreeTasks(t, "c1", "- [x] a\n- [ ] b\n")

	changes, _ := ListChanges(workspacePath)
	ch := changes[0]
	if !ch.IsStale {
		t.Fatalf("precondition: main repo change should be stale")
	}
	ApplyWorktreeProgress(&ch, workspacePath, wt)
	if ch.DaysSinceActivity != 0 || ch.IsStale {
		t.Errorf("got days=%d stale=%v, want 0/false", ch.DaysSinceActivity, ch.IsStale)
	}
}

func TestGetChangeDetail_FollowsWorktree(t *testing.T) {
	workspacePath := t.TempDir()
	changesDir := filepath.Join(workspacePath, "openspec", "changes")
	writeChangeFixture(t, changesDir, "c1", "schema: spec-driven\n", "- [ ] a\n- [ ] b\n- [ ] c\n")
	wt := writeWorktreeTasks(t, "c1", "- [x] a\n- [x] b\n- [ ] c\n")

	d, err := GetChangeDetail(workspacePath, "c1", wt)
	if err != nil {
		t.Fatal(err)
	}
	if d.TasksDone != 2 || len(d.Tasks) != 3 || !d.Tasks[0].Done || !d.Tasks[1].Done || d.KanbanStatus != "in-progress" {
		t.Errorf("worktree detail wrong: %+v", d)
	}

	d, err = GetChangeDetail(workspacePath, "c1", "")
	if err != nil {
		t.Fatal(err)
	}
	if d.TasksDone != 0 || d.Tasks[0].Done || d.KanbanStatus != "todo" {
		t.Errorf("main detail wrong: %+v", d)
	}
}

func TestParseTaskList_HumanReviewMarker(t *testing.T) {
	content := `- [ ] 4.2 Parcours manuel <!-- human review required -->
- [x] 4.3 Fait <!--Human Review Required-->
- [ ] 4.4 Ordinaire
Texte <!-- human review required --> hors tâche
`
	tasks := ParseTaskListContent(content)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	if tasks[0].Text != "4.2 Parcours manuel" || !tasks[0].HumanReview || tasks[0].Done {
		t.Errorf("unexpected task 0: %+v", tasks[0])
	}
	if tasks[1].Text != "4.3 Fait" || !tasks[1].HumanReview || !tasks[1].Done {
		t.Errorf("unexpected task 1: %+v", tasks[1])
	}
	if tasks[2].HumanReview {
		t.Errorf("task 2 must not be marked: %+v", tasks[2])
	}
}

func TestParseTaskStats_DistinguishesHumanTasks(t *testing.T) {
	content := "- [x] a\n- [x] b <!-- human review required -->\n- [ ] c <!-- human review required -->\n- [ ] d <!-- human review required -->\n- [ ] e\n"
	st := ParseTaskStatsContent(content)
	want := TaskStats{Done: 2, Total: 5, PendingHuman: 2, PendingOther: 1}
	if st != want {
		t.Errorf("got %+v, want %+v", st, want)
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "tasks.md")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if done, total := ParseTaskProgress(p); done != 2 || total != 5 {
		t.Errorf("ParseTaskProgress = %d/%d, want 2/5", done, total)
	}
	if got := ParseTaskStats(p); got != want {
		t.Errorf("ParseTaskStats = %+v, want %+v", got, want)
	}
}

func TestToggleTask_KeepsMarkerAndCleansText(t *testing.T) {
	ws := t.TempDir()
	dir := filepath.Join(ws, "openspec", "changes", "c")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "tasks.md")
	if err := os.WriteFile(p, []byte("- [ ] Parcours <!-- human review required -->\n"), 0644); err != nil {
		t.Fatal(err)
	}
	text, done, err := ToggleTask(ws, "c", 0)
	if err != nil || !done || text != "Parcours" {
		t.Fatalf("got (%q, %v, %v)", text, done, err)
	}
	data, _ := os.ReadFile(p)
	if string(data) != "- [x] Parcours <!-- human review required -->\n" {
		t.Errorf("unexpected file: %q", data)
	}
}

func TestChangeScenarios(t *testing.T) {
	root := t.TempDir()
	specs := filepath.Join(root, "openspec", "changes", "c", "specs")
	write := func(rel, content string) string {
		p := filepath.Join(specs, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	write("b-cap/spec.md", "## ADDED Requirements\n\n### Requirement: R\ntext\n\n#### Scenario: Premier\n- **WHEN** a\n- **THEN** b\n- **AND** c\n\n#### Scenario: Sans then\n- **WHEN** seul\n\n### Requirement: Autre\n- **THEN** hors scénario\n")
	write("a-cap/spec.md", "#### Scenario: Unique\n- **WHEN** x\n- **THEN** y\n")
	write("a-cap/notes.md", "#### Scenario: Ignoré\n- **WHEN** z\n")
	write("empty/spec.md", "# rien\n")
	if p := write("locked/spec.md", "#### Scenario: Illisible\n- **WHEN** q\n"); os.Chmod(p, 0) == nil && os.Geteuid() != 0 {
		defer os.Chmod(p, 0o644)
	}

	got := ChangeScenarios(root, "c")
	if len(got) < 2 || got[0].File != "a-cap/spec.md" {
		t.Fatalf("expected files in alphabetical order, got %+v", got)
	}
	var b SpecScenarios
	for _, g := range got {
		if g.File == "locked/spec.md" && os.Geteuid() != 0 {
			t.Fatal("unreadable file must be skipped")
		}
		if g.File == "b-cap/spec.md" {
			b = g
		}
		if g.File == "empty/spec.md" || strings.HasSuffix(g.File, "notes.md") {
			t.Fatalf("unexpected file %s", g.File)
		}
	}
	if len(b.Scenarios) != 2 || b.Scenarios[0].Name != "Premier" || len(b.Scenarios[0].Lines) != 3 {
		t.Fatalf("b-cap: %+v", b)
	}
	if s := b.Scenarios[1]; s.Name != "Sans then" || len(s.Lines) != 1 || s.Lines[0] != "- **WHEN** seul" {
		t.Fatalf("scenario without THEN must be kept as is: %+v", s)
	}
	if got := ChangeScenarios(root, "absent"); len(got) != 0 {
		t.Fatalf("change without delta spec: %+v", got)
	}
}
