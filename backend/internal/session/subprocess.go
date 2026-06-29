package session

import (
	"bufio"
	"context"
	"io"
	"log"
	"os/exec"

	"github.com/glefebvre/opensp8c/internal/agents"
)

const baseSystemPrompt = "Never use AskUserQuestion or interactive choice prompts. Communicate only through plain conversational text."

type Subprocess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func StartSubprocess(ctx context.Context, workspacePath string, agentCfg agents.AgentConfig, extraSystemPrompt string) (*Subprocess, error) {
	args := agentCfg.BuildSubprocessArgs(baseSystemPrompt, extraSystemPrompt)
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
			log.Printf("[subprocess stderr] %s", scanner.Text())
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
