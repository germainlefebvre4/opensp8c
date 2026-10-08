package openspec

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Scenario is one `#### Scenario:` of a delta spec with its WHEN / THEN / AND
// lines, in order.
type Scenario struct {
	Name  string
	Lines []string
}

// SpecScenarios groups the scenarios of one delta spec file. File is the path
// relative to the change's specs directory.
type SpecScenarios struct {
	File      string
	Scenarios []Scenario
}

var (
	scenarioHeadingRe = regexp.MustCompile(`^#{3,4}\s+Scenario:\s*(.*)$`)
	scenarioLineRe    = regexp.MustCompile(`(?i)^[-*]\s+\*\*(WHEN|THEN|AND|GIVEN)\*\*`)
)

// ChangeScenarios reads openspec/changes/<change>/specs/**/spec.md under root
// and returns, per file (alphabetical order), its scenarios. Unreadable files
// and files without scenario are skipped; a change without delta spec yields
// nothing.
func ChangeScenarios(root, change string) []SpecScenarios {
	specsDir := filepath.Join(root, "openspec", "changes", change, "specs")
	var files []string
	_ = filepath.WalkDir(specsDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == "spec.md" {
			files = append(files, p)
		}
		return nil
	})
	sort.Strings(files)

	var out []SpecScenarios
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		scenarios := parseScenarios(string(data))
		if len(scenarios) == 0 {
			continue
		}
		rel, err := filepath.Rel(specsDir, f)
		if err != nil {
			rel = f
		}
		out = append(out, SpecScenarios{File: filepath.ToSlash(rel), Scenarios: scenarios})
	}
	return out
}

// parseScenarios extracts the scenarios of a spec content.
func parseScenarios(content string) []Scenario {
	var out []Scenario
	cur := -1
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if m := scenarioHeadingRe.FindStringSubmatch(line); m != nil {
			out = append(out, Scenario{Name: strings.TrimSpace(m[1])})
			cur = len(out) - 1
			continue
		}
		if strings.HasPrefix(line, "#") {
			cur = -1
			continue
		}
		if cur >= 0 && scenarioLineRe.MatchString(line) {
			out[cur].Lines = append(out[cur].Lines, line)
		}
	}
	return out
}
