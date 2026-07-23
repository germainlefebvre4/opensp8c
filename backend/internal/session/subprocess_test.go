package session

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestTranslateGeminiLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // Expected output (empty means nil / skipped)
	}{
		{
			name:     "Non-JSON line",
			input:    "[ERROR] Some random stderr or log",
			expected: "[ERROR] Some random stderr or log",
		},
		{
			name:     "Invalid JSON block",
			input:    "{invalid json}",
			expected: "{invalid json}",
		},
		{
			name:     "Init event (should be skipped)",
			input:    `{"type":"init","session_id":"123","model":"auto"}`,
			expected: "",
		},
		{
			name:     "Result event (should be skipped)",
			input:    `{"type":"result","status":"success"}`,
			expected: "",
		},
		{
			name:     "User message (should be skipped)",
			input:    `{"type":"message","role":"user","content":"hello"}`,
			expected: "",
		},
		{
			name:     "Assistant message (should be translated)",
			input:    `{"type":"message","role":"assistant","content":"hello world"}`,
			expected: `{"delta":{"text":"hello world"},"type":"content_block_delta"}`,
		},
		{
			name:     "Assistant message with JSON ghost_named marker (should be translated intact)",
			input:    `{"type":"message","role":"assistant","content":"{\"event\":\"ghost_named\",\"name\":\"my-change\"}\n"}`,
			expected: `{"delta":{"text":"{\"event\":\"ghost_named\",\"name\":\"my-change\"}\n"},"type":"content_block_delta"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotBytes := translateGeminiLine([]byte(tc.input))
			if tc.expected == "" {
				if gotBytes != nil {
					t.Fatalf("expected nil (skipped), got: %s", string(gotBytes))
				}
				return
			}
			if gotBytes == nil {
				t.Fatalf("expected: %s, got: nil", tc.expected)
			}

			// Parse both as interface{} to compare structurally (ignoring key order)
			var gotObj, expObj interface{}
			if err := json.Unmarshal(gotBytes, &gotObj); err != nil {
				// If not valid JSON, we compare literally
				if string(gotBytes) != tc.expected {
					t.Errorf("expected: %q, got: %q", tc.expected, string(gotBytes))
				}
				return
			}
			if err := json.Unmarshal([]byte(tc.expected), &expObj); err != nil {
				t.Fatalf("failed to unmarshal expected JSON %q: %v", tc.expected, err)
			}

			gotStr, _ := json.Marshal(gotObj)
			expStr, _ := json.Marshal(expObj)
			if string(gotStr) != string(expStr) {
				t.Errorf("expected JSON structure: %s, got: %s", string(expStr), string(gotStr))
			}
		})
	}
}

func TestGeminiStdoutReader(t *testing.T) {
	inputLines := []string{
		`{"type":"init","session_id":"123"}`,
		`[ERROR] Failed to connect to IDE companion extension.`,
		`{"type":"message","role":"user","content":"hi"}`,
		`{"type":"message","role":"assistant","content":"Hello! How"}`,
		`{"type":"message","role":"assistant","content":" can I help?"}`,
		`{"type":"result","status":"success"}`,
	}

	rawInput := strings.Join(inputLines, "\n") + "\n"
	closer := io.NopCloser(strings.NewReader(rawInput))

	reader := newGeminiStdoutReader(closer)
	defer reader.Close()

	var outBuf bytes.Buffer
	_, err := io.Copy(&outBuf, reader)
	if err != nil && err != io.EOF {
		t.Fatalf("io.Copy failed: %v", err)
	}

	output := outBuf.String()
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")

	expectedLines := []string{
		`[ERROR] Failed to connect to IDE companion extension.`,
		`{"delta":{"text":"Hello! How"},"type":"content_block_delta"}`,
		`{"delta":{"text":" can I help?"},"type":"content_block_delta"}`,
	}

	if len(lines) != len(expectedLines) {
		t.Fatalf("expected %d lines, got %d. Output was:\n%s", len(expectedLines), len(lines), output)
	}

	for i, expected := range expectedLines {
		got := lines[i]
		if strings.HasPrefix(expected, "{") {
			var gotObj, expObj interface{}
			if err := json.Unmarshal([]byte(got), &gotObj); err != nil {
				t.Errorf("line %d is not valid JSON: %s", i, got)
				continue
			}
			_ = json.Unmarshal([]byte(expected), &expObj)
			gotStr, _ := json.Marshal(gotObj)
			expStr, _ := json.Marshal(expObj)
			if string(gotStr) != string(expStr) {
				t.Errorf("line %d mismatch:\nexpected: %s\ngot:      %s", i, string(expStr), string(gotStr))
			}
		} else {
			if got != expected {
				t.Errorf("line %d mismatch:\nexpected: %q\ngot:      %q", i, expected, got)
			}
		}
	}
}

type mockWriteCloser struct {
	bytes.Buffer
	closed bool
}

func (m *mockWriteCloser) Close() error {
	m.closed = true
	return nil
}

func TestSubprocessWrite(t *testing.T) {
	t.Run("Gemini agent with user message JSON input", func(t *testing.T) {
		mockIn := &mockWriteCloser{}
		proc := &Subprocess{
			stdin:   mockIn,
			agentID: "gemini",
		}

		inputPayload := `{"type":"user","message":{"role":"user","content":"/opsx:explore change-name"}}`
		n, err := proc.Write([]byte(inputPayload))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		if n == 0 {
			t.Fatal("expected non-zero bytes written")
		}

		expected := "/opsx:explore change-name\n"
		got := mockIn.String()
		if got != expected {
			t.Errorf("expected: %q, got: %q", expected, got)
		}
	})

	t.Run("Gemini agent with raw/plain text input", func(t *testing.T) {
		mockIn := &mockWriteCloser{}
		proc := &Subprocess{
			stdin:   mockIn,
			agentID: "gemini",
		}

		inputPayload := "plain text prompt"
		_, err := proc.Write([]byte(inputPayload))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		expected := "plain text prompt"
		got := mockIn.String()
		if got != expected {
			t.Errorf("expected: %q, got: %q", expected, got)
		}
	})

	t.Run("Claude agent with user message JSON input (pass through as-is)", func(t *testing.T) {
		mockIn := &mockWriteCloser{}
		proc := &Subprocess{
			stdin:   mockIn,
			agentID: "claude",
		}

		inputPayload := `{"type":"user","message":{"role":"user","content":"/opsx:explore change-name"}}`
		_, err := proc.Write([]byte(inputPayload))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		expected := inputPayload
		got := mockIn.String()
		if got != expected {
			t.Errorf("expected: %q, got: %q", expected, got)
		}
	})
}
