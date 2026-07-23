package agents

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type AgentConfig struct {
	ID          string
	Label       string
	CLI         string
	VersionArgs []string
}

type AgentStatus struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
}

// BuildSubprocessArgs returns the CLI args to launch this agent as a subprocess.
// Claude uses stream-json format; other agents use the same flags as placeholders
// until their actual CLI interfaces are validated.
func (a AgentConfig) BuildSubprocessArgs(basePrompt, extraPrompt string) []string {
	if a.ID == "gemini" {
		return []string{
			"--output-format", "stream-json",
			"--approval-mode", "auto_edit",
			"--skip-trust",
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
		ID:          "claude",
		Label:       "Claude",
		CLI:         "claude",
		VersionArgs: []string{"--version"},
	},
	{
		ID:          "codex",
		Label:       "Codex",
		CLI:         "codex",
		VersionArgs: []string{"--version"},
	},
	{
		ID:          "gemini",
		Label:       "Gemini",
		CLI:         "gemini",
		VersionArgs: []string{"--version"},
	},
	{
		ID:          "antigravity",
		Label:       "Antigravity v2",
		CLI:         "antigravity",
		VersionArgs: []string{"--version"},
	},
	{
		// Copilot is accessed via the gh CLI extension
		ID:          "copilot",
		Label:       "Copilot",
		CLI:         "gh",
		VersionArgs: []string{"copilot", "--version"},
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
	status := AgentStatus{ID: a.ID, Label: a.Label}

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
