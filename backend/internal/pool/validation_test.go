package pool

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

const pkgWithTest = `{"scripts":{"test":"vitest run"}}`

func cmdStrings(root string) []string {
	var out []string
	for _, c := range DetectValidationCommands(root) {
		rel, _ := filepath.Rel(root, c.Dir)
		out = append(out, rel+": "+c.String())
	}
	return out
}

func TestDetectValidationCommands(t *testing.T) {
	t.Run("go module at root", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "go.mod"), "module x\n")
		if got := cmdStrings(root); len(got) != 1 || got[0] != ".: go test ./..." {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("backend and frontend with node_modules", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "frontend", "package.json"), pkgWithTest)
		writeFile(t, filepath.Join(root, "frontend", "node_modules", ".keep"), "")
		writeFile(t, filepath.Join(root, "backend", "go.mod"), "module x\n")
		got := cmdStrings(root)
		if len(got) != 2 || got[0] != "backend: go test ./..." || got[1] != "frontend: npm test" {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("frontend without node_modules is ignored", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "frontend", "package.json"), pkgWithTest)
		writeFile(t, filepath.Join(root, "backend", "go.mod"), "module x\n")
		if got := cmdStrings(root); len(got) != 1 || got[0] != "backend: go test ./..." {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("hidden and node_modules dirs skipped, no test script", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, ".hidden", "go.mod"), "module x\n")
		writeFile(t, filepath.Join(root, "node_modules", "go.mod"), "module x\n")
		writeFile(t, filepath.Join(root, "web", "package.json"), `{"scripts":{"build":"x"}}`)
		writeFile(t, filepath.Join(root, "web", "node_modules", ".keep"), "")
		if got := cmdStrings(root); len(got) != 0 {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("nothing recognised", func(t *testing.T) {
		if got := cmdStrings(t.TempDir()); len(got) != 0 {
			t.Fatalf("got %v", got)
		}
	})
}

func validationManager(t *testing.T, command string) *Manager {
	t.Helper()
	svc := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	if command != "" {
		var patch preferences.PoolPatch
		patch.ValidationCommand = preferences.StringPatch{Set: true, Value: command}
		if err := svc.SetPoolDefaults(patch); err != nil {
			t.Fatal(err)
		}
	}
	return NewManager(nil, nil, svc, nil, nil)
}

func TestRunValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("configured command wins over detection", func(t *testing.T) {
		wt := t.TempDir()
		writeFile(t, filepath.Join(wt, "go.mod"), "module x\n") // would run go test, which would fail (no packages are fine)
		m := validationManager(t, "touch ran-configured")
		if err := m.runValidation(ctx, &Worker{WorktreePath: wt}); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(wt, "ran-configured")); err != nil {
			t.Fatalf("configured command did not run at the worktree root: %v", err)
		}
	})

	t.Run("test failure is an ordinary error", func(t *testing.T) {
		m := validationManager(t, "echo red-test; exit 1")
		err := m.runValidation(ctx, &Worker{WorktreePath: t.TempDir()})
		var envErr *ValidationEnvError
		if err == nil || errors.As(err, &envErr) || !strings.Contains(err.Error(), "red-test") {
			t.Fatalf("expected ordinary error with output, got %v", err)
		}
	})

	t.Run("command not found", func(t *testing.T) {
		m := validationManager(t, "definitely-not-a-command-xyz --flag")
		err := m.runValidation(ctx, &Worker{WorktreePath: t.TempDir()})
		var envErr *ValidationEnvError
		if !errors.As(err, &envErr) || !strings.Contains(envErr.Reason, "definitely-not-a-command-xyz") {
			t.Fatalf("expected env error naming the command, got %v", err)
		}
	})

	t.Run("no command available", func(t *testing.T) {
		m := validationManager(t, "")
		err := m.runValidation(ctx, &Worker{WorktreePath: t.TempDir()})
		var envErr *ValidationEnvError
		if !errors.As(err, &envErr) || !strings.Contains(envErr.Reason, "Aucune commande de validation") {
			t.Fatalf("expected env error, got %v", err)
		}
	})

	t.Run("missing directory", func(t *testing.T) {
		m := validationManager(t, "true")
		err := m.runValidation(ctx, &Worker{WorktreePath: filepath.Join(t.TempDir(), "gone")})
		var envErr *ValidationEnvError
		if !errors.As(err, &envErr) {
			t.Fatalf("expected env error, got %v", err)
		}
	})

	t.Run("detected go module in subdirectory", func(t *testing.T) {
		wt := t.TempDir()
		writeFile(t, filepath.Join(wt, "backend", "go.mod"), "module x\n\ngo 1.21\n")
		writeFile(t, filepath.Join(wt, "backend", "x.go"), "package x\n")
		if err := validationManager(t, "").runValidation(ctx, &Worker{WorktreePath: wt}); err != nil {
			t.Fatalf("go test in backend/ should pass: %v", err)
		}
	})
}

