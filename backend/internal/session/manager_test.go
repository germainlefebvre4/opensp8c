package session

import (
	"fmt"
	"strings"
)
import "testing"

func TestSystemPromptsIncludeFramingAndGhostQuestionMarker(t *testing.T) {
	for name, prompt := range map[string]string{
		"baseSystemPrompt": baseSystemPrompt,
		"anonSystemPrompt": anonSystemPrompt,
	} {
		if !strings.Contains(prompt, "ghost_question") {
			t.Errorf("%s: expected ghost_question marker convention to be present", name)
		}
		if !strings.Contains(prompt, "first response") && !strings.Contains(prompt, "first turn") {
			t.Errorf("%s: expected first-turn framing instruction to be present", name)
		}
	}
}

func TestExtractGhostQuestion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Untranslated top-level JSON",
			input:    `{"event":"ghost_question","question":"Quel est le périmètre exact ?"}`,
			expected: "Quel est le périmètre exact ?",
		},
		{
			name:     "Translated Gemini content_block_delta with JSON text",
			input:    `{"type":"content_block_delta","delta":{"text":"{\"event\":\"ghost_question\",\"question\":\"Quelle stack utiliser ?\"}\n"}}`,
			expected: "Quelle stack utiliser ?",
		},
		{
			name:     "Translated Gemini content_block_delta with raw text",
			input:    `{"type":"content_block_delta","delta":{"text":"some text containing \"event\":\"ghost_question\" and \"question\":\"raw text question\""}}`,
			expected: "raw text question",
		},
		{
			name:     "Fallback with escaped quotes",
			input:    `{\"event\":\"ghost_question\",\"question\":\"escaped fallback question\"}`,
			expected: "escaped fallback question",
		},
		{
			name:     "No match",
			input:    `{"type":"content_block_delta","delta":{"text":"hello world"}}`,
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractGhostQuestion([]byte(tc.input))
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestExtractAllGhostQuestions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single untranslated top-level JSON",
			input:    `{"event":"ghost_question","question":"Quel est le périmètre exact ?"}`,
			expected: []string{"Quel est le périmètre exact ?"},
		},
		{
			name:     "No match",
			input:    `{"type":"content_block_delta","delta":{"text":"hello world"}}`,
			expected: nil,
		},
		{
			name: "Two markers concatenated in the same delta text (multi-line consolidated turn)",
			input: `{"type":"content_block_delta","delta":{"text":"` +
				`{\"event\":\"ghost_question\",\"question\":\"Quelle stack utiliser ?\"}\n` +
				`{\"event\":\"ghost_question\",\"question\":\"Quel est le budget ?\"}\n` +
				`"}}`,
			expected: []string{"Quelle stack utiliser ?", "Quel est le budget ?"},
		},
		{
			name: "Two markers embedded loosely in surrounding conversational text",
			input: `{"type":"content_block_delta","delta":{"text":"J'ai deux questions. ` +
				`\"event\":\"ghost_question\" \"question\":\"Premiere question ?\" et ensuite ` +
				`\"event\":\"ghost_question\" \"question\":\"Deuxieme question ?\""}}`,
			expected: []string{"Premiere question ?", "Deuxieme question ?"},
		},
		{
			name: "Two markers in a raw JSONL blob spanning multiple lines",
			input: "{\"event\":\"ghost_question\",\"question\":\"Q1 ?\"}\n" +
				"{\"event\":\"ghost_question\",\"question\":\"Q2 ?\"}",
			expected: []string{"Q1 ?", "Q2 ?"},
		},
		{
			name:     "Fallback with escaped quotes (malformed top-level JSON)",
			input:    `{\"event\":\"ghost_question\",\"question\":\"escaped fallback question\"}`,
			expected: []string{"escaped fallback question"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractAllGhostQuestions([]byte(tc.input))
			if len(got) != len(tc.expected) {
				t.Fatalf("expected %v, got %v", tc.expected, got)
			}
			for i := range got {
				if got[i] != tc.expected[i] {
					t.Errorf("index %d: expected %q, got %q", i, tc.expected[i], got[i])
				}
			}
		})
	}
}

