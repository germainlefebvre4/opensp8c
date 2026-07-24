package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestGhostDraftCRUD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "opensp8c-test-drafts-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	h := &ExploreHandler{
		draftsDir: tmpDir,
	}

	workspaceID := "ws123"
	ghostID := "ghost456"

	// 1. GET - Should return an empty default draft
	req := httptest.NewRequest("GET", "/workspaces/"+workspaceID+"/explorations/"+ghostID+"/draft", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workspaceID)
	rctx.URLParams.Add("ghostId", ghostID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.GetGhostDraft(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected GET code 200, got %d", rec.Code)
	}

	var defaultDraft ExplorationDraft
	if err := json.NewDecoder(rec.Body).Decode(&defaultDraft); err != nil {
		t.Fatalf("failed to decode GET body: %v", err)
	}

	if defaultDraft.GhostID != ghostID || defaultDraft.WorkspaceID != workspaceID {
		t.Errorf("default draft mismatch: %+v", defaultDraft)
	}
	if len(defaultDraft.Tasks) != 0 {
		t.Errorf("expected no default tasks, got %d", len(defaultDraft.Tasks))
	}

	// 2. PUT - Update/Create draft
	payload := ExplorationDraft{
		Name:        "test-name",
		Description: "test desc",
		Tasks: []DraftTask{
			{ID: "t1", Text: "Task 1", Done: false},
			{ID: "t2", Text: "Task 2", Done: true},
		},
	}
	bodyBytes, _ := json.Marshal(payload)
	putReq := httptest.NewRequest("PUT", "/workspaces/"+workspaceID+"/explorations/"+ghostID+"/draft", bytes.NewReader(bodyBytes))
	putReq = putReq.WithContext(context.WithValue(putReq.Context(), chi.RouteCtxKey, rctx))

	putRec := httptest.NewRecorder()
	h.UpdateGhostDraft(putRec, putReq)

	if putRec.Code != http.StatusOK {
		t.Errorf("expected PUT code 200, got %d", putRec.Code)
	}

	var savedDraft ExplorationDraft
	if err := json.NewDecoder(putRec.Body).Decode(&savedDraft); err != nil {
		t.Fatalf("failed to decode PUT response: %v", err)
	}

	if savedDraft.Name != "test-name" || len(savedDraft.Tasks) != 2 {
		t.Errorf("saved draft incorrect: %+v", savedDraft)
	}

	// Verify file is actually on disk
	filePath := filepath.Join(tmpDir, ghostID+".json")
	if _, err := os.Stat(filePath); err != nil {
		t.Errorf("expected file to exist at %s, got err: %v", filePath, err)
	}

	// 3. GET - Retrieve again, should return the saved draft from disk
	getReq2 := httptest.NewRequest("GET", "/workspaces/"+workspaceID+"/explorations/"+ghostID+"/draft", nil)
	getReq2 = getReq2.WithContext(context.WithValue(getReq2.Context(), chi.RouteCtxKey, rctx))

	getRec2 := httptest.NewRecorder()
	h.GetGhostDraft(getRec2, getReq2)

	if getRec2.Code != http.StatusOK {
		t.Errorf("expected GET (2) code 200, got %d", getRec2.Code)
	}

	var retrievedDraft ExplorationDraft
	if err := json.NewDecoder(getRec2.Body).Decode(&retrievedDraft); err != nil {
		t.Fatalf("failed to decode GET (2) response: %v", err)
	}

	if retrievedDraft.Name != "test-name" || len(retrievedDraft.Tasks) != 2 || retrievedDraft.Tasks[1].Done != true {
		t.Errorf("retrieved draft incorrect: %+v", retrievedDraft)
	}

	// 4. DELETE - Delete the draft
	delReq := httptest.NewRequest("DELETE", "/workspaces/"+workspaceID+"/explorations/"+ghostID+"/draft", nil)
	delReq = delReq.WithContext(context.WithValue(delReq.Context(), chi.RouteCtxKey, rctx))

	delRec := httptest.NewRecorder()
	h.DeleteGhostDraft(delRec, delReq)

	if delRec.Code != http.StatusNoContent {
		t.Errorf("expected DELETE code 204, got %d", delRec.Code)
	}

	// Verify file is deleted from disk
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted, got err: %v", err)
	}

	// 5. GET - Retrieve after delete, should fall back to empty default
	getReq3 := httptest.NewRequest("GET", "/workspaces/"+workspaceID+"/explorations/"+ghostID+"/draft", nil)
	getReq3 = getReq3.WithContext(context.WithValue(getReq3.Context(), chi.RouteCtxKey, rctx))

	getRec3 := httptest.NewRecorder()
	h.GetGhostDraft(getRec3, getReq3)

	var finalDraft ExplorationDraft
	_ = json.NewDecoder(getRec3.Body).Decode(&finalDraft)
	if len(finalDraft.Tasks) != 0 {
		t.Errorf("expected empty tasks after delete, got %d", len(finalDraft.Tasks))
	}
}
