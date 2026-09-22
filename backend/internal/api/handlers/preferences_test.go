package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/glefebvre/opensp8c/internal/preferences"
)

func TestGetPreferencesWithSystemEnv(t *testing.T) {
	// Set test environment variables
	t.Setenv("GOOGLE_CLOUD_PROJECT", "test-gcp-project-123")
	t.Setenv("GEMINI_MODEL", "test-gemini-2.0-flash")
	t.Setenv("GEMINI_SANDBOX", "false")

	// Set up a temporary preferences service
	tmpDir := t.TempDir()
	prefSvc := preferences.NewService(filepath.Join(tmpDir, "preferences.json"))

	handler := NewPreferencesHandler(prefSvc)

	req := httptest.NewRequest("GET", "/api/preferences", nil)
	rec := httptest.NewRecorder()

	handler.GetPreferences(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected code 200, got %d", rec.Code)
	}

	var resp struct {
		DefaultAgent string            `json:"defaultAgent"`
		Env          map[string]string `json:"env"`
		SystemEnv    map[string]string `json:"systemEnv"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.SystemEnv == nil {
		t.Fatal("expected systemEnv to be present, got nil")
	}

	if resp.SystemEnv["GOOGLE_CLOUD_PROJECT"] != "test-gcp-project-123" {
		t.Errorf("expected GOOGLE_CLOUD_PROJECT test-gcp-project-123, got %q", resp.SystemEnv["GOOGLE_CLOUD_PROJECT"])
	}

	if resp.SystemEnv["GEMINI_MODEL"] != "test-gemini-2.0-flash" {
		t.Errorf("expected GEMINI_MODEL test-gemini-2.0-flash, got %q", resp.SystemEnv["GEMINI_MODEL"])
	}

	if resp.SystemEnv["GEMINI_SANDBOX"] != "false" {
		t.Errorf("expected GEMINI_SANDBOX false, got %q", resp.SystemEnv["GEMINI_SANDBOX"])
	}
}

func TestNativeQuestionModeDefaultAndPatch(t *testing.T) {
	tmpDir := t.TempDir()
	prefSvc := preferences.NewService(filepath.Join(tmpDir, "preferences.json"))
	handler := NewPreferencesHandler(prefSvc)

	// Default is false.
	req := httptest.NewRequest("GET", "/api/preferences", nil)
	rec := httptest.NewRecorder()
	handler.GetPreferences(rec, req)

	var resp struct {
		NativeQuestionMode bool `json:"nativeQuestionMode"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.NativeQuestionMode != false {
		t.Errorf("expected nativeQuestionMode to default to false, got %v", resp.NativeQuestionMode)
	}

	// PATCH persists true.
	patchBody := bytes.NewBufferString(`{"nativeQuestionMode":true}`)
	patchReq := httptest.NewRequest("PATCH", "/api/preferences", patchBody)
	patchRec := httptest.NewRecorder()
	handler.PatchPreferences(patchRec, patchReq)

	if patchRec.Code != http.StatusNoContent {
		t.Fatalf("expected PATCH code 204, got %d", patchRec.Code)
	}

	req2 := httptest.NewRequest("GET", "/api/preferences", nil)
	rec2 := httptest.NewRecorder()
	handler.GetPreferences(rec2, req2)

	var resp2 struct {
		NativeQuestionMode bool `json:"nativeQuestionMode"`
	}
	if err := json.NewDecoder(rec2.Body).Decode(&resp2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp2.NativeQuestionMode != true {
		t.Errorf("expected nativeQuestionMode to persist as true after PATCH, got %v", resp2.NativeQuestionMode)
	}
}
