//go:build unix

package session

import (
	"os/exec"
	"syscall"
	"time"
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

// StopProcessGroup terminates the process group led by cmd: SIGTERM first, then
// SIGKILL once grace has elapsed with members still alive. It returns when no
// member is left (the leader included, once reaped by cmd.Wait elsewhere).
func StopProcessGroup(cmd *exec.Cmd, grace time.Duration) {
	if cmd.Process == nil {
		return
	}
	pgid := cmd.Process.Pid
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		return // no member left
	}
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if syscall.Kill(-pgid, 0) != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	for i := 0; i < 100 && syscall.Kill(-pgid, 0) == nil; i++ {
		time.Sleep(20 * time.Millisecond)
	}
}
