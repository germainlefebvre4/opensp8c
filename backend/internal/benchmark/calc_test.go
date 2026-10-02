package benchmark

import (
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

func at(sec int) time.Time { return t0.Add(time.Duration(sec) * time.Second) }

func ev(k PhaseKind, sec int) PhaseEvent { return PhaseEvent{Kind: k, At: at(sec)} }
func end(sec int, outcome, reason string) PhaseEvent {
	return PhaseEvent{Kind: PoolEnded, At: at(sec), Outcome: outcome, Reason: reason}
}

func TestComputePlatform(t *testing.T) {
	cases := []struct {
		name        string
		data        PhaseData
		wantInvalid string // substring, "" = valid
		check       func(t *testing.T, r PlatformTimings)
	}{
		{
			name: "nominal run",
			data: PhaseData{Events: []PhaseEvent{
				ev(ExploreStarted, 0), ev(FFStarted, 100), ev(FFDone, 160), ev(Launched, 200),
				ev(PoolStarted, 205), end(700, "completed", ""), ev(Merged, 710),
			}},
			check: func(t *testing.T, r PlatformTimings) {
				tm := r.Timings
				if tm.ExploreSec != 100 || tm.FFSec != 60 || tm.LaunchWaitSec != 40 || tm.PoolSec != 510 || tm.PoolActiveSec != 495 {
					t.Fatalf("breakdown: %+v", tm)
				}
				if tm.HumanSec != 140 || tm.MachineSec != 570 || tm.TotalSec != 710 || tm.HumanSec+tm.MachineSec != tm.TotalSec {
					t.Fatalf("totals: %+v", tm)
				}
				if tm.PoolRuns != 1 || len(r.Phases) != 4 {
					t.Fatalf("runs/phases: %+v", r)
				}
			},
		},
		{
			name: "paused pool",
			data: PhaseData{Events: []PhaseEvent{
				ev(ExploreStarted, 0), ev(FFStarted, 10), ev(FFDone, 20), ev(Launched, 30),
				ev(PoolStarted, 31), end(90, "paused", "validation command missing"),
			}},
			wantInvalid: "paused",
			check: func(t *testing.T, r PlatformTimings) {
				if r.PauseReason != "validation command missing" {
					t.Fatalf("pause reason: %q", r.PauseReason)
				}
			},
		},
		{
			name: "several pool runs are cumulated",
			data: PhaseData{Events: []PhaseEvent{
				ev(ExploreStarted, 0), ev(FFStarted, 10), ev(FFDone, 20), ev(Launched, 30),
				ev(PoolStarted, 31), end(100, "paused", "x"),
				ev(PoolStarted, 150), end(250, "completed", ""), ev(Merged, 255),
			}},
			check: func(t *testing.T, r PlatformTimings) {
				if r.Timings.PoolRuns != 2 || r.Timings.PoolActiveSec != 69+100 || r.Timings.PoolSec != 225 {
					t.Fatalf("cumulation: %+v", r.Timings)
				}
				if len(r.Notes) != 1 {
					t.Fatalf("notes: %v", r.Notes)
				}
			},
		},
		{
			name:        "incomplete observation",
			data:        PhaseData{ObservationIncomplete: true, Events: []PhaseEvent{ev(ExploreStarted, 0)}},
			wantInvalid: "observation incomplète",
		},
		{
			name: "missing merge",
			data: PhaseData{Events: []PhaseEvent{
				ev(ExploreStarted, 0), ev(FFStarted, 10), ev(FFDone, 20), ev(Launched, 30),
				ev(PoolStarted, 31), end(90, "completed", ""),
			}},
			wantInvalid: "merged absent",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputePlatform(tc.data)
			if tc.wantInvalid == "" && len(got.Invalid) > 0 {
				t.Fatalf("unexpected invalid: %v", got.Invalid)
			}
			if tc.wantInvalid != "" && !strings.Contains(strings.Join(got.Invalid, ";"), tc.wantInvalid) {
				t.Fatalf("want invalid %q, got %v", tc.wantInvalid, got.Invalid)
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

func TestComputePlatformSameFromAnySource(t *testing.T) {
	data := PhaseData{Events: []PhaseEvent{
		ev(ExploreStarted, 0), ev(FFStarted, 5), ev(FFDone, 9), ev(Launched, 12),
		ev(PoolStarted, 13), end(60, "completed", ""), ev(Merged, 61),
	}}
	a, _ := MemorySource{Data: data}.Load()
	b, _ := MemorySource{Data: data}.Load()
	if ComputePlatform(a).Timings != ComputePlatform(b).Timings {
		t.Fatal("results differ for identical events")
	}
}

func TestComputeBaselineWithRetry(t *testing.T) {
	got := ComputeBaseline([]Attempt{
		{AgentStart: at(0), AgentEnd: at(100), ValStart: at(100), ValEnd: at(120)},
		{AgentStart: at(121), AgentEnd: at(160), ValStart: at(160), ValEnd: at(170), ValidationPassed: true},
	})
	tm := got.Timings
	if got.Retries != 1 || !got.Passed || tm.AgentSec != 139 || tm.ValidationSec != 30 || tm.TotalSec != 170 {
		t.Fatalf("got %+v", got)
	}
}
