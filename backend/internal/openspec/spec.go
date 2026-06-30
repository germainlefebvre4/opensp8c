package openspec

import (
	"os"
	"path/filepath"
)

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
