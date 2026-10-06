package pool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCommitType(t *testing.T) {
	cases := map[string]string{
		"fix-login": "fix", "refactor-x": "refactor", "perf-x": "perf", "doc-x": "docs",
		"docs-x": "docs", "test-x": "test", "chore-x": "chore", "ci-x": "ci", "build-x": "build",
		"style-x": "style", "add-auth": "feat", "improve-x": "feat", "Fix-Mixed": "fix", "FIX_x": "fix",
		"": "feat",
	}
	for in, want := range cases {
		if got := commitType(in); got != want {
			t.Errorf("commitType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCommitSubject(t *testing.T) {
	if got := commitSubject("improve-matrix-change-drilldown-nav"); got != "Improve matrix change drilldown nav" {
		t.Fatalf("subject = %q", got)
	}
	if got := commitSubject("fix_the-Thing"); got != "Fix the Thing" {
		t.Fatalf("subject = %q", got)
	}
}

func TestNormalizeScope(t *testing.T) {
	cases := map[string]string{
		"Agent Pool": "agent-pool", "timeline-spec-matrix": "timeline-spec-matrix",
		"  --a__b!!c-- ": "a__b-c", "!!!": "", "": "", "a/b.c": "a/b.c",
	}
	for in, want := range cases {
		if got := normalizeScope(in); got != want {
			t.Errorf("normalizeScope(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestChangeCommitMessage(t *testing.T) {
	if got := changeCommitMessage("fix-login-redirect", ""); got != "fix: Fix login redirect\n\nChange: fix-login-redirect" {
		t.Fatalf("no scope: %q", got)
	}
	got := changeCommitMessage("improve-matrix-change-drilldown-nav", "Timeline Spec Matrix")
	want := "feat(timeline-spec-matrix): Improve matrix change drilldown nav\n\nChange: improve-matrix-change-drilldown-nav"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := correctionCommitMessage("fix-x", "auth"); got != "chore(auth): Add review correction\n\nChange: fix-x" {
		t.Fatalf("correction: %q", got)
	}
}

func TestCommitHeaderTruncatesOnWordBoundary(t *testing.T) {
	name := "add-a-very-long-change-name-that-describes-many-many-different-things-at-once"
	msg := changeCommitMessage(name, "scope")
	header, body, _ := strings.Cut(msg, "\n\n")
	if n := utf8.RuneCountInString(header); n > maxHeaderLen {
		t.Fatalf("header %d chars: %q", n, header)
	}
	if !strings.HasPrefix(header, "feat(scope): Add a very long") || strings.HasSuffix(header, " ") {
		t.Fatalf("header = %q", header)
	}
	full := commitSubject(name)
	subject := strings.TrimPrefix(header, "feat(scope): ")
	if !strings.HasPrefix(full, subject) || full[len(subject)] != ' ' {
		t.Fatalf("not cut on a word boundary: %q", header)
	}
	if body != "Change: "+name {
		t.Fatalf("body = %q", body)
	}
}

func writeScopeFiles(t *testing.T, dir string, yaml string, specs ...string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if yaml != "" {
		writeFile(t, filepath.Join(dir, ".openspec.yaml"), yaml)
	}
	for _, s := range specs {
		if err := os.MkdirAll(filepath.Join(dir, "specs", s), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestChangeScope(t *testing.T) {
	repo := newTestRepo(t)
	wc := newTestWC(t, repo)
	dir := func(n string) string { return filepath.Join(repo, "openspec", "changes", n) }

	writeScopeFiles(t, dir("tagged"), "schema: spec-driven\ntags:\n  components: [\"\", Timeline Matrix, other]\n", "zzz")
	writeScopeFiles(t, dir("specsonly"), "schema: spec-driven\n", "task-toggle", "kanban-board")
	writeScopeFiles(t, dir("badyaml"), "tags: [: not yaml", "alpha")
	writeScopeFiles(t, dir("empty"), "")

	for name, want := range map[string]string{
		"tagged": "timeline-matrix", "specsonly": "kanban-board", "badyaml": "alpha", "empty": "", "missing": "",
	} {
		if got := wc.changeScope(name); got != want {
			t.Errorf("changeScope(%q) = %q, want %q", name, got, want)
		}
	}

	// Change folder present only in the worktree.
	path, err := wc.Provision("wtonly")
	if err != nil {
		t.Fatal(err)
	}
	writeScopeFiles(t, filepath.Join(path, "openspec", "changes", "wtonly"), "tags:\n  components: [auth]\n")
	if got := wc.changeScope("wtonly"); got != "auth" {
		t.Fatalf("worktree-only scope = %q", got)
	}
}
