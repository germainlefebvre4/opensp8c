package verification

import (
	"encoding/json"
	"testing"
)

func b(v bool) *bool     { return &v }
func s(v string) *string { return &v }

func TestResolve(t *testing.T) {
	cases := []struct {
		name                string
		platform, workspace *Level
		change              *Override
		want                Resolved
	}{
		{"nothing", nil, nil, nil, Resolved{}},
		{"platform only", &Level{Override: Override{Conformity: b(true)}}, nil, nil, Resolved{Conformity: true}},
		{"workspace disables", &Level{Override: Override{Conformity: b(true)}}, &Level{Override: Override{Conformity: b(false)}}, nil, Resolved{}},
		{"change enables", nil, nil, &Override{Conformity: b(true)}, Resolved{Conformity: true}},
		{"change disables over workspace", nil, &Level{Override: Override{UI: b(true)}}, &Override{UI: b(false)}, Resolved{}},
		{"mixed steps", nil, &Level{Override: Override{UI: b(true)}}, &Override{Conformity: b(true)}, Resolved{Conformity: true, UI: true}},
		{"empty levels", &Level{}, &Level{}, &Override{}, Resolved{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Resolve(c.platform, c.workspace, c.change); got != c.want {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestResolveLaunchParamsFieldByField(t *testing.T) {
	platform := &Level{LaunchOverride: LaunchOverride{UIStartCommand: s("make dev"), UIBaseURL: s("http://localhost:5173")}}
	ws := &Level{LaunchOverride: LaunchOverride{UIBaseURL: s("http://localhost:3000")}}
	got := Resolve(platform, ws, nil)
	if got.UIStartCommand != "make dev" || got.UIBaseURL != "http://localhost:3000" {
		t.Fatalf("got %+v", got.LaunchParams)
	}
	// A blank workspace value counts as absent.
	got = Resolve(platform, &Level{LaunchOverride: LaunchOverride{UIStartCommand: s("   ")}}, nil)
	if got.UIStartCommand != "make dev" {
		t.Fatalf("blank workspace value must inherit, got %q", got.UIStartCommand)
	}
}

func TestValidateBaseURL(t *testing.T) {
	for _, ok := range []string{"", "   ", "http://localhost:3000", "https://example.com/app"} {
		if err := ValidateBaseURL(ok); err != nil {
			t.Errorf("%q should be valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"localhost:5173", "ftp://hote", "/relative", "http://"} {
		if err := ValidateBaseURL(bad); err == nil {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestNormalizeText(t *testing.T) {
	if NormalizeText("   ") != "" || NormalizeText("  make dev ") != "make dev" {
		t.Fatal("unexpected normalization")
	}
}

func TestPatch(t *testing.T) {
	var p Patch
	if err := json.Unmarshal([]byte(`{"conformity": true, "ui": null}`), &p); err != nil {
		t.Fatal(err)
	}
	got := p.Apply(&Override{UI: b(false)})
	if got == nil || got.Conformity == nil || !*got.Conformity || got.UI != nil {
		t.Fatalf("got %+v", got)
	}
	// Absent field is untouched; clearing the last value yields nil.
	var q Patch
	_ = json.Unmarshal([]byte(`{"conformity": null}`), &q)
	if got := q.Apply(&Override{Conformity: b(true)}); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
	var bad Patch
	if err := json.Unmarshal([]byte(`{"ui": "maybe"}`), &bad); err == nil {
		t.Fatal("non-boolean must be rejected")
	}
}
