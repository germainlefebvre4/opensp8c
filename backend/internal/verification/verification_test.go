package verification

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func b(v bool) *bool          { return &v }
func s(v string) *string      { return &v }
func l(v ...string) *[]string { return &v }

func TestResolve(t *testing.T) {
	cases := []struct {
		name                string
		platform, workspace *Level
		change              *Override
		want                Resolved
	}{
		{"nothing", nil, nil, nil, Resolved{LaunchParams: LaunchParams{UIDriver: DriverAuto}}},
		{"platform only", &Level{Override: Override{Conformity: b(true)}}, nil, nil, Resolved{Conformity: true, LaunchParams: LaunchParams{UIDriver: DriverAuto}}},
		{"workspace disables", &Level{Override: Override{Conformity: b(true)}}, &Level{Override: Override{Conformity: b(false)}}, nil, Resolved{LaunchParams: LaunchParams{UIDriver: DriverAuto}}},
		{"change enables", nil, nil, &Override{Conformity: b(true)}, Resolved{Conformity: true, LaunchParams: LaunchParams{UIDriver: DriverAuto}}},
		{"change disables over workspace", nil, &Level{Override: Override{UI: b(true)}}, &Override{UI: b(false)}, Resolved{LaunchParams: LaunchParams{UIDriver: DriverAuto}}},
		{"mixed steps", nil, &Level{Override: Override{UI: b(true)}}, &Override{Conformity: b(true)}, Resolved{Conformity: true, UI: true, LaunchParams: LaunchParams{UIDriver: DriverAuto}}},
		{"empty levels", &Level{}, &Level{}, &Override{}, Resolved{LaunchParams: LaunchParams{UIDriver: DriverAuto}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Resolve(c.platform, c.workspace, c.change); !reflect.DeepEqual(got, c.want) {
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
	for _, ok := range []string{"", "   ", "http://localhost:3000", "https://example.com/app", "http://localhost:{port}", "https://h:{port}/app"} {
		if err := ValidateBaseURL(ok); err != nil {
			t.Errorf("%q should be valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"localhost:5173", "ftp://hote", "/relative", "http://", "{port}://localhost", "http://{port}", "http://localhost:{port}/{port}"} {
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

func TestSubstitute(t *testing.T) {
	if got := Substitute("run --port {port} # {port}", 4242); got != "run --port 4242 # 4242" {
		t.Fatalf("got %q", got)
	}
	if got := Substitute("make dev", 4242); got != "make dev" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveDriver(t *testing.T) {
	d := func(v string) *Level { return &Level{LaunchOverride: LaunchOverride{UIDriver: s(v)}} }
	cases := []struct {
		name                string
		platform, workspace *Level
		want                string
	}{
		{"built-in", nil, nil, DriverAuto},
		{"platform", d("playwright"), nil, DriverPlaywright},
		{"workspace overrides", d("playwright"), d("custom"), DriverCustom},
		{"platform chrome ignored", d("chrome"), nil, DriverAuto},
		{"platform chrome falls back to nothing, workspace wins", d("chrome"), d("playwright"), DriverPlaywright},
		{"workspace chrome kept", nil, d("chrome"), DriverChrome},
		{"workspace chrome over platform playwright", d("playwright"), d("chrome"), DriverChrome},
		{"unknown value ignored", d("selenium"), nil, DriverAuto},
		{"blank workspace inherits", d("playwright"), d("  "), DriverPlaywright},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Resolve(c.platform, c.workspace, nil).UIDriver; got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestParseDriver(t *testing.T) {
	for in, want := range map[string]string{"auto": "auto", " playwright ": "playwright", "chrome": "chrome", "custom": "custom", "  ": ""} {
		if got, err := ParseDriver(in); err != nil || got != want {
			t.Errorf("ParseDriver(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := ParseDriver("selenium"); err == nil {
		t.Fatal("unknown driver must be refused")
	}
}

func TestResolveToolsAndMcpConfig(t *testing.T) {
	platform := &Level{LaunchOverride: LaunchOverride{UIMcpConfig: s("/etc/mcp/ui.json"), UIAllowedTools: l("mcp__cypress")}}
	ws := &Level{LaunchOverride: LaunchOverride{UIAllowedTools: l("mcp__cypress", " mcp__db ")}}
	got := Resolve(platform, ws, nil)
	if got.UIMcpConfig != "/etc/mcp/ui.json" || !reflect.DeepEqual(got.UIAllowedTools, []string{"mcp__cypress", "mcp__db"}) {
		t.Fatalf("got %+v", got.LaunchParams)
	}
	// A blank list is absent: the platform list applies.
	got = Resolve(platform, &Level{LaunchOverride: LaunchOverride{UIAllowedTools: l("  ", "")}}, nil)
	if !reflect.DeepEqual(got.UIAllowedTools, []string{"mcp__cypress"}) {
		t.Fatalf("blank list must inherit, got %v", got.UIAllowedTools)
	}
	if got := Resolve(&Level{LaunchOverride: LaunchOverride{UIAllowedTools: l()}}, nil, nil); got.UIAllowedTools != nil {
		t.Fatalf("empty list must be absent, got %v", got.UIAllowedTools)
	}
}

func TestNormalizeTools(t *testing.T) {
	got, err := NormalizeTools([]string{" a ", "", "  ", "b"})
	if err != nil || !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("got %v, %v", got, err)
	}
	if got, err := NormalizeTools([]string{" "}); err != nil || got != nil {
		t.Fatalf("blank list must be absent, got %v, %v", got, err)
	}
	fifty := make([]string, 50)
	for i := range fifty {
		fifty[i] = "t"
	}
	if _, err := NormalizeTools(fifty); err != nil {
		t.Fatalf("50 entries must be accepted: %v", err)
	}
	if _, err := NormalizeTools(append(fifty, "t")); err == nil {
		t.Fatal("51 entries must be refused")
	}
	// Blank entries do not count towards the limit.
	if _, err := NormalizeTools(append(fifty, "", " ")); err != nil {
		t.Fatalf("blank entries must not count: %v", err)
	}
}

func TestResolveGuidanceAccumulates(t *testing.T) {
	g := func(v string) *Level { return &Level{LaunchOverride: LaunchOverride{UIGuidance: s(v)}} }
	if got := Resolve(g("  A  "), g("B"), nil).UIGuidance; got != "A\n\nB" {
		t.Fatalf("two levels: %q", got)
	}
	if got := Resolve(nil, g("B"), nil).UIGuidance; got != "B" {
		t.Fatalf("workspace only: %q", got)
	}
	if got := Resolve(g("A"), g("   "), nil).UIGuidance; got != "A" {
		t.Fatalf("blank workspace: %q", got)
	}
	if got := Resolve(nil, nil, nil).UIGuidance; got != "" {
		t.Fatalf("none: %q", got)
	}
}

func TestValidateGuidance(t *testing.T) {
	if err := ValidateGuidance(strings.Repeat("é", 4000)); err != nil {
		t.Fatalf("4000 characters must be accepted: %v", err)
	}
	if err := ValidateGuidance(strings.Repeat("é", 4001)); err == nil {
		t.Fatal("4001 characters must be refused")
	}
}
