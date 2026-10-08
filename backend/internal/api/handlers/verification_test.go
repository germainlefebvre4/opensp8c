package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/go-chi/chi/v5"
)

// verifyFixture is a branchFixture whose change "c" carries the verification
// marker state.
func newVerifyFixture(t *testing.T, state, branchTasks string) (*branchFixture, *VerificationHandler) {
	t.Helper()
	f := newBranchFixture(t, true, mainTasks0of10, branchTasks, false)
	if state != "" {
		bgit(t, f.repo, "config", openspec.VerifyKey(branchChange), state)
	}
	ws := NewWorkspaceHandler(&config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: f.repo}}}, "", f.reg)
	return f, NewVerificationHandler(ws, f.reg, conversation.NewStore(t.TempDir()))
}

func (f *branchFixture) marker() string {
	return openspec.VerifyMarkers(f.repo)[branchChange]
}

func vcall(f *branchFixture, h http.HandlerFunc, method, change, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/", strings.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", f.workspaceID)
	rctx.URLParams.Add("name", change)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

const completeBranchTasks = "- [x] 1\n- [x] 2\n"

func TestVerificationState_ListAndDetail(t *testing.T) {
	cases := []struct {
		marker, wantState, wantStep string
		running                     bool
	}{
		{"pending", "queued", "", false},
		{"pending", "running", "conformity", true},
		{"failed", "failed", "", false},
		{"passed", "passed", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.marker+"/"+tc.wantState, func(t *testing.T) {
			f, _ := newVerifyFixture(t, tc.marker, branchTasks8of)
			if tc.running {
				release := pool.SeedVerificationForTest(f.reg.For(f.workspaceID), branchChange, "conformity")
				defer release()
			}
			c := listedChange(t, f.kanban, f.workspaceID, branchChange)
			if c.KanbanStatus != "verifying" || c.VerificationState != tc.wantState || c.VerificationStep != tc.wantStep {
				t.Errorf("list = %s/%s/%s", c.KanbanStatus, c.VerificationState, c.VerificationStep)
			}
			if c.TasksDone != 8 || c.TasksTotal != 10 {
				t.Errorf("progress from the branch expected, got %d/%d", c.TasksDone, c.TasksTotal)
			}
			d := f.detail(t)
			if d.KanbanStatus != "verifying" || d.VerificationState != tc.wantState || d.VerificationStep != tc.wantStep {
				t.Errorf("detail = %s/%s/%s", d.KanbanStatus, d.VerificationState, d.VerificationStep)
			}
		})
	}
}

func TestVerificationState_AbsentOutsideVerifying(t *testing.T) {
	f, _ := newVerifyFixture(t, "", branchTasks8of)
	rec, req := launchRequest("GET", f.workspaceID, branchChange, "", nil)
	f.kanban.GetChange(rec, req)
	if strings.Contains(rec.Body.String(), "verification_state") || strings.Contains(rec.Body.String(), "verification_step") {
		t.Errorf("fields must be absent: %s", rec.Body)
	}
}

