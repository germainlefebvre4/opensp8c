// Package acceptance is the black-box judge of the benchmark: it builds the
// server of the clone it is copied into, serves a fixture workspace and checks
// the HTTP contract of GET /api/workspaces/{id}/changes/{name}/stats.
// It only uses the standard library and never imports the implementation.
package acceptance

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const change = "change-stats-endpoint"

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func line(ts, dir, data string) string {
	return fmt.Sprintf(`{"ts":%q,"dir":%q,"data":%s}`, ts, dir, data) + "\n"
}

// startServer builds the clone's server and starts it on the fixture.
// It returns the base URL and the workspace id.
func startServer(t *testing.T) (string, string) {
	t.Helper()
	tmp := t.TempDir()

	bin := filepath.Join(tmp, "server")
	build := exec.Command("go", "build", "-o", bin, "./cmd/server")
	build.Dir = ".." // backend/
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build server: %v\n%s", err, out)
	}

	ws := filepath.Join(tmp, "workspace")
	write(t, filepath.Join(ws, "openspec", "config.yaml"), "schema: spec-driven\n")
	changeDir := filepath.Join(ws, "openspec", "changes", change)
	write(t, filepath.Join(changeDir, ".openspec.yaml"), "schema: spec-driven\ncreated: \"2026-01-01\"\n")
	write(t, filepath.Join(changeDir, "proposal.md"), "# Proposal\n")
	write(t, filepath.Join(changeDir, "tasks.md"), "# Tasks\n\n- [x] one\n- [x] two\n- [X] three\n- [ ] four\n\nnot a task\n")
	// A change with no run at all.
	empty := filepath.Join(ws, "openspec", "changes", "no-runs")
	write(t, filepath.Join(empty, ".openspec.yaml"), "schema: spec-driven\ncreated: \"2026-01-01\"\n")
	write(t, filepath.Join(empty, "tasks.md"), "# Tasks\n")

	cfgDir := filepath.Join(tmp, "platform")
	cfgPath := filepath.Join(cfgDir, "config.yaml")
	write(t, cfgPath, "workspaces:\n  - name: fixture\n    path: "+ws+"\n")

	port := freePort(t)
	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), "CONFIG_PATH="+cfgPath, fmt.Sprintf("PORT=%d", port), "HOST=127.0.0.1")
	cmd.Dir = tmp
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	var wsID string
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/api/workspaces")
		if err == nil {
			var list []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			json.NewDecoder(resp.Body).Decode(&list)
			resp.Body.Close()
			if len(list) == 1 {
				wsID = list[0].ID
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if wsID == "" {
		t.Fatal("server did not become ready")
	}

	// Pool runs: 90 s + 30 s + one unfinished run. A chat run must not count.
	pool := filepath.Join(cfgDir, "conversations", wsID, change, "pool")
	write(t, filepath.Join(pool, "2026-01-01T10-00-00Z.jsonl"),
		line("2026-01-01T10:00:00.000Z", "meta", `{"type":"pool_run_start","worker_id":1}`)+
			line("2026-01-01T10:00:30.000Z", "out", `{"type":"assistant"}`)+
			line("2026-01-01T10:01:30.000Z", "meta", `{"type":"pool_run_end","outcome":"paused","reason":"x"}`))
	write(t, filepath.Join(pool, "2026-01-01T10-10-00Z.jsonl"),
		line("2026-01-01T10:10:00.000Z", "meta", `{"type":"pool_run_start","worker_id":1}`)+
			line("2026-01-01T10:10:30.000Z", "meta", `{"type":"pool_run_end","outcome":"completed","reason":""}`))
	write(t, filepath.Join(pool, "2026-01-01T10-20-00Z.jsonl"),
		line("2026-01-01T10:20:00.000Z", "meta", `{"type":"pool_run_start","worker_id":1}`)+
			line("2026-01-01T10:20:05.000Z", "out", `{"type":"assistant"}`))
	write(t, filepath.Join(cfgDir, "conversations", wsID, change, "chat", "2026-01-01T09-00-00Z.jsonl"),
		line("2026-01-01T09:00:00.000Z", "in", `{"type":"user"}`)+line("2026-01-01T09:05:00.000Z", "out", `{"type":"assistant"}`))
	return base, wsID
}

func get(t *testing.T, url string) (int, []byte) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}

type stats struct {
	TasksTotal           *int     `json:"tasks_total"`
	TasksDone            *int     `json:"tasks_done"`
	ProgressPercent      *int     `json:"progress_percent"`
	RunsCount            *int     `json:"runs_count"`
	TotalDurationSeconds *float64 `json:"total_duration_seconds"`
}

func decode(t *testing.T, body []byte) stats {
	t.Helper()
	var s stats
	if err := json.Unmarshal(body, &s); err != nil {
		t.Fatalf("response is not the expected JSON: %v\n%s", err, body)
	}
	if s.TasksTotal == nil || s.TasksDone == nil || s.ProgressPercent == nil || s.RunsCount == nil || s.TotalDurationSeconds == nil {
		t.Fatalf("missing field in %s", body)
	}
	return s
}

func TestStatsEndpoint(t *testing.T) {
	base, id := startServer(t)

	t.Run("change with runs", func(t *testing.T) {
		code, body := get(t, base+"/api/workspaces/"+id+"/changes/"+change+"/stats")
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", code, body)
		}
		s := decode(t, body)
		if *s.TasksTotal != 4 || *s.TasksDone != 3 || *s.ProgressPercent != 75 {
			t.Errorf("tasks: total=%d done=%d progress=%d, want 4/3/75", *s.TasksTotal, *s.TasksDone, *s.ProgressPercent)
		}
		if *s.RunsCount != 3 {
			t.Errorf("runs_count = %d, want 3 (pool runs only, unfinished one included)", *s.RunsCount)
		}
		if d := *s.TotalDurationSeconds - 120; d < -0.001 || d > 0.001 {
			t.Errorf("total_duration_seconds = %v, want 120", *s.TotalDurationSeconds)
		}
	})

	t.Run("change without run", func(t *testing.T) {
		code, body := get(t, base+"/api/workspaces/"+id+"/changes/no-runs/stats")
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", code, body)
		}
		s := decode(t, body)
		if *s.TasksTotal != 0 || *s.ProgressPercent != 0 || *s.RunsCount != 0 || *s.TotalDurationSeconds != 0 {
			t.Errorf("unexpected stats for an empty change: %s", body)
		}
	})

	t.Run("unknown change", func(t *testing.T) {
		code, _ := get(t, base+"/api/workspaces/"+id+"/changes/does-not-exist/stats")
		if code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", code)
		}
	})
}
