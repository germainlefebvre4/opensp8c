package pool

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRepoFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// newReviewRepo builds a repo with a feature branch (created through Provision
// so the base is recorded) that adds, modifies, deletes, renames, and adds a
// binary and an awkward file name.
func newReviewRepo(t *testing.T) (*WorktreeController, string, string) {
	t.Helper()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "keep.txt", "keep\n")
	writeRepoFile(t, repo, "gone.txt", "bye\n")
	writeRepoFile(t, repo, "old-name.txt", "same content\nline2\nline3\nline4\n")
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-q", "-m", "base files")
	wc := newTestWC(t, repo)
	wt, err := wc.Provision("add-auth")
	if err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, wt, "README.md", "x\nmore\n")
	writeRepoFile(t, wt, "new.go", "package x\n")
	writeRepoFile(t, wt, "with space.txt", "a\n")
	writeRepoFile(t, wt, "-dash.txt", "d\n")
	if err := os.WriteFile(filepath.Join(wt, "img.bin"), []byte{0, 1, 2, 0, 3}, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(wt, "gone.txt")); err != nil {
		t.Fatal(err)
	}
	gitIn(t, wt, "mv", "old-name.txt", "new-name.txt")
	gitIn(t, wt, "add", "-A")
	gitIn(t, wt, "commit", "-q", "-m", "work")
	return wc, repo, wt
}

func reviewByPath(l ReviewList) map[string]ReviewFileEntry {
	m := map[string]ReviewFileEntry{}
	for _, f := range l.Files {
		m[f.Path] = f
	}
	return m
}

func TestReviewFiles(t *testing.T) {
	wc, _, _ := newReviewRepo(t)
	list, err := wc.ReviewFiles("add-auth")
	if err != nil {
		t.Fatal(err)
	}
	if list.Base != "main" {
		t.Fatalf("base = %q", list.Base)
	}
	got := reviewByPath(list)
	cases := map[string]string{
		"README.md":      "modified",
		"new.go":         "added",
		"with space.txt": "added",
		"-dash.txt":      "added",
		"img.bin":        "added",
		"gone.txt":       "deleted",
		"old-name.txt":   "deleted", // rename = deletion + addition
		"new-name.txt":   "added",
	}
	if len(got) != len(cases) {
		t.Fatalf("files = %+v", list.Files)
	}
	for path, status := range cases {
		if got[path].Status != status {
			t.Errorf("%s status = %q, want %q", path, got[path].Status, status)
		}
	}
	if !got["img.bin"].Binary || got["img.bin"].Additions != 0 {
		t.Errorf("img.bin = %+v", got["img.bin"])
	}
	if r := got["README.md"]; r.Additions != 2 || r.Deletions != 1 || r.Binary {
		t.Errorf("README.md = %+v", r)
	}
	if _, ok := got["keep.txt"]; ok {
		t.Error("untouched file listed")
	}
}

func TestReviewFilesBaseAdvancedAndWorktreeRemoved(t *testing.T) {
	wc, repo, wt := newReviewRepo(t)
	writeRepoFile(t, repo, "main-only.txt", "m\n")
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-q", "-m", "main advances")
	gitIn(t, repo, "worktree", "remove", "--force", wt)

	list, err := wc.ReviewFiles("add-auth")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reviewByPath(list)["main-only.txt"]; ok {
		t.Fatal("file added on main after the divergence must not be listed")
	}
	if len(list.Files) != 8 {
		t.Fatalf("files = %+v", list.Files)
	}
}

func TestReviewBaseFallbacks(t *testing.T) {
	wc, repo, _ := newReviewRepo(t)
	gitIn(t, repo, "config", baseConfigKey("add-auth"), "vanished")
	list, err := wc.ReviewFiles("add-auth")
	if err != nil {
		t.Fatal(err)
	}
	if list.Base != "HEAD" {
		t.Fatalf("base = %q, want HEAD fallback", list.Base)
	}

	gitIn(t, repo, "config", "--unset", baseConfigKey("add-auth"))
	list, err = wc.ReviewFiles("add-auth")
	if err != nil || list.Base != "main" {
		t.Fatalf("base = %q err = %v, want current branch", list.Base, err)
	}
}

