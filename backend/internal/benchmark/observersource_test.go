package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func jsonl(t *testing.T, path string, lines ...string) {
	t.Helper()
	writeFile(t, path, strings.Join(lines, "\n")+"\n")
}

func convLineJSON(ts time.Time, dir, data string) string {
	return fmt.Sprintf(`{"ts":%q,"dir":%q,"data":%s}`, ts.Format(time.RFC3339Nano), dir, data)
}

func obsJSON(r ObserverRecord) string { b, _ := json.Marshal(r); return string(b) }

// fixtureRun lays out a method-A run in the current platform log format.
func fixtureRun(t *testing.T, poolOutcome string, withMerge bool) ObserverSource {
	src, _, _ := fixtureRunFull(t, poolOutcome, withMerge)
	return src
}

func fixtureRunFull(t *testing.T, poolOutcome string, withMerge bool) (ObserverSource, *Config, *RunEnv) {
	t.Helper()
	cfg, sha := testConfig(t)
	env, err := PrepareRun(cfg, MethodPlatform, 1)
	if err != nil {
		t.Fatal(err)
	}
	base := time.Now().UTC().Truncate(time.Second).Add(-10 * time.Minute)
	logDir := filepath.Join(env.RunDir, "logs")

	jsonl(t, filepath.Join(logDir, "chat", "2026-01-01T10-00-00Z.jsonl"),
		convLineJSON(base, "out", `{"type":"system"}`), // system line before the user speaks
		convLineJSON(base.Add(5*time.Second), "in", `{"type":"user"}`),
		convLineJSON(base.Add(90*time.Second), "out", `{"type":"assistant"}`),
	)
	jsonl(t, filepath.Join(logDir, "pool", "2026-01-01T10-05-00Z.jsonl"),
		convLineJSON(base.Add(200*time.Second), "meta", `{"type":"pool_run_start","worker_id":1}`),
		convLineJSON(base.Add(250*time.Second), "out", `{"type":"assistant"}`),
		convLineJSON(base.Add(400*time.Second), "meta", fmt.Sprintf(`{"type":"pool_run_end","outcome":%q,"reason":""}`, poolOutcome)),
	)
	obsLog := filepath.Join(env.RunDir, "events.jsonl")
	jsonl(t, obsLog,
		obsJSON(ObserverRecord{Ts: base.Add(-time.Second), Kind: RecObserverStarted}),
		obsJSON(ObserverRecord{Ts: base.Add(100 * time.Second), Kind: RecEvent, Event: "ff_started", Name: "c1"}),
		obsJSON(ObserverRecord{Ts: base.Add(160 * time.Second), Kind: RecEvent, Event: "ff_done", Name: "c1"}),
		obsJSON(ObserverRecord{Ts: base.Add(165 * time.Second), Kind: RecTransition, From: "", To: "ready"}),
		obsJSON(ObserverRecord{Ts: base.Add(198 * time.Second), Kind: RecTransition, From: "ready", To: "todo"}),
		obsJSON(ObserverRecord{Ts: base.Add(450 * time.Second), Kind: RecObserverStopped}),
	)
	if withMerge {
		// the merge commit, dated within the run window
		date := base.Add(410 * time.Second).Format(time.RFC3339)
		writeFile(t, filepath.Join(env.CloneDir, "merged.txt"), "x\n")
		gitRun(t, env.CloneDir, "add", "-A")
		gitRun(t, env.CloneDir, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "merge", "--date="+date)
		// commit date (committer) is what we read
		gitRunEnv(t, env.CloneDir, []string{"GIT_COMMITTER_DATE=" + date}, "commit", "-q", "--amend", "--no-edit", "--date="+date)
	}
	return ObserverSource{ObserverLog: obsLog, ChangeLogDir: logDir, CloneDir: env.CloneDir, StartSHA: sha}, cfg, env
}

func TestObserverSourceProducesOrderedPhaseEvents(t *testing.T) {
	src := fixtureRun(t, "completed", true)
	data, err := src.Load()
	if err != nil {
		t.Fatal(err)
	}
	if data.ObservationIncomplete {
		t.Fatal("observation should be complete")
	}
	want := []PhaseKind{ExploreStarted, FFStarted, FFDone, Launched, PoolStarted, PoolEnded, Merged}
	got := map[PhaseKind]time.Time{}
	for _, e := range data.Events {
		got[e.Kind] = e.At
	}
	var prev time.Time
	for _, k := range want {
		at, ok := got[k]
		if !ok {
			t.Fatalf("missing event %s (have %v)", k, data.Events)
		}
		if at.Before(prev) {
			t.Fatalf("event %s out of order: %v before %v", k, at, prev)
		}
		prev = at
	}
	// The first "in" message, not the earlier system line, starts the explore.
	if d := got[FFStarted].Sub(got[ExploreStarted]); d != 95*time.Second {
		t.Fatalf("explore duration = %v, want 95s", d)
	}
	timings := ComputePlatform(data)
	if len(timings.Invalid) != 0 {
		t.Fatalf("unexpected invalid: %v", timings.Invalid)
	}
}

func TestObserverSourceNoMergeCommit(t *testing.T) {
	data, err := fixtureRun(t, "completed", false).Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range data.Events {
		if e.Kind == Merged {
			t.Fatal("no merge commit expected")
		}
	}
}

func TestObserverSourceIncompleteObservation(t *testing.T) {
	src := fixtureRun(t, "completed", true)
	// The observer stopped before the merge happened.
	recs, _ := ReadObserverLog(src.ObserverLog)
	var lines []string
	for _, r := range recs {
		if r.Kind == RecObserverStopped {
			continue
		}
		if r.Kind == RecTransition && r.To == "todo" {
			r.Ts = r.Ts.Add(0)
		}
		lines = append(lines, obsJSON(r))
	}
	lines = append(lines, obsJSON(ObserverRecord{Ts: recs[0].Ts.Add(300 * time.Second), Kind: RecObserverStopped}))
	if err := os.WriteFile(src.ObserverLog, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, _ := src.Load()
	if !data.ObservationIncomplete {
		t.Fatal("observer stopped before the merge: observation must be incomplete")
	}
}
