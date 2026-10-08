package workspace

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

type Workspace struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Path       string         `json:"path"`
	TaskCounts map[string]int `json:"task_counts"`
	// Attention lists the changes waiting for a user action; never null.
	Attention []Attention `json:"attention"`
}

// Signal kinds carried by an Attention entry.
const (
	SignalReview       = "review"
	SignalPaused       = "paused"
	SignalHITL         = "hitl"
	SignalVerifyFailed = "verify-failed"
)

// Signal is one reason a change waits for the user.
type Signal struct {
	Kind   string `json:"kind"`
	Reason string `json:"reason,omitempty"`
}

// Attention is a change carrying at least one Signal, most blocking first.
type Attention struct {
	Change  string   `json:"change"`
	Signals []Signal `json:"signals"`
}

func StableID(absPath string) string {
	h := sha256.Sum256([]byte(absPath))
	return fmt.Sprintf("%x", h[:4])
}

func Validate(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path does not exist: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory")
	}
	openspecPath := filepath.Join(path, "openspec")
	if _, err := os.Stat(openspecPath); err != nil {
		return fmt.Errorf("directory does not contain an openspec/ folder")
	}
	return nil
}

func FromPath(path string) (*Workspace, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return &Workspace{
		ID:   StableID(absPath),
		Name: filepath.Base(absPath),
		Path: absPath,
	}, nil
}
