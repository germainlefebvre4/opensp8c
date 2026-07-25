package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/glefebvre/opensp8c/internal/openspec"
)

// Manager orchestrates the agent pool.
type Manager struct {
	mu            sync.Mutex
	workspacePath string
	config        AgentPoolConfig
	activeWorkers map[int]*Worker
	cancelLoop    context.CancelFunc
	isRunning     bool
}

// NewManager creates a new pool manager.
func NewManager() *Manager {
	return &Manager{
		activeWorkers: make(map[int]*Worker),
	}
}

// Start begins the orchestration loop with the given configuration.
func (m *Manager) Start(cfg AgentPoolConfig, workspacePath string) error {
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
	m.workspacePath = workspacePath
	m.isRunning = true

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelLoop = cancel

	go m.orchestrationLoop(ctx)

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
}

// Status returns the current status of the pool and its workers.
func (m *Manager) Status() (AgentPoolConfig, bool, []Worker) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var workers []Worker
	for _, w := range m.activeWorkers {
		workers = append(workers, *w)
	}

	return m.config, m.isRunning, workers
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
		ID:           id,
		ActiveChange: changeName,
		Status:       StatusWorking,
		CancelFunc:   cancel,
	}
	m.activeWorkers[id] = worker

	// Start worker routine asynchronously
	go m.runWorker(ctx, worker)
}
