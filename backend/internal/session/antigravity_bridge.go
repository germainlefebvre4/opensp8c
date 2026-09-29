package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

// antigravityWriter wraps a subprocess stdin io.WriteCloser, translating input
// messages to the format expected by Antigravity CLI (`agy`), and injecting
// framing instructions on the first user turn.
type antigravityWriter struct {
	mu            sync.Mutex
	underlying    io.WriteCloser
	framingPrompt string
	firstTurn     bool
}

// newAntigravityWriter constructs a new writer wrapping underlying.
// If framingPrompt is non-empty, it will be prefixed to the first turn's user content.
func newAntigravityWriter(underlying io.WriteCloser, framingPrompt string) *antigravityWriter {
	return &antigravityWriter{
		underlying:    underlying,
		framingPrompt: strings.TrimSpace(framingPrompt),
		firstTurn:     true,
	}
}

func (w *antigravityWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	trimmed := strings.TrimSpace(string(p))
	if trimmed == "" {
		return len(p), nil
	}

	// Try parsing as JSON payload
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &payload); err == nil {
			typ, _ := payload["type"].(string)
			evt, _ := payload["event"].(string)
			if typ == "user" || evt == "user" {
				payload["event"] = "user"
				delete(payload, "type")

				if w.firstTurn {
					w.firstTurn = false
					if w.framingPrompt != "" {
						if msgMap, ok := payload["message"].(map[string]interface{}); ok {
							if contentStr, ok := msgMap["content"].(string); ok {
								if !strings.HasPrefix(contentStr, "[System Instructions:\n") {
									msgMap["content"] = fmt.Sprintf("[System Instructions:\n%s]\n\n%s", w.framingPrompt, contentStr)
								}
							}
						}
					}
				}

				out, err := json.Marshal(payload)
				if err == nil {
					out = append(out, '\n')
					_, writeErr := w.underlying.Write(out)
					return len(p), writeErr
				}
			}
		}
	}

	// Plain text fallback
	content := trimmed
	if w.firstTurn {
		w.firstTurn = false
		if w.framingPrompt != "" {
			if !strings.HasPrefix(content, "[System Instructions:\n") {
				content = fmt.Sprintf("[System Instructions:\n%s]\n\n%s", w.framingPrompt, content)
			}
		}
	}
	payload := map[string]interface{}{
		"event": "user",
		"message": map[string]interface{}{
			"role":    "user",
			"content": content,
		},
	}
	out, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	out = append(out, '\n')
	_, writeErr := w.underlying.Write(out)
	return len(p), writeErr
}

func (w *antigravityWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.underlying != nil {
		return w.underlying.Close()
	}
	return nil
}

// antigravityStdoutReader adapts the NDJSON stdout of Antigravity CLI to the standard Claude format.
type antigravityStdoutReader struct {
	original io.ReadCloser
	scanner  *bufio.Scanner
	buffer   []byte
	convID   string
	mu       sync.Mutex
}

func newAntigravityStdoutReader(original io.ReadCloser) *antigravityStdoutReader {
	scanner := bufio.NewScanner(original)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)
	return &antigravityStdoutReader{
		original: original,
		scanner:  scanner,
	}
}

// ConversationID returns the conversation ID observed from init or result events, if any.
func (r *antigravityStdoutReader) ConversationID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.convID
}

func (r *antigravityStdoutReader) translateLine(line []byte) []byte {
	var data struct {
		Event          string `json:"event"`
		ConversationID string `json:"conversation_id"`
	}
	if err := json.Unmarshal(line, &data); err == nil {
		if data.ConversationID != "" {
			r.mu.Lock()
			r.convID = data.ConversationID
			r.mu.Unlock()
		}
	}
	return translateAntigravityLine(line)
}

func (r *antigravityStdoutReader) Read(p []byte) (int, error) {
	if len(r.buffer) == 0 {
		for {
			if !r.scanner.Scan() {
				if err := r.scanner.Err(); err != nil {
					return 0, err
				}
				return 0, io.EOF
			}
			line := r.scanner.Bytes()
			translated := r.translateLine(line)
			if translated != nil {
				r.buffer = append(translated, '\n')
				break
			}
		}
	}

	n := copy(p, r.buffer)
	r.buffer = r.buffer[n:]
	return n, nil
}

func (r *antigravityStdoutReader) Close() error {
	return r.original.Close()
}

