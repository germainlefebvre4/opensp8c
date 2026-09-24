package activity

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/watcher"
)

type mockBroadcaster struct {
	mu     sync.Mutex
	events []struct {
		wsID string
		ev   watcher.Event
	}
}

func (m *mockBroadcaster) Broadcast(wsID string, ev watcher.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, struct {
		wsID string
		ev   watcher.Event
	}{wsID: wsID, ev: ev})
}

func (m *mockBroadcaster) getEvents() []struct {
	wsID string
	ev   watcher.Event
} {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]struct {
		wsID string
		ev   watcher.Event
	}, len(m.events))
	copy(copied, m.events)
	return copied
}

func TestStore_AppendThreeEntries(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, nil)

	wsID := "ws-1"
	changeName := "change-a"

	entries := []Entry{
		{
			Ts:       "2026-09-24T10:00:00Z",
			Type:     "kanban.task_toggled",
			Category: "kanban",
			Summary:  "Task 1 completed",
			Meta:     map[string]any{"task": "Task 1", "done": true},
		},
		{
			Ts:       "2026-09-24T10:01:00Z",
			Type:     "kanban.ff_triggered",
			Category: "kanban",
			Summary:  "Run ff started",
		},
		{
			Ts:       "2026-09-24T10:02:00Z",
			Type:     "pool.worker_status",
			Category: "pool",
			Summary:  "Worker changed to working",
		},
	}

	for _, e := range entries {
		if err := store.Append(wsID, changeName, e); err != nil {
			t.Fatalf("unexpected error on Append: %v", err)
		}
	}

	filePath := store.FilePath(wsID, changeName)
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to open activity.jsonl: %v", err)
	}
	defer f.Close()

	var readEntries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("failed to unmarshal entry line: %v", err)
		}
		readEntries = append(readEntries, entry)
	}

	if len(readEntries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(readEntries))
	}
	for i, expected := range entries {
		if readEntries[i].Summary != expected.Summary {
			t.Errorf("entry %d: expected summary %q, got %q", i, expected.Summary, readEntries[i].Summary)
		}
		if readEntries[i].Type != expected.Type {
			t.Errorf("entry %d: expected type %q, got %q", i, expected.Type, readEntries[i].Type)
		}
	}
}

func TestStore_Read(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, nil)

	wsID := "ws-1"
	changeName := "change-b"

	// 1.2: Absent file returns empty list, no error
	entries, err := store.Read(wsID, changeName)
	if err != nil {
		t.Fatalf("unexpected error reading absent file: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries for absent file, got %d", len(entries))
	}

	// 1.2: File with multiple lines: order preserved (sorted by Ts ascending)
	e1 := Entry{Ts: "2026-09-24T10:05:00Z", Type: "b", Category: "cat", Summary: "Second"}
	e2 := Entry{Ts: "2026-09-24T10:01:00Z", Type: "a", Category: "cat", Summary: "First"}
	e3 := Entry{Ts: "2026-09-24T10:10:00Z", Type: "c", Category: "cat", Summary: "Third"}

	if err := store.Append(wsID, changeName, e1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := store.Append(wsID, changeName, e2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := store.Append(wsID, changeName, e3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	read, err := store.Read(wsID, changeName)
	if err != nil {
		t.Fatalf("unexpected error on Read: %v", err)
	}
	if len(read) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(read))
	}
	if read[0].Summary != "First" || read[1].Summary != "Second" || read[2].Summary != "Third" {
		t.Errorf("entries not sorted by ts ascending: %+v", read)
	}
}

func TestStore_Append_NonBlockingOnError(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a regular file where the workspace directory would be,
	// so MkdirAll or OpenFile will fail
	blockingFile := filepath.Join(tmpDir, "blocked-ws")
	if err := os.WriteFile(blockingFile, []byte("blocker"), 0644); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

	store := NewStore(blockingFile, nil)
	entry := Entry{
		Type:    "kanban.task_toggled",
		Summary: "Should not block caller",
	}

	// 1.3: Failure must not return error to caller
	err := store.Append("sub-path", "change", entry)
	if err != nil {
		t.Fatalf("expected Append to return nil on write failure, got: %v", err)
	}
}

func TestStore_BroadcastOnAppend(t *testing.T) {
	tmpDir := t.TempDir()
	broadcaster := &mockBroadcaster{}
	store := NewStore(tmpDir, broadcaster)

	wsID := "ws-test"
	changeName := "feature-x"

	entry := Entry{
		Ts:       time.Now().UTC().Format(time.RFC3339Nano),
		Type:     "git.commit",
		Category: "git",
		Summary:  "commit abc1234: test",
	}

	if err := store.Append(wsID, changeName, entry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := broadcaster.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 broadcast event, got %d", len(events))
	}
	if events[0].wsID != wsID {
		t.Errorf("expected wsID %q, got %q", wsID, events[0].wsID)
	}
	if events[0].ev.Type != "activity_appended" {
		t.Errorf("expected event Type %q, got %q", "activity_appended", events[0].ev.Type)
	}
	if events[0].ev.Name != changeName {
		t.Errorf("expected event Name %q, got %q", changeName, events[0].ev.Name)
	}
}
