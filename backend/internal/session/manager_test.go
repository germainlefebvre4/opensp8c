package session

import "strings"
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
