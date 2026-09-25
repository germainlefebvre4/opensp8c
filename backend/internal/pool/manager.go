package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/glefebvre/opensp8c/internal/activity"
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
	lastWorkerStatus map[int]WorkerStatus
	cancelLoop       context.CancelFunc
	isRunning        bool
	broadcaster      Broadcaster
	sessionMgr       *session.Manager
	prefs            *preferences.Service
	activityStore    *activity.Store
}

// NewManager creates a new pool manager. broadcaster may be nil, in which
// case pool state changes are simply not published as events. sessionMgr and
// prefs are used to resolve which agent CLI (and custom env) to invoke for a
// given workspace/change, the same resolution used by interactive sessions.
func NewManager(broadcaster Broadcaster, sessionMgr *session.Manager, prefs *preferences.Service, actStore *activity.Store) *Manager {
	return &Manager{
		activeWorkers:    make(map[int]*Worker),
		lastWorkerStatus: make(map[int]WorkerStatus),
		broadcaster:      broadcaster,
		sessionMgr:       sessionMgr,
		prefs:            prefs,
		activityStore:    actStore,
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

	if cfg.Size <= 0 {
		cfg.Size = 1
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}

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
	m.isRunning = false

	m.broadcastLocked()

	m.workspaceID = ""
	m.workspaceName = ""
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

	return m.config, m.isRunning, workers
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
	// Find next available ID
	id := 1
	for {
		if _, exists := m.activeWorkers[id]; !exists {
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
	}
	m.activeWorkers[id] = worker

	// Start worker routine asynchronously
	go m.runWorker(ctx, worker)

	m.broadcastLocked()
}
