package pool

import (
	"context"
	"time"
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
	DelegationMode DelegationMode     `json:"delegation_mode"`
	StartedAt      time.Time          `json:"started_at"`
	CancelFunc     context.CancelFunc `json:"-"`
}
