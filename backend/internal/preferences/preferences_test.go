package preferences

import (
	"os"
	"path/filepath"
	"testing"
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
