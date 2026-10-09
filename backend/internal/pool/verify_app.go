package pool

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/verification"
)

// Timings of the application started by the UI step. Variables so tests can
// shorten them.
var (
	uiStartTimeout   = 2 * time.Minute
	uiReadyInterval  = 500 * time.Millisecond
	uiReadyReqTimout = 2 * time.Second
)

// appTailLines is how many of the last output lines explain a failed start.
const appTailLines = 40

// ringBuffer keeps the last n lines written to it.
type ringBuffer struct {
	mu    sync.Mutex
	n     int
	lines []string
}

func (r *ringBuffer) add(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.lines) == r.n {
		copy(r.lines, r.lines[1:])
		r.lines = r.lines[:r.n-1]
	}
	r.lines = append(r.lines, line)
}

func (r *ringBuffer) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return strings.Join(r.lines, "\n")
}

// appRunner is the application under verification: a command running in its
// own process group on a free port.
type appRunner struct {
	cmd    *exec.Cmd
	port   int
	url    string
	exited chan struct{} // closed once the command is reaped
	waitEr error         // its Wait error, valid after exited is closed
	outEnd chan struct{} // closed once its output is drained
	tail   *ringBuffer
}

// freePort returns a TCP port nobody listens on right now. A race remains
// until the application binds it; it surfaces as a premature exit.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// startApp picks a free port, substitutes it in command and baseURL, then
// starts command with `sh -c` in dir. Its output goes to journal, line by line,
// and to a buffer of the last lines. The caller must call Stop.
func startApp(ctx context.Context, dir, command, baseURL, workspacePath string, journal func(string)) (*appRunner, error) {
	port, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("aucun port libre : %w", err)
	}
	command = verification.Substitute(command, port)
	url := verification.Substitute(baseURL, port)

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"PORT="+strconv.Itoa(port),
		"OPENSP8C_UI_PORT="+strconv.Itoa(port),
		"OPENSP8C_UI_URL="+url,
		"OPENSP8C_WORKSPACE_PATH="+workspacePath,
	)
	session.ApplyProcessGroup(cmd)

	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout, cmd.Stderr = w, w
	if err := cmd.Start(); err != nil {
		r.Close()
		w.Close()
		return nil, fmt.Errorf("impossible de lancer la commande : %w", err)
	}
	w.Close() // the child holds the write end

	a := &appRunner{
		cmd: cmd, port: port, url: url,
		exited: make(chan struct{}), outEnd: make(chan struct{}),
		tail: &ringBuffer{n: appTailLines},
	}
	go func() {
		defer close(a.outEnd)
		defer r.Close()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Text()
			a.tail.add(line)
			if journal != nil {
				journal(line)
			}
		}
	}()
	go func() {
		a.waitEr = cmd.Wait()
		close(a.exited)
	}()
	return a, nil
}

// URL is the base URL with the port substituted.
func (a *appRunner) URL() string { return a.url }

// WaitReady blocks until the base URL answers with a status below 500, the
// command exits, the timeout elapses or ctx ends.
func (a *appRunner) WaitReady(ctx context.Context, timeout time.Duration) error {
	client := &http.Client{
		Timeout: uiReadyReqTimout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(uiReadyInterval)
	defer tick.Stop()
	for {
		select {
		case <-a.exited:
			return a.exitError()
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if a.probe(ctx, client) {
			return nil
		}
		select {
		case <-a.exited:
			return a.exitError()
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("l'application n'est pas prête dans le délai (%s)%s", timeout, a.tailSuffix())
		case <-tick.C:
		}
	}
}

func (a *appRunner) probe(ctx context.Context, client *http.Client) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.url, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

// exitError describes a command that ended before the application was ready.
func (a *appRunner) exitError() error {
	select {
	case <-a.outEnd:
	case <-time.After(time.Second):
	}
	code := "?"
	if a.cmd.ProcessState != nil {
		code = strconv.Itoa(a.cmd.ProcessState.ExitCode())
	}
	return fmt.Errorf("la commande de lancement s'est terminée avant que l'application ne réponde (code %s)%s", code, a.tailSuffix())
}

func (a *appRunner) tailSuffix() string {
	if t := a.tail.String(); t != "" {
		return " :\n" + t
	}
	return ""
}

// Stop terminates the whole process group (SIGTERM, then SIGKILL after grace)
// and waits for the command and its output to end. It is safe to call after
// the command already exited.
func (a *appRunner) Stop(grace time.Duration) {
	session.StopProcessGroup(a.cmd, grace)
	select {
	case <-a.exited:
	case <-time.After(grace + 2*time.Second):
	}
	select {
	case <-a.outEnd:
	case <-time.After(2 * time.Second):
	}
}