func TestReviewFilesMissingBranch(t *testing.T) {
	wc := newTestWC(t, newTestRepo(t))
	if _, err := wc.ReviewFiles("nope"); !errors.Is(err, ErrReviewBranchMissing) {
		t.Fatalf("err = %v", err)
	}
}

func TestReviewPatch(t *testing.T) {
	wc, _, _ := newReviewRepo(t)
	p, err := wc.ReviewPatch("add-auth", "README.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p.Content, "@@") || !strings.Contains(p.Content, "+more") || p.Binary || p.Truncated {
		t.Fatalf("patch = %+v", p)
	}
	for _, path := range []string{"with space.txt", "-dash.txt", "gone.txt"} {
		if p, err := wc.ReviewPatch("add-auth", path); err != nil || p.Content == "" {
			t.Errorf("%s: patch = %+v err = %v", path, p, err)
		}
	}
	bin, err := wc.ReviewPatch("add-auth", "img.bin")
	if err != nil || !bin.Binary || bin.Content != "" {
		t.Fatalf("binary patch = %+v err = %v", bin, err)
	}
}

func TestReviewPatchTruncated(t *testing.T) {
	wc, _, wt := newReviewRepo(t)
	writeRepoFile(t, wt, "big.txt", strings.Repeat("0123456789abcdef\n", MaxReviewBytes/8))
	gitIn(t, wt, "add", "-A")
	gitIn(t, wt, "commit", "-q", "-m", "big")
	p, err := wc.ReviewPatch("add-auth", "big.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !p.Truncated || len(p.Content) > MaxReviewBytes {
		t.Fatalf("truncated = %v len = %d", p.Truncated, len(p.Content))
	}
	f, err := wc.ReviewFile("add-auth", "big.txt")
	if err != nil || !f.Truncated || len(f.Content) > MaxReviewBytes {
		t.Fatalf("file truncated = %v len = %d err = %v", f.Truncated, len(f.Content), err)
	}
}

func TestReviewPathValidation(t *testing.T) {
	wc, _, _ := newReviewRepo(t)
	for _, path := range []string{"", "../../etc/passwd", "a/../b", "/etc/passwd", "a\\b"} {
		if _, err := wc.ReviewPatch("add-auth", path); !errors.Is(err, ErrInvalidReviewPath) {
			t.Errorf("patch %q: err = %v", path, err)
		}
		if _, err := wc.ReviewFile("add-auth", path); !errors.Is(err, ErrInvalidReviewPath) {
			t.Errorf("file %q: err = %v", path, err)
		}
	}
	for _, path := range []string{"keep.txt", "*.go", "--output=x", "nonexistent"} {
		if _, err := wc.ReviewPatch("add-auth", path); !errors.Is(err, ErrReviewPathNotInReview) {
			t.Errorf("patch %q: err = %v", path, err)
		}
	}
}

func TestReviewFile(t *testing.T) {
	wc, _, _ := newReviewRepo(t)
	f, err := wc.ReviewFile("add-auth", "README.md")
	if err != nil || f.Content != "x\nmore\n" {
		t.Fatalf("file = %+v err = %v", f, err)
	}
	if f, err := wc.ReviewFile("add-auth", "-dash.txt"); err != nil || f.Content != "d\n" {
		t.Fatalf("dash file = %+v err = %v", f, err)
	}
	if b, err := wc.ReviewFile("add-auth", "img.bin"); err != nil || !b.Binary || b.Content != "" {
		t.Fatalf("binary file = %+v err = %v", b, err)
	}
	if _, err := wc.ReviewFile("add-auth", "gone.txt"); !errors.Is(err, ErrReviewFileDeleted) {
		t.Fatalf("deleted err = %v", err)
	}
}
