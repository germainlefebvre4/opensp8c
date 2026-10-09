package pool

import (
	"errors"
	"io"
	"testing"
	"time"
)

func textTarget() (turnTarget, *string) {
	var activity string
	return turnTarget{
		setActivity: func(a string) { activity = a },
		notify:      func() {},
	}, &activity
}

func runTextTurn(t *testing.T, lines ...string) (string, error) {
	t.Helper()
	proc, stdinR, stdoutW := newPipeSubprocess()
	go func() { _, _ = io.Copy(io.Discard, stdinR) }()
	go func() {
		for _, l := range lines {
			_, _ = stdoutW.Write([]byte(l + "\n"))
		}
		_ = stdoutW.Close()
	}()
	tt, _ := textTarget()
	return (&Manager{}).runTurnText(tt, proc, "go")
}

func TestRunTurnText_ResultText(t *testing.T) {
	got, err := runTextTurn(t,
		`{"type":"content_block_delta","delta":{"text":"partial"}}`,
		`{"type":"result","subtype":"success","result":"final report\nVERDICT: PASS"}`)
	if err != nil || got != "final report\nVERDICT: PASS" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestRunTurnText_FallsBackToDeltas(t *testing.T) {
	got, err := runTextTurn(t,
		`{"type":"content_block_delta","delta":{"text":"hello "}}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"text":"world"}}}`,
		`{"type":"content_block_delta","delta":{"thinking":"ignored"}}`,
		`{"type":"message_complete","result":" "}`)
	if err != nil || got != "hello world" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestRunTurnText_NoText(t *testing.T) {
	got, err := runTextTurn(t, `{"type":"result","subtype":"success"}`)
	if err != nil || got != "" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestRunTurnText_TurnError(t *testing.T) {
	_, err := runTextTurn(t, `{"type":"result","subtype":"error_max_turns","is_error":true,"result":"boom"}`)
	var te *agentTurnError
	if !errors.As(err, &te) {
		t.Fatalf("expected agentTurnError, got %v", err)
	}
}

func TestRunTurnText_Idle(t *testing.T) {
	prev := agentIdleTimeout
	agentIdleTimeout = 50 * time.Millisecond
	defer func() { agentIdleTimeout = prev }()

	proc, stdinR, _ := newPipeSubprocess()
	go func() { _, _ = io.Copy(io.Discard, stdinR) }()
	cancelled := false
	tt, _ := textTarget()
	tt.procCancel = func() { cancelled = true }
	_, err := (&Manager{}).runTurnText(tt, proc, "go")
	var ie *agentIdleError
	if !errors.As(err, &ie) || !cancelled {
		t.Fatalf("expected idle error and cancellation, got %v (cancelled=%v)", err, cancelled)
	}
}
