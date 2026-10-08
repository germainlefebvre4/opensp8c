package pool

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// TestHelperHTTPApp is not a test: re-executed by the app tests, it serves
// HTTP on $PORT answering $HELPER_APP_STATUS, then runs until killed.
func TestHelperHTTPApp(t *testing.T) {
	status := os.Getenv("HELPER_APP_STATUS")
	if status == "" {
		t.Skip("helper process")
	}
	code, _ := strconv.Atoi(status)
	fmt.Println("helper app listening on", os.Getenv("PORT"))
	_ = http.ListenAndServe("127.0.0.1:"+os.Getenv("PORT"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if code == 302 {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		w.WriteHeader(code)
	}))
}

// helperAppCommand is a `sh -c` command running the helper app.
func helperAppCommand(status int, extra string) string {
	return fmt.Sprintf("%sHELPER_APP_STATUS=%d exec %s -test.run=TestHelperHTTPApp -test.v", extra, status, os.Args[0])
}

func shortUITimings(t *testing.T) {
	t.Helper()
	pi, pt := uiReadyInterval, uiStartTimeout
	uiReadyInterval = 20 * time.Millisecond
	t.Cleanup(func() { uiReadyInterval, uiStartTimeout = pi, pt })
}

func collectJournal() (func(string), func() []string) {
	var mu sync.Mutex
	var lines []string
	return func(l string) {
			mu.Lock()
			lines = append(lines, l)
			mu.Unlock()
		}, func() []string {
			mu.Lock()
			defer mu.Unlock()
			return append([]string(nil), lines...)
		}
}

func TestAppReadyStatuses(t *testing.T) {
	shortUITimings(t)
	for _, status := range []int{200, 302, 404} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			a, err := startApp(context.Background(), t.TempDir(), helperAppCommand(status, ""), "http://127.0.0.1:{port}", "/main", nil)
			if err != nil {
				t.Fatal(err)
			}
			defer a.Stop(time.Second)
			if err := a.WaitReady(context.Background(), 10*time.Second); err != nil {
				t.Fatalf("status %d should be ready: %v", status, err)
			}
		})
	}
}

func TestAppNeverReadyOn500(t *testing.T) {
	shortUITimings(t)
	a, err := startApp(context.Background(), t.TempDir(), helperAppCommand(500, ""), "http://127.0.0.1:{port}", "/main", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Stop(time.Second)
	err = a.WaitReady(context.Background(), 300*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "n'est pas prête dans le délai") {
		t.Fatalf("expected a start timeout, got %v", err)
	}
}

func TestAppEarlyExit(t *testing.T) {
	shortUITimings(t)
	a, err := startApp(context.Background(), t.TempDir(), "echo boom-line; echo oops >&2; exit 3", "http://127.0.0.1:{port}", "/main", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Stop(time.Second)
	err = a.WaitReady(context.Background(), 10*time.Second)
	if err == nil || !strings.Contains(err.Error(), "code 3") || !strings.Contains(err.Error(), "boom-line") || !strings.Contains(err.Error(), "oops") {
		t.Fatalf("expected the exit code and the output tail, got %v", err)
	}
}

func TestAppPortAndEnvironment(t *testing.T) {
	shortUITimings(t)
	journal, lines := collectJournal()
	cmd := `echo "ARG=$1 PORT=$PORT UIPORT=$OPENSP8C_UI_PORT URL=$OPENSP8C_UI_URL WS=$OPENSP8C_WORKSPACE_PATH PWD=$(pwd)"; ` + helperAppCommand(200, "")
	cmd = strings.Replace(cmd, `$1`, `{port}`, 1)
	dir := t.TempDir()
	a, err := startApp(context.Background(), dir, cmd, "http://127.0.0.1:{port}/app", "/main/repo", journal)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Stop(time.Second)
	if err := a.WaitReady(context.Background(), 10*time.Second); err != nil {
		t.Fatal(err)
	}
	p := strconv.Itoa(a.port)
	if a.URL() != "http://127.0.0.1:"+p+"/app" {
		t.Fatalf("url = %s", a.URL())
	}
	var got string
	for _, l := range lines() {
		if strings.HasPrefix(l, "ARG=") {
			got = l
		}
	}
	for _, want := range []string{"ARG=" + p, "PORT=" + p, "UIPORT=" + p, "URL=http://127.0.0.1:" + p + "/app", "WS=/main/repo"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q missing in %q", want, got)
		}
	}
	if resolved, _ := filepath.EvalSymlinks(dir); !strings.Contains(got, "PWD="+resolved) && !strings.Contains(got, "PWD="+dir) {
		t.Errorf("command should run in the worktree: %q", got)
	}
}

func TestAppStopKillsTheWholeGroup(t *testing.T) {
	shortUITimings(t)
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	// A child outliving the command's own shell.
	cmd := "sleep 300 & echo $! > " + pidFile + "; " + helperAppCommand(200, "")
	a, err := startApp(context.Background(), t.TempDir(), cmd, "http://127.0.0.1:{port}", "/main", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.WaitReady(context.Background(), 10*time.Second); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	child, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	a.Stop(time.Second)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && syscall.Kill(child, 0) == nil {
		time.Sleep(20 * time.Millisecond)
	}
	if syscall.Kill(child, 0) == nil {
		t.Fatal("the child of the command is still alive after Stop")
	}
	if _, err := http.Get(a.URL()); err == nil {
		t.Fatal("the application still answers after Stop")
	}
}

func TestAppStopEscalatesToKill(t *testing.T) {
	shortUITimings(t)
	// Ignores SIGTERM: only SIGKILL stops it.
	a, err := startApp(context.Background(), t.TempDir(), `trap '' TERM; while true; do sleep 1; done`, "http://127.0.0.1:{port}", "/main", nil)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	start := time.Now()
	a.Stop(200 * time.Millisecond)
	select {
	case <-a.exited:
	default:
		t.Fatal("process still running after Stop")
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("Stop took too long")
	}
}
