package pool

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/session"
)

type stubFn = func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error)

func neverStart(t *testing.T) stubFn {
	return func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
		return nil, errors.New("subprocess must not start")
	}
}

func TestRunWorker_PausesWhenProvisionFails(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	notRepo := t.TempDir() // not a git repository: Provision fails
	m := newWorkerTestManager(t, notRepo, AgentPoolConfig{Size: 1, MaxAttempts: 1}, neverStart(t))
	m.workspaceID = "ws"
	m.isRunning = true

	w := &Worker{ID: 1, WorkspaceID: "ws", ActiveChange: "some-change"}
	m.activeWorkers[1] = w
	m.runWorker(context.Background(), w)

	_, _, workers := m.Status("ws")
	if len(workers) != 1 || workers[0].Status != StatusPaused || workers[0].BlockedReason == "" {
		t.Fatalf("expected one paused worker with a reason, got %+v", workers)
	}
}

func TestTick_DoesNotRedistributePausedChange(t *testing.T) {
	repo := t.TempDir()
	writeChangeForPoolTest(t, repo+"/openspec/changes", "change-a", boolPtr(true), intPtr(1))
	writeChangeForPoolTest(t, repo+"/openspec/changes", "change-b", boolPtr(true), intPtr(2))

	block := make(chan struct{})
	defer close(block)
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 2, MaxAttempts: 1},
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
			<-block
			return nil, context.Canceled
		})
	m.workspaceID = "ws"
	m.isRunning = true
	m.mu.Lock()
	m.pausedWorkers[1] = &Worker{ID: 1, ActiveChange: "change-a", Status: StatusPaused, BlockedReason: "r"}
	m.mu.Unlock()

	m.tick()
	m.tick()

	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.activeWorkers) != 1 {
		t.Fatalf("expected exactly one new worker, got %d", len(m.activeWorkers))
	}
	for _, w := range m.activeWorkers {
		if w.ActiveChange != "change-b" {
			t.Fatalf("paused change-a must not be redistributed, got worker on %q", w.ActiveChange)
		}
	}
}

func TestStopStart_MakesPausedChangeEligibleAgain(t *testing.T) {
	repo := t.TempDir()
	writeChangeForPoolTest(t, repo+"/openspec/changes", "change-a", boolPtr(true), intPtr(1))
	m := NewManager(nil, nil, nil, nil, nil)
	if err := m.Start(AgentPoolConfig{Size: 1, MaxAttempts: 1}, "ws", "ws", repo); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.pausedWorkers[1] = &Worker{ID: 1, ActiveChange: "change-a", Status: StatusPaused}
	m.mu.Unlock()
	m.Stop()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.pausedWorkers) != 0 {
		t.Fatalf("Stop must clear paused workers, got %+v", m.pausedWorkers)
	}
}

func TestResumeWorker(t *testing.T) {
	repo := t.TempDir()
	writeChangeForPoolTest(t, repo+"/openspec/changes", "change-a", boolPtr(true), intPtr(1))

	block := make(chan struct{})
	defer close(block)
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 2, MaxAttempts: 1},
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
			<-block
			return nil, context.Canceled
		})

	if err := m.ResumeWorker(1); !errors.Is(err, ErrPoolNotRunning) {
		t.Fatalf("stopped pool: got %v", err)
	}

	m.workspaceID = "ws"
	m.isRunning = true
	other := &Worker{ID: 2, ActiveChange: "other", Status: StatusWorking}
	m.mu.Lock()
	m.activeWorkers[2] = other
	m.pausedWorkers[1] = &Worker{ID: 1, ActiveChange: "change-a", Status: StatusPaused, BlockedReason: "r"}
	m.mu.Unlock()

	if err := m.ResumeWorker(2); !errors.Is(err, ErrWorkerNotPaused) {
		t.Fatalf("active worker: got %v", err)
	}
	if err := m.ResumeWorker(99); !errors.Is(err, ErrWorkerNotPaused) {
		t.Fatalf("unknown worker: got %v", err)
	}

	if err := m.ResumeWorker(1); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	if _, ok := m.pausedWorkers[1]; ok {
		t.Fatal("worker 1 should no longer be paused")
	}
	if m.activeWorkers[2] != other || other.Status != StatusWorking {
		t.Fatal("other active workers must be untouched")
	}
	m.mu.Unlock()

	m.tick() // change-a is distributable again
	deadline := time.Now().Add(2 * time.Second)
	for {
		m.mu.Lock()
		got := false
		for _, w := range m.activeWorkers {
			if w.ActiveChange == "change-a" {
				got = true
			}
		}
		m.mu.Unlock()
		if got {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("change-a was not redistributed after resume")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
