package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/workspace"
)

func resetTasksRequest(workspaceID, changeName string) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest("PATCH", "/workspaces/"+workspaceID+"/changes/"+changeName+"/tasks/reset", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workspaceID)
	rctx.URLParams.Add("name", changeName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return httptest.NewRecorder(), req
}

func TestFFHandler_ResetTasks_ClearsKanbanState(t *testing.T) {
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
	h := NewFFHandler(ws, nil, nil, nil)

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
}
