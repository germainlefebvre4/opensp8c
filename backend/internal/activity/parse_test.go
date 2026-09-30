package activity

import (
	"strings"
	"testing"
)

func TestParseConversationLines_RepresentativeFixture(t *testing.T) {
	lines := [][]byte{
		// 1. Text deltas (consecutive)
		[]byte(`{"ts":"2026-09-24T10:00:00.000Z","dir":"out","data":{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"I will "}}}}`),
		[]byte(`{"ts":"2026-09-24T10:00:00.200Z","dir":"out","data":{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"inspect the codebase."}}}}`),

		// 2. tool_use paired with tool_result
		[]byte(`{"ts":"2026-09-24T10:00:01.000Z","dir":"out","data":{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"tool_1","name":"Read","input":{"file_path":"backend/main.go"}}}}}`),
		[]byte(`{"ts":"2026-09-24T10:00:03.500Z","dir":"out","data":{"timestamp":"2026-09-24T10:00:03.500Z","type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool_1","content":"package main..."}]}}}`),

		// 3. tool_use without tool_result (interrupted run)
		[]byte(`{"ts":"2026-09-24T10:00:04.000Z","dir":"out","data":{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tool_unpaired","name":"Bash","input":{"command":"go test ./..."}}]}}}`),

		// 4. Another narration block via flat content_block_delta
		[]byte(`{"ts":"2026-09-24T10:00:05.000Z","dir":"out","data":{"type":"content_block_delta","delta":{"text":"All tests "}}}`),
		[]byte(`{"ts":"2026-09-24T10:00:05.100Z","dir":"out","data":{"type":"content_block_delta","delta":{"text":"are passing."}}}`),
		[]byte(`{"ts":"2026-09-24T10:00:05.200Z","dir":"out","data":{"type":"message_complete"}}`),
	}

	entries, err := ParseConversationLines(lines)
	if err != nil {
		t.Fatalf("unexpected error parsing lines: %v", err)
	}

	// We expect 4 entries:
	// 1: Narration: "I will inspect the codebase."
	// 2: Tool: Read backend/main.go (duration 2500ms)
	// 3: Tool: Bash go test ./... (duration nil)
	// 4: Narration: "All tests are passing."
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d: %+v", len(entries), entries)
	}

	// Check 1: Narration
	if entries[0].Type != "agent" || entries[0].Category != "agent" {
		t.Errorf("entry 0: expected type=agent, category=agent, got type=%s, cat=%s", entries[0].Type, entries[0].Category)
	}
	if entries[0].Summary != "I will inspect the codebase." {
		t.Errorf("entry 0: expected summary 'I will inspect the codebase.', got %q", entries[0].Summary)
	}
	if entries[0].DurationMs != nil {
		t.Errorf("entry 0: expected duration nil, got %v", *entries[0].DurationMs)
	}

	// Check 2: Paired tool_use with duration
	if entries[1].Type != "Read" || entries[1].Category != "tool" {
		t.Errorf("entry 1: expected type=Read, category=tool, got type=%s, cat=%s", entries[1].Type, entries[1].Category)
	}
	if entries[1].Summary != "Read backend/main.go" {
		t.Errorf("entry 1: expected summary 'Read backend/main.go', got %q", entries[1].Summary)
	}
	if entries[1].DurationMs == nil {
		t.Fatalf("entry 1: expected duration non-nil")
	}
	if *entries[1].DurationMs != 2500 {
		t.Errorf("entry 1: expected duration 2500ms, got %dms", *entries[1].DurationMs)
	}

	// Check 3: Unpaired tool_use (run interrupted)
	if entries[2].Type != "Bash" || entries[2].Category != "tool" {
		t.Errorf("entry 2: expected type=Bash, category=tool, got type=%s, cat=%s", entries[2].Type, entries[2].Category)
	}
	if entries[2].Summary != "Bash go test ./..." {
		t.Errorf("entry 2: expected summary 'Bash go test ./...', got %q", entries[2].Summary)
	}
	if entries[2].DurationMs != nil {
		t.Errorf("entry 2: expected duration nil for unpaired tool, got %v", *entries[2].DurationMs)
	}

	// Check 4: Narration assembled from flat deltas
	if entries[3].Type != "agent" || entries[3].Summary != "All tests are passing." {
		t.Errorf("entry 3: expected agent narration 'All tests are passing.', got %+v", entries[3])
	}
	if entries[3].DurationMs != nil {
		t.Errorf("entry 3: expected duration nil, got %v", *entries[3].DurationMs)
	}
}

func TestParseConversationLines_AssistantTurnWithFullContent(t *testing.T) {
	lines := [][]byte{
		[]byte(`{"ts":"2026-09-24T11:00:00.000Z","dir":"out","data":{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Direct answer without streaming deltas"}]}}}`),
	}

	entries, err := ParseConversationLines(lines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Summary != "Direct answer without streaming deltas" {
		t.Errorf("unexpected summary: %q", entries[0].Summary)
	}
}

func TestParseConversationLines_ToolResultKeptAndTruncated(t *testing.T) {
	long := strings.Repeat("x", 5000)
	lines := [][]byte{
		[]byte(`{"ts":"2026-09-24T10:00:01.000Z","dir":"out","data":{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"ls"}}]}}}`),
		[]byte(`{"ts":"2026-09-24T10:00:02.000Z","dir":"out","data":{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":"` + long + `"}]}}}`),
		[]byte(`{"ts":"2026-09-24T10:00:03.000Z","dir":"out","data":{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t2","name":"Read","input":{"file_path":"a.go"}}]}}}`),
		[]byte(`{"ts":"2026-09-24T10:00:04.000Z","dir":"out","data":{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t2","content":[{"type":"text","text":"hello"}]}]}}}`),
	}
	entries, _ := ParseConversationLines(lines)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	r1, _ := entries[0].Meta["result"].(string)
	if len(r1) < maxToolResultLen || len(r1) > maxToolResultLen+4 {
		t.Errorf("expected truncated result ~%d bytes, got %d", maxToolResultLen, len(r1))
	}
	if r2, _ := entries[1].Meta["result"].(string); r2 != "hello" {
		t.Errorf("expected block-array result 'hello', got %q", r2)
	}
}

func TestParseConversationLines_IgnoresMetaMarkers(t *testing.T) {
	lines := [][]byte{
		[]byte(`{"ts":"2026-09-24T10:00:00Z","dir":"meta","data":{"type":"pool_run_start","worker_id":1}}`),
		[]byte(`{"ts":"2026-09-24T10:00:01Z","dir":"out","data":{"type":"content_block_delta","delta":{"text":"hi"}}}`),
		[]byte(`{"ts":"2026-09-24T10:00:02Z","dir":"meta","data":{"type":"pool_run_end","outcome":"completed"}}`),
	}
	entries, _ := ParseConversationLines(lines)
	if len(entries) != 1 || entries[0].Summary != "hi" {
		t.Fatalf("expected only the narration entry, got %+v", entries)
	}
}
