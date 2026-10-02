package pool

import (
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/glefebvre/opensp8c/internal/watcher"
)

// worktreeWatchDebounce matches the debounce of the main-repo watcher so a
// burst of writes yields a single change_updated.
const worktreeWatchDebounce = 150 * time.Millisecond

// watchWorktreeTasks watches the change directory inside a worker's worktree
// and broadcasts a debounced change_updated whenever its tasks.md is written,
// so the Kanban refreshes live while the agent ticks tasks. The directory is
// watched rather than the file because a file replaced by an atomic rename
// would drop a watch placed on the file itself. The returned stop function
// is idempotent, waits for the watcher goroutine to exit, and guarantees no
// event is broadcast after it returns. Failures are logged and yield a no-op
// stop: live refresh is best effort, pool_updated still reloads the list.
func (m *Manager) watchWorktreeTasks(workspaceID, worktreePath, change string) (stop func()) {
	noop := func() {}
	if m.broadcaster == nil {
		return noop
	}
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("[pool] worktree watcher for %s: %v\n", change, err)
		return noop
	}
	dir := filepath.Join(worktreePath, "openspec", "changes", change)
	if err := fw.Add(dir); err != nil {
		log.Printf("[pool] worktree watcher for %s: %v\n", change, err)
		fw.Close()
		return noop
	}

	quit := make(chan struct{})
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		var timer *time.Timer
		var fire <-chan time.Time
		defer func() {
			if timer != nil {
				timer.Stop()
			}
		}()
		for {
			select {
			case <-quit:
				return
			case ev, ok := <-fw.Events:
				if !ok {
					return
				}
				if filepath.Base(ev.Name) != "tasks.md" || !ev.Has(fsnotify.Write|fsnotify.Create|fsnotify.Rename) {
					continue
				}
				if timer == nil {
					timer = time.NewTimer(worktreeWatchDebounce)
				} else {
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timer.Reset(worktreeWatchDebounce)
				}
				fire = timer.C
			case <-fw.Errors:
			case <-fire:
				fire = nil
				m.broadcaster.Broadcast(workspaceID, watcher.Event{Type: "change_updated", Name: change})
			}
		}
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			close(quit)
			<-exited
			fw.Close()
		})
	}
}