func TestReconstructPendingQuestion(t *testing.T) {
	t.Run("marker question left unanswered at end of buffer", func(t *testing.T) {
		lines := []LogLine{
			{Dir: "out", Data: []byte(`{"type":"content_block_delta","delta":{"text":"Let's frame this."}}`)},
			{Dir: "out", Data: []byte(`{"event":"ghost_question","question":"Quel est le périmètre ?"}`)},
		}
		got := ReconstructPendingQuestion(lines)
		if got != "Quel est le périmètre ?" {
			t.Errorf("expected pending question, got %q", got)
		}
	})

	t.Run("marker question answered by a later user message", func(t *testing.T) {
		lines := []LogLine{
			{Dir: "out", Data: []byte(`{"event":"ghost_question","question":"Quel est le périmètre ?"}`)},
			{Dir: "in", Data: []byte(`{"type":"user","message":{"role":"user","content":"Le périmètre X"}}`)},
		}
		got := ReconstructPendingQuestion(lines)
		if got != "" {
			t.Errorf("expected no pending question after a user reply, got %q", got)
		}
	})

	t.Run("native AskUserQuestion block left unanswered, split across lines", func(t *testing.T) {
		lines := []LogLine{
			{Dir: "out", Data: []byte(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"AskUserQuestion"}}`)},
			{Dir: "out", Data: []byte(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"questions\":[{\"question\":\"Quel format ?\"}]}"}}`)},
			{Dir: "out", Data: []byte(`{"type":"content_block_stop","index":0}`)},
		}
		got := ReconstructPendingQuestion(lines)
		if got != "Quel format ?" {
			t.Errorf("expected pending native question, got %q", got)
		}
	})

	t.Run("later question supersedes an earlier answered one", func(t *testing.T) {
		lines := []LogLine{
			{Dir: "out", Data: []byte(`{"event":"ghost_question","question":"Premiere question ?"}`)},
			{Dir: "in", Data: []byte(`{"type":"user","message":{"role":"user","content":"Reponse"}}`)},
			{Dir: "out", Data: []byte(`{"event":"ghost_question","question":"Deuxieme question ?"}`)},
		}
		got := ReconstructPendingQuestion(lines)
		if got != "Deuxieme question ?" {
			t.Errorf("expected the later unanswered question, got %q", got)
		}
	})

	t.Run("empty buffer has no pending question", func(t *testing.T) {
		if got := ReconstructPendingQuestion(nil); got != "" {
			t.Errorf("expected no pending question for an empty buffer, got %q", got)
		}
	})
}

func TestParseLogLines(t *testing.T) {
	raw := [][]byte{
		[]byte(`{"ts":"2026-01-01T00:00:00Z","dir":"out","data":{"event":"ghost_question","question":"Q?"}}`),
		[]byte(`{"ts":"2026-01-01T00:00:01Z","dir":"in","data":{"type":"user","message":{"role":"user","content":"A"}}}`),
		[]byte(`not valid json`),
	}
	lines := ParseLogLines(raw)
	if len(lines) != 2 {
		t.Fatalf("expected 2 valid lines (invalid one skipped), got %d", len(lines))
	}
	if lines[0].Dir != "out" || !strings.Contains(string(lines[0].Data), "ghost_question") {
		t.Errorf("unexpected first line: %+v", lines[0])
	}
	if lines[1].Dir != "in" {
		t.Errorf("unexpected second line: %+v", lines[1])
	}
}

func TestSessionPendingQuestion(t *testing.T) {
	s := &Session{
		messages: make([][]byte, 0),
		notify:   make(chan struct{}, 10),
	}

	if got := s.PendingQuestion(); got != "" {
		t.Fatalf("expected no pending question initially, got %q", got)
	}

	s.SetPendingQuestion("Quel est le périmètre ?")
	if got := s.PendingQuestion(); got != "Quel est le périmètre ?" {
		t.Fatalf("expected pending question to be set, got %q", got)
	}

	s.ClearPendingQuestion()
	if got := s.PendingQuestion(); got != "" {
		t.Fatalf("expected pending question to be cleared, got %q", got)
	}
}

