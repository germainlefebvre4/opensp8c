//go:build !unix

package session

import (
	"os/exec"
	"time"
)

// ApplyProcessGroup is a no-op where process groups are unavailable.
func ApplyProcessGroup(cmd *exec.Cmd) {}

// StopProcessGroup kills the process where process groups are unavailable.
func StopProcessGroup(cmd *exec.Cmd, grace time.Duration) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
