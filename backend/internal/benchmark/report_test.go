package benchmark

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files")

func platformRun(id string, valid bool, human, machine float64, reason string) *Result {
	r := &Result{
		Method: "A", RunID: id, StartSHA: "abc123", Valid: valid,
		Config:  RunConfig{Agent: "claude", Model: "opus", Effort: "high", MaxRetries: 2},
		Timings: Timings{HumanSec: human, MachineSec: machine, TotalSec: human + machine, PoolRuns: 1},
	}
	if !valid {
		r.Status = "invalid"
		r.InvalidReasons = []string{reason}
	}
	return r
}

func baselineRun(id string, valid bool, total float64, retries int) *Result {
	r := &Result{
		Method: "B", RunID: id, StartSHA: "abc123", Valid: valid, Retries: retries,
		Config:  RunConfig{Agent: "claude", Model: "opus", Effort: "high", MaxRetries: 2},
		Timings: Timings{TotalSec: total, AgentSec: total - 30, ValidationSec: 30},
	}
	if !valid {
		r.Status = "invalid"
		r.InvalidReasons = []string{"test d'acceptation en échec"}
	}
	return r
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden.md")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update)", err)
	}
	if string(want) != got {
		t.Fatalf("report differs from %s:\n--- got ---\n%s", path, got)
	}
}

func TestReportGoldenTwoOfThreeValid(t *testing.T) {
	got := RenderReport([]*Result{
		platformRun("a-01", true, 120, 600, ""),
		platformRun("a-02", false, 100, 900, "test d'acceptation en échec"),
		platformRun("a-03", true, 140, 700, ""),
		baselineRun("b-01", true, 500, 0),
		baselineRun("b-02", true, 900, 1),
		baselineRun("b-03", true, 700, 0),
	})
	golden(t, "two-of-three", got)
	for _, want := range []string{"2/3 valides", "3/3 valides", "Limites", "biais de familiarité"} {
		if !strings.Contains(strings.ToLower(got), strings.ToLower(want)) {
			t.Errorf("report lacks %q", want)
		}
	}
}

func TestReportAggregatesIgnoreInvalidRuns(t *testing.T) {
	got := RenderReport([]*Result{
		platformRun("a-01", true, 100, 100, ""),
		platformRun("a-02", false, 1, 1, "x"), // would skew min if included
		baselineRun("b-01", true, 400, 0),
	})
	if !strings.Contains(got, "| A (plateforme) | 1/2 valides | 3m20s | 3m20s | 3m20s |") {
		t.Fatalf("aggregate must only use the valid run:\n%s", got)
	}
}

func TestReportGoldenNoValidData(t *testing.T) {
	got := RenderReport([]*Result{
		platformRun("a-01", false, 100, 100, "run de pool terminé en \"paused\""),
		baselineRun("b-01", true, 400, 0),
	})
	golden(t, "no-valid-data", got)
	if !strings.Contains(got, "aucune donnée valide") || !strings.Contains(got, "Aucun écart calculé") {
		t.Fatalf("expected the no-data wording:\n%s", got)
	}
}

func TestReportLimitsAlwaysPresent(t *testing.T) {
	if !strings.Contains(strings.ToLower(RenderReport(nil)), "biais de familiarité") {
		t.Fatal("limits section must mention familiarity bias even with no results")
	}
}
