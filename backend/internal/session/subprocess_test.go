package session

import (
	"bufio"
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

func TestStartSubprocessGeminiBridge_StderrErrors(t *testing.T) {
	// Create a mock executable script that writes an error to stderr and exits
	tmpDir := t.TempDir()
	mockCLIPath := filepath.Join(tmpDir, "mock-gemini-err")
	mockCLIScript := "#!/bin/sh\necho 'ProjectIdRequiredError: This account requires setting GOOGLE_CLOUD_PROJECT' >&2\ncat\n"
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

	// We trigger a turn by writing to the subprocess.
	// This will start the cmd and run our mock script, which will write the error to stderr.
	userMsg := `{"type":"user","message":{"role":"user","content":"hello"}}`
	_, err = proc.Write([]byte(userMsg))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Close stdin immediately so the virtual runner scanner knows there are no more turns.
	_ = proc.CloseStdin()

	// Read all output until EOF. This unblocks any concurrent writes to virtualStdoutWriter.
	var outBuf bytes.Buffer
	_, err = io.Copy(&outBuf, proc.Stdout())
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	got := outBuf.String()
	expectedSub := `"type":"session_warning"`
	if !strings.Contains(got, expectedSub) {
		t.Errorf("expected output to contain %q, got: %q", expectedSub, got)
	}

	expectedText := "Erreur d'authentification Google Cloud"
	if !strings.Contains(got, expectedText) {
		t.Errorf("expected output to contain %q, got: %q", expectedText, got)
	}

	expectedFatal := `"fatal":true`
	if !strings.Contains(got, expectedFatal) {
		t.Errorf("expected output to contain %q, got: %q", expectedFatal, got)
	}

	_ = proc.Wait()
}

func TestStartSubprocessGeminiBridge_ThrottledIDEWarning(t *testing.T) {
	// Create a mock executable script that writes the companion connection error to stderr and exits
	tmpDir := t.TempDir()
	mockCLIPath := filepath.Join(tmpDir, "mock-gemini-ide-err")
	mockCLIScript := "#!/bin/sh\necho 'Failed to connect to IDE companion extension' >&2\necho '{\"type\":\"message\",\"role\":\"assistant\",\"content\":\"response chunk\"}'\n"
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

	proc, err := StartSubprocess(ctx, tmpDir, agentCfg, "system prompt", "test-session-789", false, nil)
	if err != nil {
		t.Fatalf("StartSubprocess failed: %v", err)
	}

	scanner := bufio.NewScanner(proc.Stdout())

	// === FIRST TURN ===
	userMsg1 := `{"type":"user","message":{"role":"user","content":"hello 1"}}`
	_, err = proc.Write([]byte(userMsg1))
	if err != nil {
		t.Fatalf("First Write failed: %v", err)
	}

	// First turn expects 2 lines: session_warning and content_block_delta, in any order due to concurrency
	if !scanner.Scan() {
		t.Fatalf("First scan failed")
	}
	line1 := scanner.Text()

	if !scanner.Scan() {
		t.Fatalf("Second scan failed")
	}
	line2 := scanner.Text()

	hasWarning := false
	hasWarningNonFatal := false
	hasDelta := false

	checkLine := func(l string) {
		if strings.Contains(l, `"type":"session_warning"`) {
			hasWarning = true
			if strings.Contains(l, `"fatal":false`) {
				hasWarningNonFatal = true
			}
		}
		if strings.Contains(l, `content_block_delta`) {
			hasDelta = true
		}
	}

	checkLine(line1)
	checkLine(line2)

	if !hasWarning {
		t.Errorf("expected first turn to contain a warning, got: %q and %q", line1, line2)
	} else if !hasWarningNonFatal {
		t.Errorf("expected warning to be non-fatal, got: %q and %q", line1, line2)
	}
	if !hasDelta {
		t.Errorf("expected first turn to contain a content delta, got: %q and %q", line1, line2)
	}

	// === SECOND TURN ===
	userMsg2 := `{"type":"user","message":{"role":"user","content":"hello 2"}}`
	_, err = proc.Write([]byte(userMsg2))
	if err != nil {
		t.Fatalf("Second Write failed: %v", err)
	}

	// Second turn expects only 1 line: content_block_delta (warning is throttled)
	if !scanner.Scan() {
		t.Fatalf("Third scan failed (expected content delta line)")
	}
	line3 := scanner.Text()
	if strings.Contains(line3, `"type":"session_warning"`) {
		t.Errorf("expected warning to be throttled, but got warning line: %q", line3)
	}
	if !strings.Contains(line3, `content_block_delta`) {
		t.Errorf("expected content delta line, got: %q", line3)
	}

	_ = proc.CloseStdin()
	_ = proc.Wait()
}

func TestSessionInjectMessage(t *testing.T) {
	s := &Session{
		messages: make([][]byte, 0),
		notify:   make(chan struct{}, 10),
	}

	msg := []byte(`{"type":"ghost_card_created","name":"test"}`)
	s.InjectMessage(msg)

	s.msgMu.RLock()
	if len(s.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(s.messages))
	}
	if string(s.messages[0]) != string(msg) {
		t.Errorf("expected %s, got %s", string(msg), string(s.messages[0]))
	}
	s.msgMu.RUnlock()

	select {
	case <-s.Notify():
		// Success
	default:
		t.Errorf("expected notification on notify channel")
	}
}

func TestExtractGhostNamed(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Untranslated top-level JSON",
			input:    `{"event":"ghost_named","name":"untranslated-name"}`,
			expected: "untranslated-name",
		},
		{
			name:     "Translated Gemini content_block_delta with JSON text",
			input:    `{"type":"content_block_delta","delta":{"text":"{\"event\":\"ghost_named\",\"name\":\"translated-json-name\"}\n"}}`,
			expected: "translated-json-name",
		},
		{
			name:     "Translated Gemini content_block_delta with raw text",
			input:    `{"type":"content_block_delta","delta":{"text":"some text containing \"event\":\"ghost_named\" and \"name\":\"translated-raw-name\""}}`,
			expected: "translated-raw-name",
		},
		{
			name:     "Fallback with escaped quotes",
			input:    `{\"event\":\"ghost_named\",\"name\":\"escaped-fallback-name\"}`,
			expected: "escaped-fallback-name",
		},
		{
			name:     "No match",
			input:    `{"type":"content_block_delta","delta":{"text":"hello world"}}`,
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractGhostNamed([]byte(tc.input))
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
