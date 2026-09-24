package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/workspace"
	"github.com/go-chi/chi/v5"
)

func activityRequest(workspaceID, changeName string) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest("GET", "/api/workspaces/"+workspaceID+"/changes/"+changeName+"/activity", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workspaceID)
	rctx.URLParams.Add("name", changeName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return httptest.NewRecorder(), req
}

func TestActivityHandler_GetActivity_Merged(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "project")
	convDir := filepath.Join(tmpDir, "conversations")
	actDir := filepath.Join(tmpDir, "activity")

	absWsPath, err := filepath.Abs(wsDir)
	if err != nil {
		t.Fatalf("failed to get abs path: %v", err)
	}
	wsID := workspace.StableID(absWsPath)
	changeName := "feature-test"

	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: wsDir}}}
	ws := NewWorkspaceHandler(cfg, "", nil)

	convStore := conversation.NewStore(convDir)
	actStore := activity.NewStore(actDir, nil)

	// Write 1 conversation run with a paired tool_use & tool_result
	f, err := convStore.OpenRun(wsID, changeName, "ff", "2026-09-24T10-00-00Z")
	if err != nil {
		t.Fatalf("failed to open run: %v", err)
	}
	toolUseLine := `{"ts":"2026-09-24T10:00:01.000Z","dir":"out","data":{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"call-1","name":"Read","input":{"file_path":"test.go"}}}}}` + "\n"
	toolResultLine := `{"ts":"2026-09-24T10:00:03.000Z","dir":"out","data":{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"call-1","content":"file contents"}]}}}` + "\n"
	_, _ = f.WriteString(toolUseLine)
	_, _ = f.WriteString(toolResultLine)
	_ = f.Close()

	// Write 1 persisted activity entry
	actEntry := activity.Entry{
		Ts:       "2026-09-24T10:00:00.000Z",
		Type:     "kanban.task_toggled",
		Category: "kanban",
		Summary:  "Task 1 completed",
	}
	if err := actStore.Append(wsID, changeName, actEntry); err != nil {
		t.Fatalf("failed to append activity: %v", err)
	}

	h := NewActivityHandler(ws, convStore, actStore)
	rec, req := activityRequest(wsID, changeName)
	h.GetActivity(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var entries []activity.Entry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(entries), entries)
	}

	// First entry must be the earlier kanban entry
	if entries[0].Type != "kanban.task_toggled" {
		t.Errorf("entry 0: expected kanban.task_toggled, got %s", entries[0].Type)
	}
	if entries[0].DurationMs != nil {
		t.Errorf("entry 0: expected nil duration, got %v", *entries[0].DurationMs)
	}

	// Second entry must be the tool call with durationMs
	if entries[1].Type != "Read" {
		t.Errorf("entry 1: expected Read, got %s", entries[1].Type)
	}
	if entries[1].DurationMs == nil {
		t.Fatalf("entry 1: expected durationMs non-nil")
	}
	if *entries[1].DurationMs != 2000 {
		t.Errorf("entry 1: expected 2000ms duration, got %dms", *entries[1].DurationMs)
	}
}

func TestActivityHandler_GetActivity_NoActivity(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "project")
	convDir := filepath.Join(tmpDir, "conversations")
	actDir := filepath.Join(tmpDir, "activity")

	absWsPath, err := filepath.Abs(wsDir)
	if err != nil {
		t.Fatalf("failed to get abs path: %v", err)
	}
	wsID := workspace.StableID(absWsPath)
	changeName := "empty-change"

	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: wsDir}}}
	ws := NewWorkspaceHandler(cfg, "", nil)

	convStore := conversation.NewStore(convDir)
	actStore := activity.NewStore(actDir, nil)

	h := NewActivityHandler(ws, convStore, actStore)
	rec, req := activityRequest(wsID, changeName)
	h.GetActivity(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var entries []activity.Entry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(entries) != 0 {
		t.Fatalf("expected empty list, got %d entries", len(entries))
	}
}