func TestExtractGhostMarkers_AntigravityTranslation(t *testing.T) {
	t.Run("ExtractGhostNamed from translated Antigravity step_update", func(t *testing.T) {
		rawAgyLine := `{"event":"step_update","step_update":{"step_index":2,"state":"ACTIVE","step_type":"agent_response","text_delta":"{\"event\":\"ghost_named\",\"name\":\"add-antigravity-support\"}\n"}}`
		translated := translateAntigravityLine([]byte(rawAgyLine))
		if translated == nil {
			t.Fatalf("expected non-nil translated output")
		}

		gotName := ExtractGhostNamed(translated)
		if gotName != "add-antigravity-support" {
			t.Errorf("expected 'add-antigravity-support', got %q", gotName)
		}
	})

	t.Run("ExtractGhostQuestion from translated Antigravity step_update", func(t *testing.T) {
		rawAgyLine := `{"event":"step_update","step_update":{"step_index":3,"state":"ACTIVE","step_type":"agent_response","text_delta":"{\"event\":\"ghost_question\",\"question\":\"Quelle base de données préférez-vous ?\"}\n"}}`
		translated := translateAntigravityLine([]byte(rawAgyLine))
		if translated == nil {
			t.Fatalf("expected non-nil translated output")
		}

		gotQuestion := ExtractGhostQuestion(translated)
		if gotQuestion != "Quelle base de données préférez-vous ?" {
			t.Errorf("expected 'Quelle base de données préférez-vous ?', got %q", gotQuestion)
		}
	})

	t.Run("ExtractGhostQuestion embedded in conversational text delta", func(t *testing.T) {
		rawAgyLine := `{"event":"step_update","step_update":{"step_index":4,"state":"ACTIVE","step_type":"agent_response","text_delta":"J'ai analysé le projet.\n{\"event\":\"ghost_question\",\"question\":\"Faut-il supporter PostgreSQL ?\"}\nMerci de préciser."}}`
		translated := translateAntigravityLine([]byte(rawAgyLine))
		if translated == nil {
			t.Fatalf("expected non-nil translated output")
		}

		gotQuestion := ExtractGhostQuestion(translated)
		if gotQuestion != "Faut-il supporter PostgreSQL ?" {
			t.Errorf("expected 'Faut-il supporter PostgreSQL ?', got %q", gotQuestion)
		}
	})
}

func msgN(i int) []byte { return []byte(fmt.Sprintf("m%d", i)) }

func TestMessagesSinceDeliversAfterSaturation(t *testing.T) {
	s := NewTestSession(nil)
	_, cursor := s.Snapshot()
	total := maxMessages*2 + 37
	for i := 0; i < total; i++ {
		s.appendMessage(msgN(i))
		var got [][]byte
		got, cursor = s.MessagesSince(cursor)
		if len(got) != 1 || string(got[0]) != string(msgN(i)) {
			t.Fatalf("step %d: got %q, want exactly %q", i, got, msgN(i))
		}
	}
	if cursor != total {
		t.Fatalf("cursor = %d, want %d", cursor, total)
	}
}

func TestMessagesSinceStaleCursor(t *testing.T) {
	s := NewTestSession(nil)
	for i := 0; i < maxMessages+10; i++ {
		s.appendMessage(msgN(i))
	}
	got, cursor := s.MessagesSince(3) // entries 0..9 evicted
	if len(got) != maxMessages || string(got[0]) != string(msgN(10)) {
		t.Fatalf("got %d msgs starting %q, want %d starting %q", len(got), got[0], maxMessages, msgN(10))
	}
	if cursor != maxMessages+10 {
		t.Fatalf("cursor = %d, want %d", cursor, maxMessages+10)
	}
	got, _ = s.MessagesSince(cursor + 50) // future cursor clamps
	if len(got) != 0 {
		t.Fatalf("future cursor returned %d msgs", len(got))
	}
}

func TestSnapshotThenMessagesSinceSaturated(t *testing.T) {
	s := NewTestSession(nil)
	for i := 0; i < maxMessages+25; i++ {
		s.appendMessage(msgN(i))
	}
	snap, cursor := s.Snapshot()
	if len(snap) != maxMessages {
		t.Fatalf("snapshot len = %d", len(snap))
	}
	if got, _ := s.MessagesSince(cursor); len(got) != 0 {
		t.Fatalf("expected no duplicates after snapshot, got %d", len(got))
	}
	s.appendMessage(msgN(9999))
	got, _ := s.MessagesSince(cursor)
	if len(got) != 1 || string(got[0]) != string(msgN(9999)) {
		t.Fatalf("got %q, want only the new message", got)
	}
}

func TestInjectMessageOnSaturatedBuffer(t *testing.T) {
	s := NewTestSession(nil)
	for i := 0; i < maxMessages; i++ {
		s.appendMessage(msgN(i))
	}
	_, cursor := s.Snapshot()
	s.InjectMessage([]byte("injected"))
	got, cursor := s.MessagesSince(cursor)
	if len(got) != 1 || string(got[0]) != "injected" {
		t.Fatalf("got %q, want injected", got)
	}
	if got, _ = s.MessagesSince(cursor); len(got) != 0 {
		t.Fatalf("duplicate delivery: %q", got)
	}
}
