//go:build unix

package session

import (
	"os/exec"
	"syscall"
)

// ApplyProcessGroup runs cmd in its own process group and makes context
// cancellation kill the whole group, children included.
func ApplyProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
