package pool

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
)

func installMockClaude(t *testing.T) {
	t.Helper()
	cli := filepath.Join(t.TempDir(), "mock-claude")
	if err := os.WriteFile(cli, []byte("#!/bin/sh\necho 'mock 1.0.0'\n"), 0755); err != nil {
		t.Fatal(err)
	}
	orig := agents.SupportedAgents[0].CLI
	agents.SupportedAgents[0].CLI = cli
	t.Cleanup(func() { agents.SupportedAgents[0].CLI = orig })
}

func newRolePrefs(t *testing.T, patchJSON string) *preferences.Service {
	t.Helper()
	svc := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	if patchJSON != "" {
		p, err := jsonPatch(patchJSON)
		if err != nil {
			t.Fatal(err)
		}
		if err := svc.SetAgentSettings(p); err != nil {
			t.Fatal(err)
		}
	}
	return svc
}

type startCall struct{ model, effort string }

func roleWorkerManager(t *testing.T, repoDir string, prefs *preferences.Service, maxAttempts int, calls *[]startCall) *Manager {
	t.Helper()
	installMockClaude(t)
	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: maxAttempts},
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
			*calls = append(*calls, startCall{agentCfg.Model, agentCfg.Effort})
			return fakeAutoRespondingSubprocess(), nil
		})
	m.prefs = prefs
	m.sessionMgr = session.NewManager(prefs, nil)
	return m
}

func TestRunWorker_AppliesImplementerRoleAndHealKeepsIt(t *testing.T) {
	changeName := "role-heal-change"
	repoDir := t.TempDir()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	files := map[string]string{
		"go.mod":  "module fixture\n\ngo 1.21\n",
		"main.go": "package main\n\nfunc main() { this is not valid go }\n",
		filepath.Join("openspec", "changes", changeName, "tasks.md"): "- [x] done\n",
	}
	for name, content := range files {
		path := filepath.Join(repoDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-q")
	runGit("-c", "user.email=t@t", "-c", "user.name=t", "add", "-A")
	runGit("-c", "user.email=t@t", "-c", "user.name=t", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "init")
	t.Cleanup(func() {
		home, _ := os.UserHomeDir()
		_ = os.RemoveAll(filepath.Join(home, ".opensp8c", "worktrees", "wt-"+changeName))
	})

	var calls []startCall
	prefs := newRolePrefs(t, `{"roles":{"implementer":{"model":"opus","effort":"max"},"fixer":{"model":"haiku","effort":"low"}}}`)
	m := roleWorkerManager(t, repoDir, prefs, 2, &calls)

	w := &Worker{ID: 1, ActiveChange: changeName}
	m.activeWorkers[1] = w
	m.runWorker(context.Background(), w)

	// The validation fails, so heal turns ran on the same subprocess.
	if len(calls) != 1 {
		t.Fatalf("heal must not restart the subprocess: %d starts", len(calls))
	}
	if calls[0] != (startCall{"opus", "max"}) {
		t.Errorf("implementer settings expected, got %+v", calls[0])
	}
}

func TestRunWorker_FixerRoleAppliesFixerSettings(t *testing.T) {
	repoDir := newGoFixtureRepo(t, "fixer-change", "- [x] done\n")
	var calls []startCall
	prefs := newRolePrefs(t, `{"roles":{"implementer":{"model":"opus","effort":"max"},"fixer":{"model":"haiku","effort":"low"}}}`)
	m := roleWorkerManager(t, repoDir, prefs, 1, &calls)

	w := &Worker{ID: 1, ActiveChange: "fixer-change", Role: preferences.RoleFixer}
	m.activeWorkers[1] = w
	m.runWorker(context.Background(), w)

	if len(calls) != 1 || calls[0] != (startCall{"haiku", "low"}) {
		t.Errorf("fixer settings expected, got %+v", calls)
	}
}

func TestStart_CompletesConfigFromResolvedPool(t *testing.T) {
	prefs := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	if err := prefs.SetPoolDefaults(mustPoolPatch(t, `{"size":2,"maxAttempts":5}`)); err != nil {
		t.Fatal(err)
	}
	if err := prefs.PatchWorkspace("wsA", preferences.WorkspaceSettingsPatch{Pool: ptrPool(mustPoolPatch(t, `{"size":4,"delegationMode":"full-autonomy"}`))}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(prefs.Path())

	cases := []struct {
		ws   string
		req  AgentPoolConfig
		want AgentPoolConfig
	}{
		{"wsA", AgentPoolConfig{}, AgentPoolConfig{Size: 4, DelegationMode: ModeFullAutonomy, MaxAttempts: 5}},
		{"wsB", AgentPoolConfig{}, AgentPoolConfig{Size: 2, DelegationMode: ModeHITLReview, MaxAttempts: 5}},
		{"wsA", AgentPoolConfig{Size: 5, DelegationMode: ModeHITLReview, MaxAttempts: 1}, AgentPoolConfig{Size: 5, DelegationMode: ModeHITLReview, MaxAttempts: 1}},
	}
	for _, c := range cases {
		m := NewManager(nil, nil, prefs, nil)
		if err := m.Start(c.req, c.ws, "name", t.TempDir()); err != nil {
			t.Fatal(err)
		}
		m.Stop()
		if m.config != c.want {
			t.Errorf("%s %+v: got %+v want %+v", c.ws, c.req, m.config, c.want)
		}
	}
	after, _ := os.ReadFile(prefs.Path())
	if string(before) != string(after) {
		t.Error("explicit requests must not be persisted")
	}
}

func jsonPatch(s string) (preferences.AgentSettingsPatch, error) {
	var p preferences.AgentSettingsPatch
	err := json.Unmarshal([]byte(s), &p)
	return p, err
}

func mustPoolPatch(t *testing.T, s string) preferences.PoolPatch {
	t.Helper()
	var p preferences.PoolPatch
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func ptrPool(p preferences.PoolPatch) *preferences.PoolPatch { return &p }