func TestVerificationRerun(t *testing.T) {
	f, h := newVerifyFixture(t, "failed", completeBranchTasks)
	if rec := vcall(f, h.Rerun, "POST", branchChange, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if f.marker() != "pending" {
		t.Errorf("marker = %q", f.marker())
	}
}

func TestVerificationFinalize(t *testing.T) {
	t.Run("complete tasks", func(t *testing.T) {
		f, h := newVerifyFixture(t, "failed", completeBranchTasks)
		if rec := vcall(f, h.Finalize, "POST", branchChange, ""); rec.Code != http.StatusNoContent {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		if f.marker() != "passed" {
			t.Errorf("marker = %q", f.marker())
		}
	})
	t.Run("incomplete tasks", func(t *testing.T) {
		f, h := newVerifyFixture(t, "failed", "- [x] 1\n- [ ] 2\n- [ ] 3 manual <!-- human review required -->\n")
		rec := vcall(f, h.Finalize, "POST", branchChange, "")
		if rec.Code != http.StatusConflict {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		got := errCode(t, rec)
		if got.Code != "tasks_incomplete" || got.Remaining != 2 {
			t.Errorf("body = %+v", got)
		}
		if f.marker() != "failed" {
			t.Errorf("marker = %q", f.marker())
		}
	})
}

func TestVerificationRequestCorrection(t *testing.T) {
	f, h := newVerifyFixture(t, "failed", completeBranchTasks)
	if rec := vcall(f, h.RequestCorrection, "POST", branchChange, `{"feedback":"  "}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty feedback: %d", rec.Code)
	}
	if f.marker() != "failed" {
		t.Fatal("marker must be kept after a rejected request")
	}
	if rec := vcall(f, h.RequestCorrection, "POST", branchChange, `nope`); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid body: %d", rec.Code)
	}

	rec := vcall(f, h.RequestCorrection, "POST", branchChange, `{"feedback":"corriger le contrat de l'API"}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if f.marker() != "" {
		t.Errorf("marker must be lifted, got %q", f.marker())
	}
	got := bgit(t, f.repo, "show", "feature/c:openspec/changes/c/tasks.md")
	if !strings.Contains(got, "- [ ] Correction : corriger le contrat de l'API") {
		t.Errorf("branch tasks.md = %q", got)
	}
	// The change is eligible to a worker again.
	if c := listedChange(t, f.kanban, f.workspaceID, branchChange); c.KanbanStatus == "verifying" {
		t.Errorf("status = %s", c.KanbanStatus)
	}
}

func TestVerificationActions_ErrorCodes(t *testing.T) {
	actions := map[string]func(*VerificationHandler) http.HandlerFunc{
		"rerun":              func(h *VerificationHandler) http.HandlerFunc { return h.Rerun },
		"finalize":           func(h *VerificationHandler) http.HandlerFunc { return h.Finalize },
		"request-correction": func(h *VerificationHandler) http.HandlerFunc { return h.RequestCorrection },
	}
	for name, pick := range actions {
		t.Run(name, func(t *testing.T) {
			body := `{"feedback":"x"}`
			t.Run("unknown change", func(t *testing.T) {
				f, h := newVerifyFixture(t, "failed", completeBranchTasks)
				if rec := vcall(f, pick(h), "POST", "ghost", body); rec.Code != http.StatusNotFound {
					t.Fatalf("%d", rec.Code)
				}
			})
			t.Run("unknown workspace", func(t *testing.T) {
				f, h := newVerifyFixture(t, "failed", completeBranchTasks)
				f.workspaceID = "nope"
				if rec := vcall(f, pick(h), "POST", branchChange, body); rec.Code != http.StatusNotFound {
					t.Fatalf("%d", rec.Code)
				}
			})
			for _, state := range []string{"", "passed", "pending"} {
				t.Run("not failed/"+state, func(t *testing.T) {
					f, h := newVerifyFixture(t, state, completeBranchTasks)
					rec := vcall(f, pick(h), "POST", branchChange, body)
					if rec.Code != http.StatusConflict || errCode(t, rec).Code != "not_failed" {
						t.Fatalf("%d %s", rec.Code, rec.Body)
					}
					if f.marker() != state {
						t.Errorf("marker changed to %q", f.marker())
					}
				})
			}
			t.Run("verification running", func(t *testing.T) {
				f, h := newVerifyFixture(t, "pending", completeBranchTasks)
				release := pool.SeedVerificationForTest(f.reg.For(f.workspaceID), branchChange, "conformity")
				defer release()
				rec := vcall(f, pick(h), "POST", branchChange, body)
				if rec.Code != http.StatusConflict || errCode(t, rec).Code != "verification_running" {
					t.Fatalf("%d %s", rec.Code, rec.Body)
				}
			})
			t.Run("review busy", func(t *testing.T) {
				f, h := newVerifyFixture(t, "failed", completeBranchTasks)
				unlock, err := f.reg.For(f.workspaceID).TryLockReview(branchChange)
				if err != nil {
					t.Fatal(err)
				}
				defer unlock()
				rec := vcall(f, pick(h), "POST", branchChange, body)
				if rec.Code != http.StatusConflict || errCode(t, rec).Code != "review_busy" {
					t.Fatalf("%d %s", rec.Code, rec.Body)
				}
				if f.marker() != "failed" {
					t.Errorf("marker = %q", f.marker())
				}
			})
		})
	}
}

func TestVerificationReport(t *testing.T) {
	f, h := newVerifyFixture(t, "failed", completeBranchTasks)
	store := conversation.NewStore(t.TempDir())
	h.convStore = store

	if rec := vcall(f, h.Report, "GET", branchChange, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("no run: %d", rec.Code)
	}

	write := func(ts string, markers ...map[string]any) {
		file, err := store.OpenRun(f.workspaceID, branchChange, "verify", ts)
		if err != nil {
			t.Fatal(err)
		}
		log := conversation.NewSessionLog(file)
		for _, m := range markers {
			b, _ := json.Marshal(m)
			if err := log.WriteLine("meta", b); err != nil {
				t.Fatal(err)
			}
		}
		_ = log.Close()
	}
	write("2026-01-01T10-00-00Z",
		map[string]any{"type": "verify_run_start"},
		map[string]any{"type": "verify_run_end", "step": "conformity", "verdict": "pass", "reason": "", "report": "old"})
	write("2026-01-02T10-00-00Z",
		map[string]any{"type": "verify_run_start"},
		map[string]any{"type": "verify_run_end", "step": "conformity", "verdict": "fail", "reason": "point critique", "report": "CRITICAL issue\nVERDICT: FAIL"})

	rec := vcall(f, h.Report, "GET", branchChange, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	var got verificationReport
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Step != "conformity" || got.Verdict != "fail" || got.Reason != "point critique" || !strings.Contains(got.Report, "CRITICAL issue") || got.StartedAt != "2026-01-02T10:00:00Z" {
		t.Errorf("report = %+v", got)
	}

	// A run still in flight falls back to the last finished one.
	write("2026-01-03T10-00-00Z", map[string]any{"type": "verify_run_start"})
	rec = vcall(f, h.Report, "GET", branchChange, "")
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Verdict != "fail" {
		t.Errorf("report of an unfinished run = %+v", got)
	}
}

func TestPatchTask_RefusedWhileVerificationRuns(t *testing.T) {
	f, _ := newVerifyFixture(t, "pending", branchTasks8of)
	release := pool.SeedVerificationForTest(f.reg.For(f.workspaceID), branchChange, "conformity")
	res := f.toggle("8")
	release()
	if res.code != http.StatusConflict || !strings.Contains(res.body, `"verification_busy"`) {
		t.Fatalf("%d %s", res.code, res.body)
	}
	if got := bgit(t, f.repo, "show", "feature/c:openspec/changes/c/tasks.md"); got != strings.TrimSpace(branchTasks8of) {
		t.Errorf("branch tasks must not change: %q", got)
	}
}

func TestPatchTask_FailedVerificationTogglesBranchWithCommit(t *testing.T) {
	f, _ := newVerifyFixture(t, "failed", branchTasks8of)
	if res := f.toggle("8"); res.code != http.StatusOK {
		t.Fatalf("%d %s", res.code, res.body)
	}
	if got := bgit(t, f.repo, "show", "feature/c:openspec/changes/c/tasks.md"); !strings.Contains(got, "- [x] 9 manual") {
		t.Errorf("branch tasks = %q", got)
	}
	if f.marker() != "failed" {
		t.Errorf("marker = %q, want failed", f.marker())
	}
	if n := bgit(t, f.repo, "rev-list", "--count", "main..feature/c"); n != "2" {
		t.Errorf("one commit per tick expected, commits ahead = %s", n)
	}
}

func TestResetTasks_FailedVerificationRemovesBranchWorktreeAndMarker(t *testing.T) {
	f, _ := newVerifyFixture(t, "failed", branchTasks8of)
	if res := f.toggle("8"); res.code != http.StatusOK { // provisions the worktree
		t.Fatalf("toggle: %d %s", res.code, res.body)
	}
	if code, body := f.reset(t); code != http.StatusNoContent {
		t.Fatalf("reset: %d %s", code, body)
	}
	if branchExistsIn(t, f.repo) {
		t.Error("branch must be deleted")
	}
	f.noWorktree(t)
	if f.marker() != "" {
		t.Errorf("marker = %q", f.marker())
	}
	if c := listedChange(t, f.kanban, f.workspaceID, branchChange); c.KanbanStatus != "to-explore" {
		t.Errorf("status = %s", c.KanbanStatus)
	}
}

func TestResetTasks_RefusedWhileVerificationRuns(t *testing.T) {
	f, _ := newVerifyFixture(t, "pending", branchTasks8of)
	release := pool.SeedVerificationForTest(f.reg.For(f.workspaceID), branchChange, "conformity")
	defer release()
	code, body := f.reset(t)
	if code != http.StatusConflict || !strings.Contains(body, `"verification_busy"`) {
		t.Fatalf("%d %s", code, body)
	}
	if !branchExistsIn(t, f.repo) || f.marker() != "pending" {
		t.Error("nothing may be removed")
	}
}

func TestDeleteChange_VerifyingCleansBranchAndMarker(t *testing.T) {
	f, _ := newVerifyFixture(t, "failed", branchTasks8of)
	rec, req := deleteChangeRequest(f.workspaceID, branchChange)
	f.kanban.DeleteChange(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if branchExistsIn(t, f.repo) || f.marker() != "" {
		t.Error("branch and marker must be removed")
	}
}
