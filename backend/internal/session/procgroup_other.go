//go:build !unix

package session

import "os/exec"

// ApplyProcessGroup is a no-op where process groups are unavailable.
func ApplyProcessGroup(cmd *exec.Cmd) {}
