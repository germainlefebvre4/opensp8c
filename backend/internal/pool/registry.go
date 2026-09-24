package pool

import (
	"sync"

	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
)

// Registry manages one Manager per workspace, created lazily on first use.
// It lets independent pools run concurrently for different workspaces while
// reusing the existing single-workspace Manager implementation unchanged.
type Registry struct {
	broadcaster Broadcaster
	sessionMgr  *session.Manager
	prefs       *preferences.Service

	mu       sync.Mutex
	managers map[string]*Manager
}

// NewRegistry creates a Registry whose Managers publish pool state changes
// through broadcaster (may be nil, see NewManager), and resolve agents via
// sessionMgr/prefs.
func NewRegistry(broadcaster Broadcaster, sessionMgr *session.Manager, prefs *preferences.Service) *Registry {
	return &Registry{
		broadcaster: broadcaster,
		sessionMgr:  sessionMgr,
		prefs:       prefs,
		managers:    make(map[string]*Manager),
	}
}

// For returns the Manager for workspaceID, creating and caching one on first
// call. Repeated calls for the same workspaceID return the same instance.
func (reg *Registry) For(workspaceID string) *Manager {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	if m, ok := reg.managers[workspaceID]; ok {
		return m
	}

	m := NewManager(reg.broadcaster, reg.sessionMgr, reg.prefs)
	reg.managers[workspaceID] = m
	return m
}

// Remove stops the pool for workspaceID, if any, and drops it from the
// registry so a deleted workspace cannot leave an orphaned running pool.
func (reg *Registry) Remove(workspaceID string) {
	reg.mu.Lock()
	m, ok := reg.managers[workspaceID]
	delete(reg.managers, workspaceID)
	reg.mu.Unlock()

	if ok {
		m.Stop()
	}
}

// AllWorkers flattens the active workers of every known workspace's Manager
// into one slice, tagged with their own workspace identity.
func (reg *Registry) AllWorkers() []Worker {
	reg.mu.Lock()
	managers := make(map[string]*Manager, len(reg.managers))
	for id, m := range reg.managers {
		managers[id] = m
	}
	reg.mu.Unlock()

	var workers []Worker
	for workspaceID, m := range managers {
		_, running, ws := m.Status(workspaceID)
		if !running {
			continue
		}
		workers = append(workers, ws...)
	}
	return workers
}
