package activity

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/glefebvre/opensp8c/internal/watcher"
)

// Broadcaster notifies subscribers of a workspace about an in-memory event.
// watcher.WatcherService satisfies this interface.
type Broadcaster interface {
	Broadcast(workspaceID string, ev watcher.Event)
}

// Entry represents an activity item in the chronological timeline.
type Entry struct {
	Ts         string         `json:"ts"`
	Type       string         `json:"type"`
	Category   string         `json:"category"`
	Summary    string         `json:"summary"`
	DurationMs *int64         `json:"durationMs,omitempty"`
	Meta       map[string]any `json:"meta,omitempty"`
}

// Store persists and retrieves non-agent activity entries as JSONL.
type Store struct {
	basePath    string
	broadcaster Broadcaster
	mu          sync.Mutex
}

// NewStore creates a new Store rooted at basePath. broadcaster may be nil.
func NewStore(basePath string, broadcaster Broadcaster) *Store {
	return &Store{
		basePath:    basePath,
		broadcaster: broadcaster,
	}
}

// Dir returns the directory for a workspace and change's activity logs.
func (s *Store) Dir(wsID, changeName string) string {
	return filepath.Join(s.basePath, wsID, changeName)
}

// FilePath returns the full path to activity.jsonl for a workspace and change.
func (s *Store) FilePath(wsID, changeName string) string {
	return filepath.Join(s.Dir(wsID, changeName), "activity.jsonl")
}

// Append writes an entry to <basePath>/<wsID>/<changeName>/activity.jsonl.
// Failures are logged server-side and do not return an error to the caller.
// On success, an activity_appended event is broadcasted if a Broadcaster is set.
func (s *Store) Append(wsID, changeName string, entry Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry.Ts == "" {
		entry.Ts = time.Now().UTC().Format(time.RFC3339Nano)
	}

	dir := s.Dir(wsID, changeName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("activity: failed to create dir %s: %v", dir, err)
		return nil
	}

	path := s.FilePath(wsID, changeName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("activity: failed to open %s: %v", path, err)
		return nil
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("activity: failed to marshal entry: %v", err)
		return nil
	}
	data = append(data, '\n')

	if _, err := f.Write(data); err != nil {
		log.Printf("activity: failed to write entry to %s: %v", path, err)
		return nil
	}

	if s.broadcaster != nil {
		s.broadcaster.Broadcast(wsID, watcher.Event{
			Type: "activity_appended",
			Name: changeName,
		})
	}

	return nil
}

// Read returns all persisted entries for a change, sorted by Ts ascending.
// If the file does not exist, an empty slice and nil error are returned.
func (s *Store) Read(wsID, changeName string) ([]Entry, error) {
	path := s.FilePath(wsID, changeName)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			log.Printf("activity: skipping corrupt entry in %s: %v", path, err)
			continue
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Ts < entries[j].Ts
	})

	if entries == nil {
		entries = []Entry{}
	}
	return entries, nil
}

// DeleteChangeActivity removes the activity directory for the given workspace and change.
func (s *Store) DeleteChangeActivity(wsID, changeName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.RemoveAll(filepath.Join(s.basePath, wsID, changeName))
}
