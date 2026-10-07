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

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/workspace"
	"github.com/go-chi/chi/v5"
)

func patchPrefs(t *testing.T, h *PreferencesHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.PatchPreferences(rec, httptest.NewRequest(http.MethodPatch, "/api/preferences", bytes.NewBufferString(body)))
	return rec
}

func getPrefs(t *testing.T, h *PreferencesHandler) map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	h.GetPreferences(rec, httptest.NewRequest(http.MethodGet, "/api/preferences", nil))
	var out map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPreferencesAgentSettingsRoundTrip(t *testing.T) {
	prefs := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	h := NewPreferencesHandler(prefs)

	before := getPrefs(t, h)
	roles := before["agentSettings"].(map[string]any)["roles"].(map[string]any)
	if len(roles) != 6 {
		t.Fatalf("expected the six roles: %v", roles)
	}
	resolved := before["resolvedAgentSettings"].(map[string]any)["roles"].(map[string]any)
	if resolved["explorer"].(map[string]any)["model"] != "opus" {
		t.Errorf("preset not exposed: %v", resolved["explorer"])
	}
	if before["poolDefaults"].(map[string]any)["size"].(float64) != 3 {
		t.Errorf("pool default: %v", before["poolDefaults"])
	}

	rec := patchPrefs(t, h, `{"agentSettings":{"roles":{"documenter":{"model":"haiku"}}},"poolDefaults":{"size":2}}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("code %d: %s", rec.Code, rec.Body.String())
	}
	after := getPrefs(t, h)
	doc := after["agentSettings"].(map[string]any)["roles"].(map[string]any)["documenter"].(map[string]any)
	if doc["model"] != "haiku" {
		t.Errorf("documenter: %v", doc)
	}
	if other := after["agentSettings"].(map[string]any)["roles"].(map[string]any)["explorer"].(map[string]any); len(other) != 0 {
		t.Errorf("other roles must be unchanged: %v", other)
	}
	if after["poolDefaults"].(map[string]any)["size"].(float64) != 2 {
		t.Errorf("pool defaults: %v", after["poolDefaults"])
	}
}

func TestPreferencesPatchRejections(t *testing.T) {
	prefs := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	h := NewPreferencesHandler(prefs)
	if rec := patchPrefs(t, h, `{"defaultAgent":"gemini"}`); rec.Code != http.StatusNoContent {
		t.Fatal(rec.Body.String())
	}
	before, _ := os.ReadFile(prefs.Path())

	for _, body := range []string{
		`{"agentSettings":{"global":{"agent":"nope"}}}`,
		`{"agentSettings":{"roles":{"explorer":{"model":"--flag"}}}}`,
		`{"agentSettings":{"roles":{"unknown":{"model":"x"}}}}`,
		`{"agentSettings":{"roles":{"ff":{"effort":"bogus"}}}}`,
		`{"poolDefaults":{"size":0}}`,
		// a valid change combined with an invalid one must not be partially applied
		`{"defaultAgent":"codex","agentSettings":{"roles":{"explorer":{"model":"--x"}}}}`,
	} {
		rec := patchPrefs(t, h, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", body, rec.Code)
		}
		after, _ := os.ReadFile(prefs.Path())
		if string(after) != string(before) {
			t.Fatalf("%s: file modified by a rejected update", body)
		}
	}
}

func wsSettingsFixture(t *testing.T) (*WorkspaceSettingsHandler, *preferences.Service, string, string) {
	t.Helper()
	dirA, dirB := t.TempDir(), t.TempDir()
	ws := NewWorkspaceHandler(&config.Config{Workspaces: []config.WorkspaceConfig{{Name: "a", Path: dirA}, {Name: "b", Path: dirB}}}, "", nil)
	prefs := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	return NewWorkspaceSettingsHandler(ws, prefs), prefs, workspace.StableID(mustAbs(t, dirA)), workspace.StableID(mustAbs(t, dirB))
}

func wsReq(method, id, body string) *http.Request {
	req := httptest.NewRequest(method, "/api/workspaces/"+id+"/settings", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func decodeMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	return out
}

func roleModel(m map[string]any, section, role string) string {
	return m[section].(map[string]any)["agentSettings"].(map[string]any)["roles"].(map[string]any)[role].(map[string]any)["model"].(string)
}

func TestWorkspaceSettingsOverrideResetAndIsolation(t *testing.T) {
	h, prefs, idA, idB := wsSettingsFixture(t)

	// Configuration level sets sonnet for implementer.
	var cfgPatch preferences.AgentSettingsPatch
	_ = json.Unmarshal([]byte(`{"roles":{"implementer":{"model":"sonnet"}}}`), &cfgPatch)
	if err := prefs.SetAgentSettings(cfgPatch); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.Patch(rec, wsReq(http.MethodPatch, idA, `{"agentSettings":{"roles":{"implementer":{"model":"opus"}}},"pool":{"size":4}}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d: %s", rec.Code, rec.Body.String())
	}
	out := decodeMap(t, rec)
	if roleModel(out, "resolved", "implementer") != "opus" || roleModel(out, "overrides", "implementer") != "opus" || roleModel(out, "inherited", "implementer") != "sonnet" {
		t.Errorf("unexpected A view: %v", out)
	}
	if out["resolved"].(map[string]any)["pool"].(map[string]any)["size"].(float64) != 4 {
		t.Errorf("pool: %v", out["resolved"])
	}

	// Isolation: workspace B is untouched.
	rec = httptest.NewRecorder()
	h.Get(rec, wsReq(http.MethodGet, idB, ""))
	outB := decodeMap(t, rec)
	if roleModel(outB, "resolved", "implementer") != "sonnet" || outB["resolved"].(map[string]any)["pool"].(map[string]any)["size"].(float64) != 3 {
		t.Errorf("B leaked: %v", outB)
	}

	// Reset via null removes the section from the file.
	rec = httptest.NewRecorder()
	h.Patch(rec, wsReq(http.MethodPatch, idA, `{"agentSettings":{"roles":{"implementer":{"model":null}}},"pool":{"size":null}}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("reset: %d %s", rec.Code, rec.Body.String())
	}
	if roleModel(decodeMap(t, rec), "resolved", "implementer") != "sonnet" {
		t.Error("reset did not restore inheritance")
	}
	p, _ := prefs.Load()
	if len(p.Workspaces) != 0 {
		t.Errorf("workspace section should be gone: %+v", p.Workspaces)
	}
}

func TestWorkspaceSettingsErrors(t *testing.T) {
	h, prefs, idA, _ := wsSettingsFixture(t)

	rec := httptest.NewRecorder()
	h.Get(rec, wsReq(http.MethodGet, "unknown", ""))
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET unknown: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.Patch(rec, wsReq(http.MethodPatch, "unknown", `{}`))
	if rec.Code != http.StatusNotFound {
		t.Errorf("PATCH unknown: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.Patch(rec, wsReq(http.MethodPatch, idA, `{"pool":{"size":9}}`))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid pool size: %d", rec.Code)
	}
	p, _ := prefs.Load()
	if len(p.Workspaces) != 0 {
		t.Errorf("rejected update modified prefs: %+v", p.Workspaces)
	}

	// A section for a removed workspace is ignored, not an error.
	if err := prefs.PatchWorkspace("gone", preferences.WorkspaceSettingsPatch{Env: map[string]string{"X": "1"}}); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	h.Get(rec, wsReq(http.MethodGet, idA, ""))
	if rec.Code != http.StatusOK {
		t.Errorf("GET with stale section: %d", rec.Code)
	}
}

func TestListAgentModels(t *testing.T) {
	h := NewPreferencesHandler(preferences.NewService(filepath.Join(t.TempDir(), "preferences.json")))
	// Fake the antigravity discovery.
	agents.ResetModelCache()
	t.Cleanup(agents.ResetModelCache)
	for i := range agents.SupportedAgents {
		if agents.SupportedAgents[i].ID == "antigravity" {
			orig := agents.SupportedAgents[i].ListModels
			agents.SupportedAgents[i].ListModels = func(ctx context.Context) ([]agents.Model, error) {
				return []agents.Model{{ID: "dyn-1", Label: "Dynamic One", Source: agents.ModelSourceCLI}}, nil
			}
			t.Cleanup(func() { agents.SupportedAgents[i].ListModels = orig })
		}
	}

	rec := httptest.NewRecorder()
	h.ListAgentModels(rec, httptest.NewRequest(http.MethodGet, "/api/agents/models", nil))
	var out map[string]struct {
		Models         []agents.Model `json:"models"`
		EffortLevels   []string       `json:"effortLevels"`
		SupportsModel  bool           `json:"supportsModel"`
		SupportsEffort bool           `json:"supportsEffort"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out) != len(agents.SupportedAgents) {
		t.Fatalf("expected every agent: %v", out)
	}
	agy1 := out["antigravity"]
	if len(agy1.Models) != 1 || agy1.Models[0].Source != "cli" || !agy1.SupportsEffort {
		t.Errorf("antigravity: %+v", agy1)
	}
	claude := out["claude"]
	if len(claude.Models) != 4 || claude.Models[0].Source != "seed" || len(claude.EffortLevels) != 5 {
		t.Errorf("claude: %+v", claude)
	}
	copilot := out["copilot"]
	if copilot.SupportsModel || copilot.SupportsEffort || len(copilot.Models) != 0 || copilot.EffortLevels == nil {
		t.Errorf("copilot: %+v", copilot)
	}
	if g := out["gemini"]; !g.SupportsModel || g.SupportsEffort || len(g.EffortLevels) != 0 {
		t.Errorf("gemini: %+v", g)
	}
}

func TestValidationCommandExposed(t *testing.T) {
	h, prefs, idA, _ := wsSettingsFixture(t)
	ph := NewPreferencesHandler(prefs)

	if rec := patchPrefs(t, ph, `{"poolDefaults":{"validationCommand":"make test"}}`); rec.Code != http.StatusNoContent {
		t.Fatalf("patch defaults: %d %s", rec.Code, rec.Body.String())
	}
	if got := getPrefs(t, ph)["poolDefaults"].(map[string]any)["validationCommand"]; got != "make test" {
		t.Errorf("GET /api/preferences: %v", got)
	}

	rec := httptest.NewRecorder()
	h.Patch(rec, wsReq(http.MethodPatch, idA, `{"pool":{"validationCommand":"cd backend && go test ./..."}}`))
	out := decodeMap(t, rec)
	if out["overrides"].(map[string]any)["pool"].(map[string]any)["validationCommand"] != "cd backend && go test ./..." {
		t.Errorf("override: %v", out["overrides"])
	}
	if out["inherited"].(map[string]any)["pool"].(map[string]any)["validationCommand"] != "make test" {
		t.Errorf("inherited: %v", out["inherited"])
	}
	if out["resolved"].(map[string]any)["pool"].(map[string]any)["validationCommand"] != "cd backend && go test ./..." {
		t.Errorf("resolved: %v", out["resolved"])
	}
}
