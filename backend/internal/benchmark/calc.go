package benchmark

import (
	"fmt"
	"sort"
	"time"
)

// PlatformTimings is the output of the method-A calculation.
type PlatformTimings struct {
	Timings Timings
	Phases  []Phase
	// Invalid lists the reasons the run cannot be used for aggregates.
	Invalid     []string
	PauseReason string
	Notes       []string
}

func secs(d time.Duration) float64 { return d.Seconds() }

// ComputePlatform derives the method-A timings from normalized phase events.
// It is pure: it never touches files or the network.
func ComputePlatform(data PhaseData) PlatformTimings {
	var out PlatformTimings
	if data.ObservationIncomplete {
		out.Invalid = append(out.Invalid, "observation incomplète")
	}
	events := append([]PhaseEvent(nil), data.Events...)
	sort.SliceStable(events, func(i, j int) bool { return events[i].At.Before(events[j].At) })

	var (
		exploreAt, ffStart, ffDone, launched, merged time.Time
		poolStarts                                   []time.Time
		poolEnds                                     []PhaseEvent
	)
	for _, e := range events {
		switch e.Kind {
		case ExploreStarted:
			if exploreAt.IsZero() {
				exploreAt = e.At
			}
		case FFStarted:
			if ffStart.IsZero() {
				ffStart = e.At
			}
		case FFDone:
			if ffDone.IsZero() && !ffStart.IsZero() {
				ffDone = e.At
			}
		case Launched:
			if launched.IsZero() {
				launched = e.At
			}
		case PoolStarted:
			poolStarts = append(poolStarts, e.At)
		case PoolEnded:
			poolEnds = append(poolEnds, e)
		case Merged:
			merged = e.At
		}
	}

	for _, m := range []struct {
		name string
		t    time.Time
	}{
		{string(ExploreStarted), exploreAt}, {string(FFStarted), ffStart}, {string(FFDone), ffDone},
		{string(Launched), launched}, {string(Merged), merged},
	} {
		if m.t.IsZero() {
			out.Invalid = append(out.Invalid, fmt.Sprintf("événement %s absent", m.name))
		}
	}

	// A pool run whose end marker is not "completed" invalidates the run.
	// With several pool runs (resume after pause) the last outcome decides;
	// earlier non-completed outcomes are kept as notes.
	if len(poolEnds) == 0 {
		out.Invalid = append(out.Invalid, "aucun run de pool terminé")
	} else {
		last := poolEnds[len(poolEnds)-1]
		for _, e := range poolEnds[:len(poolEnds)-1] {
			if e.Outcome != OutcomeCompleted {
				out.Notes = append(out.Notes, fmt.Sprintf("run de pool précédent terminé en %q", e.Outcome))
			}
		}
		if last.Outcome != OutcomeCompleted {
			out.PauseReason = last.Reason
			reason := fmt.Sprintf("run de pool terminé en %q", last.Outcome)
			if last.Reason != "" {
				reason += " : " + last.Reason
			}
			out.Invalid = append(out.Invalid, reason)
		}
	}

	if len(out.Invalid) > 0 && (exploreAt.IsZero() || ffStart.IsZero() || ffDone.IsZero() || launched.IsZero() || merged.IsZero()) {
		return out
	}
	if !(!exploreAt.After(ffStart) && !ffStart.After(ffDone) && !ffDone.After(launched) && !launched.After(merged)) {
		out.Invalid = append(out.Invalid, "événements de phase dans un ordre incohérent")
		return out
	}

	t := &out.Timings
	t.ExploreSec = secs(ffStart.Sub(exploreAt))
	t.LaunchWaitSec = secs(launched.Sub(ffDone))
	t.FFSec = secs(ffDone.Sub(ffStart))
	t.PoolSec = secs(merged.Sub(launched))
	var active time.Duration
	for i, s := range poolStarts {
		if i < len(poolEnds) && poolEnds[i].At.After(s) {
			active += poolEnds[i].At.Sub(s)
		}
	}
	t.PoolActiveSec = secs(active)
	t.PoolRuns = len(poolStarts)
	t.HumanSec = t.ExploreSec + t.LaunchWaitSec
	t.MachineSec = t.FFSec + t.PoolSec
	t.TotalSec = secs(merged.Sub(exploreAt))

	out.Phases = []Phase{
		{"explore", exploreAt, ffStart},
		{"ff", ffStart, ffDone},
		{"launch_wait", ffDone, launched},
		{"pool", launched, merged},
	}
	return out
}

// Attempt is one agent execution followed by its validation (method B).
type Attempt struct {
	AgentStart, AgentEnd time.Time
	ValStart, ValEnd     time.Time
	ValidationPassed     bool
}

// BaselineTimings is the output of the method-B calculation.
type BaselineTimings struct {
	Timings Timings
	Phases  []Phase
	Retries int
	Passed  bool
}

// ComputeBaseline derives the method-B timings from the recorded attempts.
func ComputeBaseline(attempts []Attempt) BaselineTimings {
	var out BaselineTimings
	if len(attempts) == 0 {
		return out
	}
	var agent, val time.Duration
	for i, a := range attempts {
		agent += a.AgentEnd.Sub(a.AgentStart)
		val += a.ValEnd.Sub(a.ValStart)
		out.Phases = append(out.Phases,
			Phase{fmt.Sprintf("agent_%d", i+1), a.AgentStart, a.AgentEnd},
			Phase{fmt.Sprintf("validation_%d", i+1), a.ValStart, a.ValEnd})
	}
	last := attempts[len(attempts)-1]
	out.Retries = len(attempts) - 1
	out.Passed = last.ValidationPassed
	out.Timings = Timings{
		AgentSec:      secs(agent),
		ValidationSec: secs(val),
		TotalSec:      secs(last.ValEnd.Sub(attempts[0].AgentStart)),
	}
	return out
}
