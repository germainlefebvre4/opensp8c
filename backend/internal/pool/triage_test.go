package pool

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/session"
)

// turnAgent answers every turn with okResult and calls onTurn(n, worktree)
// first, n being the 1-based turn number. It records the system prompts it
// was started with.
type turnAgent struct {
	mu      sync.Mutex
	turns   int
	prompts []string
}

func (a *turnAgent) start(onTurn func(n int, ws string)) startFn {
	return func(ctx context.Context, ws string, cfg agents.AgentConfig, extra, sid string, resume bool, l *conversation.SessionLog, env map[string]string, nq bool, lang string) (*session.Subprocess, error) {
		a.mu.Lock()
		a.prompts = append(a.prompts, extra+"|"+lang)
		a.mu.Unlock()
		return pipeAgent(func(ws string) {
			a.mu.Lock()
			a.turns++
			n := a.turns
			a.mu.Unlock()
			if onTurn != nil {
				onTurn(n, ws)
			}
		}, okResult, nil)(ctx, ws, cfg, extra, sid, resume, l, env, nq, lang)
	}
}

func (a *turnAgent) turnCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.turns
}

func rewriteTasks(ws, change, content string) {
	_ = os.WriteFile(filepath.Join(ws, "openspec", "changes", change, "tasks.md"), []byte(content), 0644)
}

func triageManager(t *testing.T, repo string, mode DelegationMode, a *turnAgent, onTurn func(int, string)) (*Manager, *activity.Store) {
	t.Helper()
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: mode, MaxAttempts: 1}, a.start(onTurn))
	store := activity.NewStore(t.TempDir(), nil)
	m.activityStore = store
	return m, store
}

func runTriage(t *testing.T, m *Manager, change string) *Worker {
	t.Helper()
	w := &Worker{ID: 1, WorkspaceID: "ws1", ActiveChange: change}
	m.activeWorkers[1] = w
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.runWorker(ctx, w)
	return w
}

func flaggedEntries(t *testing.T, store *activity.Store, change string) []activity.Entry {
	t.Helper()
	entries, _ := store.Read("ws1", change)
	var out []activity.Entry
	for _, e := range entries {
		if e.Type == "pool.task_flagged" {
			out = append(out, e)
		}
	}
	return out
}

func TestTriage_AgentFlagsTask_GoesToReview(t *testing.T) {
	const change = "triage-flag"
	repo := newGoFixtureRepo(t, change, "- [x] one\n- [ ] Parcours manuel\n")
	a := &turnAgent{}
	m, store := triageManager(t, repo, ModeHITLReview, a, func(n int, ws string) {
		if n == 2 {
			rewriteTasks(ws, change, "- [x] one\n- [ ] Parcours manuel <!-- human review required -->\n")
		}
	})
	w := runTriage(t, m, change)
	if w.Status == StatusPaused {
		t.Fatalf("worker paused: %s", w.BlockedReason)
	}
	if !openspec.ReviewMarkers(repo)[change] {
		t.Error("change should be in review")
	}
	if got := flaggedEntries(t, store, change); len(got) != 1 || !strings.Contains(got[0].Summary, "Parcours manuel") || strings.Contains(got[0].Summary, "<!--") {
		t.Errorf("unexpected flagged entries: %+v", got)
	}
}

func TestTriage_AgentCompletesTask_GoesToReview(t *testing.T) {
	const change = "triage-complete"
	repo := newGoFixtureRepo(t, change, "- [ ] one\n")
	a := &turnAgent{}
	m, store := triageManager(t, repo, ModeHITLReview, a, func(n int, ws string) {
		if n == 2 {
			rewriteTasks(ws, change, "- [x] one\n")
		}
	})
	w := runTriage(t, m, change)
	if w.Status == StatusPaused {
		t.Fatalf("worker paused: %s", w.BlockedReason)
	}
	if len(flaggedEntries(t, store, change)) != 0 {
		t.Error("no flagged entry expected")
	}
	if !openspec.ReviewMarkers(repo)[change] {
		t.Error("change should be in review")
	}
}

func TestTriage_AgentDoesNothing_Pauses(t *testing.T) {
	const change = "triage-nothing"
	repo := newGoFixtureRepo(t, change, "- [x] a\n- [ ] b\n")
	a := &turnAgent{}
	m, _ := triageManager(t, repo, ModeHITLReview, a, nil)
	w := runTriage(t, m, change)
	if w.Status != StatusPaused || !strings.Contains(w.BlockedReason, "tâches restantes incomplètes") {
		t.Errorf("expected pause, got %q / %q", w.Status, w.BlockedReason)
	}
	if a.turnCount() != 2 {
		t.Errorf("expected apply + one triage turn, got %d", a.turnCount())
	}
}

