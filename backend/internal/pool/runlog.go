package pool

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/watcher"
)

// runEventInterval bounds pool_run_appended to one event per change per
// interval during a burst of journal lines.
const runEventInterval = time.Second

// Run outcomes recorded in the pool_run_end marker.
const (
	OutcomeCompleted      = "completed"
	OutcomeAwaitingReview = "awaiting-review"
	OutcomePaused         = "paused"
	OutcomeStopped        = "stopped"
)

// runTimestampLayout is the run identifier format shared with chat/ff runs.
const runTimestampLayout = "2006-01-02T15-04-05Z"

// poolRunLog is the journal of one worker execution. It is only used by the
// worker's own goroutine, except for the wrapped SessionLog which serializes
// its writes.
type poolRunLog struct {
	sess   *conversation.SessionLog
	failed bool
}

// openRunLog creates the "pool" run for w's change. It returns nil, "" when no
// conversation store is configured or the file cannot be created: journaling
// must never prevent a worker from running.
func (m *Manager) openRunLog(w *Worker) (*poolRunLog, string) {
	if m.convStore == nil {
		return nil, ""
	}
	ts := time.Now().UTC().Format(runTimestampLayout)
	f, err := m.convStore.OpenRun(w.WorkspaceID, w.ActiveChange, "pool", ts)
	if err != nil {
		log.Printf("[worker %d] failed to open pool run journal: %v\n", w.ID, err)
		return nil, ""
	}
	return &poolRunLog{sess: conversation.NewSessionLog(f)}, ts
}

// logRun appends one line to w's run journal. A write failure is logged once
// and otherwise ignored: the worker keeps running.
func (m *Manager) logRun(w *Worker, dir string, data []byte) {
	rl := w.runLog
	if rl == nil {
		return
	}
	if !json.Valid(data) {
		data, _ = json.Marshal(string(data))
	}
	if err := rl.sess.WriteLine(dir, data); err != nil {
		if !rl.failed {
			rl.failed = true
			log.Printf("[worker %d] pool run journal write failed (further failures not logged): %v\n", w.ID, err)
		}
		return
	}
	m.noteRunAppended(w)
}

func (m *Manager) logRunMarker(w *Worker, marker map[string]any) {
	data, err := json.Marshal(marker)
	if err != nil {
		return
	}
	m.logRun(w, "meta", data)
}

// stopper is the subset of *time.Timer the run-event limiter needs.
type stopper interface{ Stop() bool }

// runThrottle limits emit to once per interval, with a trailing emit after
// the last call of a burst.
type runThrottle struct {
	mu        sync.Mutex
	interval  time.Duration
	now       func() time.Time
	afterFunc func(d time.Duration, f func()) stopper
	emit      func()
	last      time.Time
	timer     stopper
}

// Note requests an event: immediate if the interval elapsed, else deferred to
// the end of the interval (at most one deferred event is ever pending).
func (t *runThrottle) Note() {
	t.mu.Lock()
	n := t.now()
	if t.timer != nil {
		t.mu.Unlock()
		return
	}
	if t.last.IsZero() || n.Sub(t.last) >= t.interval {
		t.last = n
		t.mu.Unlock()
		t.emit()
		return
	}
	t.timer = t.afterFunc(t.last.Add(t.interval).Sub(n), func() {
		t.mu.Lock()
		t.timer = nil
		t.last = t.now()
		t.mu.Unlock()
		t.emit()
	})
	t.mu.Unlock()
}

// Flush cancels any deferred event and emits immediately.
func (t *runThrottle) Flush() {
	t.mu.Lock()
	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
	t.last = t.now()
	t.mu.Unlock()
	t.emit()
}

func (m *Manager) throttleFor(w *Worker) *runThrottle {
	m.runMu.Lock()
	defer m.runMu.Unlock()
	if t, ok := m.runThrottles[w.ActiveChange]; ok {
		return t
	}
	now := m.clock
	if now == nil {
		now = time.Now
	}
	after := m.afterFunc
	if after == nil {
		after = func(d time.Duration, f func()) stopper { return time.AfterFunc(d, f) }
	}
	wsID, change := w.WorkspaceID, w.ActiveChange
	t := &runThrottle{
		interval:  runEventInterval,
		now:       now,
		afterFunc: after,
		emit: func() {
			if m.broadcaster != nil {
				m.broadcaster.Broadcast(wsID, watcher.Event{Type: "pool_run_appended", Name: change})
			}
		},
	}
	if m.runThrottles == nil {
		m.runThrottles = make(map[string]*runThrottle)
	}
	m.runThrottles[change] = t
	return t
}

// noteRunAppended signals that w's run received a new line (rate limited).
func (m *Manager) noteRunAppended(w *Worker) { m.throttleFor(w).Note() }

// finishRunEvents emits the final pool_run_appended of w's run and drops its limiter.
func (m *Manager) finishRunEvents(w *Worker) {
	t := m.throttleFor(w)
	t.Flush()
	m.runMu.Lock()
	delete(m.runThrottles, w.ActiveChange)
	m.runMu.Unlock()
}
