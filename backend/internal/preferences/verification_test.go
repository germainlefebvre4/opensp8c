package preferences

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/glefebvre/opensp8c/internal/verification"
)

func vpatch(t *testing.T, j string) VerificationPatch {
	t.Helper()
	var p VerificationPatch
	if err := json.Unmarshal([]byte(j), &p); err != nil {
		t.Fatalf("%s: %v", j, err)
	}
	return p
}

func TestVerificationPersistenceAndReset(t *testing.T) {
	svc := newTestService(t)
	// Existing file without the field loads, everything is off.
	if err := os.WriteFile(svc.Path(), []byte(`{"defaultAgent":"claude"}`), 0644); err != nil {
		t.Fatal(err)
	}
	p, err := svc.Load()
	if err != nil {
		t.Fatal(err)
	}
	if r := p.ResolveVerification("A", nil); r.Conformity || r.UI {
		t.Fatalf("expected off: %+v", r)
	}

	if err := svc.SetVerificationDefaults(vpatch(t, `{"conformity":true,"uiStartCommand":" make dev ","uiBaseUrl":"http://localhost:5173"}`)); err != nil {
		t.Fatal(err)
	}
	if err := svc.PatchWorkspace("A", WorkspaceSettingsPatch{Verification: ptr(vpatch(t, `{"conformity":false,"uiBaseUrl":"http://localhost:3000"}`))}); err != nil {
		t.Fatal(err)
	}
	p, _ = svc.Load()
	if a := p.ResolveVerification("A", nil); a.Conformity || a.UIStartCommand != "make dev" || a.UIBaseURL != "http://localhost:3000" {
		t.Errorf("A: %+v", a)
	}
	if b := p.ResolveVerification("B", nil); !b.Conformity || b.UIBaseURL != "http://localhost:5173" {
		t.Errorf("B must inherit: %+v", b)
	}
	// A change-level value wins over the workspace.
	tr := true
	if a := p.ResolveVerification("A", &verification.Override{Conformity: &tr}); !a.Conformity {
		t.Errorf("change must win: %+v", a)
	}

	// Resetting the last override removes the section.
	if err := svc.PatchWorkspace("A", WorkspaceSettingsPatch{Verification: ptr(vpatch(t, `{"conformity":null,"uiBaseUrl":null}`))}); err != nil {
		t.Fatal(err)
	}
	p, _ = svc.Load()
	if len(p.Workspaces) != 0 {
		t.Errorf("section should be removed: %+v", p.Workspaces)
	}
	// Disabling every default removes the section from the file.
	if err := svc.SetVerificationDefaults(vpatch(t, `{"conformity":false,"uiStartCommand":null,"uiBaseUrl":"  "}`)); err != nil {
		t.Fatal(err)
	}
	p, _ = svc.Load()
	if p.VerificationDefaults != nil {
		t.Errorf("defaults should be removed: %+v", p.VerificationDefaults)
	}
	// Unknown workspace sections are ignored.
	if r := p.ResolveVerification("ghost", nil); r.Conformity || r.UI {
		t.Errorf("ghost: %+v", r)
	}
}

func TestVerificationPatchValidation(t *testing.T) {
	svc := newTestService(t)
	for _, j := range []string{`{"uiBaseUrl":"localhost:5173"}`, `{"uiBaseUrl":"ftp://hote"}`} {
		var ve *ValidationError
		if err := svc.SetVerificationDefaults(vpatch(t, j)); !errors.As(err, &ve) {
			t.Errorf("defaults %s: %v", j, err)
		}
		if err := svc.PatchWorkspace("A", WorkspaceSettingsPatch{Verification: ptr(vpatch(t, j))}); !errors.As(err, &ve) {
			t.Errorf("workspace %s: %v", j, err)
		}
		if err := svc.ValidateGlobalUpdate(nil, nil, ptr(vpatch(t, j))); !errors.As(err, &ve) {
			t.Errorf("dry run %s: %v", j, err)
		}
	}
	if _, err := os.Stat(svc.Path()); err == nil {
		t.Error("rejected updates must not create the file")
	}
	var bad VerificationPatch
	if err := json.Unmarshal([]byte(`{"ui":"maybe"}`), &bad); err == nil {
		t.Error("non-boolean must not decode")
	}
	// Blank command is absent.
	if err := svc.SetVerificationDefaults(vpatch(t, `{"uiStartCommand":"   "}`)); err != nil {
		t.Fatal(err)
	}
	p, _ := svc.Load()
	if p.VerificationDefaults != nil {
		t.Errorf("blank command stored: %+v", p.VerificationDefaults)
	}
}
