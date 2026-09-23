package agents

import (
	"strings"
	"testing"
)

// TestDocsFormalismPromptLoads verifies the opensp8c documentation-generation
// formalism prompt is non-empty and states the fixed page list, the
// deterministic workflows.md skip rule, the Mermaid convention, and the
// docs/opensp8c/ output path.
func TestDocsFormalismPromptLoads(t *testing.T) {
	prompt := DocsFormalismPrompt
	if prompt == "" {
		t.Fatal("expected DocsFormalismPrompt to be non-empty")
	}

	for _, want := range []string{
		"overview.md",
		"architecture.md",
		"domain-model.md",
		"workflows.md",
		"Mermaid",
		"docs/opensp8c/",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("expected DocsFormalismPrompt to mention %q", want)
		}
	}
}
