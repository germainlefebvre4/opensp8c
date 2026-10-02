package benchmark

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeAgent writes a shell script standing in for `claude`. Each call appends
// its stdin to calls.log and creates ok.flag once it has been called `okAfter`
// times (0 = never). The validation command checks for ok.flag.
func fakeAgent(t *testing.T, okAfter int) (bin, callsLog string) {
	t.Helper()
	dir := t.TempDir()
	callsLog = filepath.Join(dir, "calls.log")
	bin = filepath.Join(dir, "fake-agent")
	script := `#!/bin/sh
cat >> ` + callsLog + `
echo "-----" >> ` + callsLog + `
n=$(grep -c -e '-----' ` + callsLog + `)
if [ "` + itoa(okAfter) + `" -gt 0 ] && [ "$n" -ge "` + itoa(okAfter) + `" ]; then touch ok.flag; fi
exit 0
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin, callsLog
}

func itoa(n int) string { return string(rune('0' + n)) }

func baselineSetup(t *testing.T, retries int) (*Config, *RunEnv) {
	t.Helper()
	cfg, _ := testConfig(t)
	cfg.ValidationCommand = "test -f ok.flag || { echo 'FAIL: missing ok.flag'; exit 1; }"
	cfg.ValidationDir = "."
	r := retries
	cfg.MaxRetries = &r
	env, err := PrepareRun(cfg, MethodBaseline, 1)
	if err != nil {
		t.Fatal(err)
	}
	return cfg, env
}

func TestBaselineNoRetry(t *testing.T) {
	cfg, env := baselineSetup(t, 2)
	bin, calls := fakeAgent(t, 1)
	rec, err := RunBaseline(context.Background(), cfg, env, BaselineOptions{AgentBin: bin})
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Attempts) != 1 || rec.Status != "validation_passed" {
		t.Fatalf("got %d attempts, status %s", len(rec.Attempts), rec.Status)
	}
	log, _ := os.ReadFile(calls)
	// Same brief and same answers as method A's operator input.
	if !strings.Contains(string(log), "brief") || !strings.Contains(string(log), "answers") {
		t.Fatalf("prompt lacks brief or answers: %s", log)
	}
	if got := ComputeBaseline(rec.CalcAttempts()); got.Retries != 0 {
		t.Fatalf("retries = %d", got.Retries)
	}
}

func TestBaselineOneRetry(t *testing.T) {
	cfg, env := baselineSetup(t, 2)
	bin, calls := fakeAgent(t, 2)
	rec, err := RunBaseline(context.Background(), cfg, env, BaselineOptions{AgentBin: bin})
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Attempts) != 2 || rec.Status != "validation_passed" {
		t.Fatalf("got %d attempts, status %s", len(rec.Attempts), rec.Status)
	}
	log, _ := os.ReadFile(calls)
	if !strings.Contains(string(log), "Les tests suivants échouent") || !strings.Contains(string(log), "FAIL: missing ok.flag") {
		t.Fatalf("retry prompt must carry the failing output: %s", log)
	}
	if got := ComputeBaseline(rec.CalcAttempts()); got.Retries != 1 {
		t.Fatalf("retries = %d", got.Retries)
	}
}

func TestBaselineMaxRetriesReached(t *testing.T) {
	cfg, env := baselineSetup(t, 2)
	bin, _ := fakeAgent(t, 0) // never fixes anything
	rec, err := RunBaseline(context.Background(), cfg, env, BaselineOptions{AgentBin: bin})
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Attempts) != 3 || rec.Status != "validation_failed" {
		t.Fatalf("got %d attempts, status %s", len(rec.Attempts), rec.Status)
	}
	if _, err := ReadBaselineRecord(env.RunDir); err != nil {
		t.Fatal(err)
	}
}
