package session

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

type mockCloseWriter struct {
	bytes.Buffer
	closed bool
}

func (m *mockCloseWriter) Close() error {
	m.closed = true
	return nil
}

func TestAntigravityWriter(t *testing.T) {
	t.Run("First turn prefixes framing instructions and converts type to event", func(t *testing.T) {
		mockOut := &mockCloseWriter{}
		framing := "Explore the codebase carefully."
		w := newAntigravityWriter(mockOut, framing)

		inputPayload := `{"type":"user","message":{"role":"user","content":"Hello, please check the app."}}`
		n, err := w.Write([]byte(inputPayload))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		if n != len(inputPayload) {
			t.Fatalf("expected written len %d, got %d", len(inputPayload), n)
		}

		outBytes := mockOut.Bytes()
		var parsed struct {
			Type    string `json:"type"`
			Event   string `json:"event"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(outBytes, &parsed); err != nil {
			t.Fatalf("failed to parse output: %v", err)
		}

		if parsed.Type != "" {
			t.Errorf("expected type to be removed, got: %q", parsed.Type)
		}
		if parsed.Event != "user" {
			t.Errorf("expected event: user, got: %q", parsed.Event)
		}
		if parsed.Message.Role != "user" {
			t.Errorf("expected role: user, got: %q", parsed.Message.Role)
		}
		expectedPrefix := "[System Instructions:\nExplore the codebase carefully.]\n\nHello, please check the app."
		if parsed.Message.Content != expectedPrefix {
			t.Errorf("expected content:\n%q\ngot:\n%q", expectedPrefix, parsed.Message.Content)
		}

		// Second turn: should NOT have framing instructions
		mockOut.Reset()
		secondInput := `{"type":"user","message":{"role":"user","content":"Second question."}}`
		_, err = w.Write([]byte(secondInput))
		if err != nil {
			t.Fatalf("Second write failed: %v", err)
		}

		var parsedSecond struct {
			Event   string `json:"event"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(mockOut.Bytes(), &parsedSecond); err != nil {
			t.Fatalf("failed to parse second output: %v", err)
		}
		if parsedSecond.Event != "user" {
			t.Errorf("expected event: user, got: %q", parsedSecond.Event)
		}
		if parsedSecond.Message.Content != "Second question." {
			t.Errorf("expected content: 'Second question.', got: %q", parsedSecond.Message.Content)
		}
	})

	t.Run("Empty framing prompt does not inject prefix on first turn", func(t *testing.T) {
		mockOut := &mockCloseWriter{}
		w := newAntigravityWriter(mockOut, "")

		inputPayload := `{"type":"user","message":{"role":"user","content":"Run tests."}}`
		_, err := w.Write([]byte(inputPayload))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		var parsed struct {
			Event   string `json:"event"`
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(mockOut.Bytes(), &parsed); err != nil {
			t.Fatalf("failed to parse output: %v", err)
		}
		if parsed.Event != "user" {
			t.Errorf("expected event: user, got: %q", parsed.Event)
		}
		if parsed.Message.Content != "Run tests." {
			t.Errorf("expected content: 'Run tests.', got: %q", parsed.Message.Content)
		}
	})

	t.Run("Plain text input is wrapped into user event with framing on first turn", func(t *testing.T) {
		mockOut := &mockCloseWriter{}
		framing := "System instructions here."
		w := newAntigravityWriter(mockOut, framing)

		_, err := w.Write([]byte("plain text query\n"))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		var parsed struct {
			Event   string `json:"event"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(mockOut.Bytes(), &parsed); err != nil {
			t.Fatalf("failed to parse output: %v", err)
		}
		if parsed.Event != "user" {
			t.Errorf("expected event: user, got: %q", parsed.Event)
		}
		expectedContent := "[System Instructions:\nSystem instructions here.]\n\nplain text query"
		if parsed.Message.Content != expectedContent {
			t.Errorf("expected content:\n%q\ngot:\n%q", expectedContent, parsed.Message.Content)
		}
	})

	t.Run("Tool result array content is preserved and event converted", func(t *testing.T) {
		mockOut := &mockCloseWriter{}
		w := newAntigravityWriter(mockOut, "some framing")

		toolResultInput := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool-1","content":"done"}]}}`
		_, err := w.Write([]byte(toolResultInput))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(mockOut.Bytes(), &parsed); err != nil {
			t.Fatalf("failed to parse output: %v", err)
		}
		if parsed["event"] != "user" {
			t.Errorf("expected event: user, got: %v", parsed["event"])
		}
		if _, hasType := parsed["type"]; hasType {
			t.Errorf("expected type to be removed")
		}
		msg, ok := parsed["message"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected message object")
		}
		contentList, ok := msg["content"].([]interface{})
		if !ok || len(contentList) != 1 {
			t.Fatalf("expected content array with 1 element, got: %v", msg["content"])
		}
		block, ok := contentList[0].(map[string]interface{})
		if !ok || block["tool_use_id"] != "tool-1" {
			t.Errorf("expected tool_result block with tool-1, got: %v", contentList[0])
		}
	})

	t.Run("Close closes underlying writer", func(t *testing.T) {
		mockOut := &mockCloseWriter{}
		w := newAntigravityWriter(mockOut, "")
		if err := w.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
		if !mockOut.closed {
			t.Errorf("expected underlying writer to be closed")
		}
	})
}

func TestTranslateAntigravityLine(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectNil   bool
		checkOutput func(t *testing.T, out []byte)
	}{
		{
			name:      "init event is skipped",
			input:     `{"event":"init","conversation_id":"conv-abc","init":{"cwd":"/dir"}}`,
			expectNil: true,
		},
		{
			name:      "user_input step_update is skipped",
			input:     `{"event":"step_update","step_update":{"step_index":0,"state":"DONE","step_type":"user_input"}}`,
			expectNil: true,
		},
		{
			name:      "empty text_delta agent_response is skipped",
			input:     `{"event":"step_update","step_update":{"step_index":1,"state":"DONE","step_type":"agent_response","text_delta":""}}`,
			expectNil: true,
		},
		{
			name:  "agent_response with text_delta translates to content_block_delta",
			input: `{"event":"step_update","step_update":{"step_index":1,"state":"ACTIVE","step_type":"agent_response","text_delta":"Hello world!"}}`,
			checkOutput: func(t *testing.T, out []byte) {
				var data map[string]interface{}
				if err := json.Unmarshal(out, &data); err != nil {
					t.Fatalf("failed to parse: %v", err)
				}
				if data["type"] != "content_block_delta" {
					t.Errorf("expected type content_block_delta, got: %v", data["type"])
				}
				delta, ok := data["delta"].(map[string]interface{})
				if !ok || delta["text"] != "Hello world!" {
					t.Errorf("expected text 'Hello world!', got: %v", delta["text"])
				}
			},
		},
		{
			name:  "tool ACTIVE translates to content_block_start with tool_use and content array",
			input: `{"event":"step_update","step_update":{"step_index":2,"step_id":"step-42","state":"ACTIVE","step_type":"tool","tool_name":"run_command","tool_info":{"parameters":{"CommandLine":"echo 1"}}}}`,
			checkOutput: func(t *testing.T, out []byte) {
				var data map[string]interface{}
				if err := json.Unmarshal(out, &data); err != nil {
					t.Fatalf("failed to parse: %v", err)
				}
				if data["type"] != "content_block_start" {
					t.Errorf("expected type content_block_start, got: %v", data["type"])
				}
				cb, ok := data["content_block"].(map[string]interface{})
				if !ok || cb["type"] != "tool_use" {
					t.Fatalf("expected content_block.type == tool_use, got: %v", cb)
				}
				if cb["id"] != "step-42" {
					t.Errorf("expected id step-42, got: %v", cb["id"])
				}
				if cb["name"] != "run_command" {
					t.Errorf("expected name run_command, got: %v", cb["name"])
				}
				params, ok := cb["input"].(map[string]interface{})
				if !ok || params["CommandLine"] != "echo 1" {
					t.Errorf("expected params.CommandLine == 'echo 1', got: %v", cb["input"])
				}

				// Check content array for exploreChat.ts compatibility
				contentList, ok := data["content"].([]interface{})
				if !ok || len(contentList) != 1 {
					t.Fatalf("expected content array with 1 item, got: %v", data["content"])
				}
				item := contentList[0].(map[string]interface{})
				if item["type"] != "tool_use" || item["id"] != "step-42" {
					t.Errorf("expected content[0] to match tool_use, got: %v", item)
				}
			},
		},
		{
			name:  "tool DONE translates to content_block_start with tool_result and content array",
			input: `{"event":"step_update","step_update":{"step_index":2,"step_id":"step-42","state":"DONE","step_type":"tool","tool_name":"run_command","tool_info":{"output":"1\n"}}}`,
			checkOutput: func(t *testing.T, out []byte) {
				var data map[string]interface{}
				if err := json.Unmarshal(out, &data); err != nil {
					t.Fatalf("failed to parse: %v", err)
				}
				if data["type"] != "content_block_start" {
					t.Errorf("expected type content_block_start, got: %v", data["type"])
				}
				cb, ok := data["content_block"].(map[string]interface{})
				if !ok || cb["type"] != "tool_result" {
					t.Fatalf("expected content_block.type == tool_result, got: %v", cb)
				}
				if cb["tool_use_id"] != "step-42" {
					t.Errorf("expected tool_use_id step-42, got: %v", cb["tool_use_id"])
				}
				if cb["content"] != "1\n" {
					t.Errorf("expected content '1\\n', got: %v", cb["content"])
				}

				contentList, ok := data["content"].([]interface{})
				if !ok || len(contentList) != 1 {
					t.Fatalf("expected content array with 1 item, got: %v", data["content"])
				}
				item := contentList[0].(map[string]interface{})
				if item["type"] != "tool_result" || item["tool_use_id"] != "step-42" {
					t.Errorf("expected content[0] to match tool_result, got: %v", item)
				}
			},
		},
		{
			name:  "result event translates to message_complete",
			input: `{"event":"result","result":{"conversation_id":"conv-1","status":"SUCCESS","response":"Done"}}`,
			checkOutput: func(t *testing.T, out []byte) {
				var data map[string]interface{}
				if err := json.Unmarshal(out, &data); err != nil {
					t.Fatalf("failed to parse: %v", err)
				}
				if data["type"] != "message_complete" {
					t.Errorf("expected type message_complete, got: %v", data["type"])
				}
				if data["result"] != " " {
					t.Errorf("expected result ' ', got: %v", data["result"])
				}
			},
		},
		{
			name:  "non-JSON line is returned unchanged",
			input: `[DEBUG] Starting agent execution`,
			checkOutput: func(t *testing.T, out []byte) {
				if string(out) != `[DEBUG] Starting agent execution` {
					t.Errorf("expected plain text preserved, got: %s", string(out))
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := translateAntigravityLine([]byte(tc.input))
			if tc.expectNil {
				if got != nil {
					t.Errorf("expected nil, got: %s", string(got))
				}
				return
			}
			if got == nil {
				t.Fatalf("expected non-nil output")
			}
			if tc.checkOutput != nil {
				tc.checkOutput(t, got)
			}
		})
	}
}

func TestAntigravityStdoutReader(t *testing.T) {
	inputLines := []string{
		`{"event":"init","conversation_id":"conv-stream-123","init":{"cwd":"/dir"}}`,
		`{"event":"step_update","step_update":{"step_index":0,"state":"DONE","step_type":"user_input"}}`,
		`{"event":"step_update","step_update":{"step_index":1,"state":"ACTIVE","step_type":"agent_response","text_delta":"First token"}}`,
		`{"event":"step_update","step_update":{"step_index":2,"step_id":"step-tool","state":"ACTIVE","step_type":"tool","tool_name":"Read","tool_info":{"parameters":{"file_path":"/foo.txt"}}}}`,
		`{"event":"step_update","step_update":{"step_index":2,"step_id":"step-tool","state":"DONE","step_type":"tool","tool_name":"Read","tool_info":{"output":"content of foo"}}}`,
		`{"event":"step_update","step_update":{"step_index":3,"state":"ACTIVE","step_type":"agent_response","text_delta":"Second token"}}`,
		`{"event":"result","result":{"conversation_id":"conv-stream-123","status":"SUCCESS"}}`,
	}

	inputData := strings.Join(inputLines, "\n") + "\n"
	reader := newAntigravityStdoutReader(io.NopCloser(bytes.NewReader([]byte(inputData))))

	scanner := bufio.NewScanner(reader)
	var outputLines []string
	for scanner.Scan() {
		outputLines = append(outputLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}

	// Verify conversation ID was captured from the init event
	if reader.ConversationID() != "conv-stream-123" {
		t.Errorf("expected conversation ID 'conv-stream-123', got: %q", reader.ConversationID())
	}

	// We expect 5 output lines (init and user_input omitted):
	// 1. content_block_delta ("First token")
	// 2. content_block_start tool_use ("Read")
	// 3. content_block_start tool_result ("content of foo")
	// 4. content_block_delta ("Second token")
	// 5. message_complete
	if len(outputLines) != 5 {
		t.Fatalf("expected 5 output lines, got %d:\n%s", len(outputLines), strings.Join(outputLines, "\n"))
	}

	// Line 1: delta
	var line1 map[string]interface{}
	json.Unmarshal([]byte(outputLines[0]), &line1)
	if line1["type"] != "content_block_delta" {
		t.Errorf("line 1 expected content_block_delta, got: %v", line1["type"])
	}

	// Line 2: tool_use
	var line2 map[string]interface{}
	json.Unmarshal([]byte(outputLines[1]), &line2)
	cb2, _ := line2["content_block"].(map[string]interface{})
	if cb2["type"] != "tool_use" || cb2["name"] != "Read" {
		t.Errorf("line 2 expected tool_use Read, got: %v", cb2)
	}

	// Line 3: tool_result
	var line3 map[string]interface{}
	json.Unmarshal([]byte(outputLines[2]), &line3)
	cb3, _ := line3["content_block"].(map[string]interface{})
	if cb3["type"] != "tool_result" || cb3["content"] != "content of foo" {
		t.Errorf("line 3 expected tool_result, got: %v", cb3)
	}

	// Line 4: delta
	var line4 map[string]interface{}
	json.Unmarshal([]byte(outputLines[3]), &line4)
	if line4["type"] != "content_block_delta" {
		t.Errorf("line 4 expected content_block_delta, got: %v", line4["type"])
	}

	// Line 5: message_complete
	var line5 map[string]interface{}
	json.Unmarshal([]byte(outputLines[4]), &line5)
	if line5["type"] != "message_complete" {
		t.Errorf("line 5 expected message_complete, got: %v", line5["type"])
	}
}

