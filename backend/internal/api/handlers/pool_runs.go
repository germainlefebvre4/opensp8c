package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/go-chi/chi/v5"
)

// maxPoolRuns caps the list of recent pool runs returned to the client.
const maxPoolRuns = 50

// outcomeRunning and outcomeInterrupted are derived, never written to a
// journal: a run has no end marker and, respectively, a live or no worker.
const (
	outcomeRunning     = "running"
	outcomeInterrupted = "interrupted"
)

// PoolRunsHandler serves the persisted runs of the workers of a workspace pool.
type PoolRunsHandler struct {
	ws            *WorkspaceHandler
	reg           *pool.Registry
	convStore     *conversation.Store
	activityStore *activity.Store
	// activeRunsFn returns the "<change>/<ts>" keys of live workers; a seam
	// over the registry so tests can simulate a running worker.
	activeRunsFn func(wsID string) map[string]bool
}

func NewPoolRunsHandler(ws *WorkspaceHandler, reg *pool.Registry, convStore *conversation.Store, activityStore *activity.Store) *PoolRunsHandler {
	h := &PoolRunsHandler{ws: ws, reg: reg, convStore: convStore, activityStore: activityStore}
	h.activeRunsFn = h.registryActiveRuns
	return h
}

// validSegment rejects URL segments that could escape the run directory.
func validSegment(s string) bool {
	return s != "" && s != "." && s != ".." && !strings.ContainsAny(s, `/\`)
}

// PoolRun summarizes one worker execution.
type PoolRun struct {
	Change    string `json:"change"`
	Ts        string `json:"ts"`
	WorkerID  int    `json:"worker_id"`
	Outcome   string `json:"outcome"`
	Reason    string `json:"reason,omitempty"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at,omitempty"`
	LineCount int    `json:"line_count"`
}

// PoolRunDetail is a PoolRun with its chronological activity entries.
type PoolRunDetail struct {
	PoolRun
	Entries []activity.Entry `json:"entries"`
}

type journalEnvelope struct {
	Ts   string          `json:"ts"`
	Dir  string          `json:"dir"`
	Data json.RawMessage `json:"data"`
}

// summarizeRun derives a run's summary from its raw journal lines.
// activeRunTS is the set of "<change>/<ts>" of live workers.
func summarizeRun(change, ts string, lines [][]byte, active map[string]bool) (PoolRun, time.Time, time.Time) {
	run := PoolRun{Change: change, Ts: ts, LineCount: len(lines)}

	// The run ts (second resolution) is a floor of the real start.
	start, _ := time.Parse("2006-01-02T15-04-05Z", ts)
	run.StartedAt = start.UTC().Format(time.RFC3339Nano)

	var lastTs time.Time
	ended := false
	for _, raw := range lines {
		var env journalEnvelope
		if json.Unmarshal(raw, &env) != nil {
			continue
		}
		if t, err := time.Parse(time.RFC3339Nano, env.Ts); err == nil {
			lastTs = t
		}
		if env.Dir != "meta" {
			continue
		}
		var marker struct {
			Type     string `json:"type"`
			WorkerID int    `json:"worker_id"`
			Outcome  string `json:"outcome"`
			Reason   string `json:"reason"`
		}
		if json.Unmarshal(env.Data, &marker) != nil {
			continue
		}
		switch marker.Type {
		case "pool_run_start":
			run.WorkerID = marker.WorkerID
			if t, err := time.Parse(time.RFC3339Nano, env.Ts); err == nil {
				run.StartedAt = t.UTC().Format(time.RFC3339Nano)
			}
		case "pool_run_end":
			ended = true
			run.Outcome = marker.Outcome
			run.Reason = marker.Reason
			if !lastTs.IsZero() {
				run.EndedAt = lastTs.UTC().Format(time.RFC3339Nano)
			}
		}
	}
	if !ended {
		if active[change+"/"+ts] {
			run.Outcome = outcomeRunning
		} else {
			run.Outcome = outcomeInterrupted
		}
	}

	// Window used to attach the change's non-agent activity to this run.
	windowEnd := lastTs
	if run.Outcome == outcomeRunning {
		windowEnd = time.Time{} // open-ended
	}
	return run, start, windowEnd
}

func (h *PoolRunsHandler) registryActiveRuns(wsID string) map[string]bool {
	active := map[string]bool{}
	_, _, workers := h.reg.For(wsID).Status(wsID)
	for _, w := range workers {
		if w.RunTS != "" {
			active[w.ActiveChange+"/"+w.RunTS] = true
		}
	}
	return active
}

// ListRuns serves GET /api/workspaces/{id}/pool/runs.
func (h *PoolRunsHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	wsID := chi.URLParam(r, "id")
	if _, ok := h.ws.workspacePath(wsID); !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	out := []PoolRun{}
	if h.convStore != nil {
		metas, err := h.convStore.ListKindAcrossChanges(wsID, "pool")
		if err != nil {
			http.Error(w, "failed to list pool runs: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if len(metas) > maxPoolRuns {
			metas = metas[:maxPoolRuns]
		}
		active := h.activeRunsFn(wsID)
		for _, m := range metas {
			lines, err := h.convStore.Load(wsID, m.Change, "pool", m.Ts)
			if err != nil {
				continue
			}
			run, _, _ := summarizeRun(m.Change, m.Ts, lines, active)
			out = append(out, run)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// GetRun serves GET /api/workspaces/{id}/pool/runs/{change}/{ts}.
func (h *PoolRunsHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	wsID := chi.URLParam(r, "id")
	change := chi.URLParam(r, "change")
	ts := chi.URLParam(r, "ts")
	if _, ok := h.ws.workspacePath(wsID); !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}
	if h.convStore == nil || !validSegment(change) || !validSegment(ts) {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	lines, err := h.convStore.Load(wsID, change, "pool", ts)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "run not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to load run: "+err.Error(), http.StatusInternalServerError)
		return
	}

	run, windowStart, windowEnd := summarizeRun(change, ts, lines, h.activeRunsFn(wsID))

	entries, err := activity.ParseConversationLines(lines)
	if err != nil {
		http.Error(w, "failed to parse run: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if h.activityStore != nil {
		persisted, err := h.activityStore.Read(wsID, change)
		if err != nil {
			http.Error(w, "failed to read activity: "+err.Error(), http.StatusInternalServerError)
			return
		}
		for _, e := range persisted {
			t, err := time.Parse(time.RFC3339Nano, e.Ts)
			if err != nil || t.Before(windowStart) {
				continue
			}
			if !windowEnd.IsZero() && t.After(windowEnd) {
				continue
			}
			entries = append(entries, e)
		}
	}
	sortEntriesChronologically(entries)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PoolRunDetail{PoolRun: run, Entries: entries})
}

// sortEntriesChronologically orders entries by parsed timestamp (string order
// is wrong for RFC3339Nano values with a varying number of fraction digits).
func sortEntriesChronologically(entries []activity.Entry) {
	parsed := make([]time.Time, len(entries))
	for i, e := range entries {
		parsed[i], _ = time.Parse(time.RFC3339Nano, e.Ts)
	}
	idx := make([]int, len(entries))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return parsed[idx[a]].Before(parsed[idx[b]]) })
	sorted := make([]activity.Entry, len(entries))
	for i, j := range idx {
		sorted[i] = entries[j]
	}
	copy(entries, sorted)
}
