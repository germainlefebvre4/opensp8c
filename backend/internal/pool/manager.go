package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
)

// Broadcaster notifies subscribers of a workspace about an in-memory event,
// independently of any filesystem change. watcher.WatcherService satisfies it.
type Broadcaster interface {
	Broadcast(workspaceID string, ev watcher.Event)
}

// Manager orchestrates the agent pool.
type Manager struct {
	mu               sync.Mutex
	workspaceID      string
	workspaceName    string
	workspacePath    string
	config           AgentPoolConfig
	activeWorkers    map[int]*Worker
	pausedWorkers    map[int]*Worker
	lastWorkerStatus map[int]WorkerStatus
	cancelLoop       context.CancelFunc
	isRunning        bool
	broadcaster      Broadcaster
	sessionMgr       *session.Manager
	prefs            *preferences.Service
	activityStore    *activity.Store
	convStore        *conversation.Store

	// worktreesRoot is the root directory of the worktrees, read once from
	// OPENSP8C_WORKTREES_DIR (or the default) at construction.
	worktreesRoot string
	// mergeMu serializes the merges into the workspace repository.
	mergeMu sync.Mutex

	// reviewOps holds one lock per change serializing its review actions.
	reviewOps sync.Map

	// workers tracks running runWorker goroutines so tests can wait for them
	// after Stop (which only cancels them).
	workers sync.WaitGroup

	// runMu guards runThrottles, the per-change limiters of pool_run_appended.
	runMu        sync.Mutex
	runThrottles map[string]*runThrottle
	// clock and afterFunc are seams for the run-event limiter.
	clock     func() time.Time
	afterFunc func(d time.Duration, f func()) stopper
}

// NewManager creates a new pool manager. broadcaster may be nil, in which
// case pool state changes are simply not published as events. sessionMgr and
// prefs are used to resolve which agent CLI (and custom env) to invoke for a
// given workspace/change, the same resolution used by interactive sessions.
func NewManager(broadcaster Broadcaster, sessionMgr *session.Manager, prefs *preferences.Service, actStore *activity.Store, convStore *conversation.Store) *Manager {
	return &Manager{
		activeWorkers:    make(map[int]*Worker),
		pausedWorkers:    make(map[int]*Worker),
		worktreesRoot:    DefaultWorktreesRoot(),
		lastWorkerStatus: make(map[int]WorkerStatus),
		broadcaster:      broadcaster,
		sessionMgr:       sessionMgr,
		prefs:            prefs,
		activityStore:    actStore,
		convStore:        convStore,
	}
}

// Start begins the orchestration loop with the given configuration, on
// behalf of workspaceID.
func (m *Manager) Start(cfg AgentPoolConfig, workspaceID, workspaceName, workspacePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("pool is already running")
	}

	// Omitted fields are completed from the resolved (workspace, then
	// Configuration) pool settings; explicit request values win and nothing
	// is persisted.
	if m.prefs != nil {
		if p, err := m.prefs.Load(); err == nil {
			resolved := p.ResolvePool(workspaceID)
			if cfg.Size <= 0 {
				cfg.Size = resolved.Size
			}
			if cfg.DelegationMode == "" {
				cfg.DelegationMode = DelegationMode(resolved.DelegationMode)
			}
			if cfg.MaxAttempts <= 0 {
				cfg.MaxAttempts = resolved.MaxAttempts
			}
		}
	}
	if cfg.Size <= 0 {
		cfg.Size = 1
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}

	m.pausedWorkers = make(map[int]*Worker)

	m.config = cfg
	m.workspaceID = workspaceID
	m.workspaceName = workspaceName
	m.workspacePath = workspacePath
	m.isRunning = true

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelLoop = cancel

	go m.orchestrationLoop(ctx)

	m.broadcastLocked()

	return nil
}

// waitWorkers blocks until every started worker goroutine has returned.
func (m *Manager) waitWorkers() { m.workers.Wait() }

