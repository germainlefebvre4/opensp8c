package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/go-chi/chi/v5"
)

func newActionsHandler(t *testing.T, e *reviewEnv) *ReviewActionsHandler {
	t.Helper()
	// HOME is a throwaway: give git an identity for the merge commits.
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME"} {
		t.Setenv(k, "t")
	}
	for _, k := range []string{"GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "t@t")
	}
	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: e.repo}}}
	return NewReviewActionsHandler(NewWorkspaceHandler(cfg, "", nil), pool.NewRegistry(nil, nil, nil, nil, nil))
}

func (e *reviewEnv) post(h http.HandlerFunc, change, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", e.wsID)
	rctx.URLParams.Add("name", change)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func errCode(t *testing.T, rec *httptest.ResponseRecorder) reviewActionError {
	t.Helper()
	var got reviewActionError
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body %q: %v", rec.Body, err)
	}
	return got
}

func mainCommit(t *testing.T, e *reviewEnv, rel, content string) {
	t.Helper()
	writeReviewFile(t, e.repo, rel, content)
	reviewGit(t, e.repo, "add", ".")
	reviewGit(t, e.repo, "commit", "-q", "-m", "main "+rel)
}

func branchKept(t *testing.T, e *reviewEnv) {
	t.Helper()
	if reviewGit(t, e.repo, "branch", "--list", "feature/add-auth") == "" {
		t.Fatal("branch must be kept")
	}
	if has, _ := e.wc.HasReview("add-auth"); !has {
		t.Fatal("review marker must be kept")
	}
}

func TestApprove_Success(t *testing.T) {
	e := newReviewEnv(t, true)
	h := newActionsHandler(t, e)
	rec := e.post(h.Approve, "add-auth", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body)
	}
	var got map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["target"] != "main" {
		t.Fatalf("target = %q", got["target"])
	}
	b, _ := os.ReadFile(filepath.Join(e.repo, "openspec/changes/add-auth/tasks.md"))
	if !strings.Contains(string(b), "- [x] b") {
		t.Fatalf("tasks.md of main not merged: %q", b)
	}
	if reviewGit(t, e.repo, "branch", "--list", "feature/add-auth") != "" {
		t.Fatal("branch must be deleted")
	}
}

func TestApprove_SuccessHasNoWarningKey(t *testing.T) {
	e := newReviewEnv(t, true)
	rec := e.post(newActionsHandler(t, e).Approve, "add-auth", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "warning") {
		t.Fatalf("no warning expected: %s", rec.Body)
	}
}

func TestApprove_CleanupIncompleteIsAWarning(t *testing.T) {
	e := newReviewEnv(t, true)
	// An untracked file makes `git worktree remove` refuse the removal.
	writeReviewFile(t, e.wt, "junk.txt", "untracked")

	rec := e.post(newActionsHandler(t, e).Approve, "add-auth", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body)
	}
	var got approveResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Target != "main" || got.Warning == nil || got.Warning.Code != "cleanup_incomplete" ||
		len(got.Warning.Remaining) != 1 || got.Warning.Remaining[0] != "worktree" || got.Warning.Message == "" {
		t.Fatalf("body = %s", rec.Body)
	}
	if has, _ := e.wc.HasReview("add-auth"); has {
		t.Fatal("review marker must be lifted")
	}
}

