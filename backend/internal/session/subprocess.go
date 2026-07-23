package session

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"os/exec"
	"strings"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
)

const baseSystemPrompt = "Never use AskUserQuestion or interactive choice prompts. Communicate only through plain conversational text."

type Subprocess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	agentID string
}

type geminiStdoutReader struct {
	original io.ReadCloser
	scanner  *bufio.Scanner
	buffer   []byte
}

func newGeminiStdoutReader(original io.ReadCloser) *geminiStdoutReader {
	scanner := bufio.NewScanner(original)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	return &geminiStdoutReader{
		original: original,
		scanner:  scanner,
	}
}

func translateGeminiLine(line []byte) []byte {
	trimmed := strings.TrimSpace(string(line))
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return line
	}

	var data map[string]interface{}
	if err := json.Unmarshal(line, &data); err != nil {
		return line
	}

	typ, _ := data["type"].(string)
	if typ == "message" {
		role, _ := data["role"].(string)
		if role == "assistant" {
			content, _ := data["content"].(string)
			claudeMsg := map[string]interface{}{
				"type": "content_block_delta",
				"delta": map[string]interface{}{
					"text": content,
				},
			}
			claudeBytes, err := json.Marshal(claudeMsg)
			if err == nil {
				return claudeBytes
			}
		} else if role == "user" {
			return nil
		}
	} else if typ == "init" || typ == "result" {
		return nil
	}

	return line
}

func (r *geminiStdoutReader) Read(p []byte) (int, error) {
	if len(r.buffer) == 0 {
		for {
			if !r.scanner.Scan() {
				if err := r.scanner.Err(); err != nil {
					return 0, err
				}
				return 0, io.EOF
			}
			line := r.scanner.Bytes()
			translated := translateGeminiLine(line)
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

func (r *geminiStdoutReader) Close() error {
	return r.original.Close()
}

// StartSubprocess launches the agent CLI as a subprocess.
// claudeSessionID controls session continuity:
//   - empty: no session flags (anonymous sessions)
//   - non-empty, resume=false: passes --session-id <claudeSessionID>
//   - non-empty, resume=true: passes --resume <claudeSessionID>
//
// sessionLog is optional (nil-safe): when provided, stderr lines are also
// written to it in addition to the existing log.Printf.
func StartSubprocess(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog) (*Subprocess, error) {
	args := agentCfg.BuildSubprocessArgs(baseSystemPrompt, extraSystemPrompt)
	if claudeSessionID != "" {
		if agentCfg.ID == "claude" || agentCfg.ID == "gemini" {
			if resume {
				args = append(args, "--resume", claudeSessionID)
			} else {
				args = append(args, "--session-id", claudeSessionID)
			}
		}
	}
	cmd := exec.CommandContext(ctx, agentCfg.CLI, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// Propagate environment variables (like GOOGLE_CLOUD_PROJECT for Gemini CLI)
	// transparently by leaving cmd.Env as nil, allowing os.exec to inherit parent env.
	cmd.Dir = workspacePath

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			text := scanner.Text()
			log.Printf("[subprocess stderr] %s", text)
			sessionLog.WriteErr(text)
		}
	}()

	var adaptedStdout io.ReadCloser = stdout
	if agentCfg.ID == "gemini" {
		adaptedStdout = newGeminiStdoutReader(stdout)
	}

	return &Subprocess{cmd: cmd, stdin: stdin, stdout: adaptedStdout, agentID: agentCfg.ID}, nil
}

func (s *Subprocess) Write(p []byte) (int, error) {
	if s.agentID == "gemini" {
		var payload struct {
			Type    string `json:"type"`
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		trimmed := strings.TrimSpace(string(p))
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			if err := json.Unmarshal([]byte(trimmed), &payload); err == nil && payload.Type == "user" && payload.Message.Content != "" {
				content := payload.Message.Content
				if !strings.HasSuffix(content, "\n") {
					content += "\n"
				}
				return s.stdin.Write([]byte(content))
			}
		}
	}
	return s.stdin.Write(p)
}

func (s *Subprocess) Read(p []byte) (int, error) {
	return s.stdout.Read(p)
}

func (s *Subprocess) CloseStdin() error {
	return s.stdin.Close()
}

func (s *Subprocess) Wait() error {
	return s.cmd.Wait()
}

func (s *Subprocess) Stdout() io.ReadCloser {
	return s.stdout
}
