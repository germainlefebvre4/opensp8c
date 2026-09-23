// Package docsgen assembles the generation prompt and reads back the fixed
// set of documentation pages ("overview", "architecture", "domain-model",
// "workflows") that opensp8c generates from a workspace's raw OpenSpec
// specs and persists under docs/opensp8c/ at the root of the target
// project, per the spec-documentation-generation capability.
package docsgen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/glefebvre/opensp8c/internal/agents"
)

// PageOrder is the fixed, invariable page list and display order.
var PageOrder = []string{"overview", "architecture", "domain-model", "workflows"}

// ErrPageNotFound is returned by ReadPage when the requested page does not
// exist under docs/opensp8c/, and by ReadPage/OutputDir callers guarding
// against an unknown page id.
var ErrPageNotFound = errors.New("docs page not found")

// OutputDir returns the fixed, invariable output directory for generated
// documentation pages, relative to workspacePath.
func OutputDir(workspacePath string) string {
	return filepath.Join(workspacePath, "docs", "opensp8c")
}

func pagePath(workspacePath, page string) string {
	return filepath.Join(OutputDir(workspacePath), page+".md")
}

func specsGlob(workspacePath string) string {
	return filepath.Join(workspacePath, "openspec", "specs", "*", "spec.md")
}

// BuildPrompt assembles the final generation prompt: opensp8c's fixed
// formalism (agents.DocsFormalismPrompt) followed by the concatenated
// content of every openspec/specs/**/spec.md file found in workspacePath.
func BuildPrompt(workspacePath string) (string, error) {
	matches, err := filepath.Glob(specsGlob(workspacePath))
	if err != nil {
		return "", err
	}
	sort.Strings(matches)

	var b strings.Builder
	b.WriteString(agents.DocsFormalismPrompt)
	b.WriteString("\n\nHere are the raw OpenSpec capability specs for this project:\n")

	for _, specPath := range matches {
		data, err := os.ReadFile(specPath)
		if err != nil {
			continue
		}
		capability := filepath.Base(filepath.Dir(specPath))
		fmt.Fprintf(&b, "\n---\n## Capability: %s\n\n%s\n", capability, string(data))
	}

	return b.String(), nil
}

// ListPages returns the subset of PageOrder whose file is actually present
// under docs/opensp8c/, in fixed order, filtering out anything absent.
func ListPages(workspacePath string) ([]string, error) {
	var pages []string
	for _, page := range PageOrder {
		if _, err := os.Stat(pagePath(workspacePath, page)); err == nil {
			pages = append(pages, page)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if pages == nil {
		pages = []string{}
	}
	return pages, nil
}

// ReadPage returns the raw Markdown content of one generated page. It
// returns ErrPageNotFound (wrapped) if page isn't a known page id or the
// file doesn't exist.
func ReadPage(workspacePath, page string) (string, error) {
	if !isKnownPage(page) {
		return "", ErrPageNotFound
	}
	data, err := os.ReadFile(pagePath(workspacePath, page))
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrPageNotFound
		}
		return "", err
	}
	return string(data), nil
}

func isKnownPage(page string) bool {
	for _, p := range PageOrder {
		if p == page {
			return true
		}
	}
	return false
}

// IsStale reports whether the generated documentation is potentially
// outdated: at least one openspec/specs/**/spec.md file has a modification
// time strictly later than the most recently modified generated page.
// It reports false when no pages have been generated yet.
func IsStale(workspacePath string) (bool, error) {
	pages, err := ListPages(workspacePath)
	if err != nil {
		return false, err
	}
	if len(pages) == 0 {
		return false, nil
	}

	var newestDoc int64
	for _, page := range pages {
		info, err := os.Stat(pagePath(workspacePath, page))
		if err != nil {
			return false, err
		}
		if mtime := info.ModTime().UnixNano(); mtime > newestDoc {
			newestDoc = mtime
		}
	}

	specPaths, err := filepath.Glob(specsGlob(workspacePath))
	if err != nil {
		return false, err
	}
	for _, specPath := range specPaths {
		info, err := os.Stat(specPath)
		if err != nil {
			continue
		}
		if info.ModTime().UnixNano() > newestDoc {
			return true, nil
		}
	}
	return false, nil
}
