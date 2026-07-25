package agents

import (
	"reflect"
	"testing"
)

func TestBuildSubprocessArgs_Gemini(t *testing.T) {
	cfg := AgentConfig{
		ID: "gemini",
	}
	args := cfg.BuildSubprocessArgs("base prompt", "extra prompt")
	expected := []string{
		"--output-format", "stream-json",
		"--approval-mode", "auto_edit",
		"--skip-trust",
	}

	if !reflect.DeepEqual(args, expected) {
		t.Errorf("expected gemini args: %v, got: %v", expected, args)
	}
}

func TestBuildSubprocessArgs_Other(t *testing.T) {
	cfg := AgentConfig{
		ID: "claude",
	}
	args := cfg.BuildSubprocessArgs("base prompt", "extra prompt")
	expected := []string{
		"--print",
		"--verbose",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--include-partial-messages",
		"--append-system-prompt", "base prompt",
		"--append-system-prompt", "extra prompt",
	}

	if !reflect.DeepEqual(args, expected) {
		t.Errorf("expected claude args: %v, got: %v", expected, args)
	}
}

func TestByID(t *testing.T) {
	gemini, ok := ByID("gemini")
	if !ok {
		t.Fatal("expected to find gemini")
	}
	if gemini.ID != "gemini" {
		t.Errorf("expected ID 'gemini', got: %s", gemini.ID)
	}

	_, ok = ByID("nonexistent")
	if ok {
		t.Fatal("expected nonexistent agent to not be found")
	}
}
