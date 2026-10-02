package benchmark

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir,
		"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// newRepo creates a temporary git repository with one commit and the
// benchmark input files, returning its path and HEAD sha.
func newRepo(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q", "-b", "main")
	writeFile(t, filepath.Join(dir, "README.md"), "hello\n")
	gitRun(t, dir, "add", "-A")
	gitRun(t, dir, "commit", "-q", "-m", "init")
	sha := gitRun(t, dir, "rev-parse", "HEAD")
	// benchmark inputs are untracked: they must never reach a clone.
	writeFile(t, filepath.Join(dir, "benchmark", "brief.md"), "brief\n")
	writeFile(t, filepath.Join(dir, "benchmark", "answers.md"), "answers\n")
	writeFile(t, filepath.Join(dir, "benchmark", "acceptance", "stats_test.go"), "package acceptance\n")
	return dir, sha
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeConfig(t *testing.T, repo, body string) string {
	t.Helper()
	p := filepath.Join(repo, "benchmark", "benchmark.yaml")
	writeFile(t, p, body)
	return p
}

func baseYAML(sha string) string {
	return "start_sha: " + sha + "\nbrief: benchmark/brief.md\nanswers: benchmark/answers.md\nacceptance: benchmark/acceptance\n"
}

func TestLoadConfigDefaults(t *testing.T) {
	repo, sha := newRepo(t)
	cfg, err := LoadConfig(writeConfig(t, repo, baseYAML(sha)), repo)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Runs != 3 || cfg.Retries() != 2 || cfg.OutputDir != "benchmark/results" {
		t.Fatalf("defaults not applied: %+v", cfg)
	}
}

func TestLoadConfigOverrideRuns(t *testing.T) {
	repo, sha := newRepo(t)
	cfg, err := LoadConfig(writeConfig(t, repo, baseYAML(sha)+"runs: 5\nmax_retries: 0\n"), repo)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Runs != 5 || cfg.Retries() != 0 {
		t.Fatalf("got runs=%d retries=%d", cfg.Runs, cfg.Retries())
	}
}

func TestLoadConfigInvalid(t *testing.T) {
	repo, sha := newRepo(t)
	cases := []struct {
		name, yaml, field string
	}{
		{"runs zero", baseYAML(sha) + "runs: 0\n", "runs"},
		{"unknown sha", strings.Replace(baseYAML(sha), sha, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", 1), "start_sha"},
		{"missing brief", strings.Replace(baseYAML(sha), "benchmark/brief.md", "benchmark/nope.md", 1), "brief"},
		{"missing answers", strings.Replace(baseYAML(sha), "benchmark/answers.md", "benchmark/nope.md", 1), "answers"},
		{"missing acceptance", strings.Replace(baseYAML(sha), "benchmark/acceptance", "benchmark/nope", 1), "acceptance"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadConfig(writeConfig(t, repo, tc.yaml), repo)
			if err == nil {
				t.Fatal("expected error")
			}
			fe, ok := err.(*FieldError)
			if !ok || fe.Field != tc.field || !strings.Contains(err.Error(), `"`+tc.field+`"`) {
				t.Fatalf("error does not name %q: %v", tc.field, err)
			}
		})
	}
}

func gitRunEnv(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir,
		"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
