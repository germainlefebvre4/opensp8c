package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/glefebvre/opensp8c/internal/workspace"
	"github.com/go-chi/chi/v5"
)

type poolRunsEnv struct {
	h         *PoolRunsHandler
	wsID      string
	convStore *conversation.Store
	actStore  *activity.Store
	activity  *ActivityHandler
}

func newPoolRunsEnv(t *testing.T) *poolRunsEnv {
	t.Helper()
	tmp := t.TempDir()
	wsDir := filepath.Join(tmp, "project")
	abs, _ := filepath.Abs(wsDir)
	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: wsDir}}}
	ws := NewWorkspaceHandler(cfg, "", nil)
	convStore := conversation.NewStore(filepath.Join(tmp, "conversations"))
	actStore := activity.NewStore(filepath.Join(tmp, "activity"), nil)
	return &poolRunsEnv{
		h:         NewPoolRunsHandler(ws, pool.NewRegistry(nil, nil, nil, nil, nil), convStore, actStore),
		wsID:      workspace.StableID(abs),
		convStore: convStore,
		actStore:  actStore,
		activity:  NewActivityHandler(ws, convStore, actStore),
	}
}

// writeRun writes a pool run journal with a start marker at startTs, the given
// extra lines and, when endOutcome != "", an end marker at endTs.
func (e *poolRunsEnv) writeRun(t *testing.T, change, ts, startTs string, extra []string, endTs, endOutcome, endReason string) {
	t.Helper()
	f, err := e.convStore.OpenRun(e.wsID, change, "pool", ts)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fmt.Fprintf(f, `{"ts":%q,"dir":"meta","data":{"type":"pool_run_start","worker_id":1,"change":%q}}`+"\n", startTs, change)
	for _, l := range extra {
		fmt.Fprintln(f, l)
	}
	if endOutcome != "" {
		fmt.Fprintf(f, `{"ts":%q,"dir":"meta","data":{"type":"pool_run_end","outcome":%q,"reason":%q}}`+"\n", endTs, endOutcome, endReason)
	}
}

func (e *poolRunsEnv) do(handler http.HandlerFunc, path string, params map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", path, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", e.wsID)
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func (e *poolRunsEnv) list(t *testing.T) []PoolRun {
	t.Helper()
	rec := e.do(e.h.ListRuns, "/pool/runs", nil)
	if rec.Code != 200 {
		t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
	}
	var runs []PoolRun
	if err := json.Unmarshal(rec.Body.Bytes(), &runs); err != nil {
		t.Fatal(err)
	}
	return runs
}

func (e *poolRunsEnv) detail(change, ts string) *httptest.ResponseRecorder {
	return e.do(e.h.GetRun, "/pool/runs/"+change+"/"+ts, map[string]string{"change": change, "ts": ts})
}

const narration = `{"ts":"2026-09-24T10:00:01.000Z","dir":"out","data":{"type":"content_block_delta","delta":{"text":"working"}}}`

func TestPoolRuns_List_NoRunsReturnsEmptyArray(t *testing.T) {
	e := newPoolRunsEnv(t)
	rec := e.do(e.h.ListRuns, "/pool/runs", nil)
	if rec.Code != 200 || rec.Body.String() != "[]\n" {
		t.Fatalf("expected [] got %d %q", rec.Code, rec.Body.String())
	}
}

func TestPoolRuns_List_MultipleChangesNewestFirstWithOutcomes(t *testing.T) {
	e := newPoolRunsEnv(t)
	e.writeRun(t, "add-user-auth", "2026-09-24T10-00-00Z", "2026-09-24T10:00:00.5Z", []string{narration}, "2026-09-24T10:05:00Z", "paused", "tests failed")
	e.writeRun(t, "fix-docs", "2026-09-24T11-00-00Z", "2026-09-24T11:00:00.5Z", nil, "2026-09-24T11:01:00Z", "completed", "")
	e.writeRun(t, "crashed", "2026-09-24T12-00-00Z", "2026-09-24T12:00:00.5Z", []string{narration}, "", "", "")
	e.writeRun(t, "live", "2026-09-24T13-00-00Z", "2026-09-24T13:00:00.5Z", nil, "", "", "")
	e.h.activeRunsFn = func(string) map[string]bool { return map[string]bool{"live/2026-09-24T13-00-00Z": true} }

	runs := e.list(t)
	if len(runs) != 4 {
		t.Fatalf("expected 4 runs, got %d", len(runs))
	}
	want := []struct{ change, outcome string }{{"live", "running"}, {"crashed", "interrupted"}, {"fix-docs", "completed"}, {"add-user-auth", "paused"}}
	for i, w := range want {
		if runs[i].Change != w.change || runs[i].Outcome != w.outcome {
			t.Errorf("run %d: want %+v got %+v", i, w, runs[i])
		}
	}
	if runs[3].Reason != "tests failed" || runs[3].EndedAt == "" || runs[3].WorkerID != 1 || runs[3].LineCount != 3 {
		t.Errorf("unexpected paused run summary: %+v", runs[3])
	}
	if runs[0].EndedAt != "" || runs[1].EndedAt != "" {
		t.Errorf("running/interrupted runs must have no end date: %+v %+v", runs[0], runs[1])
	}
}

