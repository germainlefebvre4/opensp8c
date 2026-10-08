package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/go-chi/chi/v5"
)

func verifSection(m map[string]any, section string) map[string]any {
	return m[section].(map[string]any)["verification"].(map[string]any)
}

func TestVerificationDefaultsRoundTrip(t *testing.T) {
	h, prefs, _, _ := wsSettingsFixture(t)
	_ = h
	ph := NewPreferencesHandler(prefs)

	d := getPrefs(t, ph)["verificationDefaults"].(map[string]any)
	if d["conformity"] != false || d["ui"] != false {
		t.Fatalf("defaults must be off: %v", d)
	}
	if rec := patchPrefs(t, ph, `{"verificationDefaults":{"conformity":true,"uiStartCommand":"make dev","uiBaseUrl":"http://localhost:5173"}}`); rec.Code != http.StatusNoContent {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}
	d = getPrefs(t, ph)["verificationDefaults"].(map[string]any)
	if d["conformity"] != true || d["ui"] != false || d["uiStartCommand"] != "make dev" {
		t.Errorf("defaults: %v", d)
	}

	before, _ := os.ReadFile(prefs.Path())
	for _, body := range []string{
		`{"verificationDefaults":{"uiBaseUrl":"localhost:5173"}}`,
		`{"verificationDefaults":{"ui":"maybe"}}`,
		`{"defaultAgent":"codex","verificationDefaults":{"uiBaseUrl":"ftp://hote"}}`,
	} {
		rec := patchPrefs(t, ph, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", body, rec.Code)
		}
	}
	// Rejected updates leave the file untouched.
	after, _ := os.ReadFile(prefs.Path())
	if string(after) != string(before) {
		t.Errorf("rejected updates modified the file:\n%s\n%s", before, after)
	}
}

func TestVerificationPortTokenKeptAsIs(t *testing.T) {
	h, prefs, idA, _ := wsSettingsFixture(t)
	ph := NewPreferencesHandler(prefs)
	if rec := patchPrefs(t, ph, `{"verificationDefaults":{"uiStartCommand":"npm run dev -- --port {port}","uiBaseUrl":"http://localhost:{port}"}}`); rec.Code != http.StatusNoContent {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}
	d := getPrefs(t, ph)["verificationDefaults"].(map[string]any)
	if d["uiStartCommand"] != "npm run dev -- --port {port}" || d["uiBaseUrl"] != "http://localhost:{port}" {
		t.Errorf("defaults: %v", d)
	}
	rec := httptest.NewRecorder()
	h.Patch(rec, wsReq(http.MethodPatch, idA, `{"verification":{"uiBaseUrl":"http://127.0.0.1:{port}","uiStartCommand":"make dev PORT={port}"}}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}
	if r := verifSection(decodeMap(t, rec), "resolved"); r["uiBaseUrl"] != "http://127.0.0.1:{port}" || r["uiStartCommand"] != "make dev PORT={port}" {
		t.Errorf("resolved: %v", r)
	}
	rec = httptest.NewRecorder()
	h.Patch(rec, wsReq(http.MethodPatch, idA, `{"verification":{"uiBaseUrl":"{port}://localhost"}}`))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("misplaced token: %d", rec.Code)
	}
}

func TestWorkspaceVerificationSettings(t *testing.T) {
	h, prefs, idA, idB := wsSettingsFixture(t)
	ph := NewPreferencesHandler(prefs)
	if rec := patchPrefs(t, ph, `{"verificationDefaults":{"conformity":true}}`); rec.Code != http.StatusNoContent {
		t.Fatal(rec.Body.String())
	}

	rec := httptest.NewRecorder()
	h.Patch(rec, wsReq(http.MethodPatch, idA, `{"verification":{"conformity":false,"ui":true}}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}
	out := decodeMap(t, rec)
	if o := verifSection(out, "overrides"); o["conformity"] != false || o["ui"] != true {
		t.Errorf("overrides: %v", o)
	}
	if i := verifSection(out, "inherited"); i["conformity"] != true || i["ui"] != false {
		t.Errorf("inherited: %v", i)
	}
	if r := verifSection(out, "resolved"); r["conformity"] != false || r["ui"] != true {
		t.Errorf("resolved: %v", r)
	}

	// Another workspace is unaffected.
	rec = httptest.NewRecorder()
	h.Get(rec, wsReq(http.MethodGet, idB, ""))
	outB := decodeMap(t, rec)
	if r := verifSection(outB, "resolved"); r["conformity"] != true || r["ui"] != false {
		t.Errorf("B leaked: %v", r)
	}
	if o := verifSection(outB, "overrides"); len(o) != 0 {
		t.Errorf("B overrides: %v", o)
	}

	// Invalid values are rejected without any change.
	for _, body := range []string{`{"verification":{"ui":"maybe"}}`, `{"verification":{"uiBaseUrl":"nope"}}`} {
		rec = httptest.NewRecorder()
		h.Patch(rec, wsReq(http.MethodPatch, idA, body))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d", body, rec.Code)
		}
	}
	rec = httptest.NewRecorder()
	h.Get(rec, wsReq(http.MethodGet, idA, ""))
	if r := verifSection(decodeMap(t, rec), "overrides"); r["ui"] != true {
		t.Errorf("rejected update changed state: %v", r)
	}

	// null resets.
	rec = httptest.NewRecorder()
	h.Patch(rec, wsReq(http.MethodPatch, idA, `{"verification":{"conformity":null,"ui":null}}`))
	out = decodeMap(t, rec)
	if r := verifSection(out, "resolved"); r["conformity"] != true || r["ui"] != false {
		t.Errorf("after reset: %v", r)
	}
	if p, _ := prefs.Load(); len(p.Workspaces) != 0 {
		t.Errorf("section should be removed: %+v", p.Workspaces)
	}

	rec = httptest.NewRecorder()
	h.Patch(rec, wsReq(http.MethodPatch, "unknown", `{"verification":{"ui":true}}`))
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown workspace: %d", rec.Code)
	}
}

