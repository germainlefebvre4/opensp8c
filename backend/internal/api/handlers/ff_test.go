package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
	"github.com/glefebvre/opensp8c/internal/workspace"
	"github.com/go-chi/chi/v5"
)

func resetTasksRequest(workspaceID, changeName string) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest("PATCH", "/workspaces/"+workspaceID+"/changes/"+changeName+"/tasks/reset", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workspaceID)
	rctx.URLParams.Add("name", changeName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return httptest.NewRecorder(), req
}

func triggerFFRequest(workspaceID, changeName string) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest("POST", "/workspaces/"+workspaceID+"/changes/"+changeName+"/ff", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workspaceID)
	rctx.URLParams.Add("name", changeName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return httptest.NewRecorder(), req
}

func TestFFHandler_ResetTasks_ClearsKanbanStateAndAppendsActivity(t *testing.T) {
	tmpDir := t.TempDir()
	changesDir := filepath.Join(tmpDir, "openspec", "changes")
	changeDir := filepath.Join(changesDir, "my-change")
	if err := os.MkdirAll(changeDir, 0755); err != nil {
		t.Fatalf("failed to create change dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, ".openspec.yaml"), []byte("schema: spec-driven\ncreated: \"2024-01-01\"\nlaunched: true\norder: 2\n"), 0644); err != nil {
		t.Fatalf("failed to write .openspec.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("- [x] done\n"), 0644); err != nil {
		t.Fatalf("failed to write tasks.md: %v", err)
	}

	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: tmpDir}}}
	ws := NewWorkspaceHandler(cfg, "", nil)
	absPath, err := filepath.Abs(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve abs path: %v", err)
	}
	workspaceID := workspace.StableID(absPath)
	actStore := activity.NewStore(filepath.Join(tmpDir, "activity"), nil)
	h := NewFFHandler(ws, nil, nil, actStore, nil, nil)

	rec, req := resetTasksRequest(workspaceID, "my-change")
	h.ResetTasks(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	changes, err := openspec.ListChanges(tmpDir)
	if err != nil {
		t.Fatalf("ListChanges: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].KanbanStatus != "to-explore" {
		t.Errorf("expected kanban_status to-explore after reset, got %q", changes[0].KanbanStatus)
	}

	data, err := os.ReadFile(filepath.Join(changeDir, ".openspec.yaml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "launched:") || strings.Contains(content, "order:") {
		t.Errorf("expected launched/order fields to be removed from .openspec.yaml, got: %s", content)
	}

	entries, err := actStore.Read(workspaceID, "my-change")
	if err != nil {
		t.Fatalf("actStore.Read: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 activity entry, got %d", len(entries))
	}
	if entries[0].Type != "kanban.tasks_reset" {
		t.Errorf("expected type kanban.tasks_reset, got %s", entries[0].Type)
	}
	if entries[0].Category != "kanban" {
		t.Errorf("expected category kanban, got %s", entries[0].Category)
	}
}

func TestFFHandler_TriggerFF_AppendsActivity(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	claudeScript := filepath.Join(binDir, "claude")
	if err := os.WriteFile(claudeScript, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write mock claude script: %v", err)
	}
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	changeDir := filepath.Join(tmpDir, "openspec", "changes", "ff-change")
	if err := os.MkdirAll(changeDir, 0755); err != nil {
		t.Fatalf("failed to create change dir: %v", err)
	}

	absPath, err := filepath.Abs(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve abs path: %v", err)
	}
	workspaceID := workspace.StableID(absPath)
	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: tmpDir}}}
	ws := NewWorkspaceHandler(cfg, "", nil)

	prefDir := filepath.Join(tmpDir, "prefs.json")
	prefSvc := preferences.NewService(prefDir)

	convStore := conversation.NewStore(filepath.Join(tmpDir, "conversations"))
	actStore := activity.NewStore(filepath.Join(tmpDir, "activity"), nil)
	mgr := session.NewManager(prefSvc, convStore)
	watcherSvc := watcher.NewWatcherService()

	h := NewFFHandler(ws, mgr, convStore, actStore, watcherSvc, nil)

	rec, req := triggerFFRequest(workspaceID, "ff-change")
	h.TriggerFF(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d: %s", rec.Code, rec.Body.String())
	}

	// Wait briefly for goroutine to finish
	time.Sleep(50 * time.Millisecond)

	entries, err := actStore.Read(workspaceID, "ff-change")
	if err != nil {
		t.Fatalf("failed to read activity: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 activity entry, got %d", len(entries))
	}
	if entries[0].Type != "kanban.ff_triggered" {
		t.Errorf("expected type kanban.ff_triggered, got %s", entries[0].Type)
	}
	if entries[0].Category != "kanban" {
		t.Errorf("expected category kanban, got %s", entries[0].Category)
	}
}
