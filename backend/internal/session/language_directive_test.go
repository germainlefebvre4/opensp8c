package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/agents"
)

const testDirective = "Unless the user explicitly asks otherwise, converse in French."

func argsContain(args []string, want string) bool {
	for i, a := range args {
		if a == "--append-system-prompt" && i+1 < len(args) && strings.Contains(args[i+1], want) {
			return true
		}
	}
	return false
}

func TestDirectiveInSystemPromptFreshAndResume(t *testing.T) {
	for _, id := range []string{"claude", "codex", "copilot"} {
		cfg, ok := agents.ByID(id)
		if !ok {
			t.Fatalf("agent %s missing", id)
		}
		for _, resume := range []bool{false, true} {
			args := buildSubprocessArgs(cfg, "base", joinPrompts("framing", testDirective), "sid", resume)
			if !argsContain(args, testDirective) || !argsContain(args, "framing") {
				t.Errorf("%s resume=%v: directive or framing missing in %v", id, resume, args)
			}
		}
	}
}

func TestJoinPromptsSkipsEmpty(t *testing.T) {
	if got := joinPrompts("", "  ", testDirective); got != testDirective {
		t.Errorf("got %q", got)
	}
}

func TestAntigravityFramingWithDirective(t *testing.T) {
	// first message: framing + directive
	out := &mockCloseWriter{}
	w := newAntigravityWriter(out, antigravityFraming("Frame.", testDirective, false))
	_, _ = w.Write([]byte(`{"type":"user","message":{"role":"user","content":"hi"}}`))
	if s := string(out.Bytes()); !strings.Contains(s, "Frame.") || !strings.Contains(s, testDirective) {
		t.Errorf("first message: %s", s)
	}

	// resume: directive alone
	out2 := &mockCloseWriter{}
	w2 := newAntigravityWriter(out2, antigravityFraming("Frame.", testDirective, true))
	_, _ = w2.Write([]byte(`{"type":"user","message":{"role":"user","content":"hi"}}`))
	s := string(out2.Bytes())
	if !strings.Contains(s, testDirective) || strings.Contains(s, "Frame.") {
		t.Errorf("resume: %s", s)
	}

	// resume without directive: no framing at all
	if got := antigravityFraming("Frame.", "", true); got != "" {
		t.Errorf("resume without directive = %q", got)
	}
}

func TestAppendLanguageDirectiveKeepsSlashCommandIntact(t *testing.T) {
	got := appendLanguageDirective("/opsx:ff my-change", testDirective)
	lines := strings.Split(got, "\n")
	if lines[0] != "/opsx:ff my-change" {
		t.Errorf("first line = %q", lines[0])
	}
	if !strings.HasSuffix(got, "\n\n"+testDirective) {
		t.Errorf("directive not in separate paragraph: %q", got)
	}
	if got := appendLanguageDirective("free prompt", ""); got != "free prompt" {
		t.Errorf("empty directive changed prompt: %q", got)
	}
}

func TestGeminiBridgeSendsDirectiveEveryTurn(t *testing.T) {
	tmpDir := t.TempDir()
	out := filepath.Join(tmpDir, "stdin.txt")
	script := "#!/bin/sh\ncat >> " + out + "\n"
	cli := filepath.Join(tmpDir, "mock-gemini")
	if err := os.WriteFile(cli, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	proc, err := StartSubprocess(ctx, tmpDir, agents.AgentConfig{ID: "gemini", Label: "Gemini", CLI: cli}, "", "sid", false, nil, nil, false, testDirective)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = proc.Write([]byte("free text\n"))
	_, _ = proc.Write([]byte("/opsx:ff my-change\n"))
	_ = proc.CloseStdin()
	_ = proc.Wait()

	var got string
	for i := 0; i < 50; i++ {
		b, _ := os.ReadFile(out)
		got = string(b)
		if strings.Count(got, testDirective) == 2 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if strings.Count(got, testDirective) != 2 {
		t.Fatalf("expected directive on both turns, got %q", got)
	}
	if !strings.Contains(got, "/opsx:explore free text\n\n"+testDirective) {
		t.Errorf("free prompt: %q", got)
	}
	if !strings.Contains(got, "/opsx:ff my-change\n\n"+testDirective) {
		t.Errorf("slash prompt: %q", got)
	}
}