func TestApprove_Refusals(t *testing.T) {
	t.Run("not in review", func(t *testing.T) {
		e := newReviewEnv(t, false)
		rec := e.post(newActionsHandler(t, e).Approve, "add-auth", "")
		if rec.Code != http.StatusConflict || errCode(t, rec).Code != "not_in_review" {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
	})
	t.Run("unknown change", func(t *testing.T) {
		e := newReviewEnv(t, true)
		if rec := e.post(newActionsHandler(t, e).Approve, "ghost", ""); rec.Code != http.StatusNotFound {
			t.Fatalf("%d", rec.Code)
		}
	})
	t.Run("base mismatch", func(t *testing.T) {
		e := newReviewEnv(t, true)
		reviewGit(t, e.repo, "checkout", "-q", "-b", "other")
		rec := e.post(newActionsHandler(t, e).Approve, "add-auth", "")
		if rec.Code != http.StatusConflict || errCode(t, rec).Code != "base_branch_mismatch" {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		branchKept(t, e)
	})
	t.Run("merge in progress", func(t *testing.T) {
		e := newReviewEnv(t, true)
		writeReviewFile(t, e.repo, ".git/MERGE_HEAD", reviewGit(t, e.repo, "rev-parse", "HEAD")+"\n")
		rec := e.post(newActionsHandler(t, e).Approve, "add-auth", "")
		if rec.Code != http.StatusConflict || errCode(t, rec).Code != "merge_in_progress" {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		branchKept(t, e)
	})
	t.Run("integration conflict", func(t *testing.T) {
		e := newReviewEnv(t, true)
		mainCommit(t, e, "backend/auth.go", "package other\n")
		rec := e.post(newActionsHandler(t, e).Approve, "add-auth", "")
		if rec.Code != http.StatusConflict || errCode(t, rec).Code != "integration_conflict" {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		branchKept(t, e)
	})
	t.Run("validation failed", func(t *testing.T) {
		e := newReviewEnv(t, true)
		// main advanced and the fixture has no validation command to run.
		mainCommit(t, e, "other.txt", "o\n")
		rec := e.post(newActionsHandler(t, e).Approve, "add-auth", "")
		got := errCode(t, rec)
		if rec.Code != http.StatusUnprocessableEntity || got.Code != "validation_failed" || got.Output == "" {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		branchKept(t, e)
	})
}

func TestApproveFailureMapping(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{pool.ErrNotInReview, 409, "not_in_review"},
		{pool.ErrWorkerActive, 409, "worker_active"},
		{pool.ErrMergeInProgress, 409, "merge_in_progress"},
		{&pool.BaseBranchMismatchError{Base: "main", Current: "x"}, 409, "base_branch_mismatch"},
		{&pool.IntegrationConflictError{Target: "main", Err: errors.New("c")}, 409, "integration_conflict"},
		{&pool.TargetMovingError{Rounds: 3}, 409, "target_moving"},
		{&pool.ValidationFailedError{Err: errors.New("tests failed")}, 422, "validation_failed"},
		{&pool.ValidationEnvError{Reason: "no command"}, 422, "validation_failed"},
		{&pool.TasksPendingError{Remaining: 2}, 409, "tasks_pending"},
		{errors.New("boom"), 500, "approve_failed"},
	}
	for _, c := range cases {
		status, body := approveFailure(c.err)
		if status != c.status || body.Code != c.code {
			t.Errorf("%v: got %d %q, want %d %q", c.err, status, body.Code, c.status, c.code)
		}
	}
}

func TestApproveFailureMapping_TasksPendingRemaining(t *testing.T) {
	_, body := approveFailure(&pool.TasksPendingError{Remaining: 2})
	if body.Remaining != 2 || body.Message == "" {
		t.Errorf("unexpected body: %+v", body)
	}
	raw, _ := json.Marshal(reviewActionError{Code: "not_in_review"})
	if strings.Contains(string(raw), "remaining") {
		t.Errorf("remaining must be omitted when absent: %s", raw)
	}
}

func TestRequestCorrection_Handler(t *testing.T) {
	e := newReviewEnv(t, true)
	h := newActionsHandler(t, e)

	if rec := e.post(h.RequestCorrection, "add-auth", `{"feedback":"  "}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty feedback: %d", rec.Code)
	}
	if rec := e.post(h.RequestCorrection, "add-auth", `nope`); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid body: %d", rec.Code)
	}
	if has, _ := e.wc.HasReview("add-auth"); !has {
		t.Fatal("marker must be kept after a rejected request")
	}

	rec := e.post(h.RequestCorrection, "add-auth", `{"feedback":"Le bouton ne ferme pas\nle dialogue"}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body)
	}
	if has, _ := e.wc.HasReview("add-auth"); has {
		t.Fatal("marker must be lifted")
	}
	if got := reviewGit(t, e.repo, "show", "feature/add-auth:openspec/changes/add-auth/tasks.md"); !strings.Contains(got, "- [ ] Correction : Le bouton ne ferme pas") {
		t.Fatalf("tasks.md of the branch: %q", got)
	}

	rec = e.post(h.RequestCorrection, "add-auth", `{"feedback":"encore"}`)
	if rec.Code != http.StatusConflict || errCode(t, rec).Code != "not_in_review" {
		t.Fatalf("second request: %d %s", rec.Code, rec.Body)
	}
}

func TestRequestCorrection_NotInReview(t *testing.T) {
	e := newReviewEnv(t, false)
	rec := e.post(newActionsHandler(t, e).RequestCorrection, "add-auth", `{"feedback":"x"}`)
	if rec.Code != http.StatusConflict || errCode(t, rec).Code != "not_in_review" {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
}

func TestWriteRefusal_WorkerActive(t *testing.T) {
	rec := httptest.NewRecorder()
	if !writeRefusal(rec, pool.ErrWorkerActive) || rec.Code != http.StatusConflict || errCode(t, rec).Code != "worker_active" {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
}

func deleteChange(t *testing.T, e *reviewEnv) *httptest.ResponseRecorder {
	t.Helper()
	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: e.repo}}}
	h := NewKanbanHandler(NewWorkspaceHandler(cfg, "", nil), nil, pool.NewRegistry(nil, nil, nil, nil, nil), nil, nil, nil, "")
	rec, req := deleteChangeRequest(e.wsID, "add-auth")
	h.DeleteChange(rec, req)
	return rec
}

func TestDeleteChange_InReviewCleansBranchWorktreeAndMarker(t *testing.T) {
	e := newReviewEnv(t, true)
	if rec := deleteChange(t, e); rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body)
	}
	if _, err := os.Stat(filepath.Join(e.repo, "openspec", "changes", "add-auth")); !os.IsNotExist(err) {
		t.Fatal("change folder must be gone")
	}
	if _, err := os.Stat(e.wt); !os.IsNotExist(err) {
		t.Fatal("worktree must be gone")
	}
	if reviewGit(t, e.repo, "branch", "--list", "feature/add-auth") != "" {
		t.Fatal("branch must be gone")
	}
	if has, _ := e.wc.HasReview("add-auth"); has {
		t.Fatal("marker must be gone")
	}
}

func TestDeleteChange_InReviewCleanupFailureKeepsChange(t *testing.T) {
	e := newReviewEnv(t, true)
	// A locked worktree resists a single --force removal.
	reviewGit(t, e.repo, "worktree", "lock", e.wt)
	rec := deleteChange(t, e)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body)
	}
	if _, err := os.Stat(filepath.Join(e.repo, "openspec", "changes", "add-auth")); err != nil {
		t.Fatal("change folder must be kept")
	}
	branchKept(t, e)
}
