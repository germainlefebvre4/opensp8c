package pool

import (
	"sync"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
)

// PoolSummary groups a single workspace's pool configuration with its
// currently active and paused workers, for the "all workspaces" list view.
type PoolSummary struct {
	WorkspaceID    string         `json:"workspace_id"`
	WorkspaceName  string         `json:"workspace_name"`
	Size           int            `json:"size"`
	DelegationMode DelegationMode `json:"delegation_mode"`
	Workers        []Worker       `json:"workers"`
}

// Registry manages one Manager per workspace, created lazily on first use.
// It lets independent pools run concurrently for different workspaces while
// reusing the existing single-workspace Manager implementation unchanged.
type Registry struct {
	broadcaster   Broadcaster
	sessionMgr    *session.Manager
	prefs         *preferences.Service
	activityStore *activity.Store
	convStore     *conversation.Store

	mu       sync.Mutex
	managers map[string]*Manager
}

// NewRegistry creates a Registry whose Managers publish pool state changes
// through broadcaster (may be nil, see NewManager), and resolve agents via
// sessionMgr/prefs. convStore (may be nil) receives the per-worker run journals.
func NewRegistry(broadcaster Broadcaster, sessionMgr *session.Manager, prefs *preferences.Service, actStore *activity.Store, convStore *conversation.Store) *Registry {
	return &Registry{
		broadcaster:   broadcaster,
		sessionMgr:    sessionMgr,
		prefs:         prefs,
		activityStore: actStore,
		convStore:     convStore,
		managers:      make(map[string]*Manager),
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

	m := NewManager(reg.broadcaster, reg.sessionMgr, reg.prefs, reg.activityStore, reg.convStore)
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

// AllPools returns one PoolSummary per currently running pool across every
// known workspace, each carrying its configured size, delegation mode, and
// the detail of its active and paused workers (see Manager.Status).
func (reg *Registry) AllPools() []PoolSummary {
	reg.mu.Lock()
	managers := make(map[string]*Manager, len(reg.managers))
	for id, m := range reg.managers {
		managers[id] = m
	}
	reg.mu.Unlock()

	var pools []PoolSummary
	for workspaceID, m := range managers {
		cfg, running, workers := m.Status(workspaceID)
		if !running {
			continue
		}
		pools = append(pools, PoolSummary{
			WorkspaceID:    workspaceID,
			WorkspaceName:  m.WorkspaceName(),
			Size:           cfg.Size,
			DelegationMode: cfg.DelegationMode,
			Workers:        workers,
		})
	}
	return pools
}
