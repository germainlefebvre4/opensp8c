package docsgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeSpec(t *testing.T, workspacePath, capability, content string) string {
	t.Helper()
	dir := filepath.Join(workspacePath, "openspec", "specs", capability)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}
	path := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write spec.md: %v", err)
	}
	return path
}

func writePage(t *testing.T, workspacePath, page, content string) string {
	t.Helper()
	dir := OutputDir(workspacePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}
	path := filepath.Join(dir, page+".md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write page: %v", err)
	}
	return path
}

// TestBuildPromptIncludesAllSpecFiles verifies the assembled prompt includes
// opensp8c's formalism plus the content of every spec.md found under a test
// openspec/specs/ directory.
func TestBuildPromptIncludesAllSpecFiles(t *testing.T) {
	tmpDir := t.TempDir()
	writeSpec(t, tmpDir, "capability-a", "Requirement A content")
	writeSpec(t, tmpDir, "capability-b", "Requirement B content")

	prompt, err := BuildPrompt(tmpDir)
	if err != nil {
		t.Fatalf("BuildPrompt failed: %v", err)
	}

	for _, want := range []string{"Requirement A content", "Requirement B content", "capability-a", "capability-b"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("expected prompt to contain %q, got: %s", want, prompt)
		}
	}
}

// TestListPagesOrderAndFiltering verifies ListPages returns only the pages
// actually present on disk, in the fixed order.
func TestListPagesOrderAndFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	writePage(t, tmpDir, "workflows", "w")
	writePage(t, tmpDir, "overview", "o")
	writePage(t, tmpDir, "domain-model", "d")

	pages, err := ListPages(tmpDir)
	if err != nil {
		t.Fatalf("ListPages failed: %v", err)
	}

	want := []string{"overview", "domain-model", "workflows"}
	if len(pages) != len(want) {
		t.Fatalf("expected pages %v, got %v", want, pages)
	}
	for i := range want {
		if pages[i] != want[i] {
			t.Errorf("expected pages %v, got %v", want, pages)
			break
		}
	}
}

// TestListPagesEmptyWhenNoneGenerated verifies an absent docs/opensp8c/
// directory yields an empty (non-nil) page list.
func TestListPagesEmptyWhenNoneGenerated(t *testing.T) {
	tmpDir := t.TempDir()
	pages, err := ListPages(tmpDir)
	if err != nil {
		t.Fatalf("ListPages failed: %v", err)
	}
	if len(pages) != 0 {
		t.Fatalf("expected no pages, got %v", pages)
	}
}

// TestIsStale_SpecNewerThanDocs verifies staleness is signaled when a spec
// file was modified after all generated pages.
func TestIsStale_SpecNewerThanDocs(t *testing.T) {
	tmpDir := t.TempDir()
	pagePath := writePage(t, tmpDir, "overview", "content")
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(pagePath, old, old); err != nil {
		t.Fatalf("failed to set page mtime: %v", err)
	}

	writeSpec(t, tmpDir, "capability-a", "content")

	stale, err := IsStale(tmpDir)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if !stale {
		t.Fatal("expected documentation to be reported as stale")
	}
}

// TestIsStale_DocsUpToDate verifies no staleness is signaled when every
// generated page is newer than every spec file.
func TestIsStale_DocsUpToDate(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := writeSpec(t, tmpDir, "capability-a", "content")
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(specPath, old, old); err != nil {
		t.Fatalf("failed to set spec mtime: %v", err)
	}

	writePage(t, tmpDir, "overview", "content")

	stale, err := IsStale(tmpDir)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if stale {
		t.Fatal("expected documentation to be reported as up to date")
	}
}

// TestIsStale_NoPagesGenerated verifies no staleness is signaled when no
// documentation page has ever been generated.
func TestIsStale_NoPagesGenerated(t *testing.T) {
	tmpDir := t.TempDir()
	writeSpec(t, tmpDir, "capability-a", "content")

	stale, err := IsStale(tmpDir)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if stale {
		t.Fatal("expected no staleness signal when no pages were ever generated")
	}
}

// TestReadPage_NotFound verifies ReadPage reports ErrPageNotFound for a page
// that doesn't exist on disk.
func TestReadPage_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	if _, err := ReadPage(tmpDir, "overview"); err != ErrPageNotFound {
		t.Fatalf("expected ErrPageNotFound, got %v", err)
	}
}

// TestReadPage_UnknownPageID verifies ReadPage rejects a page id outside the
// fixed PageOrder rather than reading an arbitrary path.
func TestReadPage_UnknownPageID(t *testing.T) {
	tmpDir := t.TempDir()
	if _, err := ReadPage(tmpDir, "../../etc/passwd"); err != ErrPageNotFound {
		t.Fatalf("expected ErrPageNotFound for unknown page id, got %v", err)
	}
}
