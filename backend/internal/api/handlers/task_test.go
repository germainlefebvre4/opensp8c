package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/glefebvre/opensp8c/internal/activity"
	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/workspace"
	"github.com/go-chi/chi/v5"
)

func patchTaskRequest(workspaceID, changeName, index string) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest("PATCH", "/workspaces/"+workspaceID+"/changes/"+changeName+"/tasks/"+index, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workspaceID)
	rctx.URLParams.Add("name", changeName)
	rctx.URLParams.Add("index", index)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return httptest.NewRecorder(), req
}

func TestTaskHandler_PatchTask_AppendsActivity(t *testing.T) {
	tmpDir := t.TempDir()
	changeDir := filepath.Join(tmpDir, "openspec", "changes", "my-change")
	if err := os.MkdirAll(changeDir, 0755); err != nil {
		t.Fatalf("failed to create change dir: %v", err)
	}
	tasksContent := "- [ ] First task\n- [ ] Second task\n"
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte(tasksContent), 0644); err != nil {
		t.Fatalf("failed to write tasks.md: %v", err)
	}

	absPath, err := filepath.Abs(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve abs path: %v", err)
	}
	wsID := workspace.StableID(absPath)
	cfg := &config.Config{Workspaces: []config.WorkspaceConfig{{Name: "test", Path: tmpDir}}}
	ws := NewWorkspaceHandler(cfg, "", nil)

	actDir := filepath.Join(tmpDir, "activity")
	actStore := activity.NewStore(actDir, nil)

	h := NewTaskHandler(ws, actStore)

	// Toggle first task (index 0)
	rec, req := patchTaskRequest(wsID, "my-change", "0")
	h.PatchTask(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	entries, err := actStore.Read(wsID, "my-change")
	if err != nil {
		t.Fatalf("failed to read activity: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 activity entry, got %d", len(entries))
	}
	if entries[0].Type != "kanban.task_toggled" {
		t.Errorf("expected type kanban.task_toggled, got %s", entries[0].Type)
	}
	if entries[0].Category != "kanban" {
		t.Errorf("expected category kanban, got %s", entries[0].Category)
	}
	if entries[0].Summary != "Completed: First task" {
		t.Errorf("expected summary 'Completed: First task', got %q", entries[0].Summary)
	}
	if done, ok := entries[0].Meta["done"].(bool); !ok || !done {
		t.Errorf("expected meta.done == true, got %v", entries[0].Meta["done"])
	}
}
