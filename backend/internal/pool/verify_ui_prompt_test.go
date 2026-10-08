package pool

import (
	"reflect"
	"strings"
	"testing"

	"github.com/glefebvre/opensp8c/internal/openspec"
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
	turn := buildUITurn("http://localhost:4321", "/art/dir", scenarios, []string{"4.2 Parcours manuel de la navigation"})
	for _, want := range []string{"http://localhost:4321", "/art/dir", "Scenario: Premier", "Scenario: Second", "- **THEN** b", "4.2 Parcours manuel de la navigation", "TASK-VERIFIED"} {
		if !strings.Contains(turn, want) {
			t.Errorf("%q missing from the turn:\n%s", want, turn)
		}
	}
	if strings.Contains(turn, "human review required") {
		t.Error("the marker must be removed")
	}

	empty := buildUITurn("http://x", "/a", nil, nil)
	if strings.Contains(empty, "Scenario:") || !strings.Contains(empty, "VERDICT: SKIP") {
		t.Errorf("no scenario: invite to SKIP, got:\n%s", empty)
	}
}
