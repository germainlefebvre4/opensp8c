package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakePlatform serves an SSE stream plus the change detail endpoint.
type fakePlatform struct {
	mu     sync.Mutex
	column string
	events chan string // raw SSE blocks; closing a connection = send "" on drop
	drop   chan struct{}
	conns  int
	refuse bool
}

func newFake() *fakePlatform {
	return &fakePlatform{events: make(chan string, 16), drop: make(chan struct{}, 4)}
}

func (f *fakePlatform) setColumn(c string) { f.mu.Lock(); f.column = c; f.mu.Unlock() }

func (f *fakePlatform) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/workspaces/ws/events", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.conns++
		refuse := f.refuse
		f.mu.Unlock()
		if refuse {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		fl.Flush()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-f.drop:
				return
			case b := <-f.events:
				fmt.Fprint(w, b)
				fl.Flush()
			}
		}
	})
	mux.HandleFunc("/api/workspaces/ws/changes/c1", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		col := f.column
		f.mu.Unlock()
		if col == "" {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"kanban_status": col})
	})
	return mux
}

func sse(typ, name string) string {
	return fmt.Sprintf("event: %s\ndata: {\"type\":%q,\"name\":%q}\n\n", typ, typ, name)
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not reached in time")
}

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

func TestObserveRecordsEventsAndTransitions(t *testing.T) {
	f := newFake()
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	var out syncBuf
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Observe(ctx, ObserverOptions{BaseURL: srv.URL, WorkspaceID: "ws", Change: "c1", Out: &out, Backoff: time.Millisecond})
	}()

	f.events <- sse("ff_started", "c1")
	f.setColumn("ready")
	f.events <- sse("ff_done", "c1")
	waitFor(t, func() bool { return strings.Contains(out.String(), `"to":"ready"`) })
	f.setColumn("todo")
	f.events <- sse("change_updated", "c1")
	waitFor(t, func() bool { return strings.Contains(out.String(), `"to":"todo"`) })
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	recs, err := parseRecords(out.String())
	if err != nil {
		t.Fatal(err)
	}
	var sawFFStart, sawFFDone, sawTransition bool
	for _, r := range recs {
		if r.Ts.IsZero() {
			t.Fatal("record without timestamp")
		}
		switch {
		case r.Kind == RecEvent && r.Event == "ff_started":
			sawFFStart = true
		case r.Kind == RecEvent && r.Event == "ff_done":
			sawFFDone = true
		case r.Kind == RecTransition && r.From == "ready" && r.To == "todo":
			sawTransition = true
		}
	}
	if !sawFFStart || !sawFFDone || !sawTransition {
		t.Fatalf("missing records: ff_started=%v ff_done=%v ready->todo=%v\n%s", sawFFStart, sawFFDone, sawTransition, out.String())
	}
	if recs[len(recs)-1].Kind != RecObserverStopped {
		t.Fatal("expected observer_stopped last")
	}
}

func TestObserveConnectionLostMarksIncomplete(t *testing.T) {
	f := newFake()
	srv := httptest.NewServer(f.handler())
	defer srv.Close()

	var out syncBuf
	done := make(chan error, 1)
	go func() {
		done <- Observe(context.Background(), ObserverOptions{
			BaseURL: srv.URL, WorkspaceID: "ws", Change: "c1", Out: &out,
			Backoff: time.Millisecond, MaxReconnects: 2,
		})
	}()
	f.events <- sse("ff_started", "c1")
	waitFor(t, func() bool { return strings.Contains(out.String(), "ff_started") })
	f.mu.Lock()
	f.refuse = true // reconnection attempts will fail
	f.mu.Unlock()
	f.drop <- struct{}{}

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "observation incomplète") {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("observer did not give up")
	}
	if !strings.Contains(out.String(), RecObservationIncomplete) {
		t.Fatalf("no observation_incomplete record:\n%s", out.String())
	}
}

func TestObserveReconnects(t *testing.T) {
	f := newFake()
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	var out syncBuf
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Observe(ctx, ObserverOptions{BaseURL: srv.URL, WorkspaceID: "ws", Change: "c1", Out: &out, Backoff: time.Millisecond})
	}()
	f.events <- sse("ff_started", "c1")
	waitFor(t, func() bool { return strings.Contains(out.String(), "ff_started") })
	f.drop <- struct{}{}
	waitFor(t, func() bool { return strings.Contains(out.String(), RecReconnected) })
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), RecObservationIncomplete) {
		t.Fatal("a restored connection must not mark the run incomplete")
	}
}

func parseRecords(s string) ([]ObserverRecord, error) {
	var out []ObserverRecord
	for _, l := range strings.Split(strings.TrimSpace(s), "\n") {
		var r ObserverRecord
		if err := json.Unmarshal([]byte(l), &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}
