package pool

import (
	"context"
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
	ID           int                `json:"id"`
	ActiveChange string             `json:"active_change"`
	WorktreePath string             `json:"worktree_path,omitempty"`
	BranchName   string             `json:"branch_name,omitempty"`
	Status       WorkerStatus       `json:"status"`
	CancelFunc   context.CancelFunc `json:"-"`
}
