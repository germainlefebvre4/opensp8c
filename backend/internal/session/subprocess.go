package session

import (
	"context"
	"io"
	"os/exec"
)

const systemPrompt = "Never use AskUserQuestion or interactive choice prompts. Communicate only through plain conversational text."

type Subprocess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func StartSubprocess(ctx context.Context, workspacePath string, extraSystemPrompt string) (*Subprocess, error) {
	args := []string{
		"--print",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--include-partial-messages",
		"--append-system-prompt", systemPrompt,
		"--cwd", workspacePath,
	}
	if extraSystemPrompt != "" {
		args = append(args, "--append-system-prompt", extraSystemPrompt)
	}
	cmd := exec.CommandContext(ctx, "claude", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	cmd.Dir = workspacePath

	if err := cmd.Start(); err != nil {
		return nil, err
	}

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
