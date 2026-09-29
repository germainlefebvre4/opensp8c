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

type langResp struct {
	AgentLanguages     map[string]string `json:"agentLanguages"`
	UILocale           string            `json:"uiLocale"`
	SupportedLanguages []struct {
		Code       string `json:"code"`
		NativeName string `json:"nativeName"`
	} `json:"supportedLanguages"`
}

func langHandler(t *testing.T) *PreferencesHandler {
	t.Helper()
	return NewPreferencesHandler(preferences.NewService(filepath.Join(t.TempDir(), "preferences.json")))
}

func getLang(t *testing.T, h *PreferencesHandler) langResp {
	t.Helper()
	rec := httptest.NewRecorder()
	h.GetPreferences(rec, httptest.NewRequest("GET", "/api/preferences", nil))
	var r langResp
	if err := json.NewDecoder(rec.Body).Decode(&r); err != nil {
		t.Fatal(err)
	}
	return r
}

func patchLang(h *PreferencesHandler, body string) int {
	rec := httptest.NewRecorder()
	h.PatchPreferences(rec, httptest.NewRequest("PATCH", "/api/preferences", bytes.NewBufferString(body)))
	return rec.Code
}

func TestGetPreferencesLanguageDefaults(t *testing.T) {
	r := getLang(t, langHandler(t))
	if r.AgentLanguages["chat"] != "auto" || r.AgentLanguages["documentation"] != "auto" || r.AgentLanguages["code"] != "en" {
		t.Errorf("defaults: %+v", r.AgentLanguages)
	}
	if r.UILocale != "" {
		t.Errorf("uiLocale = %q", r.UILocale)
	}
	if len(r.SupportedLanguages) < 2 || r.SupportedLanguages[0].Code != "en" || r.SupportedLanguages[1].NativeName != "Français" {
		t.Errorf("supported: %+v", r.SupportedLanguages)
	}
}

func TestPatchLanguagePartialAndUILocale(t *testing.T) {
	h := langHandler(t)
	if c := patchLang(h, `{"agentLanguages":{"chat":"fr","code":"fr"},"uiLocale":"fr"}`); c != http.StatusNoContent {
		t.Fatalf("code %d", c)
	}
	if c := patchLang(h, `{"agentLanguages":{"documentation":"en"}}`); c != http.StatusNoContent {
		t.Fatalf("code %d", c)
	}
	r := getLang(t, h)
	if r.AgentLanguages["chat"] != "fr" || r.AgentLanguages["documentation"] != "en" || r.AgentLanguages["code"] != "fr" || r.UILocale != "fr" {
		t.Errorf("got %+v locale %q", r.AgentLanguages, r.UILocale)
	}
}

func TestPatchLanguageRejections(t *testing.T) {
	cases := map[string]string{
		"unknown code":   `{"agentLanguages":{"chat":"xx"}}`,
		"auto for code":  `{"agentLanguages":{"code":"auto"}}`,
		"bad ui locale":  `{"uiLocale":"xx"}`,
		"partial reject": `{"agentLanguages":{"chat":"fr","code":"auto"},"uiLocale":"fr"}`,
	}
	for name, body := range cases {
		h := langHandler(t)
		if c := patchLang(h, body); c != http.StatusBadRequest {
			t.Errorf("%s: code %d", name, c)
		}
		r := getLang(t, h)
		if r.AgentLanguages["chat"] != "auto" || r.AgentLanguages["code"] != "en" || r.UILocale != "" {
			t.Errorf("%s: state modified: %+v %q", name, r.AgentLanguages, r.UILocale)
		}
	}
}