// translateAntigravityLine converts a single line of Antigravity NDJSON output
// to the standard Claude stream-json format.
// Returns nil if the line should be skipped (e.g. init or internal events).
// Returns line unchanged if not valid JSON or unrecognized.
func translateAntigravityLine(line []byte) []byte {
	trimmed := strings.TrimSpace(string(line))
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return line
	}

	var data map[string]interface{}
	if err := json.Unmarshal(line, &data); err != nil {
		return line
	}

	evt, _ := data["event"].(string)
	typ, _ := data["type"].(string)

	// Filter out init event
	if evt == "init" || typ == "init" {
		return nil
	}

	// Result event -> message_complete
	if evt == "result" || typ == "result" {
		completeMsg := map[string]interface{}{
			"type":   "message_complete",
			"result": " ",
		}
		out, err := json.Marshal(completeMsg)
		if err == nil {
			return out
		}
		return line
	}

	// Step update event
	if evt == "step_update" {
		stepUpdate, ok := data["step_update"].(map[string]interface{})
		if !ok {
			return nil
		}

		stepType, _ := stepUpdate["step_type"].(string)

		switch stepType {
		case "user_input":
			return nil

		case "agent_response":
			textDelta, _ := stepUpdate["text_delta"].(string)
			if textDelta == "" {
				return nil
			}
			claudeMsg := map[string]interface{}{
				"type": "content_block_delta",
				"delta": map[string]interface{}{
					"text": textDelta,
				},
			}
			out, err := json.Marshal(claudeMsg)
			if err == nil {
				return out
			}

		case "tool":
			state, _ := stepUpdate["state"].(string)
			toolName, _ := stepUpdate["tool_name"].(string)
			toolInfo, _ := stepUpdate["tool_info"].(map[string]interface{})

			toolID := ""
			if sid, ok := stepUpdate["step_id"].(string); ok && sid != "" {
				toolID = sid
			} else if id, ok := stepUpdate["id"].(string); ok && id != "" {
				toolID = id
			} else if sidx, ok := stepUpdate["step_index"].(float64); ok {
				toolID = fmt.Sprintf("toolu_%d", int(sidx))
			} else if toolInfo != nil {
				if tid, ok := toolInfo["id"].(string); ok && tid != "" {
					toolID = tid
				}
			}
			if toolID == "" {
				toolID = "toolu_1"
			}

			if toolName == "" && toolInfo != nil {
				if tname, ok := toolInfo["name"].(string); ok {
					toolName = tname
				} else if tname, ok := toolInfo["tool_name"].(string); ok {
					toolName = tname
				}
			}

			if state == "ACTIVE" {
				var params interface{}
				if toolInfo != nil {
					if p, ok := toolInfo["parameters"]; ok && p != nil {
						params = p
					} else if p, ok := toolInfo["input"]; ok && p != nil {
						params = p
					} else if p, ok := toolInfo["args"]; ok && p != nil {
						params = p
					}
				}
				if params == nil {
					params = map[string]interface{}{}
				}

				toolBlock := map[string]interface{}{
					"type":  "tool_use",
					"id":    toolID,
					"name":  toolName,
					"input": params,
				}
				msg := map[string]interface{}{
					"type":          "content_block_start",
					"content_block": toolBlock,
					"content":       []interface{}{toolBlock},
				}
				out, err := json.Marshal(msg)
				if err == nil {
					return out
				}
			} else if state == "DONE" {
				var outputStr string
				if toolInfo != nil {
					if outVal, ok := toolInfo["output"]; ok {
						if s, ok := outVal.(string); ok {
							outputStr = s
						} else {
							b, _ := json.Marshal(outVal)
							outputStr = string(b)
						}
					} else if resVal, ok := toolInfo["result"]; ok {
						if s, ok := resVal.(string); ok {
							outputStr = s
						} else {
							b, _ := json.Marshal(resVal)
							outputStr = string(b)
						}
					}
				}

				resultBlock := map[string]interface{}{
					"type":        "tool_result",
					"tool_use_id": toolID,
					"content":     outputStr,
				}
				msg := map[string]interface{}{
					"type":          "content_block_start",
					"content_block": resultBlock,
					"content":       []interface{}{resultBlock},
				}
				out, err := json.Marshal(msg)
				if err == nil {
					return out
				}
			}
			return nil
		}
	}

	return line
}