func TestTriage_NotSentWhenAllChecked(t *testing.T) {
	const change = "triage-allchecked"
	repo := newGoFixtureRepo(t, change, "- [x] a\n")
	a := &turnAgent{}
	m, _ := triageManager(t, repo, ModeHITLReview, a, nil)
	runTriage(t, m, change)
	if a.turnCount() != 1 {
		t.Errorf("expected only the apply turn, got %d", a.turnCount())
	}
}

func TestTriage_NotSentInFullAutonomy(t *testing.T) {
	const change = "triage-fullauto"
	repo := newGoFixtureRepo(t, change, "- [x] a\n- [ ] b <!-- human review required -->\n")
	a := &turnAgent{}
	m, _ := triageManager(t, repo, ModeFullAutonomy, a, nil)
	w := runTriage(t, m, change)
	if a.turnCount() != 1 {
		t.Errorf("expected only the apply turn, got %d", a.turnCount())
	}
	if w.Status != StatusPaused {
		t.Errorf("full-autonomy must pause with a flagged task left, got %q", w.Status)
	}
}

func TestTriage_AlreadyFlaggedTask_NoTriageNoEntry(t *testing.T) {
	const change = "triage-preflagged"
	repo := newGoFixtureRepo(t, change, "- [x] a\n- [ ] b <!-- human review required -->\n")
	a := &turnAgent{}
	m, store := triageManager(t, repo, ModeHITLReview, a, func(_ int, ws string) { writeInWorktree("w.txt")(ws) })
	w := runTriage(t, m, change)
	if w.Status == StatusPaused {
		t.Fatalf("worker paused: %s", w.BlockedReason)
	}
	if a.turnCount() != 1 || len(flaggedEntries(t, store, change)) != 0 {
		t.Errorf("no triage nor entry expected (turns=%d)", a.turnCount())
	}
	if !openspec.ReviewMarkers(repo)[change] {
		t.Error("change should be in review")
	}
}

func TestTriage_TurnError_Pauses(t *testing.T) {
	const change = "triage-error"
	repo := newGoFixtureRepo(t, change, "- [ ] a\n")
	m := newWorkerTestManager(t, repo, AgentPoolConfig{Size: 1, DelegationMode: ModeHITLReview, MaxAttempts: 1},
		func(ctx context.Context, ws string, cfg agents.AgentConfig, extra, sid string, resume bool, l *conversation.SessionLog, env map[string]string, nq bool, lang string) (*session.Subprocess, error) {
			inR, inW := io.Pipe()
			outR, outW := io.Pipe()
			go func() {
				r := bufio.NewReader(inR)
				for turn := 1; ; turn++ {
					if _, err := r.ReadString('\n'); err != nil {
						_ = outW.Close()
						return
					}
					line := okResult
					if turn == 2 {
						line = `{"type":"result","subtype":"error_during_execution","is_error":true,"result":"agent exploded"}`
					}
					_, _ = outW.Write([]byte(line + "\n"))
				}
			}()
			return session.NewTestSubprocess(inW, outR, "claude"), nil
		})
	w := runTriage(t, m, change)
	if w.Status != StatusPaused || !strings.Contains(w.BlockedReason, "agent exploded") {
		t.Errorf("expected pause with the agent's reason, got %q / %q", w.Status, w.BlockedReason)
	}
}

func TestHumanReviewDirective_OnlyInHITL(t *testing.T) {
	for mode, want := range map[DelegationMode]bool{ModeHITLReview: true, ModeFullAutonomy: false} {
		change := "directive-" + string(mode)
		repo := newGoFixtureRepo(t, change, "- [x] a\n")
		a := &turnAgent{}
		m, _ := triageManager(t, repo, mode, a, nil)
		runTriage(t, m, change)
		if len(a.prompts) != 1 {
			t.Fatalf("%s: expected 1 start, got %d", mode, len(a.prompts))
		}
		if got := strings.Contains(a.prompts[0], openspec.HumanReviewMarker); got != want {
			t.Errorf("%s: directive present = %v, want %v", mode, got, want)
		}
		if want && !strings.Contains(a.prompts[0], "|") {
			t.Error("language separator missing")
		}
	}
}

func TestTasksBlockCompletion(t *testing.T) {
	st := openspec.TaskStats{Done: 8, Total: 10, PendingHuman: 2}
	if tasksBlockCompletion(ModeHITLReview, st) {
		t.Error("hitl-review: flagged-only remaining must not block")
	}
	if !tasksBlockCompletion(ModeFullAutonomy, st) {
		t.Error("full-autonomy: any unchecked task blocks")
	}
	st = openspec.TaskStats{Done: 9, Total: 10, PendingOther: 1}
	if !tasksBlockCompletion(ModeHITLReview, st) {
		t.Error("hitl-review: unflagged remaining blocks")
	}
}
