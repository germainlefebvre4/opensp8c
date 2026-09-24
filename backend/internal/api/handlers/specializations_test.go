package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/preferences"
)

func TestGetAgentSpecializationsReturnsBaseAndCustomSeparately(t *testing.T) {
	tmpDir := t.TempDir()
	prefSvc := preferences.NewService(filepath.Join(tmpDir, "preferences.json"))
	if err := prefSvc.SetCustomAgentSpecializations([]string{"ml-ops"}); err != nil {
		t.Fatalf("SetCustomAgentSpecializations: %v", err)
	}

	handler := NewSpecializationsHandler(prefSvc)

	req := httptest.NewRequest("GET", "/api/agent-specializations", nil)
	rec := httptest.NewRecorder()
	handler.GetAgentSpecializations(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected code 200, got %d", rec.Code)
	}

	var resp struct {
		Base   []string `json:"base"`
		Custom []string `json:"custom"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Base) != len(openspec.BaseAgentSpecializations) {
		t.Errorf("expected base list of length %d, got %d: %v", len(openspec.BaseAgentSpecializations), len(resp.Base), resp.Base)
	}
	if len(resp.Custom) != 1 || resp.Custom[0] != "ml-ops" {
		t.Errorf("expected custom list [ml-ops], got %v", resp.Custom)
	}
}
