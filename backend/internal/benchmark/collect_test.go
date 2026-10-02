package benchmark

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func acceptanceConfig(t *testing.T, cfg *Config, verdict string) {
	t.Helper()
	// The acceptance "test" is a marker file; the command greps it.
	acc := cfg.Resolve(cfg.Acceptance)
	writeFile(t, filepath.Join(acc, "marker.txt"), verdict+"\n")
	cfg.AcceptanceDest = "acceptance"
	cfg.AcceptanceCommand = "grep -q PASS acceptance/marker.txt"
	cfg.ValidationDir = "."
}

func TestCollectBaselineInvalidAcceptanceIsKept(t *testing.T) {
	cfg, env := baselineSetup(t, 2)
	acceptanceConfig(t, cfg, "FAIL")
	bin, _ := fakeAgent(t, 1)
	if _, err := RunBaseline(context.Background(), cfg, env, BaselineOptions{AgentBin: bin}); err != nil {
		t.Fatal(err)
	}
	res, err := Collect(context.Background(), cfg, env.RunID, CollectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Valid || res.Status != "invalid" || !res.AcceptanceRan || res.AcceptancePassed {
		t.Fatalf("expected an invalid run: %+v", res)
	}
	if res.Timings.TotalSec <= 0 && res.Timings.AgentSec < 0 {
		t.Fatalf("timings must remain readable: %+v", res.Timings)
	}
	if _, err := os.Stat(filepath.Join(env.RunDir, "acceptance.log")); err != nil {
		t.Fatal(err)
	}
}

func TestCollectBaselineValid(t *testing.T) {
	cfg, env := baselineSetup(t, 2)
	acceptanceConfig(t, cfg, "PASS")
	bin, _ := fakeAgent(t, 2)
	if _, err := RunBaseline(context.Background(), cfg, env, BaselineOptions{AgentBin: bin}); err != nil {
		t.Fatal(err)
	}
	res, err := Collect(context.Background(), cfg, env.RunID, CollectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Valid || res.Retries != 1 || res.Method != "B" || res.StartSHA != cfg.StartSHA {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestCollectPlatformCopiesRawDataAndResultIsSelfContained(t *testing.T) {
	src, cfg, env := fixtureRunFull(t, "completed", true)
	acceptanceConfig(t, cfg, "PASS")
	// Lay the platform logs out where the platform really writes them.
	convDst := filepath.Join(env.ConfigDir, "conversations", env.WorkspaceID, "c1")
	if err := os.MkdirAll(filepath.Dir(convDst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(src.ChangeLogDir, convDst); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(env.ConfigDir, "activity", env.WorkspaceID, "c1", "activity.jsonl"), "{}\n")

	res, err := Collect(context.Background(), cfg, env.RunID, CollectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Valid {
		t.Fatalf("expected valid: %+v", res.InvalidReasons)
	}
	if tm := res.Timings; tm.HumanSec+tm.MachineSec != tm.TotalSec || tm.TotalSec <= 0 {
		t.Fatalf("timings: %+v", tm)
	}
	for _, p := range []string{
		"raw/conversations/chat", "raw/conversations/pool", "raw/activity/activity.jsonl", "raw/git-log.txt", "events.jsonl", "result.json",
	} {
		if _, err := os.Stat(filepath.Join(env.RunDir, p)); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}

	// Simulate retention: delete the original platform data and the clone.
	os.RemoveAll(env.ConfigDir)
	os.RemoveAll(env.CloneDir)
	reread, err := ReadResult(filepath.Join(env.RunDir, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	if reread.Timings != res.Timings || !reread.Valid || len(reread.Phases) != 4 {
		t.Fatalf("result.json must be self-sufficient: %+v", reread)
	}
}
