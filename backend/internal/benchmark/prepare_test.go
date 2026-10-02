package benchmark

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testConfig(t *testing.T) (*Config, string) {
	t.Helper()
	repo, sha := newRepo(t)
	cfg, err := LoadConfig(writeConfig(t, repo, baseYAML(sha)), repo)
	if err != nil {
		t.Fatal(err)
	}
	return cfg, sha
}

func TestPrepareRunClone(t *testing.T) {
	cfg, sha := testConfig(t)
	env, err := PrepareRun(cfg, MethodBaseline, 1)
	if err != nil {
		t.Fatal(err)
	}
	if head := gitRun(t, env.CloneDir, "rev-parse", "HEAD"); head != sha {
		t.Fatalf("HEAD = %s, want %s", head, sha)
	}
	if st := gitRun(t, env.CloneDir, "status", "--porcelain"); st != "" {
		t.Fatalf("tree not clean: %s", st)
	}
	// Acceptance test and benchmark/ must be nowhere in the clone.
	filepath.Walk(env.CloneDir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !strings.Contains(p, "/.git") && (info.Name() == "stats_test.go" || info.Name() == "benchmark") {
			t.Errorf("forbidden file in clone: %s", p)
		}
		return nil
	})
	if env.ConfigDir != "" {
		t.Fatal("method B must not get a platform config dir")
	}
}

func TestPrepareTwoPlatformRunsAreIsolated(t *testing.T) {
	cfg, _ := testConfig(t)
	a, err := PrepareRun(cfg, MethodPlatform, 1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := PrepareRun(cfg, MethodPlatform, 2)
	if err != nil {
		t.Fatal(err)
	}
	if a.ConfigDir == b.ConfigDir || a.CloneDir == b.CloneDir {
		t.Fatal("runs share directories")
	}
	if a.Port == 0 || a.Port == b.Port {
		t.Fatalf("ports not distinct: %d %d", a.Port, b.Port)
	}
	data, err := os.ReadFile(a.ConfigYML)
	if err != nil || !strings.Contains(string(data), a.CloneDir) {
		t.Fatalf("config.yaml must point at the clone: %v %s", err, data)
	}
	if !strings.Contains(a.StartCmd, "CONFIG_PATH="+a.ConfigYML) || a.WorkspaceID == "" {
		t.Fatalf("start cmd / workspace id: %+v", a)
	}
}

func TestPrepareRejectsExistingRun(t *testing.T) {
	cfg, _ := testConfig(t)
	if _, err := PrepareRun(cfg, MethodBaseline, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareRun(cfg, MethodBaseline, 1); err == nil {
		t.Fatal("expected error for an already prepared run")
	}
}
