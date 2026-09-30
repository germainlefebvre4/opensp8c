//go:build unix

package session

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func readPid(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(b)) != "" {
			pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
			if err == nil {
				return pid
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("pid file %s never written", path)
	return 0
}

func TestApplyProcessGroupKillsChildrenOnCancel(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", "sleep 300 & echo $! > "+pidFile+"; wait")
	ApplyProcessGroup(cmd)
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	child := readPid(t, pidFile)

	cancel()
	_ = cmd.Wait()

	deadline := time.Now().Add(3 * time.Second)
	for {
		err := syscall.Kill(child, 0)
		if err == syscall.ESRCH {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("child %d survived the cancellation (kill -0: %v)", child, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
