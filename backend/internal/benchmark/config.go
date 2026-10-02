package benchmark

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultRuns           = 3
	defaultMaxRetries     = 2
	defaultOutputDir      = "benchmark/results"
	defaultValidationCmd  = "go test ./..."
	defaultValidationDir  = "backend"
	defaultAgent          = "claude"
	MethodPlatform        = "A"
	MethodBaseline        = "B"
	statusValid           = "valid"
	statusInvalid         = "invalid"
	statusValidationError = "validation_failed"
)

// Config is the benchmark configuration (benchmark/benchmark.yaml).
type Config struct {
	Runs              int    `yaml:"runs"`
	StartSHA          string `yaml:"start_sha"`
	Agent             string `yaml:"agent"`
	Model             string `yaml:"model"`
	Effort            string `yaml:"effort"`
	MaxRetries        *int   `yaml:"max_retries"`
	Brief             string `yaml:"brief"`
	Answers           string `yaml:"answers"`
	Acceptance        string `yaml:"acceptance"`
	OutputDir         string `yaml:"output_dir"`
	ValidationCommand string `yaml:"validation_command"`
	ValidationDir     string `yaml:"validation_dir"`
	// AcceptanceDest is where the acceptance test is copied in the clone once
	// the run is over; AcceptanceCommand runs it from ValidationDir.
	AcceptanceDest    string `yaml:"acceptance_dest"`
	AcceptanceCommand string `yaml:"acceptance_command"`

	// RepoDir is the git repository the benchmark clones. Not serialized.
	RepoDir string `yaml:"-"`
}

// Retries returns the effective maximum number of baseline retries.
func (c *Config) Retries() int {
	if c.MaxRetries == nil {
		return defaultMaxRetries
	}
	return *c.MaxRetries
}

// FieldError names the configuration field responsible for a failure.
type FieldError struct {
	Field string
	Msg   string
}

func (e *FieldError) Error() string { return fmt.Sprintf("config field %q: %s", e.Field, e.Msg) }

// LoadConfig reads, defaults and validates the benchmark configuration. Relative
// paths are resolved against repoDir. Nothing is created on disk.
func LoadConfig(path, repoDir string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	absRepo, err := filepath.Abs(repoDir)
	if err != nil {
		return nil, err
	}
	c.RepoDir = absRepo
	c.applyDefaults(data)
	if err := c.validate(data); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) applyDefaults(raw []byte) {
	// runs: 0 must be refused, not defaulted: only an absent key defaults.
	if !hasKey(raw, "runs") {
		c.Runs = defaultRuns
	}
	if c.OutputDir == "" {
		c.OutputDir = defaultOutputDir
	}
	if c.ValidationCommand == "" {
		c.ValidationCommand = defaultValidationCmd
	}
	if c.ValidationDir == "" {
		c.ValidationDir = defaultValidationDir
	}
	if c.AcceptanceDest == "" {
		c.AcceptanceDest = "backend/acceptance"
	}
	if c.AcceptanceCommand == "" {
		c.AcceptanceCommand = "go test -count=1 ./acceptance/..."
	}
	if c.Agent == "" {
		c.Agent = defaultAgent
	}
}

func hasKey(raw []byte, key string) bool {
	var m map[string]any
	if yaml.Unmarshal(raw, &m) != nil {
		return false
	}
	_, ok := m[key]
	return ok
}

func (c *Config) validate(raw []byte) error {
	if c.Runs < 1 {
		return &FieldError{"runs", fmt.Sprintf("must be at least 1 (got %d)", c.Runs)}
	}
	if c.MaxRetries != nil && *c.MaxRetries < 0 {
		return &FieldError{"max_retries", "must not be negative"}
	}
	if strings.TrimSpace(c.StartSHA) == "" {
		return &FieldError{"start_sha", "is required"}
	}
	if !c.commitExists(c.StartSHA) {
		return &FieldError{"start_sha", fmt.Sprintf("commit %q not found in %s", c.StartSHA, c.RepoDir)}
	}
	for _, f := range []struct{ name, val string }{
		{"brief", c.Brief}, {"answers", c.Answers}, {"acceptance", c.Acceptance},
	} {
		if f.val == "" {
			return &FieldError{f.name, "is required"}
		}
		if _, err := os.Stat(c.Resolve(f.val)); err != nil {
			return &FieldError{f.name, fmt.Sprintf("file %q not found", f.val)}
		}
	}
	return nil
}

func (c *Config) commitExists(sha string) bool {
	cmd := exec.Command("git", "-C", c.RepoDir, "cat-file", "-e", sha+"^{commit}")
	return cmd.Run() == nil
}

// Resolve turns a configured path into an absolute one.
func (c *Config) Resolve(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(c.RepoDir, p)
}
