package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/glefebvre/opensp8c/internal/workspace"
	"github.com/go-chi/chi/v5"
)

type reviewEnv struct {
	h    *ReviewHandler
	repo string
	wsID string
	wt   string
	wc   *pool.WorktreeController
}

func reviewGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeReviewFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// newReviewEnv builds a workspace whose change "add-auth" has a branch with
// commits and, when inReview, a review marker.
func newReviewEnv(t *testing.T, inReview bool) *reviewEnv {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo, _ := filepath.Abs(t.TempDir())
	reviewGit(t, repo, "init", "-q", "-b", "main")
	writeReviewFile(t, repo, "README.md", "x\n")
	writeReviewFile(t, repo, "openspec/changes/add-auth/tasks.md", "- [x] a\n")
	reviewGit(t, repo, "add", ".")
	reviewGit(t, repo, "commit", "-q", "-m", "init")

	id := workspace.StableID(repo)
	wc := pool.NewWorktreeController(repo, id, t.TempDir())
	wt, err := wc.Provision("add-auth")
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFile(t, wt, "backend/auth.go", "package auth\n")
	writeReviewFile(t, wt, "openspec/changes/add-auth/tasks.md", "- [x] a\n- [x] b\n")
	if err := os.WriteFile(filepath.Join(wt, "logo.png"), []byte{0, 1, 0, 2}, 0644); err != nil {
		t.Fatal(err)
	}
	reviewGit(t, wt, "add", "-A")
	reviewGit(t, wt, "commit", "-q", "-m", "work")
	if inReview {
		if err := wc.MarkReview("add-auth"); err != nil {
			t.Fatal(err)
		}
	}

	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: repo}}}
	return &reviewEnv{h: NewReviewHandler(NewWorkspaceHandler(cfg, "", nil)), repo: repo, wsID: id, wt: wt, wc: wc}
}

func (e *reviewEnv) call(handler http.HandlerFunc, change, path string, withPath bool) *httptest.ResponseRecorder {
	target := "/review"
	if withPath {
		target += "?path=" + url.QueryEscape(path)
	}
	req := httptest.NewRequest("GET", target, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", e.wsID)
	rctx.URLParams.Add("name", change)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func TestReviewListSuccess(t *testing.T) {
	e := newReviewEnv(t, true)
	rec := e.call(e.h.List, "add-auth", "", false)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body)
	}
	var got reviewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Branch != "feature/add-auth" || got.Base != "main" || got.TargetAhead || len(got.Files) != 3 {
		t.Fatalf("response = %+v", got)
	}
}

func TestReviewListTargetAhead(t *testing.T) {
	e := newReviewEnv(t, true)
	writeReviewFile(t, e.repo, "other.txt", "o\n")
	reviewGit(t, e.repo, "add", ".")
	reviewGit(t, e.repo, "commit", "-q", "-m", "main advances")
	var got reviewResponse
	rec := e.call(e.h.List, "add-auth", "", false)
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if !got.TargetAhead {
		t.Fatal("target_ahead should be true")
	}
	for _, f := range got.Files {
		if f.Path == "other.txt" {
			t.Fatal("main's file listed")
		}
	}
}

func TestReviewStatusCodes(t *testing.T) {
	e := newReviewEnv(t, true)
	if rec := e.call(e.h.List, "ghost", "", false); rec.Code != http.StatusNotFound {
		t.Errorf("unknown change: %d", rec.Code)
	}
	if rec := e.call(e.h.Diff, "add-auth", "../../etc/passwd", true); rec.Code != http.StatusBadRequest {
		t.Errorf("traversal: %d", rec.Code)
	}
	if rec := e.call(e.h.Diff, "add-auth", "/etc/passwd", true); rec.Code != http.StatusBadRequest {
		t.Errorf("absolute: %d", rec.Code)
	}
	if rec := e.call(e.h.Diff, "add-auth", "README.md", true); rec.Code != http.StatusNotFound {
		t.Errorf("outside review diff: %d", rec.Code)
	}
	if rec := e.call(e.h.File, "add-auth", "README.md", true); rec.Code != http.StatusNotFound {
		t.Errorf("outside review file: %d", rec.Code)
	}
	if rec := e.call(e.h.Diff, "add-auth", "", false); rec.Code != http.StatusBadRequest {
		t.Errorf("missing path: %d", rec.Code)
	}
}

func TestReviewNotInReview(t *testing.T) {
	e := newReviewEnv(t, false)
	for name, h := range map[string]http.HandlerFunc{"list": e.h.List, "diff": e.h.Diff, "file": e.h.File} {
		rec := e.call(h, "add-auth", "backend/auth.go", true)
		if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"not_in_review"`) {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body)
		}
	}
}

func TestReviewDiffAndFile(t *testing.T) {
	e := newReviewEnv(t, true)
	rec := e.call(e.h.Diff, "add-auth", "backend/auth.go", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "+package auth") {
		t.Fatalf("diff: %d %s", rec.Code, rec.Body)
	}
	rec = e.call(e.h.File, "add-auth", "openspec/changes/add-auth/tasks.md", true)
	var file struct {
		Content string `json:"content"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &file)
	if rec.Code != http.StatusOK || file.Content != "- [x] a\n- [x] b\n" {
		t.Fatalf("file: %d %s", rec.Code, rec.Body)
	}
	rec = e.call(e.h.Diff, "add-auth", "logo.png", true)
	if !strings.Contains(rec.Body.String(), `"binary":true`) {
		t.Fatalf("binary: %s", rec.Body)
	}
}

func TestReviewWorksWithoutWorktree(t *testing.T) {
	e := newReviewEnv(t, true)
	reviewGit(t, e.repo, "worktree", "remove", "--force", e.wt)
	if rec := e.call(e.h.List, "add-auth", "", false); rec.Code != http.StatusOK {
		t.Fatalf("list without worktree: %d %s", rec.Code, rec.Body)
	}
	if rec := e.call(e.h.Diff, "add-auth", "backend/auth.go", true); rec.Code != http.StatusOK {
		t.Fatalf("diff without worktree: %d", rec.Code)
	}
}

// TestReviewIsReadOnly checks that the endpoints leave branches, the worktree
// and the review marker as they were.
func TestReviewIsReadOnly(t *testing.T) {
	e := newReviewEnv(t, true)
	snapshot := func() string {
		return strings.Join([]string{
			reviewGit(t, e.repo, "rev-parse", "main", "feature/add-auth"),
			reviewGit(t, e.wt, "status", "--porcelain"),
			reviewGit(t, e.repo, "config", "--get-regexp", "^branch\\."),
			reviewGit(t, e.repo, "for-each-ref"),
		}, "\n--\n")
	}
	before := snapshot()
	e.call(e.h.List, "add-auth", "", false)
	e.call(e.h.Diff, "add-auth", "backend/auth.go", true)
	e.call(e.h.File, "add-auth", "backend/auth.go", true)
	e.call(e.h.Diff, "add-auth", "../x", true)
	if after := snapshot(); after != before {
		t.Fatalf("repository changed:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}