// StopAndWait stops the pool and waits for every worker goroutine (and so
// their process groups) to finish, within ctx.
func (m *Manager) StopAndWait(ctx context.Context) error {
	m.Stop()
	done := make(chan struct{})
	go func() {
		m.waitWorkers()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop halts the orchestration loop and cancels all active workers.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return
	}

	if m.cancelLoop != nil {
		m.cancelLoop()
		m.cancelLoop = nil
	}

	for _, w := range m.activeWorkers {
		if w.CancelFunc != nil {
			w.CancelFunc()
		}
	}
	m.activeWorkers = make(map[int]*Worker)
	m.pausedWorkers = make(map[int]*Worker)
	m.isRunning = false

	m.broadcastLocked()

	m.workspaceID = ""
	m.workspaceName = ""
}

// Errors returned by ResumeWorker.
var (
	ErrPoolNotRunning  = errors.New("pool is not running")
	ErrWorkerNotPaused = errors.New("worker is not paused")
)

// ResumeWorker lifts the pause of a worker: its change becomes eligible again
// and the next tick redistributes it (tick stays the single dispatch point).
func (m *Manager) ResumeWorker(workerID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return ErrPoolNotRunning
	}
	if _, ok := m.pausedWorkers[workerID]; !ok {
		return ErrWorkerNotPaused
	}
	delete(m.pausedWorkers, workerID)
	m.broadcastLocked()
	return nil
}

// ReleasePausedForChange lifts the pause of every paused worker holding
// change, so the change is no longer held and the next tick can redistribute
// it. It reports whether a pause was released. Active workers are untouched.
func (m *Manager) ReleasePausedForChange(change string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	released := false
	for id, w := range m.pausedWorkers {
		if w.ActiveChange == change {
			delete(m.pausedWorkers, id)
			released = true
		}
	}
	if released {
		m.broadcastLocked()
	}
	return released
}

// CancelWorkerForChange cancels the active worker assigned to changeName, if any.
// It returns true if an active worker was found and its cancel function invoked,
// or false if no worker was active for that change.
func (m *Manager) CancelWorkerForChange(changeName string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, w := range m.activeWorkers {
		if w.ActiveChange == changeName {
			if w.CancelFunc != nil {
				w.CancelFunc()
			}
			return true
		}
	}
	return false
}

// defaultCancelWaitTimeout bounds how long CancelAndWaitForChange waits.
var defaultCancelWaitTimeout = 30 * time.Second

// SetCancelWaitTimeout overrides the maximum wait of CancelAndWaitForChange and
// returns a function restoring the previous value. For tests.
func SetCancelWaitTimeout(d time.Duration) (restore func()) {
	prev := defaultCancelWaitTimeout
	defaultCancelWaitTimeout = d
	return func() { defaultCancelWaitTimeout = prev }
}

// CancelAndWaitForChange cancels the active worker assigned to changeName and
// waits for it to really finish, until ctx is done or the wait timeout
// elapses. found is false when no worker was active; timedOut is true when the
// worker was still running at the deadline (result is then zero).
func (m *Manager) CancelAndWaitForChange(ctx context.Context, changeName string) (found bool, result WorkerResult, timedOut bool) {
	m.mu.Lock()
	var target *Worker
	for _, w := range m.activeWorkers {
		if w.ActiveChange == changeName {
			target = w
			break
		}
	}
	if target == nil {
		m.mu.Unlock()
		return false, WorkerResult{}, false
	}
	if target.CancelFunc != nil {
		target.CancelFunc()
	}
	done := target.done
	m.mu.Unlock()

	if done == nil {
		return true, WorkerResult{}, false
	}
	timer := time.NewTimer(defaultCancelWaitTimeout)
	defer timer.Stop()
	select {
	case <-done:
		return true, target.result, false
	case <-ctx.Done():
		return true, WorkerResult{}, true
	case <-timer.C:
		return true, WorkerResult{}, true
	}
}

