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
	ID           int
	ActiveChange string
	WorktreePath string
	BranchName   string
	Status       WorkerStatus
	CancelFunc   context.CancelFunc
}
