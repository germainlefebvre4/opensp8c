package benchmark

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ObserverSource is the default MetricsSource: it combines the observer
// journal (events.jsonl) with the platform's persisted conversation logs and
// the clone's git history. It is the only place that reads platform files.
type ObserverSource struct {
	// ObserverLog is the path of events.jsonl.
	ObserverLog string
	// ChangeLogDir is <conversations>/<workspaceID>/<change>, holding the
	// "chat" and "pool" run folders (possibly a copy made by collect).
	ChangeLogDir string
	// CloneDir is the run's clone, used to find the merge commit.
	CloneDir string
	// StartSHA is the commit the clone started from.
	StartSHA string
}

type convLine struct {
	Ts   string          `json:"ts"`
	Dir  string          `json:"dir"`
	Data json.RawMessage `json:"data"`
}

func readConvLines(dir string) ([]convLine, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	var out []convLine
	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			return nil, err
		}
		sc := bufio.NewScanner(fh)
		sc.Buffer(make([]byte, 0, 64*1024), 16<<20)
		for sc.Scan() {
			if len(sc.Bytes()) == 0 {
				continue
			}
			var l convLine
			if json.Unmarshal(sc.Bytes(), &l) != nil {
				continue // corrupt line: skip, like the platform does
			}
			out = append(out, l)
		}
		fh.Close()
		if err := sc.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func parseTS(s string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, s)
	return t.UTC(), err == nil
}

// Load implements MetricsSource.
func (s ObserverSource) Load() (PhaseData, error) {
	var data PhaseData
	recs, err := ReadObserverLog(s.ObserverLog)
	if err != nil {
		return data, fmt.Errorf("observer journal: %w", err)
	}

	// Explore start: first message sent by the user in the chat run(s).
	chat, err := readConvLines(filepath.Join(s.ChangeLogDir, "chat"))
	if err != nil {
		return data, err
	}
	var exploreAt time.Time
	pick := func(onlyIn bool) {
		for _, l := range chat {
			t, ok := parseTS(l.Ts)
			if !ok || (onlyIn && l.Dir != "in") {
				continue
			}
			if exploreAt.IsZero() || t.Before(exploreAt) {
				exploreAt = t
			}
		}
	}
	pick(true)
	if exploreAt.IsZero() {
		pick(false)
	}
	if !exploreAt.IsZero() {
		data.Events = append(data.Events, PhaseEvent{Kind: ExploreStarted, At: exploreAt})
	}

	// ff and launch boundaries come from the observer only: the platform
	// does not persist them.
	var ffStarted, ffDone, launched bool
	for _, r := range recs {
		switch {
		case r.Kind == RecObservationIncomplete:
			data.ObservationIncomplete = true
		case r.Kind == RecEvent && r.Event == "ff_started" && !ffStarted:
			ffStarted = true
			data.Events = append(data.Events, PhaseEvent{Kind: FFStarted, At: r.Ts})
		case r.Kind == RecEvent && r.Event == "ff_done" && ffStarted && !ffDone:
			ffDone = true
			data.Events = append(data.Events, PhaseEvent{Kind: FFDone, At: r.Ts})
		case r.Kind == RecTransition && r.To == "todo" && !launched:
			launched = true
			data.Events = append(data.Events, PhaseEvent{Kind: Launched, At: r.Ts})
		}
	}

	// Pool run boundaries: markers written by the worker.
	pool, err := readConvLines(filepath.Join(s.ChangeLogDir, "pool"))
	if err != nil {
		return data, err
	}
	for _, l := range pool {
		if l.Dir != "meta" {
			continue
		}
		var m struct {
			Type    string `json:"type"`
			Outcome string `json:"outcome"`
			Reason  string `json:"reason"`
		}
		t, ok := parseTS(l.Ts)
		if !ok || json.Unmarshal(l.Data, &m) != nil {
			continue
		}
		switch m.Type {
		case "pool_run_start":
			data.Events = append(data.Events, PhaseEvent{Kind: PoolStarted, At: t})
		case "pool_run_end":
			data.Events = append(data.Events, PhaseEvent{Kind: PoolEnded, At: t, Outcome: m.Outcome, Reason: m.Reason})
		}
	}

	// Merge: latest commit on the clone's HEAD made after the launch.
	var launchAt time.Time
	for _, e := range data.Events {
		if e.Kind == Launched {
			launchAt = e.At
		}
	}
	var mergedAt time.Time
	if s.CloneDir != "" && !launchAt.IsZero() {
		if t, ok := latestCommitAfter(s.CloneDir, s.StartSHA, launchAt); ok {
			mergedAt = t
			data.Events = append(data.Events, PhaseEvent{Kind: Merged, At: t})
		}
	}

	// The observer must have covered the whole run, explore to merge.
	if len(recs) == 0 {
		data.ObservationIncomplete = true
	} else {
		if !exploreAt.IsZero() && recs[0].Ts.After(exploreAt) {
			data.ObservationIncomplete = true
		}
		if !mergedAt.IsZero() && recs[len(recs)-1].Ts.Before(mergedAt.Truncate(time.Second)) {
			data.ObservationIncomplete = true
		}
	}
	return data, nil
}

// latestCommitAfter returns the committer date of the newest commit between
// startSHA and HEAD that is not older than the given instant (git dates have a
// one-second resolution).
func latestCommitAfter(clone, startSHA string, after time.Time) (time.Time, bool) {
	out, err := exec.Command("git", "-C", clone, "log", "--format=%cI", startSHA+"..HEAD").Output()
	if err != nil {
		return time.Time{}, false
	}
	var best time.Time
	for _, line := range strings.Fields(string(out)) {
		t, err := time.Parse(time.RFC3339, line)
		if err != nil {
			continue
		}
		t = t.UTC()
		if t.Before(after.Truncate(time.Second)) {
			continue
		}
		if t.After(best) {
			best = t
		}
	}
	return best, !best.IsZero()
}
