package watcher

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Event struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Error string `json:"error,omitempty"`
}

type WatcherService struct {
	mu       sync.Mutex
	watchers map[string]*workspaceWatcher
}

func NewWatcherService() *WatcherService {
	return &WatcherService{
		watchers: make(map[string]*workspaceWatcher),
	}
}

func (s *WatcherService) StartWatching(workspaceID, workspacePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.watchers[workspaceID]; exists {
		return nil
	}

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	ww := &workspaceWatcher{
		fw:            fw,
		workspacePath: workspacePath,
		debouncers:    make(map[string]*time.Timer),
		subscribers:   nil,
	}

	openspecDir := filepath.Join(workspacePath, "openspec")
	if err := fw.Add(openspecDir); err != nil {
		fw.Close()
		return err
	}

	changesDir := filepath.Join(workspacePath, "openspec", "changes")
	ww.tryAddChangesDir(changesDir)

	s.watchers[workspaceID] = ww
	go ww.run()
	return nil
}

func (s *WatcherService) StopWatching(workspaceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ww, ok := s.watchers[workspaceID]
	if !ok {
		return
	}
	ww.stop()
	delete(s.watchers, workspaceID)
}

func (s *WatcherService) Subscribe(workspaceID string) chan Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	ww, ok := s.watchers[workspaceID]
	if !ok {
		return nil
	}
	return ww.subscribe()
}

func (s *WatcherService) Broadcast(workspaceID string, ev Event) {
	s.mu.Lock()
	ww, ok := s.watchers[workspaceID]
	s.mu.Unlock()
	if ok {
		ww.broadcast(ev)
	}
}

func (s *WatcherService) Unsubscribe(workspaceID string, ch chan Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ww, ok := s.watchers[workspaceID]
	if !ok {
		return
	}
	ww.unsubscribe(ch)
}

type workspaceWatcher struct {
	mu            sync.Mutex
	fw            *fsnotify.Watcher
	workspacePath string
	debouncers    map[string]*time.Timer
	subscribers   []chan Event
}

func (ww *workspaceWatcher) subscribe() chan Event {
	ch := make(chan Event, 8)
	ww.mu.Lock()
	ww.subscribers = append(ww.subscribers, ch)
	ww.mu.Unlock()
	return ch
}

func (ww *workspaceWatcher) unsubscribe(ch chan Event) {
	ww.mu.Lock()
	defer ww.mu.Unlock()
	updated := ww.subscribers[:0]
	for _, s := range ww.subscribers {
		if s != ch {
			updated = append(updated, s)
		}
	}
	ww.subscribers = updated
	close(ch)
}

func (ww *workspaceWatcher) broadcast(ev Event) {
	ww.mu.Lock()
	defer ww.mu.Unlock()
	for _, ch := range ww.subscribers {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (ww *workspaceWatcher) debounce(changeName string, ev Event) {
	ww.mu.Lock()
	if t, ok := ww.debouncers[changeName]; ok {
		t.Stop()
	}
	ww.debouncers[changeName] = time.AfterFunc(150*time.Millisecond, func() {
		ww.broadcast(ev)
		ww.mu.Lock()
		delete(ww.debouncers, changeName)
		ww.mu.Unlock()
	})
	ww.mu.Unlock()
}

func (ww *workspaceWatcher) stop() {
	ww.mu.Lock()
	for _, t := range ww.debouncers {
		t.Stop()
	}
	ww.mu.Unlock()
	ww.fw.Close()
}

func (ww *workspaceWatcher) tryAddChangesDir(changesDir string) {
	if _, err := os.Stat(changesDir); err != nil {
		return
	}
	_ = ww.fw.Add(changesDir)

	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		_ = ww.fw.Add(filepath.Join(changesDir, e.Name()))
	}

	archiveDir := filepath.Join(changesDir, "archive")
	if _, err := os.Stat(archiveDir); err == nil {
		_ = ww.fw.Add(archiveDir)
	}
}

func (ww *workspaceWatcher) run() {
	openspecDir := filepath.Join(ww.workspacePath, "openspec")
	changesDir := filepath.Join(openspecDir, "changes")
	archiveDir := filepath.Join(changesDir, "archive")

	for {
		select {
		case ev, ok := <-ww.fw.Events:
			if !ok {
				return
			}
			ww.handleEvent(ev, changesDir, archiveDir)
		case _, ok := <-ww.fw.Errors:
			if !ok {
				return
			}
		}
	}
}

func (ww *workspaceWatcher) handleEvent(ev fsnotify.Event, changesDir, archiveDir string) {
	path := ev.Name

	// openspec/ → watch changes/ when created
	if path == changesDir && ev.Has(fsnotify.Create) {
		ww.tryAddChangesDir(changesDir)
		return
	}

	// changes/ → detect new change dirs
	parentDir := filepath.Dir(path)
	name := filepath.Base(path)

	if parentDir == changesDir {
		if ev.Has(fsnotify.Create) {
			info, err := os.Stat(path)
			if err == nil && info.IsDir() {
				if name == "archive" {
					_ = ww.fw.Add(path)
					return
				}
				_ = ww.fw.Add(path)
				ww.broadcast(Event{Type: "change_created", Name: name})
			}
			return
		}
		if ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename) {
			if name != "archive" {
				ww.broadcast(Event{Type: "change_deleted", Name: name})
			}
			return
		}
	}

	// archive/ → detect archival (change moved here = change_deleted from active)
	if parentDir == archiveDir {
		if ev.Has(fsnotify.Create) {
			info, err := os.Stat(path)
			if err == nil && info.IsDir() {
				ww.broadcast(Event{Type: "change_deleted", Name: name})
			}
		}
		return
	}

	// changes/<name>/ → debounce on relevant file writes
	grandParent := filepath.Dir(parentDir)
	if grandParent == changesDir {
		changeName := filepath.Base(parentDir)
		if ev.Has(fsnotify.Write) || ev.Has(fsnotify.Create) {
			if name == "tasks.md" || name == ".openspec.yaml" {
				ww.debounce(changeName, Event{Type: "change_updated", Name: changeName})
			}
		}
	}
}
