package benchmark

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RunEnv describes a prepared run: an isolated clone and, for method A, a
// dedicated platform configuration directory and port.
type RunEnv struct {
	Method    string `json:"method"`
	RunID     string `json:"run_id"`
	RunDir    string `json:"run_dir"`
	CloneDir  string `json:"clone_dir"`
	ConfigDir string `json:"config_dir,omitempty"`
	ConfigYML string `json:"config_path,omitempty"`
	Port      int    `json:"port,omitempty"`
	// WorkspaceID is the platform id of the clone workspace (method A).
	WorkspaceID string `json:"workspace_id,omitempty"`
	StartCmd    string `json:"start_cmd,omitempty"`
}

// RunID builds the identifier of the n-th run of a method.
func RunID(method string, n int) string { return fmt.Sprintf("%s-%02d", strings.ToLower(method), n) }

// PrepareRun creates the isolated clone at the configured SHA and, for method
// A, the per-run platform configuration. The acceptance test and the
// benchmark/ folder never exist in the clone.
func PrepareRun(cfg *Config, method string, n int) (*RunEnv, error) {
	if method != MethodPlatform && method != MethodBaseline {
		return nil, fmt.Errorf("unknown method %q (want A or B)", method)
	}
	id := RunID(method, n)
	runDir := filepath.Join(cfg.Resolve(cfg.OutputDir), id)
	clone := filepath.Join(runDir, "clone")
	if _, err := os.Stat(clone); err == nil {
		return nil, fmt.Errorf("run %s already prepared (%s exists)", id, clone)
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, err
	}
	if out, err := exec.Command("git", "clone", "--quiet", "--no-hardlinks", cfg.RepoDir, clone).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git clone: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", clone, "checkout", "--quiet", "-B", "benchmark-start", cfg.StartSHA).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git checkout %s: %v\n%s", cfg.StartSHA, err, out)
	}
	if err := checkCloneClean(clone, cfg.StartSHA); err != nil {
		return nil, err
	}

	env := &RunEnv{Method: method, RunID: id, RunDir: runDir, CloneDir: clone}
	if method == MethodPlatform {
		if err := prepareConfigDir(cfg, env); err != nil {
			return nil, err
		}
	}
	data, _ := json.MarshalIndent(env, "", "  ")
	if err := os.WriteFile(filepath.Join(runDir, runFile), data, 0o644); err != nil {
		return nil, err
	}
	return env, nil
}

const runFile = "run.json"

// LoadRunEnv reads the run description written by PrepareRun.
func LoadRunEnv(cfg *Config, runID string) (*RunEnv, error) {
	data, err := os.ReadFile(filepath.Join(cfg.Resolve(cfg.OutputDir), runID, runFile))
	if err != nil {
		return nil, fmt.Errorf("run %s is not prepared: %w", runID, err)
	}
	var env RunEnv
	return &env, json.Unmarshal(data, &env)
}

// checkCloneClean enforces the protocol: HEAD is the start SHA, the tree is
// clean, and neither the acceptance test nor benchmark/ is present.
func checkCloneClean(clone, sha string) error {
	head, err := exec.Command("git", "-C", clone, "rev-parse", "HEAD").Output()
	if err != nil {
		return err
	}
	full, err := exec.Command("git", "-C", clone, "rev-parse", sha+"^{commit}").Output()
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(head)) != strings.TrimSpace(string(full)) {
		return fmt.Errorf("clone HEAD %s is not the start sha %s", strings.TrimSpace(string(head)), sha)
	}
	status, err := exec.Command("git", "-C", clone, "status", "--porcelain").Output()
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(status))) > 0 {
		return fmt.Errorf("clone working tree is not clean")
	}
	if _, err := os.Stat(filepath.Join(clone, "benchmark")); err == nil {
		return fmt.Errorf("start sha contains a benchmark/ folder: the clone must not hold benchmark files (pick an earlier start_sha)")
	}
	return nil
}

func prepareConfigDir(cfg *Config, env *RunEnv) error {
	env.ConfigDir = filepath.Join(env.RunDir, "platform")
	if err := os.MkdirAll(env.ConfigDir, 0o755); err != nil {
		return err
	}
	env.ConfigYML = filepath.Join(env.ConfigDir, "config.yaml")
	port, err := freePort()
	if err != nil {
		return err
	}
	env.Port = port
	env.WorkspaceID = workspaceID(env.CloneDir)

	body := map[string]any{
		"workspaces": []map[string]string{{"name": "benchmark-" + env.RunID, "path": env.CloneDir}},
	}
	data, err := yaml.Marshal(body)
	if err != nil {
		return err
	}
	if err := os.WriteFile(env.ConfigYML, data, 0o644); err != nil {
		return err
	}
	env.StartCmd = fmt.Sprintf("cd %s/backend && CONFIG_PATH=%s PORT=%d go run ./cmd/server",
		env.CloneDir, env.ConfigYML, env.Port)
	return nil
}

// workspaceID mirrors workspace.StableID (sha256 of the absolute path, first
// 4 bytes in hex). It is duplicated on purpose: the benchmark package never
// imports platform packages.
func workspaceID(absPath string) string {
	h := sha256.Sum256([]byte(absPath))
	return fmt.Sprintf("%x", h[:4])
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
