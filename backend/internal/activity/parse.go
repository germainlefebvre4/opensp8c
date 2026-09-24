package activity

import (
	"encoding/json"
	"strings"
	"time"
)

type envelopeLine struct {
	Ts   string          `json:"ts"`
	Dir  string          `json:"dir"`
	Data json.RawMessage `json:"data"`
}

type toolUseRecord struct {
	id       string
	name     string
	summary  string
	tsStr    string
	useTime  time.Time
	input    map[string]any
	entryIdx int
}

// ParseConversationLines derives activity entries (agent narration and tool calls with duration)
// from the raw JSONL lines of a conversation run.
func ParseConversationLines(lines [][]byte) ([]Entry, error) {
	var entries []Entry
	toolCallMap := make(map[string]*toolUseRecord)

	var currentNarration strings.Builder
	var currentNarrationTs string

	flushNarration := func() {
		text := strings.TrimSpace(currentNarration.String())
		if text != "" {
			entries = append(entries, Entry{
				Ts:       currentNarrationTs,
				Type:     "agent",
				Category: "agent",
				Summary:  text,
			})
		}
		currentNarration.Reset()
		currentNarrationTs = ""
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" {
			continue
		}

		var env envelopeLine
		var dataBytes []byte
		lineTs := ""

		if err := json.Unmarshal([]byte(trimmed), &env); err == nil && len(env.Data) > 0 {
			dataBytes = env.Data
			lineTs = env.Ts
		} else {
			dataBytes = []byte(trimmed)
		}

		var payload map[string]any
		if err := json.Unmarshal(dataBytes, &payload); err != nil {
			continue
		}

		tsStr := lineTs
		if tStr, ok := payload["timestamp"].(string); ok && tStr != "" {
			tsStr = tStr
		}
		if tsStr == "" {
			tsStr = time.Now().UTC().Format(time.RFC3339Nano)
		}
		tTime, _ := parseTimestamp(tsStr)

		// 1. Check for text delta
		deltaText := extractDeltaText(payload)
		if deltaText != "" {
			if currentNarration.Len() == 0 {
				currentNarrationTs = tsStr
			}
			currentNarration.WriteString(deltaText)
			continue
		}

		// 2. Check for message_complete or content_block_stop or non-delta events
		pType, _ := payload["type"].(string)
		if pType == "message_complete" || pType == "message_stop" {
			flushNarration()
		}

		// 3. Check for tool_use
		toolUses := extractToolUses(payload, tsStr, tTime)
		if len(toolUses) > 0 {
			flushNarration()
			for _, tu := range toolUses {
				if existing, ok := toolCallMap[tu.id]; ok {
					// Update existing entry summary/input if it was incomplete
					if len(tu.input) > 0 && len(existing.input) == 0 {
						existing.input = tu.input
						existing.summary = deriveToolSummary(tu.name, tu.input)
						entries[existing.entryIdx].Summary = existing.summary
						entries[existing.entryIdx].Meta["input"] = tu.input
					}
					continue
				}

				entry := Entry{
					Ts:       tu.tsStr,
					Type:     tu.name,
					Category: "tool",
					Summary:  tu.summary,
					Meta: map[string]any{
						"tool_id":   tu.id,
						"tool_name": tu.name,
						"input":     tu.input,
					},
				}
				tu.entryIdx = len(entries)
				entries = append(entries, entry)
				toolCallMap[tu.id] = tu
			}
		}

		// 4. Check for tool_result
		toolResults := extractToolResults(payload, tTime)
		for _, tr := range toolResults {
			flushNarration()
			if rec, ok := toolCallMap[tr.toolUseID]; ok {
				if !rec.useTime.IsZero() && !tr.resultTime.IsZero() {
					dur := tr.resultTime.Sub(rec.useTime).Milliseconds()
					if dur < 0 {
						dur = 0
					}
					entries[rec.entryIdx].DurationMs = &dur
				}
			}
		}

		// 5. Assistant turn with full content blocks (in case no streaming deltas were present)
		if pType == "assistant" {
			if msg, ok := payload["message"].(map[string]any); ok {
				if content, ok := msg["content"].([]any); ok {
					var textBuf strings.Builder
					for _, b := range content {
						if bMap, ok := b.(map[string]any); ok {
							if bType, _ := bMap["type"].(string); bType == "text" {
								if t, ok := bMap["text"].(string); ok {
									textBuf.WriteString(t)
								}
							}
						}
					}
					assembled := strings.TrimSpace(textBuf.String())
					if assembled != "" && currentNarration.Len() == 0 {
						// Only emit if not already emitted by deltas
						entries = append(entries, Entry{
							Ts:       tsStr,
							Type:     "agent",
							Category: "agent",
							Summary:  assembled,
						})
					}
				}
			}
			flushNarration()
		}
	}

	flushNarration()

	if entries == nil {
		entries = []Entry{}
	}
	return entries, nil
}

