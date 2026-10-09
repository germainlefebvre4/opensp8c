package pool

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/verification"
)

func TestParseUIVerdict(t *testing.T) {
	cases := []struct {
		name, text, verdict string
		verified            []string
	}{
		{"pass", "ok\nVERDICT: PASS", "PASS", nil},
		{"last one wins", "VERDICT: FAIL\nrevu\nVERDICT: PASS\n", "PASS", nil},
		{"absent", "no conclusion", "", nil},
		{"case", "verdict: skip", "SKIP", nil},
		{"unknown value", "VERDICT: MAYBE", "", nil},
		{"several task lines", "TASK-VERIFIED:   4.2 Parcours  A \ntask-verified: 4.3 B\nTASK-VERIFIED:\nVERDICT: PASS", "PASS", []string{"4.2 Parcours  A", "4.3 B"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, tasks := parseUIVerdict(tc.text)
			if v != tc.verdict || !reflect.DeepEqual(tasks, tc.verified) {
				t.Fatalf("got %q %q", v, tasks)
			}
		})
	}
}

func TestPendingHumanTasks(t *testing.T) {
	content := "- [ ] 1 impl\n- [ ] 4.2 Parcours manuel de la navigation <!-- human review required -->\n- [x] 4.3 Déjà fait <!-- human review required -->\n"
	got := pendingHumanTasks(content)
	if !reflect.DeepEqual(got, []string{"4.2 Parcours manuel de la navigation"}) {
		t.Fatalf("got %q", got)
	}
}

func TestBuildUITurn(t *testing.T) {
	scenarios := []openspec.SpecScenarios{{File: "nav/spec.md", Scenarios: []openspec.Scenario{
		{Name: "Premier", Lines: []string{"- **WHEN** a", "- **THEN** b"}},
		{Name: "Second", Lines: []string{"- **WHEN** c"}},
	}}}
	turn := buildUITurn("http://localhost:4321", "/art/dir", scenarios, []string{"4.2 Parcours manuel de la navigation"}, "")
	for _, want := range []string{"http://localhost:4321", "/art/dir", "Scenario: Premier", "Scenario: Second", "- **THEN** b", "4.2 Parcours manuel de la navigation", "TASK-VERIFIED"} {
		if !strings.Contains(turn, want) {
			t.Errorf("%q missing from the turn:\n%s", want, turn)
		}
	}
	if strings.Contains(turn, "human review required") {
		t.Error("the marker must be removed")
	}

	empty := buildUITurn("http://x", "/a", nil, nil, "")
	if strings.Contains(empty, "Scenario:") || !strings.Contains(empty, "VERDICT: SKIP") {
		t.Errorf("no scenario: invite to SKIP, got:\n%s", empty)
	}
}

func TestBuildUITurnGuidance(t *testing.T) {
	scenarios := []openspec.SpecScenarios{{File: "nav/spec.md", Scenarios: []openspec.Scenario{{Name: "Premier", Lines: []string{"- **WHEN** a"}}}}}
	turn := buildUITurn("http://x", "/a", scenarios, []string{"4.2 Parcours"}, "  Viser le desktop 1280 px  ")
	if !strings.Contains(turn, uiGuidanceHeader+"Viser le desktop 1280 px\n") {
		t.Fatalf("guidance block missing:\n%s", turn)
	}
	// After the list of what to verify, before the closing instruction.
	g, s, task, end := strings.Index(turn, "Viser le desktop"), strings.Index(turn, "Scenario: Premier"), strings.Index(turn, "4.2 Parcours"), strings.Index(turn, "End with the VERDICT line")
	if !(s < g && task < g && g < end) {
		t.Errorf("guidance must follow the scenarios and tasks: %d %d %d %d", s, task, g, end)
	}
	// It says the guidance is indicative only.
	if !strings.Contains(turn, "never lifts the read-only rule") {
		t.Errorf("the guidance must be presented as indicative:\n%s", turn)
	}
	for _, blank := range []string{"", "  \n "} {
		if got := buildUITurn("http://x", "/a", scenarios, nil, blank); strings.Contains(got, "User guidance") {
			t.Errorf("no guidance block expected for %q:\n%s", blank, got)
		}
	}
	// Also present when there is nothing to verify.
	if got := buildUITurn("http://x", "/a", nil, nil, "Viser le desktop"); !strings.Contains(got, "Viser le desktop") || !strings.Contains(got, "VERDICT: SKIP") {
		t.Errorf("guidance with an empty change:\n%s", got)
	}
}

func TestBuildUIDirective(t *testing.T) {
	plans := map[string]driverPlan{
		"auto":       {Driver: "auto", Directive: autoDriverDirective},
		"playwright": {Driver: "playwright", Directive: driverDirective("Playwright")},
		"chrome":     {Driver: "chrome", Directive: driverDirective("the Chrome integration") + " " + chromeDirective},
		"custom":     {Driver: "custom", Directive: driverDirective("the MCP servers you are given")},
	}
	for name, plan := range plans {
		d := buildUIDirective(plan)
		for _, want := range []string{"NEVER create, modify, move or delete any file", "reading files is fine", "VERDICT: SKIP", artifactsEnv} {
			if !strings.Contains(d, want) {
				t.Errorf("%s: %q missing", name, want)
			}
		}
	}
	if d := buildUIDirective(plans["auto"]); !strings.Contains(d, "Choose the way you drive the browser") || strings.Contains(d, "Drive the browser only") {
		t.Errorf("auto lets the agent choose: %s", d)
	}
	for _, name := range []string{"playwright", "chrome", "custom"} {
		d := buildUIDirective(plans[name])
		if !strings.Contains(d, "Drive the browser only with the tools of") || !strings.Contains(d, "run no script") || strings.Contains(d, "Choose the way you drive") {
			t.Errorf("%s must impose the driver tools: %s", name, d)
		}
	}
	if d := buildUIDirective(plans["chrome"]); !strings.Contains(d, "only your own tabs") {
		t.Errorf("chrome: %s", d)
	}
	for _, name := range []string{"playwright", "custom"} {
		if d := buildUIDirective(plans[name]); !strings.Contains(d, "absolute path") {
			t.Errorf("%s must ask for absolute screenshot paths: %s", name, d)
		}
	}
	for _, name := range []string{"auto", "chrome"} {
		if d := buildUIDirective(plans[name]); strings.Contains(d, "absolute path") {
			t.Errorf("%s: unexpected absolute path clause", name)
		}
	}
	// A plan produced by planDriver carries the clause too.
	withFakeNpx(t)
	plan, err := planDriver(context.Background(), verification.Resolved{LaunchParams: verification.LaunchParams{UIDriver: "playwright"}}, claudeAgent, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer plan.cleanup()
	if !strings.Contains(buildUIDirective(plan), "tools of Playwright") {
		t.Errorf("planned directive: %s", buildUIDirective(plan))
	}
}
