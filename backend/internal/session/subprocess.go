package session

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
)

const baseSystemPrompt = "Never use AskUserQuestion or interactive choice prompts. Communicate only through plain conversational text.\n\n" + explorationFramingPrompt

// baseSystemPromptNativeQuestion is the Claude-only variant of baseSystemPrompt
// used when the native question mode preference is active for this session:
// it lifts the AskUserQuestion interdiction instead of forbidding the tool.
const baseSystemPromptNativeQuestion = "You may use the AskUserQuestion tool for interactive clarification questions, as an alternative to the ghost_question marker described below.\n\n" + explorationFramingPrompt

// resolveBaseSystemPrompt picks the base system prompt for a session: the
// AskUserQuestion-permitting variant only when the resolved agent is Claude
// and the native question mode preference is active for this session.
func resolveBaseSystemPrompt(agentID string, nativeQuestionMode bool) string {
	if agentID == "claude" && nativeQuestionMode {
		return baseSystemPromptNativeQuestion
	}
	return baseSystemPrompt
}

type Subprocess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	agentID string

	// exited is closed once cmd.Wait has returned (nil when there is no real
	// process); waitErr is valid after that. A single goroutine calls
	// cmd.Wait, so Wait may be called any number of times.
	exited  chan struct{}
	waitErr error
	// stdoutFile is the read end of the pipe owned by the subprocess (not by
	// exec), closed by Wait: exec's own Wait would close a StdoutPipe as
	// soon as the process exits and drop output not read yet.
	stdoutFile *os.File
}

// watchExit starts the single goroutine reaping cmd.
func (s *Subprocess) watchExit() {
	s.exited = make(chan struct{})
	go func() {
		s.waitErr = s.cmd.Wait()
		close(s.exited)
	}()
}

// resumeProbeWindow is how long a --resume start is watched for an early exit.
// Measured with the installed CLI (claude --resume <unknown id>, stdin left
// open): it prints "No conversation found", then exits with code 1 after
// ~0.85 s, without StartSubprocess ever returning an error. 3 s leaves margin
// for slow machines; it only delays a resume, never a first start.
// Override with OPENSP8C_RESUME_PROBE (a Go duration, e.g. "5s") on slow hosts.
var resumeProbeWindow = resumeProbeFromEnv(3 * time.Second)

func resumeProbeFromEnv(def time.Duration) time.Duration {
	if v := os.Getenv("OPENSP8C_RESUME_PROBE"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
		log.Printf("[session] ignoring invalid OPENSP8C_RESUME_PROBE %q", v)
	}
	return def
}

// exitedWithin reports whether the process exits within window. Only
// meaningful for a real process started by StartSubprocess.
func (s *Subprocess) exitedWithin(window time.Duration) bool {
	if s.exited == nil {
		return false
	}
	timer := time.NewTimer(window)
	defer timer.Stop()
	select {
	case <-s.exited:
		return true
	case <-timer.C:
		return false
	}
}

// startWithResumeFallback starts an agent subprocess via start, resuming
// claudeSessionID. When the resume fails — start returns an error, or (with
// probe) the process exits within resumeProbeWindow, as an unknown session
// makes the CLI do — it starts again without resume under a fresh session id
// and reports contextLost. usedID is the session id actually in use.
func startWithResumeFallback(start func(claudeSessionID string, resume bool) (*Subprocess, error), claudeSessionID string, probe bool) (proc *Subprocess, usedID string, contextLost bool, err error) {
	proc, err = start(claudeSessionID, true)
	if err == nil && probe && proc.exitedWithin(resumeProbeWindow) {
		_ = proc.CloseStdin()
		_ = proc.Wait()
		err = io.ErrUnexpectedEOF
	}
	if err == nil {
		return proc, claudeSessionID, false, nil
	}
	log.Printf("[session] --resume %s failed, starting fresh: %v", claudeSessionID, err)
	usedID = newClaudeSessionID()
	proc, err = start(usedID, false)
	return proc, usedID, true, err
}

// NewTestSubprocess constructs a Subprocess wrapping the given stdin/stdout
// (stdout may be nil if unused), for use in tests of packages that depend on
// *Subprocess via a *Session (e.g. WebSocket handlers verifying what gets
// written to stdin).
func NewTestSubprocess(stdin io.WriteCloser, stdout io.ReadCloser, agentID string) *Subprocess {
	return &Subprocess{stdin: stdin, stdout: stdout, agentID: agentID}
}

