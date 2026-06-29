package conversation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStore_OpenRunAndList(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	f1, err := s.OpenRun("ws1", "my-change", "ff", "2026-06-29T14-00-00Z")
	if err != nil {
		t.Fatalf("OpenRun: %v", err)
	}
	f1.Write([]byte(`{"type":"text"}`))
	f1.Write([]byte("\n"))
	f1.Close()

	f2, err := s.OpenRun("ws1", "my-change", "ff", "2026-06-29T15-00-00Z")
	if err != nil {
		t.Fatalf("OpenRun second: %v", err)
	}
	f2.Write([]byte(`{"type":"done"}`))
	f2.Write([]byte("\n"))
	f2.Close()

	runs, err := s.List("ws1", "my-change", "ff")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
	// antéchronologique
	if runs[0].Ts != "2026-06-29T15-00-00Z" {
		t.Errorf("expected most recent first, got %s", runs[0].Ts)
	}
	if runs[0].MessageCount != 1 {
		t.Errorf("expected 1 message, got %d", runs[0].MessageCount)
	}
}

func TestStore_Load(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	f, _ := s.OpenRun("ws1", "ch", "ff", "2026-06-29T10-00-00Z")
	f.Write([]byte(`{"a":1}` + "\n"))
	f.Write([]byte(`{"b":2}` + "\n"))
	f.Close()

	lines, err := s.Load("ws1", "ch", "ff", "2026-06-29T10-00-00Z")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if string(lines[0]) != `{"a":1}` {
		t.Errorf("unexpected line 0: %s", lines[0])
	}
}

func TestStore_LoadNotFound(t *testing.T) {
	s := NewStore(t.TempDir())
	_, err := s.Load("ws1", "ch", "ff", "2026-06-29T99-99-99Z")
	if err == nil {
		t.Fatal("expected error for missing run")
	}
}

func TestStore_ListEmpty(t *testing.T) {
	s := NewStore(t.TempDir())
	runs, err := s.List("ws1", "ch", "ff")
	if err != nil {
		t.Fatalf("List on empty: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("expected empty list, got %d", len(runs))
	}
}

func TestStore_PartialFile(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	// Write partial content (no trailing newline — simulates crash mid-write)
	runDir := filepath.Join(dir, "ws1", "ch", "ff")
	os.MkdirAll(runDir, 0755)
	os.WriteFile(filepath.Join(runDir, "2026-06-29T12-00-00Z.jsonl"), []byte(`{"partial":true}`), 0644)

	lines, err := s.Load("ws1", "ch", "ff", "2026-06-29T12-00-00Z")
	if err != nil {
		t.Fatalf("Load partial: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line from partial file, got %d", len(lines))
	}
}
