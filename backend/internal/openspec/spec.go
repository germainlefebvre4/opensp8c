package openspec

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var changeDateRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-(.+)$`)

type ChangeRef struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Date   string `json:"date"`
	Status string `json:"status"` // "active" | "archived"
}

type SpecWithHistory struct {
	Name    string      `json:"name"`
	Changes []ChangeRef `json:"changes"`
}

type SpecOverview struct {
	Specs   []SpecWithHistory `json:"specs"`
	Orphans []string          `json:"orphans"`
}

type openspecYAML struct {
	Created string `yaml:"created"`
}

func parseChangeRef(name, status string) ChangeRef {
	if m := changeDateRe.FindStringSubmatch(name); m != nil {
		return ChangeRef{Name: name, Slug: m[2], Date: m[1], Status: status}
	}
	// No date prefix — read from .openspec.yaml
	return ChangeRef{Name: name, Slug: name, Date: "", Status: status}
}

func readChangeCreated(changeDir string) string {
	data, err := os.ReadFile(filepath.Join(changeDir, ".openspec.yaml"))
	if err != nil {
		return ""
	}
	var y openspecYAML
	if err := yaml.Unmarshal(data, &y); err != nil {
		return ""
	}
	return y.Created
}

func collectChangesInDir(dir, status string, index map[string][]ChangeRef) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		ref := parseChangeRef(e.Name(), status)
		if ref.Date == "" {
			ref.Date = readChangeCreated(filepath.Join(dir, e.Name()))
		}
		specsSubDir := filepath.Join(dir, e.Name(), "specs")
		subs, err := os.ReadDir(specsSubDir)
		if err != nil {
			continue
		}
		for _, s := range subs {
			if !s.IsDir() {
				continue
			}
			index[s.Name()] = append(index[s.Name()], ref)
		}
	}
	return nil
}

func ListSpecsWithChanges(workspacePath string) (SpecOverview, error) {
	changesDir := filepath.Join(workspacePath, "openspec", "changes")
	archiveDir := filepath.Join(changesDir, "archive")
	specsDir := filepath.Join(workspacePath, "openspec", "specs")

	// Build inverted index: spec name → []ChangeRef
	index := map[string][]ChangeRef{}
	if err := collectChangesInDir(changesDir, "active", index); err != nil {
		return SpecOverview{}, err
	}
	if err := collectChangesInDir(archiveDir, "archived", index); err != nil {
		return SpecOverview{}, err
	}

	// Sort each spec's changes newest-first
	for k := range index {
		refs := index[k]
		sort.Slice(refs, func(i, j int) bool {
			return strings.Compare(refs[i].Date, refs[j].Date) > 0
		})
		index[k] = refs
	}

	// List top-level specs
	entries, err := os.ReadDir(specsDir)
	if err != nil && !os.IsNotExist(err) {
		return SpecOverview{}, err
	}

	knownSpecs := map[string]bool{}
	var specs []SpecWithHistory
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		knownSpecs[e.Name()] = true
		changes := index[e.Name()]
		if changes == nil {
			changes = []ChangeRef{}
		}
		specs = append(specs, SpecWithHistory{Name: e.Name(), Changes: changes})
	}

	// Orphans: referenced in changes but absent from openspec/specs/
	var orphans []string
	for name := range index {
		if !knownSpecs[name] {
			orphans = append(orphans, name)
		}
	}
	sort.Strings(orphans)

	if specs == nil {
		specs = []SpecWithHistory{}
	}
	if orphans == nil {
		orphans = []string{}
	}
	return SpecOverview{Specs: specs, Orphans: orphans}, nil
}

type Spec struct {
	Name    string `json:"name"`
	Content string `json:"content,omitempty"`
}

func ListSpecs(workspacePath string) ([]Spec, error) {
	specsDir := filepath.Join(workspacePath, "openspec", "specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Spec{}, nil
		}
		return nil, err
	}

	var specs []Spec
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		specs = append(specs, Spec{Name: e.Name()})
	}
	return specs, nil
}

func ReadSpec(workspacePath, specName string) (*Spec, error) {
	specPath := filepath.Join(workspacePath, "openspec", "specs", specName, "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, err
	}
	return &Spec{Name: specName, Content: string(data)}, nil
}

func WriteSpec(workspacePath, specName, content string) error {
	specPath := filepath.Join(workspacePath, "openspec", "specs", specName, "spec.md")
	return os.WriteFile(specPath, []byte(content), 0644)
}