// countingSubprocess answers every turn like fakeAutoRespondingSubprocess and
// counts the turns written to it (1 apply turn + N heal turns).
func countingSubprocess(turns *atomic.Int32) stubFn {
	return func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
		inR, inW := io.Pipe()
		outR, outW := io.Pipe()
		go func() {
			r := bufio.NewReader(inR)
			for {
				if _, err := r.ReadString('\n'); err != nil {
					_ = outW.Close()
					return
				}
				turns.Add(1)
				if _, err := outW.Write([]byte(`{"type":"result"}` + "\n")); err != nil {
					return
				}
			}
		}()
		return session.NewTestSubprocess(inW, outR, "claude"), nil
	}
}

func TestRunWorker_EnvErrorPausesWithoutHealing(t *testing.T) {
	changeName := "env-error-change"
	repoDir := newGoFixtureRepo(t, changeName, "- [x] done\n")

	var turns atomic.Int32
	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 3}, countingSubprocess(&turns))
	svc := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	if err := svc.SetPoolDefaults(preferences.PoolPatch{ValidationCommand: preferences.StringPatch{Set: true, Value: "not-a-real-binary-abc"}}); err != nil {
		t.Fatal(err)
	}
	m.prefs = svc

	w := &Worker{ID: 1, ActiveChange: changeName}
	m.activeWorkers[1] = w
	m.runWorker(context.Background(), w)

	if w.Status != StatusPaused || !strings.Contains(w.BlockedReason, "not-a-real-binary-abc") {
		t.Fatalf("expected paused with the command named, got %q / %q", w.Status, w.BlockedReason)
	}
	if got := turns.Load(); got != 1 {
		t.Fatalf("an environment error must not trigger a heal turn (only the apply turn), got %d turns", got)
	}
}

func TestRunWorker_RedTestStillHealsUpToMaxAttempts(t *testing.T) {
	changeName := "red-test-change"
	repoDir := newGoFixtureRepo(t, changeName, "- [x] done\n")

	var turns atomic.Int32
	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 2}, countingSubprocess(&turns))
	svc := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	if err := svc.SetPoolDefaults(preferences.PoolPatch{ValidationCommand: preferences.StringPatch{Set: true, Value: "exit 1"}}); err != nil {
		t.Fatal(err)
	}
	m.prefs = svc

	w := &Worker{ID: 1, ActiveChange: changeName}
	m.activeWorkers[1] = w
	m.runWorker(context.Background(), w)

	if got := turns.Load(); got != 3 {
		t.Fatalf("expected 1 apply + 2 heal turns, got %d", got)
	}
	if w.Status != StatusPaused || !strings.Contains(w.BlockedReason, "réparation épuisées") {
		t.Fatalf("expected heal exhaustion pause, got %q / %q", w.Status, w.BlockedReason)
	}
}
