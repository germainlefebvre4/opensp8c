package session

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glefebvre/opensp8c/internal/agents"
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
			name:     "Result event (should translate to message_complete)",
			input:    `{"type":"result","status":"success"}`,
			expected: `{"type":"message_complete","result":" "}`,
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
		`{"type":"message_complete","result":" "}`,
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

		expected := `{"type":"user","message":{"role":"user","content":"/opsx:explore change-name"}}` + "\n"
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

		expected := "plain text prompt\n"
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

func TestStartSubprocessGeminiBridge(t *testing.T) {
	// Create a mock executable script that acts like a mock gemini CLI
	// but ignores all command line flags (like --output-format) and just cats stdin to stdout.
	tmpDir := t.TempDir()
	mockCLIPath := filepath.Join(tmpDir, "mock-gemini")
	mockCLIScript := "#!/bin/sh\ncat\n"
	err := os.WriteFile(mockCLIPath, []byte(mockCLIScript), 0755)
	if err != nil {
		t.Fatalf("failed to write mock CLI: %v", err)
	}

	agentCfg := agents.AgentConfig{
		ID:    "gemini",
		Label: "Gemini",
		CLI:   mockCLIPath,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proc, err := StartSubprocess(ctx, tmpDir, agentCfg, "system prompt", "test-session-456", false, nil)
	if err != nil {
		t.Fatalf("StartSubprocess failed: %v", err)
	}

	// We write a mock prompt line.
	// Since we are mocking gemini, whatever prompt content is written to mock CLI's stdin
	// will be output to its stdout as-is.
	// Note: newGeminiStdoutReader (used internally) expects JSON line stream, and will skip non-JSON.
	// So we write a mocked stream-json Gemini assistant response to mock prompt!
	mockResponseJSON := `{"type":"message","role":"assistant","content":"hello from mock gemini"}`
	
	// We send this as the user content, and since our mock CLI echoes it, geminiStdoutReader will parse it,
	// translate it to content_block_delta, and send it to stdout.
	userMsg := `{"type":"user","message":{"role":"user","content":"` + strings.ReplaceAll(mockResponseJSON, `"`, `\"`) + `"}}`

	_, err = proc.Write([]byte(userMsg))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read and verify translated stdout
	buf := make([]byte, 4096)
	n, err := proc.Stdout().Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read failed: %v", err)
	}

	got := string(buf[:n])
	expected := `{"delta":{"text":"hello from mock gemini"},"type":"content_block_delta"}` + "\n"
	if got != expected {
		t.Errorf("expected: %q, got: %q", expected, got)
	}

	_ = proc.CloseStdin()
	_ = proc.Wait()
}