func newVerifFixture(t *testing.T) (*KanbanHandler, *preferences.Service, string, string) {
	t.Helper()
	dir := t.TempDir()
	changesDir := filepath.Join(dir, "openspec", "changes")
	writeChangeWithMeta(t, changesDir, "my-change", "schema: spec-driven\ncreated: \"2024-01-01\"\n")
	writeChangeWithMeta(t, filepath.Join(changesDir, "archive"), "old-change", "schema: spec-driven\ncreated: \"2024-01-01\"\n")
	prefs := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	h, id := newTestKanbanHandler(t, dir, nil, prefs, nil, "")
	return h, prefs, id, dir
}

func verifChangeReq(method, id, name, body string) *http.Request {
	req := httptest.NewRequest(method, "/workspaces/"+id+"/changes/"+name+"/verification", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	rctx.URLParams.Add("name", name)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func patchChangeVerif(h *KanbanHandler, id, name, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.PatchVerification(rec, verifChangeReq(http.MethodPatch, id, name, body))
	return rec
}

func TestPatchChangeVerification(t *testing.T) {
	h, prefs, id, dir := newVerifFixture(t)
	metaPath := filepath.Join(dir, "openspec", "changes", "my-change", ".openspec.yaml")

	rec := patchChangeVerif(h, id, "my-change", `{"conformity":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("activate: %d %s", rec.Code, rec.Body.String())
	}
	out := decodeMap(t, rec)
	if out["override"].(map[string]any)["conformity"] != true || out["resolved"].(map[string]any)["conformity"] != true || out["inherited"].(map[string]any)["conformity"] != false {
		t.Errorf("response: %v", out)
	}
	if _, has := out["override"].(map[string]any)["ui"]; has {
		t.Errorf("ui must stay inherited: %v", out)
	}
	if data, _ := os.ReadFile(metaPath); !strings.Contains(string(data), "conformity: true") {
		t.Errorf("file: %s", data)
	}

	// The workspace enables ui; the change then disables it, then goes back.
	if err := prefs.PatchWorkspace(id, preferences.WorkspaceSettingsPatch{Verification: ptrVerif(t, `{"ui":true}`)}); err != nil {
		t.Fatal(err)
	}
	out = decodeMap(t, patchChangeVerif(h, id, "my-change", `{"ui":false}`))
	if out["resolved"].(map[string]any)["ui"] != false || out["inherited"].(map[string]any)["ui"] != true {
		t.Errorf("disable: %v", out)
	}
	out = decodeMap(t, patchChangeVerif(h, id, "my-change", `{"ui":null}`))
	if out["resolved"].(map[string]any)["ui"] != true || out["override"].(map[string]any)["conformity"] != true {
		t.Errorf("back to inherit: %v", out)
	}

	// Last setting removed leaves no section.
	patchChangeVerif(h, id, "my-change", `{"conformity":null}`)
	if data, _ := os.ReadFile(metaPath); strings.Contains(string(data), "verification") {
		t.Errorf("residual section: %s", data)
	}
}

func ptrVerif(t *testing.T, j string) *preferences.VerificationPatch {
	t.Helper()
	var p preferences.VerificationPatch
	if err := json.Unmarshal([]byte(j), &p); err != nil {
		t.Fatal(err)
	}
	return &p
}

func TestPatchChangeVerificationErrors(t *testing.T) {
	h, _, id, dir := newVerifFixture(t)
	archived := filepath.Join(dir, "openspec", "changes", "archive", "old-change", ".openspec.yaml")
	before, _ := os.ReadFile(archived)

	if rec := patchChangeVerif(h, id, "old-change", `{"ui":true}`); rec.Code != http.StatusConflict {
		t.Errorf("archived: %d", rec.Code)
	}
	if after, _ := os.ReadFile(archived); string(after) != string(before) {
		t.Errorf("archived file modified")
	}
	if rec := patchChangeVerif(h, id, "nope", `{"ui":true}`); rec.Code != http.StatusNotFound {
		t.Errorf("unknown change: %d", rec.Code)
	}
	if rec := patchChangeVerif(h, "unknown", "my-change", `{"ui":true}`); rec.Code != http.StatusNotFound {
		t.Errorf("unknown workspace: %d", rec.Code)
	}
	for _, body := range []string{`{"uiStartCommand":"make dev"}`, `{"uiBaseUrl":"http://x"}`, `{"ui":"maybe"}`, `not json`} {
		if rec := patchChangeVerif(h, id, "my-change", body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d", body, rec.Code)
		}
	}
}

func getChangeDetail(t *testing.T, h *KanbanHandler, id, name string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+id+"/changes/"+name, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	rctx.URLParams.Add("name", name)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	h.GetChange(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GetChange: %d %s", rec.Code, rec.Body.String())
	}
	return decodeMap(t, rec)
}

func TestChangeDetailExposesVerification(t *testing.T) {
	h, prefs, id, dir := newVerifFixture(t)

	v := getChangeDetail(t, h, id, "my-change")["verification"].(map[string]any)
	if len(v["override"].(map[string]any)) != 0 || v["inherited"].(map[string]any)["ui"] != false || v["resolved"].(map[string]any)["conformity"] != false {
		t.Errorf("no setting: %v", v)
	}

	if err := prefs.PatchWorkspace(id, preferences.WorkspaceSettingsPatch{Verification: ptrVerif(t, `{"ui":false,"conformity":false}`)}); err != nil {
		t.Fatal(err)
	}
	patchChangeVerif(h, id, "my-change", `{"ui":true}`)
	v = getChangeDetail(t, h, id, "my-change")["verification"].(map[string]any)
	if v["override"].(map[string]any)["ui"] != true || v["inherited"].(map[string]any)["ui"] != false || v["resolved"].(map[string]any)["ui"] != true {
		t.Errorf("change over workspace: %v", v)
	}

	// Archived changes carry no verification section.
	if _, has := getChangeDetail(t, h, id, "old-change")["verification"]; has {
		t.Error("archived change must not expose verification")
	}
	_ = dir
}

func TestChangeVerificationReadsMainRepository(t *testing.T) {
	h, _, id, dir := newVerifFixture(t)
	patchChangeVerif(h, id, "my-change", `{"conformity":true}`)
	// An outdated copy lives in a stale worktree directory; it must be ignored.
	stale := filepath.Join(t.TempDir(), "openspec", "changes", "my-change")
	if err := os.MkdirAll(stale, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stale, ".openspec.yaml"), []byte("schema: spec-driven\nverification:\n    conformity: false\n"), 0644); err != nil {
		t.Fatal(err)
	}
	v := getChangeDetail(t, h, id, "my-change")["verification"].(map[string]any)
	if v["resolved"].(map[string]any)["conformity"] != true {
		t.Errorf("main repository must win: %v", v)
	}
	_ = dir
}
