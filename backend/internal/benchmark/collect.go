package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const resultFile = "result.json"

// CollectOptions tunes Collect.
type CollectOptions struct {
	// Change is the change name of a method-A run. When empty it is inferred
	// from the platform logs if there is exactly one candidate.
	Change string
}

// Collect gathers a finished run: it copies the raw data before any
// computation, copies and runs the acceptance test, computes the timings and
// writes the self-contained result.json.
func Collect(ctx context.Context, cfg *Config, runID string, opt CollectOptions) (*Result, error) {
	env, err := LoadRunEnv(cfg, runID)
	if err != nil {
		return nil, err
	}
	res := &Result{
		Method: env.Method, RunID: env.RunID, StartSHA: cfg.StartSHA,
		Config: RunConfig{Agent: cfg.Agent, Model: cfg.Model, Effort: cfg.Effort, MaxRetries: cfg.Retries()},
	}
	rawDir := filepath.Join(env.RunDir, "raw")

	// 1. Raw data first: nothing below may depend on logs subject to retention.
	if env.Method == MethodPlatform {
		if _, err = copyPlatformData(env, rawDir, opt.Change); err != nil {
			return nil, err
		}
	}
	if err := extractGitLog(env, cfg.StartSHA, rawDir); err != nil {
		return nil, err
	}
	writeEffectiveConfig(cfg, env.RunDir)

	// 2. Acceptance test: copied into the clone only now, then executed.
	res.AcceptanceRan, res.AcceptancePassed = runAcceptance(ctx, cfg, env)

	// 3. Timings, from the copies.
	switch env.Method {
	case MethodPlatform:
		src := ObserverSource{
			ObserverLog:  filepath.Join(env.RunDir, "events.jsonl"),
			ChangeLogDir: filepath.Join(rawDir, "conversations"),
			CloneDir:     env.CloneDir,
			StartSHA:     cfg.StartSHA,
		}
		data, err := src.Load()
		if err != nil {
			return nil, err
		}
		pt := ComputePlatform(data)
		res.Timings, res.Phases, res.Notes, res.PauseReason = pt.Timings, pt.Phases, pt.Notes, pt.PauseReason
		res.Status, res.Valid = statusValid, true
		for _, r := range pt.Invalid {
			res.invalidate(r)
		}
	case MethodBaseline:
		rec, err := ReadBaselineRecord(env.RunDir)
		if err != nil {
			return nil, fmt.Errorf("baseline record: %w", err)
		}
		bt := ComputeBaseline(rec.CalcAttempts())
		res.Timings, res.Phases, res.Retries = bt.Timings, bt.Phases, bt.Retries
		res.Status, res.Valid = statusValid, true
		if rec.Status != "validation_passed" {
			res.invalidate("validation en échec après le maximum de relances")
			res.Status = statusValidationError
		}
	}
	if !res.AcceptanceRan {
		res.invalidate("test d'acceptation non exécuté")
	} else if !res.AcceptancePassed {
		res.invalidate("test d'acceptation en échec")
	}

	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(env.RunDir, resultFile), data, 0o644); err != nil {
		return nil, err
	}
	return res, nil
}

func copyPlatformData(env *RunEnv, rawDir, change string) (string, error) {
	convRoot := filepath.Join(env.ConfigDir, "conversations", env.WorkspaceID)
	if change == "" {
		entries, _ := os.ReadDir(convRoot)
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), "_") {
				if change != "" {
					return "", fmt.Errorf("several changes found in %s: pass --change", convRoot)
				}
				change = e.Name()
			}
		}
		if change == "" {
			return "", fmt.Errorf("no change logs found in %s: pass --change", convRoot)
		}
	}
	if err := copyTree(filepath.Join(convRoot, change), filepath.Join(rawDir, "conversations")); err != nil {
		return "", fmt.Errorf("copy conversations: %w", err)
	}
	actSrc := filepath.Join(env.ConfigDir, "activity", env.WorkspaceID, change)
	if _, err := os.Stat(actSrc); err == nil {
		if err := copyTree(actSrc, filepath.Join(rawDir, "activity")); err != nil {
			return "", fmt.Errorf("copy activity: %w", err)
		}
	}
	return change, nil
}

func extractGitLog(env *RunEnv, startSHA, rawDir string) error {
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		return err
	}
	out, err := exec.Command("git", "-C", env.CloneDir, "log", "--format=%H%x09%cI%x09%P%x09%s", startSHA+"..HEAD").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git log: %v\n%s", err, out)
	}
	return os.WriteFile(filepath.Join(rawDir, "git-log.txt"), out, 0o644)
}

func writeEffectiveConfig(cfg *Config, runDir string) {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	_ = os.WriteFile(filepath.Join(runDir, "config.effective.json"), data, 0o644)
}

// runAcceptance copies the hidden acceptance test into the clone and runs it.
func runAcceptance(ctx context.Context, cfg *Config, env *RunEnv) (ran, passed bool) {
	logPath := filepath.Join(env.RunDir, "acceptance.log")
	src := cfg.Resolve(cfg.Acceptance)
	dest := filepath.Join(env.CloneDir, cfg.AcceptanceDest)
	if err := copyAny(src, dest); err != nil {
		_ = os.WriteFile(logPath, []byte("copy acceptance test: "+err.Error()+"\n"), 0o644)
		return false, false
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", cfg.AcceptanceCommand)
	cmd.Dir = filepath.Join(env.CloneDir, cfg.ValidationDir)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	_ = os.WriteFile(logPath, out.Bytes(), 0o644)
	if _, isExit := err.(*exec.ExitError); err != nil && !isExit {
		return false, false // could not start
	}
	return true, err == nil
}

func copyAny(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyTree(src, dest)
	}
	return copyFile(src, filepath.Join(dest, filepath.Base(src)))
}

func copyTree(src, dest string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(p, target)
	})
}

func copyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// ReadResult loads a result.json.
func ReadResult(path string) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Result
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &r, nil
}

// LoadResults reads every <outputDir>/*/result.json, sorted by run id.
func LoadResults(outputDir string) ([]*Result, error) {
	paths, err := filepath.Glob(filepath.Join(outputDir, "*", resultFile))
	if err != nil {
		return nil, err
	}
	var out []*Result
	for _, p := range paths {
		r, err := ReadResult(p)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}