func TestPoolRuns_List_CappedAt50(t *testing.T) {
	e := newPoolRunsEnv(t)
	for i := 0; i < 55; i++ {
		ts := fmt.Sprintf("2026-09-24T10-%02d-00Z", i)
		e.writeRun(t, "big", ts, "2026-09-24T10:00:00Z", nil, "2026-09-24T10:00:01Z", "completed", "")
	}
	runs := e.list(t)
	if len(runs) != 50 {
		t.Fatalf("expected 50 runs, got %d", len(runs))
	}
	if runs[0].Ts != "2026-09-24T10-54-00Z" {
		t.Errorf("expected newest first, got %s", runs[0].Ts)
	}
}

func TestPoolRuns_Detail_FinishedRunMergesWindowedActivity(t *testing.T) {
	e := newPoolRunsEnv(t)
	change := "two-runs"
	e.writeRun(t, change, "2026-09-24T10-00-00Z", "2026-09-24T10:00:00.5Z", []string{narration}, "2026-09-24T10:10:00Z", "paused", "boom")
	e.writeRun(t, change, "2026-09-24T11-00-00Z", "2026-09-24T11:00:00.5Z", nil, "", "", "")
	e.h.activeRunsFn = func(string) map[string]bool { return map[string]bool{change + "/2026-09-24T11-00-00Z": true} }

	for _, en := range []activity.Entry{
		{Ts: "2026-09-24T10:03:00Z", Type: "pool.worker_status", Category: "pool", Summary: "Worker 1: testing"},
		{Ts: "2026-09-24T11:03:00Z", Type: "pool.worker_status", Category: "pool", Summary: "Worker 1: healing"},
	} {
		if err := e.actStore.Append(e.wsID, change, en); err != nil {
			t.Fatal(err)
		}
	}

	rec := e.detail(change, "2026-09-24T10-00-00Z")
	if rec.Code != 200 {
		t.Fatalf("detail: %d %s", rec.Code, rec.Body.String())
	}
	var d PoolRunDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	if d.Outcome != "paused" || d.Reason != "boom" || d.EndedAt == "" {
		t.Errorf("unexpected summary %+v", d.PoolRun)
	}
	if len(d.Entries) != 2 || d.Entries[0].Summary != "working" || d.Entries[1].Summary != "Worker 1: testing" {
		t.Errorf("expected narration then own status entry only, got %+v", d.Entries)
	}

	// Running run: open-ended window, no end date.
	rec = e.detail(change, "2026-09-24T11-00-00Z")
	var live PoolRunDetail
	_ = json.Unmarshal(rec.Body.Bytes(), &live)
	if live.Outcome != "running" || live.EndedAt != "" {
		t.Errorf("unexpected live summary %+v", live.PoolRun)
	}
	if len(live.Entries) != 1 || live.Entries[0].Summary != "Worker 1: healing" {
		t.Errorf("live run must only contain its own entries, got %+v", live.Entries)
	}
}

func TestPoolRuns_Detail_NotFoundAndTraversal(t *testing.T) {
	e := newPoolRunsEnv(t)
	if rec := e.detail("nope", "2026-09-24T10-00-00Z"); rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
	if rec := e.detail("..", "x"); rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for traversal, got %d", rec.Code)
	}
}

func TestPoolRuns_AppearInChangeActivityWithoutMarkers(t *testing.T) {
	e := newPoolRunsEnv(t)
	change := "merged-view"
	tool := `{"ts":"2026-09-24T10:00:02.000Z","dir":"out","data":{"type":"assistant","message":{"content":[{"type":"tool_use","id":"c1","name":"Read","input":{"file_path":"a.go"}}]}}}`
	e.writeRun(t, change, "2026-09-24T10-00-00Z", "2026-09-24T10:00:00.5Z", []string{narration, tool}, "2026-09-24T10:05:00Z", "completed", "")

	req := httptest.NewRequest("GET", "/activity", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", e.wsID)
	rctx.URLParams.Add("name", change)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	e.activity.GetActivity(rec, req)

	var entries []activity.Entry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Summary != "working" || entries[1].Type != "Read" {
		t.Fatalf("expected narration + tool call only (no marker entries), got %+v", entries)
	}
}
