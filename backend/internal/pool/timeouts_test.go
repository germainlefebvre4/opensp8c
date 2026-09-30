package pool

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/session"
)

func setVar[T any](t *testing.T, p *T, v T) {
	t.Helper()
	old := *p
	*p = v
	t.Cleanup(func() { *p = old })
}

func waitPidGone(t *testing.T, pidFile string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var pid int
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(pidFile); err == nil {
			if p, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil {
				pid = p
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatalf("pid file %s never written", pidFile)
	}
	for time.Now().Before(deadline.Add(3 * time.Second)) {
		if err := syscall.Kill(pid, 0); err == syscall.ESRCH {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("process %d survived", pid)
}

// scriptAgent returns a stub that runs a real shell script as the agent CLI.
func scriptAgent(t *testing.T, script string) startFn {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agent.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	return func(ctx context.Context, ws string, a agents.AgentConfig, e, c string, r bool, l *conversation.SessionLog, env map[string]string, n bool, d string) (*session.Subprocess, error) {
		return session.StartSubprocess(ctx, ws, agents.AgentConfig{ID: "fake", CLI: path}, e, c, r, l, env, n, d)
	}
}

func TestRunValidationCommand_TimeoutKillsProcessGroup(t *testing.T) {
	setVar(t, &validationTimeout, 200*time.Millisecond)
	setVar(t, &validationWaitDelay, time.Second)
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "sleep.pid")

	start := time.Now()
	_, err := runValidationCommand(context.Background(), dir, "sh", []string{"-c", "sleep 60 & echo $! > " + pidFile + "; wait"}, "sleeper", true)

	var envErr *ValidationEnvError
	if !errors.As(err, &envErr) || !strings.Contains(envErr.Reason, "Validation trop longue") {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatalf("took %s", time.Since(start))
	}
	waitPidGone(t, pidFile)
}

func TestRunWorker_ValidationTimeoutPauses(t *testing.T) {
	setVar(t, &validationTimeout, 200*time.Millisecond)
	change := "slow-validation"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1}, pipeAgent(writeInWorktree("x.txt"), okResult, nil))
	m.prefs = validationManager(t, "sleep 60").prefs

	w := runOnce(t, m, change)

	reason, ok := pausedReason(m, w.ID)
	if !ok || !strings.Contains(reason, "Validation trop longue") {
		t.Fatalf("paused=%v reason=%q", ok, reason)
	}
}

func TestRunTurn_IdleAgentIsInterrupted(t *testing.T) {
	setVar(t, &agentIdleTimeout, 200*time.Millisecond)
	change := "idle-agent"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1},
		scriptAgent(t, "read line\nsleep 300 &\necho $! > "+pidFile+"\nwait"))

	start := time.Now()
	w := runOnce(t, m, change)

	reason, ok := pausedReason(m, w.ID)
	if !ok || !strings.Contains(reason, "Agent inactif") {
		t.Fatalf("paused=%v reason=%q", ok, reason)
	}
	if time.Since(start) > 15*time.Second {
		t.Fatalf("worker took %s to give up", time.Since(start))
	}
	waitPidGone(t, pidFile)
}

func TestRunTurn_ChattyAgentIsNotInterrupted(t *testing.T) {
	setVar(t, &agentIdleTimeout, 300*time.Millisecond)
	change := "chatty-agent"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1},
		scriptAgent(t, `read line
touch agent-output.txt
i=0
while [ $i -lt 20 ]; do echo '{"type":"assistant"}'; sleep 0.05; i=$((i+1)); done
echo '{"type":"result","subtype":"success"}'
cat >/dev/null`))

	w := runOnce(t, m, change)

	if reason, paused := pausedReason(m, w.ID); paused {
		t.Fatalf("a regularly emitting agent was interrupted: %q", reason)
	}
}

func TestRunWorker_TeardownDoesNotHangOnAgentIgnoringStdinEOF(t *testing.T) {
	setVar(t, &teardownGrace, 300*time.Millisecond)
	change := "stubborn-agent"
	repo := newGoFixtureRepo(t, change, "- [x] done\n")
	pidFile := filepath.Join(t.TempDir(), "agent.pid")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1},
		scriptAgent(t, "echo $$ > "+pidFile+"\nread line\ntouch agent-output.txt\necho '{\"type\":\"result\"}'\nexec sleep 300"))

	start := time.Now()
	runOnce(t, m, change)

	if time.Since(start) > 6*time.Second {
		t.Fatalf("runWorker took %s", time.Since(start))
	}
	waitPidGone(t, pidFile)
}

func waitFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(b)) != "" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s never written", path)
}

func TestRegistryStopAll_TerminatesWorkersAndChildren(t *testing.T) {
	reg := NewRegistry(nil, nil, nil, nil, nil)
	agentsByChange := map[string]startFn{}
	orig := startSubprocessFn
	startSubprocessFn = func(ctx context.Context, ws string, a agents.AgentConfig, e, c string, r bool, l *conversation.SessionLog, env map[string]string, n bool, d string) (*session.Subprocess, error) {
		for change, agent := range agentsByChange {
			if strings.Contains(ws, change) {
				return agent(ctx, ws, a, e, c, r, l, env, n, d)
			}
		}
		return nil, errors.New("unexpected worktree " + ws)
	}
	t.Cleanup(func() { startSubprocessFn = orig })

	var pidFiles []string
	var managers []*Manager
	names := []string{"ws-one", "ws-two"}
	repos := map[string]string{}
	// Everything the stub reads is built before any worker starts.
	for _, name := range names {
		change := "stop-all-" + name
		repos[name] = newGoFixtureRepo(t, change, "- [x] done\n")
		pidFile := filepath.Join(t.TempDir(), "child.pid")
		pidFiles = append(pidFiles, pidFile)
		agentsByChange[change] = scriptAgent(t, "read line\nsleep 300 &\necho $! > "+pidFile+"\nwait")
	}
	for _, name := range names {
		change := "stop-all-" + name
		repo := repos[name]

		m := reg.For(name)
		m.workspacePath = repo
		m.worktreesRoot = filepath.Join(os.Getenv("OPENSP8C_WORKTREES_DIR"), filepath.Base(repo))
		m.config = AgentPoolConfig{Size: 1, DelegationMode: ModeFullAutonomy, MaxAttempts: 1}
		m.workspaceID = name
		m.isRunning = true
		managers = append(managers, m)

		ctx, cancel := context.WithCancel(context.Background())
		w := &Worker{ID: 1, WorkspaceID: name, ActiveChange: change, CancelFunc: cancel}
		m.activeWorkers[1] = w
		m.workers.Add(1)
		go func() {
			defer m.workers.Done()
			m.runWorker(ctx, w)
		}()
	}
	for _, pf := range pidFiles {
		waitFile(t, pf)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	reg.StopAll(ctx)

	for _, m := range managers {
		done := make(chan struct{})
		go func() { m.waitWorkers(); close(done) }()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("workers still running after StopAll")
		}
	}
	for _, pf := range pidFiles {
		waitPidGone(t, pf)
	}
}
