package pool

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
)

func TestRunWorker_PassesWorkerLanguageDirective(t *testing.T) {
	repoDir := newGoFixtureRepo(t, "lang-change", "- [x] done\n")
	prefs := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	_ = prefs.SetUILocale("fr")
	code, docs := "en", "auto"
	_ = prefs.SetAgentLanguages(preferences.AgentLanguagesUpdate{Code: &code, Documentation: &docs})

	var received []string
	m := newWorkerTestManager(t, repoDir, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1},
		func(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*session.Subprocess, error) {
			received = append(received, languageDirective)
			return fakeAutoRespondingSubprocess(), nil
		})
	m.prefs = prefs

	run := func(id int, change string) {
		w := &Worker{ID: id, ActiveChange: change}
		m.activeWorkers[id] = w
		m.runWorker(context.Background(), w)
	}

	run(1, "lang-change")
	if len(received) != 1 {
		t.Fatalf("expected 1 launch, got %d", len(received))
	}
	first := received[0]
	// code language (English) and documentation language (French, via auto).
	if !strings.Contains(first, "in English") || !strings.Contains(first, "in French") {
		t.Errorf("worker directive should mention both languages: %s", first)
	}

	// A setting change applies to the next launch only; the earlier one keeps its directive.
	newDocs := "en"
	_ = prefs.SetAgentLanguages(preferences.AgentLanguagesUpdate{Documentation: &newDocs})
	m.workspacePath = newGoFixtureRepo(t, "lang-change-2", "- [x] done\n")
	run(2, "lang-change-2")
	if len(received) != 2 {
		t.Fatalf("expected 2 launches, got %d", len(received))
	}
	if received[0] != first {
		t.Errorf("first directive mutated after settings change")
	}
	if strings.Contains(received[1], "in French") {
		t.Errorf("second launch should use the new documentation language: %s", received[1])
	}
}
