package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	baselineFile     = "baseline.json"
	maxValidationOut = 20000
)

// BaselineAttempt is one agent execution plus the validation that followed.
type BaselineAttempt struct {
	AgentStart       time.Time `json:"agent_start"`
	AgentEnd         time.Time `json:"agent_end"`
	AgentError       string    `json:"agent_error,omitempty"`
	ValidationStart  time.Time `json:"validation_start"`
	ValidationEnd    time.Time `json:"validation_end"`
	ValidationPassed bool      `json:"validation_passed"`
}

// BaselineRecord is written to <run>/baseline.json by `benchmark baseline`.
type BaselineRecord struct {
	Attempts []BaselineAttempt `json:"attempts"`
	// Status is "validation_passed" or "validation_failed" (retries exhausted).
	Status string `json:"status"`
}

// BaselineOptions tunes RunBaseline.
type BaselineOptions struct {
	// AgentBin overrides the agent executable (defaults to cfg.Agent).
	AgentBin string
	Now      func() time.Time
}

// BuildBaselinePrompt composes the initial prompt: the brief verbatim followed
// by the scripted explore answers as clarifications. Method A gets the same
// two texts, the answers being typed by the operator during explore.
func BuildBaselinePrompt(brief, answers string) string {
	return brief + "\n\n## Clarifications\n\n" + answers + "\n"
}

// RetryPrompt is the single retry instruction of the baseline.
func RetryPrompt(validationOutput string) string {
	if len(validationOutput) > maxValidationOut {
		validationOutput = "…" + validationOutput[len(validationOutput)-maxValidationOut:]
	}
	return "Les tests suivants échouent, corrige :\n\n" + validationOutput
}

// RunBaseline executes method B in the run's clone: the agent runs
// non-interactively with the brief, the validation command follows, and on
// failure the agent is re-run with the failing output, up to cfg.Retries()
// times. The record is written to <run>/baseline.json.
func RunBaseline(ctx context.Context, cfg *Config, env *RunEnv, opt BaselineOptions) (*BaselineRecord, error) {
	if opt.Now == nil {
		opt.Now = func() time.Time { return time.Now().UTC() }
	}
	bin := opt.AgentBin
	if bin == "" {
		bin = cfg.Agent
	}
	brief, err := os.ReadFile(cfg.Resolve(cfg.Brief))
	if err != nil {
		return nil, err
	}
	answers, err := os.ReadFile(cfg.Resolve(cfg.Answers))
	if err != nil {
		return nil, err
	}
	prompt := BuildBaselinePrompt(string(brief), string(answers))
	_ = os.WriteFile(filepath.Join(env.RunDir, "prompt.md"), []byte(prompt), 0o644)

	rec := &BaselineRecord{}
	for attempt := 0; attempt <= cfg.Retries(); attempt++ {
		a := BaselineAttempt{AgentStart: opt.Now()}
		agentOut, agentErr := runAgent(ctx, cfg, bin, env.CloneDir, prompt)
		a.AgentEnd = opt.Now()
		if agentErr != nil {
			a.AgentError = agentErr.Error()
		}
		_ = os.WriteFile(filepath.Join(env.RunDir, fmt.Sprintf("agent-%d.log", attempt+1)), agentOut, 0o644)

		a.ValidationStart = opt.Now()
		valOut, valErr := runValidation(ctx, cfg, env.CloneDir)
		a.ValidationEnd = opt.Now()
		a.ValidationPassed = valErr == nil
		_ = os.WriteFile(filepath.Join(env.RunDir, fmt.Sprintf("validation-%d.log", attempt+1)), valOut, 0o644)
		rec.Attempts = append(rec.Attempts, a)

		if ctx.Err() != nil {
			return rec, ctx.Err()
		}
		if a.ValidationPassed {
			break
		}
		prompt = RetryPrompt(string(valOut))
	}
	rec.Status = "validation_failed"
	if rec.Attempts[len(rec.Attempts)-1].ValidationPassed {
		rec.Status = "validation_passed"
	}
	data, _ := json.MarshalIndent(rec, "", "  ")
	if err := os.WriteFile(filepath.Join(env.RunDir, baselineFile), data, 0o644); err != nil {
		return rec, err
	}
	return rec, nil
}

func runAgent(ctx context.Context, cfg *Config, bin, dir, prompt string) ([]byte, error) {
	args := []string{"-p", "--dangerously-skip-permissions"}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	if cfg.Effort != "" {
		args = append(args, "--effort", cfg.Effort)
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(prompt)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.Bytes(), err
}

func runValidation(ctx context.Context, cfg *Config, clone string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", cfg.ValidationCommand)
	cmd.Dir = filepath.Join(clone, cfg.ValidationDir)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.Bytes(), err
}

// ReadBaselineRecord loads <run>/baseline.json.
func ReadBaselineRecord(runDir string) (*BaselineRecord, error) {
	data, err := os.ReadFile(filepath.Join(runDir, baselineFile))
	if err != nil {
		return nil, err
	}
	var rec BaselineRecord
	return &rec, json.Unmarshal(data, &rec)
}

// Attempts converts the record to calculator input.
func (r *BaselineRecord) CalcAttempts() []Attempt {
	out := make([]Attempt, len(r.Attempts))
	for i, a := range r.Attempts {
		out[i] = Attempt{a.AgentStart, a.AgentEnd, a.ValidationStart, a.ValidationEnd, a.ValidationPassed}
	}
	return out
}
