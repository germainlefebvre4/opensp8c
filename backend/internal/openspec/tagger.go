package openspec

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DeriveType infers the application layer from file paths in tasks.md content.
func DeriveType(tasksMd string) string {
	hasFrontend := strings.Contains(tasksMd, "frontend/")
	hasBackend := strings.Contains(tasksMd, "backend/")
	hasBatch := strings.Contains(tasksMd, "scripts/") ||
		strings.Contains(tasksMd, "batch/") ||
		strings.Contains(tasksMd, "cmd/")

	switch {
	case hasFrontend && hasBackend:
		return "fullstack"
	case hasFrontend:
		return "frontend"
	case hasBackend:
		return "backend"
	case hasBatch:
		return "batch"
	default:
		return ""
	}
}

// ExtractVocabulary collects all component slugs used across the workspace.
func ExtractVocabulary(workspaceRoot string) []string {
	changesDir := filepath.Join(workspaceRoot, "openspec", "changes")
	seen := make(map[string]struct{})

	scanDir := func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			metaPath := filepath.Join(dir, e.Name(), ".openspec.yaml")
			data, err := os.ReadFile(metaPath)
			if err != nil {
				continue
			}
			var meta openspecMeta
			if err := yaml.Unmarshal(data, &meta); err != nil || meta.Tags == nil {
				continue
			}
			for _, c := range meta.Tags.Components {
				seen[c] = struct{}{}
			}
		}
	}

	scanDir(changesDir)
	scanDir(filepath.Join(changesDir, "archive"))

	vocab := make([]string, 0, len(seen))
	for c := range seen {
		vocab = append(vocab, c)
	}
	sort.Strings(vocab)
	return vocab
}

type llmTagResult struct {
	Complexity int      `json:"complexity"`
	Components []string `json:"components"`
}

// LLMDeriveComplexityAndComponents calls `claude -p` to extract semantic tags.
func LLMDeriveComplexityAndComponents(proposal, design string, vocabulary []string) (int, []string, error) {
	vocabStr := "none yet"
	if len(vocabulary) > 0 {
		vocabStr = strings.Join(vocabulary, ", ")
	}

	prompt := `You are analyzing a software change. Return ONLY a JSON object (no markdown, no explanation) with:
- "complexity": integer 1-5 (1=trivial fix, 5=major architectural change)
- "components": array of kebab-case slugs identifying application areas touched

Existing vocabulary (prefer these when matching, create new kebab-case slug only if no match):
` + vocabStr + `

Change description:
---
` + proposal + `
---

Technical design:
---
` + design + `
---

Return ONLY valid JSON like: {"complexity": 2, "components": ["kanban-board", "search-bar"]}`

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", "-p", prompt)
	out, err := cmd.Output()
	if err != nil {
		return 0, nil, err
	}

	raw := strings.TrimSpace(string(out))
	// Strip markdown code fences if present
	re := regexp.MustCompile("(?s)```[a-z]*\n?(.*?)\n?```")
	raw = re.ReplaceAllString(raw, "$1")
	raw = strings.TrimSpace(raw)

	var result llmTagResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return 0, nil, err
	}

	if result.Complexity < 1 {
		result.Complexity = 1
	}
	if result.Complexity > 5 {
		result.Complexity = 5
	}

	return result.Complexity, result.Components, nil
}

// TagChange derives and writes semantic tags for a change.
// If forceRetag is false, skips changes with _auto: false.
func TagChange(changeRoot, workspaceRoot string, forceRetag bool) error {
	metaPath := filepath.Join(changeRoot, ".openspec.yaml")

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return err
	}

	var meta openspecMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return err
	}

	// Skip if already tagged (any origin), unless explicitly forced
	if !forceRetag && meta.Tags != nil {
		return nil
	}

	// Read tasks.md for type heuristic
	tasksMd, _ := os.ReadFile(filepath.Join(changeRoot, "tasks.md"))
	tagType := DeriveType(string(tasksMd))

	// Read proposal + design for LLM
	proposal, _ := os.ReadFile(filepath.Join(changeRoot, "proposal.md"))
	design, _ := os.ReadFile(filepath.Join(changeRoot, "design.md"))

	vocabulary := ExtractVocabulary(workspaceRoot)

	tags := &Tags{
		Type:     tagType,
		Auto:     true,
		TaggedAt: time.Now().Format("2006-01-02"),
	}

	// Try LLM — graceful degradation if unavailable
	if len(proposal) > 0 {
		complexity, components, err := LLMDeriveComplexityAndComponents(
			string(proposal), string(design), vocabulary,
		)
		if err == nil {
			tags.Complexity = complexity
			tags.Components = components
		}
	}

	meta.Tags = tags

	out, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, out, 0644)
}