// BuildToolResultMessage constructs the stream-json stdin payload for a
// tool_result reply referencing the given tool_use_id, as expected by
// Claude's --input-format stream-json when answering a native AskUserQuestion
// tool_use call.
func BuildToolResultMessage(toolUseID, content string) []byte {
	payload := map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"role": "user",
			"content": []map[string]interface{}{
				{
					"type":        "tool_result",
					"tool_use_id": toolUseID,
					"content":     content,
				},
			},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return data
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
//
// nativeQuestionMode is the global preference toggle; it only takes effect
// when agentCfg is Claude (see resolveBaseSystemPrompt).
//
// languageDirective is the language instruction for this launch (may be empty).
// It is delivered through the channel each agent supports: appended to the
// extra system prompt (Claude, Codex, Copilot), sent with the first message and
// again on resume (Antigravity), or added to every turn (Gemini).
func StartSubprocess(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt, claudeSessionID string, resume bool, sessionLog *conversation.SessionLog, customEnv map[string]string, nativeQuestionMode bool, languageDirective string) (*Subprocess, error) {
	basePrompt := resolveBaseSystemPrompt(agentCfg.ID, nativeQuestionMode)

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

				// Ensure every turn is executed in explore mode to trigger/keep the openspec-explore skill active,
				// since Gemini runs as a one-shot process and does not persist active skills natively across resume runs.
				// We skip prefixing if the prompt already has a slash (e.g. other slash commands like /opsx:ff) or starts with `{` (e.g. mock JSON in unit tests).
				trimmedPrompt := strings.TrimSpace(prompt)
				if !strings.HasPrefix(trimmedPrompt, "/") && !strings.HasPrefix(trimmedPrompt, "{") {
					prompt = "/opsx:explore " + prompt
				}
				prompt = appendLanguageDirective(prompt, languageDirective)

				// Build subprocess arguments for the one-shot run
				args := agentCfg.BuildSubprocessArgs(basePrompt, extraSystemPrompt)
				if shouldResume {
					args = append(args, "--resume", activeSessionID)
				} else {
					args = append(args, "--session-id", activeSessionID)
				}

				// Start the real gemini subprocess
				subCtx, subCancel := context.WithCancel(ctx)
				cmd := exec.CommandContext(subCtx, agentCfg.CLI, args...)
				cmd.Dir = workspacePath
				cmd.Env = buildEnv(customEnv)

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
						if strings.Contains(text, "Failed to connect to IDE companion extension") {
							continue
						}
						log.Printf("[subprocess stderr] %s", text)
						if sessionLog != nil {
							sessionLog.WriteErr(text)
						}

						// Detect common fatal errors and forward to UI gracefully
						if strings.Contains(text, "TerminalQuotaError") {
							warning := map[string]interface{}{
								"type":  "session_warning",
								"text":  "Vous avez épuisé votre quota pour ce modèle (Quota Exhausted). Veuillez sélectionner un autre agent via le sélecteur en bas de la barre latérale, puis cliquez sur Relancer/Reconnecter.",
								"fatal": true,
							}
							if b, err := json.Marshal(warning); err == nil {
								_, _ = virtualStdoutWriter.Write(append(b, '\n'))
							}
						} else if strings.Contains(text, "ProjectIdRequiredError") || strings.Contains(text, "GOOGLE_CLOUD_PROJECT") {
							warning := map[string]interface{}{
								"type":  "session_warning",
								"text":  "Erreur d'authentification Google Cloud : l'identifiant du projet (ProjectId) est requis pour ce compte. Veuillez définir la variable d'environnement GOOGLE_CLOUD_PROJECT ou GOOGLE_CLOUD_PROJECT_ID.",
								"fatal": true,
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

		dummy := &Subprocess{
			cmd:     dummyCmd,
			stdin:   virtualStdinWriter,
			stdout:  virtualStdoutReader,
			agentID: "gemini",
		}
		dummy.watchExit()
		return dummy, nil
	}

	args := buildSubprocessArgs(agentCfg, basePrompt, joinPrompts(extraSystemPrompt, languageDirective), claudeSessionID, resume)
	cmd := exec.CommandContext(ctx, agentCfg.CLI, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout = stdoutW
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdoutR.Close()
		stdoutW.Close()
		return nil, err
	}
	var stdout io.ReadCloser = stdoutR

	// Propagate environment variables, combining standard global environment variables and custom user settings.
	cmd.Dir = workspacePath
	cmd.Env = buildEnv(customEnv)

	if err := cmd.Start(); err != nil {
		stdoutR.Close()
		stdoutW.Close()
		return nil, err
	}
	stdoutW.Close() // the child holds its own copy

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			text := scanner.Text()
			if strings.Contains(text, "Failed to connect to IDE companion extension") {
				continue
			}
			log.Printf("[subprocess stderr] %s", text)
			sessionLog.WriteErr(text)
		}
	}()

	var adaptedStdout io.ReadCloser = stdout
	if agentCfg.ID == "gemini" {
		adaptedStdout = newGeminiStdoutReader(stdout)
	} else if agentCfg.ID == "antigravity" {
		adaptedStdout = newAntigravityStdoutReader(stdout)
	}

	var adaptedStdin io.WriteCloser = stdin
	if agentCfg.ID == "antigravity" {
		adaptedStdin = newAntigravityWriter(stdin, antigravityFraming(extraSystemPrompt, languageDirective, resume))
	}

	proc := &Subprocess{cmd: cmd, stdin: adaptedStdin, stdout: adaptedStdout, agentID: agentCfg.ID, stdoutFile: stdoutR}
	proc.watchExit()
	return proc, nil
}

