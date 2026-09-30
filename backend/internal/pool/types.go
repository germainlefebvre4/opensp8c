package pool

import (
	"context"
	"time"

	"github.com/glefebvre/opensp8c/internal/preferences"
)

type DelegationMode string

const (
	ModeFullAutonomy DelegationMode = "full-autonomy"
	ModeHITLReview   DelegationMode = "hitl-review"
)

type AgentPoolConfig struct {
	Size           int            `json:"size"`
	DelegationMode DelegationMode `json:"delegation_mode"`
	MaxAttempts    int            `json:"max_attempts"`
}

type WorkerStatus string

const (
	StatusIdle    WorkerStatus = "idle"
	StatusWorking WorkerStatus = "working"
	StatusTesting WorkerStatus = "testing"
	StatusHealing WorkerStatus = "healing"
	StatusPaused  WorkerStatus = "paused"
)

type Worker struct {
	ID             int                `json:"id"`
	WorkspaceID    string             `json:"workspace_id"`
	WorkspaceName  string             `json:"workspace_name"`
	ActiveChange   string             `json:"active_change"`
	WorktreePath   string             `json:"worktree_path,omitempty"`
	BranchName     string             `json:"branch_name,omitempty"`
	Status         WorkerStatus       `json:"status"`
	Activity       string             `json:"activity,omitempty"`
	BlockedReason  string             `json:"blocked_reason,omitempty"`
	DelegationMode DelegationMode     `json:"delegation_mode"`
	StartedAt      time.Time          `json:"started_at"`
	CancelFunc     context.CancelFunc `json:"-"`
	// Role selects the agent/model/effort settings of the worker's subprocess.
	// Empty means implementer; a worker restarted after "Demander des
	// corrections" is created with RoleFixer.
	Role preferences.Role `json:"-"`
}

// role returns the effective role of the worker.
func (w *Worker) role() preferences.Role {
	if w.Role == "" {
		return preferences.RoleImplementer
	}
	return w.Role
}