func parseTimestamp(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15-04-05Z",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func extractDeltaText(payload map[string]any) string {
	// 1. Claude stream_event with content_block_delta
	if pType, _ := payload["type"].(string); pType == "stream_event" {
		if ev, ok := payload["event"].(map[string]any); ok {
			if evType, _ := ev["type"].(string); evType == "content_block_delta" {
				if delta, ok := ev["delta"].(map[string]any); ok {
					if dType, _ := delta["type"].(string); dType == "text_delta" {
						if t, ok := delta["text"].(string); ok {
							return t
						}
					}
				}
			}
		}
	}

	// 2. Direct content_block_delta (Gemini translation or flat)
	if pType, _ := payload["type"].(string); pType == "content_block_delta" {
		if delta, ok := payload["delta"].(map[string]any); ok {
			if t, ok := delta["text"].(string); ok {
				return t
			}
		}
	}

	return ""
}

func extractToolUses(payload map[string]any, defaultTsStr string, defaultTime time.Time) []*toolUseRecord {
	var records []*toolUseRecord

	addRecord := func(id, name string, input map[string]any) {
		if id == "" {
			return
		}
		if name == "" {
			name = "Tool"
		}
		records = append(records, &toolUseRecord{
			id:      id,
			name:    name,
			summary: deriveToolSummary(name, input),
			tsStr:   defaultTsStr,
			useTime: defaultTime,
			input:   input,
		})
	}

	// 1. stream_event -> content_block_start -> tool_use
	if pType, _ := payload["type"].(string); pType == "stream_event" {
		if ev, ok := payload["event"].(map[string]any); ok {
			if evType, _ := ev["type"].(string); evType == "content_block_start" {
				if cb, ok := ev["content_block"].(map[string]any); ok {
					if cbType, _ := cb["type"].(string); cbType == "tool_use" {
						id, _ := cb["id"].(string)
						name, _ := cb["name"].(string)
						input, _ := cb["input"].(map[string]any)
						if input == nil {
							input = make(map[string]any)
						}
						addRecord(id, name, input)
					}
				}
			}
		}
	}

	// 2. assistant message.content -> tool_use
	if msg, ok := payload["message"].(map[string]any); ok {
		if content, ok := msg["content"].([]any); ok {
			for _, item := range content {
				if bMap, ok := item.(map[string]any); ok {
					if bType, _ := bMap["type"].(string); bType == "tool_use" {
						id, _ := bMap["id"].(string)
						name, _ := bMap["name"].(string)
						input, _ := bMap["input"].(map[string]any)
						if input == nil {
							input = make(map[string]any)
						}
						addRecord(id, name, input)
					}
				}
			}
		}
	}

	// 3. Top-level tool_use
	if pType, _ := payload["type"].(string); pType == "tool_use" {
		id, _ := payload["id"].(string)
		if id == "" {
			id, _ = payload["tool_id"].(string)
		}
		name, _ := payload["name"].(string)
		if name == "" {
			name, _ = payload["tool_name"].(string)
		}
		input, _ := payload["input"].(map[string]any)
		if input == nil {
			input, _ = payload["parameters"].(map[string]any)
		}
		if input == nil {
			input = make(map[string]any)
		}
		addRecord(id, name, input)
	}

	return records
}

type toolResultRecord struct {
	toolUseID  string
	resultTime time.Time
}

func extractToolResults(payload map[string]any, defaultTime time.Time) []toolResultRecord {
	var results []toolResultRecord

	resTime := defaultTime
	if tStr, ok := payload["timestamp"].(string); ok && tStr != "" {
		if pt, ok := parseTimestamp(tStr); ok {
			resTime = pt
		}
	}

	// 1. user message.content -> tool_result
	if msg, ok := payload["message"].(map[string]any); ok {
		if content, ok := msg["content"].([]any); ok {
			for _, item := range content {
				if bMap, ok := item.(map[string]any); ok {
					if bType, _ := bMap["type"].(string); bType == "tool_result" {
						id, _ := bMap["tool_use_id"].(string)
						if id == "" {
							id, _ = bMap["tool_id"].(string)
						}
						if id != "" {
							results = append(results, toolResultRecord{
								toolUseID:  id,
								resultTime: resTime,
							})
						}
					}
				}
			}
		}
	}

	// 2. Direct top-level tool_result
	if pType, _ := payload["type"].(string); pType == "tool_result" {
		id, _ := payload["tool_use_id"].(string)
		if id == "" {
			id, _ = payload["tool_id"].(string)
		}
		if id != "" {
			results = append(results, toolResultRecord{
				toolUseID:  id,
				resultTime: resTime,
			})
		}
	}

	return results
}

func deriveToolSummary(name string, input map[string]any) string {
	switch name {
	case "Read", "Write", "Edit":
		if path, ok := input["file_path"].(string); ok && path != "" {
			return name + " " + path
		}
	case "Grep":
		if pat, ok := input["pattern"].(string); ok && pat != "" {
			return name + " " + pat
		}
	case "Bash":
		if cmd, ok := input["command"].(string); ok && cmd != "" {
			return name + " " + truncate(cmd, 80)
		}
	}

	for _, v := range input {
		if s, ok := v.(string); ok && s != "" {
			return name + " " + truncate(s, 80)
		}
	}
	return name
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