// Status returns the current status of the pool and its workers, as seen by
// workspaceID. If the pool is running on behalf of a different workspace, it
// is reported as not running rather than leaking that workspace's state.
func (m *Manager) Status(workspaceID string) (AgentPoolConfig, bool, []Worker) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning || m.workspaceID != workspaceID {
		return AgentPoolConfig{}, false, nil
	}

	var workers []Worker
	for _, w := range m.activeWorkers {
		workers = append(workers, *w)
	}
	for _, w := range m.pausedWorkers {
		workers = append(workers, *w)
	}

	return m.config, m.isRunning, workers
}

// WorkspaceName returns the name of the workspace the pool currently runs
// for (empty if not running), for callers that need it even when the pool
// has no worker yet to read it off of.
func (m *Manager) WorkspaceName() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.workspaceName
}

// broadcastLocked publishes a pool_updated event for the workspace the pool
// currently runs for. Callers must hold m.mu.
func (m *Manager) broadcastLocked() {
	if m.broadcaster != nil && m.workspaceID != "" {
		m.broadcaster.Broadcast(m.workspaceID, watcher.Event{Type: "pool_updated"})
	}

	if m.activityStore != nil && m.workspaceID != "" {
		for id, w := range m.activeWorkers {
			if w.ActiveChange != "" && m.lastWorkerStatus[id] != w.Status {
				m.lastWorkerStatus[id] = w.Status
				_ = m.activityStore.Append(m.workspaceID, w.ActiveChange, activity.Entry{
					Type:     "pool.worker_status",
					Category: "pool",
					Summary:  fmt.Sprintf("Worker %d: %s", w.ID, w.Status),
					Meta: map[string]any{
						"worker_id": w.ID,
						"status":    string(w.Status),
						"change":    w.ActiveChange,
					},
				})
			}
		}
	}
}

// notify is the lock-free-callable counterpart of broadcastLocked, for use
// by goroutines (e.g. a running worker) that don't already hold m.mu.
func (m *Manager) notify() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.broadcastLocked()
}

func (m *Manager) orchestrationLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.tick()
		}
	}
}

func (m *Manager) tick() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.activeWorkers) >= m.config.Size {
		return
	}

	changes, err := openspec.ListChanges(m.workspacePath)
	if err != nil {
		return
	}

	scheduler := NewScheduler(changes)
	runnable := scheduler.GetRunnableChanges()

	// Filter out changes already being worked on
	activeChangeSet := make(map[string]bool)
	for _, w := range m.activeWorkers {
		activeChangeSet[w.ActiveChange] = true
	}
	// A paused change stays excluded until explicitly resumed (or the pool is
	// stopped), otherwise the dispatcher would relaunch it on the next tick.
	for _, w := range m.pausedWorkers {
		activeChangeSet[w.ActiveChange] = true
	}

	for _, changeName := range runnable {
		if len(m.activeWorkers) >= m.config.Size {
			break
		}
		if activeChangeSet[changeName] {
			continue
		}

		m.startWorker(changeName)
	}
}

func (m *Manager) startWorker(changeName string) {
	// Find next available ID, skipping both active and (still displayed)
	// paused workers so a new worker never collides with a paused one's ID.
	id := 1
	for {
		_, activeExists := m.activeWorkers[id]
		_, pausedExists := m.pausedWorkers[id]
		if !activeExists && !pausedExists {
			break
		}
		id++
	}

	ctx, cancel := context.WithCancel(context.Background())
	worker := &Worker{
		ID:             id,
		WorkspaceID:    m.workspaceID,
		WorkspaceName:  m.workspaceName,
		ActiveChange:   changeName,
		Status:         StatusWorking,
		DelegationMode: m.config.DelegationMode,
		StartedAt:      time.Now(),
		CancelFunc:     cancel,
		done:           make(chan struct{}),
	}
	m.activeWorkers[id] = worker

	// Start worker routine asynchronously
	m.workers.Add(1)
	go func() {
		defer m.workers.Done()
		m.runWorker(ctx, worker)
	}()

	m.broadcastLocked()
}
