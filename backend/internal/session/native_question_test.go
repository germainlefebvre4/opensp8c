package session

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

// collectSessionMessages drains a session's message stream (live + final)
// until Done() closes, or fails the test after timeout.
func collectSessionMessages(t *testing.T, s *Session, timeout time.Duration) [][]byte {
	t.Helper()
	var got [][]byte
	deadline := time.After(timeout)
	for {
		select {
		case <-s.Notify():
			msgs, _ := s.MessagesSince(len(got))
			got = append(got, msgs...)
		case <-s.Done():
			msgs, _ := s.MessagesSince(len(got))
			got = append(got, msgs...)
			return got
		case <-deadline:
			t.Fatal("timed out waiting for fan-out to finish")
			return nil
		}
	}
}

// TestStartFanOutExtractsNativeAskUserQuestionBlock simulates a Claude
// stream-json flow containing a text block followed by a tool_use block
// named AskUserQuestion, split across content_block_start/delta/stop lines
// (with the input streamed as partial_json chunks, as Claude does). It
// verifies the tool_use block's raw lines are never forwarded as text, and
// that a single dedicated native_question event is emitted instead, once the
// block completes.
func TestStartFanOutExtractsNativeAskUserQuestionBlock(t *testing.T) {
	r, w := io.Pipe()
	proc := &Subprocess{stdout: r}
	s := &Session{
		proc:                     proc,
		messages:                 make([][]byte, 0),
		notify:                   make(chan struct{}, 10),
		done:                     make(chan struct{}),
		nativeQuestionModeActive: true,
	}
	m := &Manager{}
	m.startFanOut(s, "ws1/change", "ws1", false)

	lines := []string{
		`{"type":"content_block_start","index":0,"content_block":{"type":"text"}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Let me ask you something."}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_123","name":"AskUserQuestion","input":{}}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"questions\":[{\"question\":\"Quel est"}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":" le périmètre ?\",\"options\":[\"A\",\"B\"]}]}"}}`,
		`{"type":"content_block_stop","index":1}`,
	}
	go func() {
		for _, l := range lines {
			_, _ = w.Write([]byte(l + "\n"))
		}
		_ = w.Close()
	}()

	got := collectSessionMessages(t, s, 3*time.Second)

	if len(got) != 4 {
		t.Fatalf("expected 4 messages (3 raw text-block lines + 1 native_question event), got %d: %v", len(got), stringifyAll(got))
	}

	for i := 0; i < 3; i++ {
		if string(got[i]) != lines[i] {
			t.Errorf("expected text block line %d to pass through unchanged, got %q", i, got[i])
		}
	}

	var evt struct {
		Type      string `json:"type"`
		ToolUseID string `json:"tool_use_id"`
		Questions []struct {
			Question string   `json:"question"`
			Options  []string `json:"options"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(got[3], &evt); err != nil {
		t.Fatalf("failed to unmarshal native_question event: %v (data: %s)", err, got[3])
	}
	if evt.Type != "native_question" {
		t.Errorf("expected type native_question, got %q", evt.Type)
	}
	if evt.ToolUseID != "toolu_123" {
		t.Errorf("expected tool_use_id toolu_123, got %q", evt.ToolUseID)
	}
	if len(evt.Questions) != 1 || evt.Questions[0].Question != "Quel est le périmètre ?" {
		t.Errorf("unexpected questions: %+v", evt.Questions)
	}
	if len(evt.Questions) != 1 || len(evt.Questions[0].Options) != 2 {
		t.Errorf("expected 2 options, got: %+v", evt.Questions)
	}

	if got := s.PendingQuestion(); got != "Quel est le périmètre ?" {
		t.Errorf("expected pending question to reflect the native question, got %q", got)
	}
}

// TestStartFanOutExtractsNativeAskUserQuestionBlock_StreamEventWrapped is the
// same scenario as TestStartFanOutExtractsNativeAskUserQuestionBlock, but
// using the shape actually emitted by real, currently-installed Claude CLI
// versions: every incremental content_block_* event is wrapped as
// {"type":"stream_event","event":{...}}, not the flat shape assumed
// elsewhere. Caught via manual end-to-end testing against a live agent.
func TestStartFanOutExtractsNativeAskUserQuestionBlock_StreamEventWrapped(t *testing.T) {
	r, w := io.Pipe()
	proc := &Subprocess{stdout: r}
	s := &Session{
		proc:                     proc,
		messages:                 make([][]byte, 0),
		notify:                   make(chan struct{}, 10),
		done:                     make(chan struct{}),
		nativeQuestionModeActive: true,
	}
	m := &Manager{}
	m.startFanOut(s, "ws1/change", "ws1", false)

	lines := []string{
		`{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_789","name":"AskUserQuestion","input":{}}}}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"questions\":[{\"question\":\"Quel format ?\"}]}"}}}`,
		`{"type":"stream_event","event":{"type":"content_block_stop","index":1}}`,
	}
	go func() {
		for _, l := range lines {
			_, _ = w.Write([]byte(l + "\n"))
		}
		_ = w.Close()
	}()

	got := collectSessionMessages(t, s, 3*time.Second)

	if len(got) != 1 {
		t.Fatalf("expected only the native_question event (the wrapped tool_use lines fully suppressed), got %d: %v", len(got), stringifyAll(got))
	}

	var evt struct {
		Type      string `json:"type"`
		ToolUseID string `json:"tool_use_id"`
	}
	if err := json.Unmarshal(got[0], &evt); err != nil {
		t.Fatalf("failed to unmarshal native_question event: %v (data: %s)", err, got[0])
	}
	if evt.Type != "native_question" || evt.ToolUseID != "toolu_789" {
		t.Errorf("unexpected event: %+v", evt)
	}
}

// TestStartFanOutDropsAskUserQuestionBlockWhenNativeModeInactive verifies the
// silent fallback required by explore-native-question-mode: an AskUserQuestion
// tool_use block is fully dropped — never forwarded as raw text, and no
// native_question card emitted either — whenever native question mode isn't
// active for the session (non-Claude agent, or Claude with the preference
// off). This also covers a stray tool_use block having traversed the Gemini
// translation bridge unchanged (see TestTranslateGeminiLine's pass-through
// case for unrecognized event types in subprocess_test.go).
func TestStartFanOutDropsAskUserQuestionBlockWhenNativeModeInactive(t *testing.T) {
	r, w := io.Pipe()
	proc := &Subprocess{stdout: r}
	s := &Session{
		proc:     proc,
		messages: make([][]byte, 0),
		notify:   make(chan struct{}, 10),
		done:     make(chan struct{}),
		// nativeQuestionModeActive left false: non-Claude agent, or Claude with the preference off.
	}
	m := &Manager{}
	m.startFanOut(s, "ws1/change", "ws1", false)

	lines := []string{
		`{"type":"content_block_start","index":0,"content_block":{"type":"text"}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Let me ask you something."}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_123","name":"AskUserQuestion","input":{}}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"questions\":[{\"question\":\"Quel est le périmètre ?\"}]}"}}`,
		`{"type":"content_block_stop","index":1}`,
	}
	go func() {
		for _, l := range lines {
			_, _ = w.Write([]byte(l + "\n"))
		}
		_ = w.Close()
	}()

	got := collectSessionMessages(t, s, 3*time.Second)

	if len(got) != 3 {
		t.Fatalf("expected only the 3 text-block lines to pass through (AskUserQuestion block fully dropped), got %d: %v", len(got), stringifyAll(got))
	}
	for _, msg := range got {
		if strings.Contains(string(msg), "AskUserQuestion") || strings.Contains(string(msg), "native_question") {
			t.Errorf("expected no trace of the AskUserQuestion block in forwarded messages, got: %s", msg)
		}
	}
	if got := s.PendingQuestion(); got != "" {
		t.Errorf("expected no pending question to be recorded, got %q", got)
	}
}

func stringifyAll(msgs [][]byte) []string {
	out := make([]string, len(msgs))
	for i, m := range msgs {
		out[i] = string(m)
	}
	return out
}
