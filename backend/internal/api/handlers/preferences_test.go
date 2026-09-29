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

func TestCustomAgentSpecializationsGetReflectsPatch(t *testing.T) {
	tmpDir := t.TempDir()
	prefSvc := preferences.NewService(filepath.Join(tmpDir, "preferences.json"))
	handler := NewPreferencesHandler(prefSvc)

	patchBody := bytes.NewBufferString(`{"customAgentSpecializations":["ml-ops","embedded"]}`)
	patchReq := httptest.NewRequest("PATCH", "/api/preferences", patchBody)
	patchRec := httptest.NewRecorder()
	handler.PatchPreferences(patchRec, patchReq)

	if patchRec.Code != http.StatusNoContent {
		t.Fatalf("expected PATCH code 204, got %d", patchRec.Code)
	}

	req := httptest.NewRequest("GET", "/api/preferences", nil)
	rec := httptest.NewRecorder()
	handler.GetPreferences(rec, req)

	var resp struct {
		CustomAgentSpecializations []string `json:"customAgentSpecializations"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.CustomAgentSpecializations) != 2 || resp.CustomAgentSpecializations[0] != "ml-ops" || resp.CustomAgentSpecializations[1] != "embedded" {
		t.Errorf("expected customAgentSpecializations to reflect PATCH, got %v", resp.CustomAgentSpecializations)
	}
}

func TestCustomAgentSpecializationsPatchStripsBaseDuplicates(t *testing.T) {
	tmpDir := t.TempDir()
	prefSvc := preferences.NewService(filepath.Join(tmpDir, "preferences.json"))
	handler := NewPreferencesHandler(prefSvc)

	patchBody := bytes.NewBufferString(`{"customAgentSpecializations":["frontend","ml-ops","frontend"]}`)
	patchReq := httptest.NewRequest("PATCH", "/api/preferences", patchBody)
	patchRec := httptest.NewRecorder()
	handler.PatchPreferences(patchRec, patchReq)

	if patchRec.Code != http.StatusNoContent {
		t.Fatalf("expected PATCH code 204, got %d", patchRec.Code)
	}

	req := httptest.NewRequest("GET", "/api/preferences", nil)
	rec := httptest.NewRecorder()
	handler.GetPreferences(rec, req)

	var resp struct {
		CustomAgentSpecializations []string `json:"customAgentSpecializations"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.CustomAgentSpecializations) != 1 || resp.CustomAgentSpecializations[0] != "ml-ops" {
		t.Errorf("expected only ['ml-ops'] persisted (base duplicate and inner dup stripped), got %v", resp.CustomAgentSpecializations)
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

func TestAgentEnvGetAndPatch(t *testing.T) {
	prefSvc := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	handler := NewPreferencesHandler(prefSvc)

	get := func() map[string]map[string]string {
		rec := httptest.NewRecorder()
		handler.GetPreferences(rec, httptest.NewRequest("GET", "/api/preferences", nil))
		var resp struct {
			AgentEnv map[string]map[string]string `json:"agentEnv"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return resp.AgentEnv
	}

	initial := get()
	for _, id := range []string{"claude", "codex", "gemini", "antigravity", "copilot"} {
		if e, ok := initial[id]; !ok || e == nil || len(e) != 0 {
			t.Errorf("expected empty agentEnv entry for %s, got %v", id, e)
		}
	}

	rec := httptest.NewRecorder()
	handler.PatchPreferences(rec, httptest.NewRequest("PATCH", "/api/preferences", bytes.NewBufferString(`{"agentEnv":{"gemini":{"K":"V"}}}`)))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	got := get()
	if got["gemini"]["K"] != "V" || len(got["claude"]) != 0 {
		t.Errorf("unexpected agentEnv after patch: %v", got)
	}

	rec = httptest.NewRecorder()
	handler.PatchPreferences(rec, httptest.NewRequest("PATCH", "/api/preferences", bytes.NewBufferString(`{"agentEnv":{"nope":{"K":"V"}}}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown agent, got %d", rec.Code)
	}
}