// joinPrompts concatenates non-empty prompt parts with a blank line.
func joinPrompts(parts ...string) string {
	var kept []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, "\n\n")
}

// antigravityFraming returns the framing sent with Antigravity's first message.
// On resume the framing prompt is dropped but the language directive is kept,
// since it must apply to every launch.
func antigravityFraming(extraSystemPrompt, languageDirective string, resume bool) string {
	if resume {
		return strings.TrimSpace(languageDirective)
	}
	return joinPrompts(extraSystemPrompt, languageDirective)
}

// appendLanguageDirective adds the directive in its own paragraph after the
// prompt, so the first line of a slash-command prompt stays the intact command.
func appendLanguageDirective(prompt, languageDirective string) string {
	if strings.TrimSpace(languageDirective) == "" {
		return prompt
	}
	return prompt + "\n\n" + languageDirective
}

func buildSubprocessArgs(agentCfg agents.AgentConfig, basePrompt, extraSystemPrompt, sessionID string, resume bool) []string {
	args := agentCfg.BuildSubprocessArgs(basePrompt, extraSystemPrompt)
	if sessionID != "" {
		if agentCfg.ID == "claude" || agentCfg.ID == "gemini" {
			if resume {
				args = append(args, "--resume", sessionID)
			} else {
				args = append(args, "--session-id", sessionID)
			}
		} else if agentCfg.ID == "antigravity" {
			if resume {
				args = append(args, "--conversation", sessionID)
			}
		}
	}
	return args
}

func buildEnv(customEnv map[string]string) []string {
	parentEnv := os.Environ()
	if len(customEnv) == 0 {
		return parentEnv
	}

	envMap := make(map[string]string)
	for _, envVar := range parentEnv {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	for k, v := range customEnv {
		envMap[k] = v
	}

	reconstructed := make([]string, 0, len(envMap))
	for k, v := range envMap {
		reconstructed = append(reconstructed, k+"="+v)
	}
	return reconstructed
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

// Wait blocks until the underlying process exits. It is a no-op for a
// Subprocess built via NewTestSubprocess (no real process to wait for), so
// callers that unconditionally Wait() as part of teardown (e.g. a pool
// worker's cleanup) can be exercised in tests without a real child process.
func (s *Subprocess) Wait() error {
	if s.cmd == nil {
		return nil
	}
	if s.exited != nil {
		<-s.exited
		if s.stdoutFile != nil {
			_ = s.stdoutFile.Close()
		}
		return s.waitErr
	}
	return s.cmd.Wait()
}

func (s *Subprocess) Stdout() io.ReadCloser {
	return s.stdout
}

func (s *Subprocess) ConversationID() string {
	if agyReader, ok := s.stdout.(*antigravityStdoutReader); ok {
		return agyReader.ConversationID()
	}
	return ""
}
