package session

import (
	"bufio"
	"context"
	"io"
	"log"
	"os/exec"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/conversation"
)

const baseSystemPrompt = "Never use AskUserQuestion or interactive choice prompts. Communicate only through plain conversational text."

type Subprocess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
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
		if resume {
			args = append(args, "--resume", claudeSessionID)
		} else {
			args = append(args, "--session-id", claudeSessionID)
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

	return &Subprocess{cmd: cmd, stdin: stdin, stdout: stdout}, nil
}

func (s *Subprocess) Write(p []byte) (int, error) {
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
