package session

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
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
	} else if typ == "init" {
		return nil
	} else if typ == "result" {
		// Translate type: result to a frontend-compatible message_complete event
		// with a non-empty result string to successfully trigger state update
		// (setting partial = false) in the frontend.
		completeMsg := map[string]interface{}{
			"type":   "message_complete",
			"result": " ",
		}
		completeBytes, err := json.Marshal(completeMsg)
		if err == nil {
			return completeBytes
		}
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
	if agentCfg.ID == "gemini" {
		// Use a dummy process to satisfy Cmd and Wait requirements of Subprocess.
		// "cat" is lightweight and will run indefinitely until its stdin is closed.
		dummyCmd := exec.CommandContext(ctx, "cat")
		dummyStdin, err := dummyCmd.StdinPipe()
		if err != nil {
			return nil, err
		}
		if err := dummyCmd.Start(); err != nil {
			return nil, err
		}

		virtualStdinReader, virtualStdinWriter := io.Pipe()
		virtualStdoutReader, virtualStdoutWriter := io.Pipe()

		activeSessionID := claudeSessionID
		if activeSessionID == "" {
			// Generate a unique session ID to support multi-turn continuity
			// for anonymous/explore sessions as well.
			var b [16]byte
			_, _ = rand.Read(b[:])
			b[6] = (b[6] & 0x0f) | 0x40
			b[8] = (b[8] & 0x3f) | 0x80
			h := hex.EncodeToString(b[:])
			activeSessionID = h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
		}

		// Keep track of whether we need to resume or start a new session.
		shouldResume := resume

		go func() {
			defer dummyStdin.Close()
			defer virtualStdoutWriter.Close()

			scanner := bufio.NewScanner(virtualStdinReader)
			for scanner.Scan() {
				line := scanner.Text()
				if len(strings.TrimSpace(line)) == 0 {
					continue
				}

				var payload struct {
					Type    string `json:"type"`
					Message struct {
						Content string `json:"content"`
					} `json:"message"`
				}
				var prompt string
				if err := json.Unmarshal([]byte(line), &payload); err == nil && payload.Type == "user" {
					prompt = payload.Message.Content
				} else {
					prompt = line
				}

				if len(strings.TrimSpace(prompt)) == 0 {
					continue
				}

				// Build subprocess arguments for the one-shot run
				args := agentCfg.BuildSubprocessArgs(baseSystemPrompt, extraSystemPrompt)
				if shouldResume {
					args = append(args, "--resume", activeSessionID)
				} else {
					args = append(args, "--session-id", activeSessionID)
				}

				// Start the real gemini subprocess
				subCtx, subCancel := context.WithCancel(ctx)
				cmd := exec.CommandContext(subCtx, agentCfg.CLI, args...)
				cmd.Dir = workspacePath

				stdin, err := cmd.StdinPipe()
				if err != nil {
					log.Printf("[gemini bridge] failed to get stdin pipe: %v", err)
					subCancel()
					continue
				}

				stdout, err := cmd.StdoutPipe()
				if err != nil {
					log.Printf("[gemini bridge] failed to get stdout pipe: %v", err)
					subCancel()
					continue
				}

				stderr, err := cmd.StderrPipe()
				if err != nil {
					log.Printf("[gemini bridge] failed to get stderr pipe: %v", err)
					subCancel()
					continue
				}

				if err := cmd.Start(); err != nil {
					log.Printf("[gemini bridge] failed to start gemini: %v", err)
					subCancel()
					continue
				}

				// Handle stderr logs safely
				go func() {
					errScanner := bufio.NewScanner(stderr)
					for errScanner.Scan() {
						text := errScanner.Text()
						log.Printf("[subprocess stderr] %s", text)
						if sessionLog != nil {
							sessionLog.WriteErr(text)
						}
						
						// Detect common fatal errors and forward to UI gracefully
						if strings.Contains(text, "TerminalQuotaError") {
							warning := map[string]interface{}{
								"type": "session_warning",
								"text": "Vous avez épuisé votre quota pour ce modèle (Quota Exhausted). Veuillez sélectionner un autre agent via le sélecteur en bas de la barre latérale, puis cliquez sur Relancer/Reconnecter.",
							}
							if b, err := json.Marshal(warning); err == nil {
								_, _ = virtualStdoutWriter.Write(append(b, '\n'))
							}
						} else if strings.Contains(text, "Failed to connect to IDE companion extension") {
							warning := map[string]interface{}{
								"type": "session_warning",
								"text": "Impossible de se connecter à l'extension IDE. Veuillez vérifier qu'elle est installée et lancée dans votre éditeur.",
							}
							if b, err := json.Marshal(warning); err == nil {
								_, _ = virtualStdoutWriter.Write(append(b, '\n'))
							}
						}
					}
				}()

				// Write the prompt to gemini stdin and close it so gemini runs to completion
				go func() {
					_, _ = stdin.Write([]byte(prompt + "\n"))
					_ = stdin.Close()
				}()

				// Read stream-json stdout, translate and forward to virtual stdout
				geminiReader := newGeminiStdoutReader(stdout)
				_, _ = io.Copy(virtualStdoutWriter, geminiReader)
				_ = geminiReader.Close()

				// Wait for the one-shot run to complete
				_ = cmd.Wait()
				subCancel()

				// Subsequent turns must resume
				shouldResume = true
			}
		}()

		return &Subprocess{
			cmd:     dummyCmd,
			stdin:   virtualStdinWriter,
			stdout:  virtualStdoutReader,
			agentID: "gemini",
		}, nil
	}

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
		// Pass raw JSON string followed by a newline so the scanner in StartSubprocess
		// can process it as a single line and extract multi-line prompts correctly.
		trimmed := strings.TrimSpace(string(p))
		if !strings.HasSuffix(trimmed, "\n") {
			trimmed += "\n"
		}
		return s.stdin.Write([]byte(trimmed))
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
