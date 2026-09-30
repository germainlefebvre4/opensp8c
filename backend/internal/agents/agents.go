package agents

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Model is a selectable model identifier for an agent.
type Model struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Source string `json:"source"` // "seed" or "cli"
}

const (
	ModelSourceSeed = "seed"
	ModelSourceCLI  = "cli"
)

type AgentConfig struct {
	ID          string
	Label       string
	CLI         string
	VersionArgs []string
	DocsURL     string

	// Capabilities. An empty flag means the agent cannot be driven for that setting.
	ModelFlag    string
	EffortFlag   string
	EffortLevels []string
	SeedModels   []Model
	// ListModels discovers models from the CLI; nil when the CLI cannot list them.
	ListModels func(ctx context.Context) ([]Model, error)

	// Execution fields, set per launch (see preferences.ApplyRole).
	Model  string
	Effort string
}

type AgentStatus struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	DocsURL   string `json:"docsUrl"`
}

// BuildSubprocessArgs returns the CLI args to launch this agent as a subprocess.
// Claude uses stream-json format; other agents use the same flags as placeholders
// until their actual CLI interfaces are validated.
func (a AgentConfig) BuildSubprocessArgs(basePrompt, extraPrompt string) []string {
	return append(a.baseArgs(basePrompt, extraPrompt), a.modelEffortArgs()...)
}

// modelEffortArgs returns the model then effort flags, each only when both the
// agent's flag and the value are non-empty.
func (a AgentConfig) modelEffortArgs() []string {
	var args []string
	if a.ModelFlag != "" && a.Model != "" {
		args = append(args, a.ModelFlag, a.Model)
	}
	if a.EffortFlag != "" && a.Effort != "" {
		args = append(args, a.EffortFlag, a.Effort)
	}
	return args
}

// SupportsModel reports whether the agent declares a model flag.
func (a AgentConfig) SupportsModel() bool { return a.ModelFlag != "" }

// SupportsEffort reports whether the agent declares effort levels and a flag.
func (a AgentConfig) SupportsEffort() bool { return a.EffortFlag != "" && len(a.EffortLevels) > 0 }

// ValidEffort reports whether level is one of the agent's effort levels.
func (a AgentConfig) ValidEffort(level string) bool {
	if !a.SupportsEffort() {
		return false
	}
	for _, l := range a.EffortLevels {
		if l == level {
			return true
		}
	}
	return false
}

func (a AgentConfig) baseArgs(basePrompt, extraPrompt string) []string {
	if a.ID == "gemini" {
		return []string{
			"--output-format", "stream-json",
			"--approval-mode", "auto_edit",
			"--skip-trust",
		}
	}
	if a.ID == "antigravity" {
		return []string{
			"--input-format", "stream-json",
			"--output-format", "stream-json",
			"--dangerously-skip-permissions",
		}
	}

	args := []string{
		"--print",
		"--verbose",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--include-partial-messages",
		"--append-system-prompt", basePrompt,
	}
	if extraPrompt != "" {
		args = append(args, "--append-system-prompt", extraPrompt)
	}
	return args
}

var SupportedAgents = []AgentConfig{
	{
		ID:           "claude",
		Label:        "Claude",
		CLI:          "claude",
		VersionArgs:  []string{"--version"},
		DocsURL:      "https://docs.claude.com/en/docs/claude-code/overview",
		ModelFlag:    "--model",
		EffortFlag:   "--effort",
		EffortLevels: []string{"low", "medium", "high", "xhigh", "max"},
		SeedModels: []Model{
			{ID: "fable", Label: "Fable", Source: ModelSourceSeed},
			{ID: "opus", Label: "Opus", Source: ModelSourceSeed},
			{ID: "sonnet", Label: "Sonnet", Source: ModelSourceSeed},
			{ID: "haiku", Label: "Haiku", Source: ModelSourceSeed},
		},
	},
	{
		ID:          "codex",
		Label:       "Codex",
		CLI:         "codex",
		VersionArgs: []string{"--version"},
		DocsURL:     "https://github.com/openai/codex",
		ModelFlag:   "-m",
		SeedModels:  []Model{{ID: "gpt-5.5", Label: "GPT-5.5", Source: ModelSourceSeed}},
	},
	{
		ID:          "gemini",
		Label:       "Gemini",
		CLI:         "gemini",
		VersionArgs: []string{"--version"},
		DocsURL:     "https://github.com/google-gemini/gemini-cli",
		ModelFlag:   "-m",
		SeedModels: []Model{
			{ID: "auto", Label: "Auto", Source: ModelSourceSeed},
			{ID: "pro", Label: "Pro", Source: ModelSourceSeed},
			{ID: "flash", Label: "Flash", Source: ModelSourceSeed},
			{ID: "flash-lite", Label: "Flash Lite", Source: ModelSourceSeed},
		},
	},
	{
		ID:           "antigravity",
		Label:        "Antigravity CLI",
		CLI:          "agy",
		VersionArgs:  []string{"--version"},
		DocsURL:      "https://antigravity.google/docs",
		ModelFlag:    "--model",
		EffortFlag:   "--effort",
		EffortLevels: []string{"low", "medium", "high", "max"},
		ListModels:   listAntigravityModels,
	},
	{
		// Copilot is accessed via the gh CLI extension
		ID:          "copilot",
		Label:       "Copilot",
		CLI:         "gh",
		VersionArgs: []string{"copilot", "--version"},
		DocsURL:     "https://docs.github.com/en/copilot/concepts/agents/about-copilot-cli",
	},
}

func ByID(id string) (AgentConfig, bool) {
	for _, a := range SupportedAgents {
		if a.ID == id {
			return a, true
		}
	}
	return AgentConfig{}, false
}

func Detect(a AgentConfig) AgentStatus {
	status := AgentStatus{ID: a.ID, Label: a.Label, DocsURL: a.DocsURL}

	_, err := exec.LookPath(a.CLI)
	if err != nil {
		return status
	}
	status.Installed = true

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, a.CLI, a.VersionArgs...).Output()
	if err == nil {
		line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
		status.Version = strings.TrimSpace(line)
	}

	return status
}

func DetectAll() []AgentStatus {
	results := make([]AgentStatus, len(SupportedAgents))
	var wg sync.WaitGroup
	for i, a := range SupportedAgents {
		wg.Add(1)
		go func(idx int, cfg AgentConfig) {
			defer wg.Done()
			results[idx] = Detect(cfg)
		}(i, a)
	}
	wg.Wait()
	return results
}
