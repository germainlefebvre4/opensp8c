package benchmark

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Record kinds written to events.jsonl by the observer.
const (
	RecEvent                 = "event"
	RecTransition            = "transition"
	RecObserverStarted       = "observer_started"
	RecObserverStopped       = "observer_stopped"
	RecConnectionLost        = "connection_lost"
	RecReconnected           = "reconnected"
	RecObservationIncomplete = "observation_incomplete"
)

// ObserverRecord is one line of events.jsonl.
type ObserverRecord struct {
	Ts    time.Time `json:"ts"`
	Kind  string    `json:"kind"`
	Event string    `json:"event,omitempty"` // SSE event type
	Name  string    `json:"name,omitempty"`  // change name carried by the event
	From  string    `json:"from,omitempty"`  // previous kanban column
	To    string    `json:"to,omitempty"`    // new kanban column
	Note  string    `json:"note,omitempty"`
}

// ObserverOptions configures Observe.
type ObserverOptions struct {
	BaseURL     string // e.g. http://127.0.0.1:8123
	WorkspaceID string
	Change      string // change whose kanban column is tracked
	Out         io.Writer

	MaxReconnects int           // consecutive failed attempts before giving up
	Backoff       time.Duration // delay between attempts
	Now           func() time.Time
	HTTPClient    *http.Client
}

type observer struct {
	opt  ObserverOptions
	mu   sync.Mutex
	last string // last known kanban column of the tracked change
}

// Observe subscribes to the workspace SSE stream, records every event with its
// reception time and every kanban column transition of the tracked change. It
// returns nil when ctx is cancelled, and an error (after writing an
// observation_incomplete record) when the connection cannot be restored.
// It never modifies the platform.
func Observe(ctx context.Context, opt ObserverOptions) error {
	if opt.Now == nil {
		opt.Now = func() time.Time { return time.Now().UTC() }
	}
	if opt.Backoff == 0 {
		opt.Backoff = time.Second
	}
	if opt.MaxReconnects == 0 {
		opt.MaxReconnects = 5
	}
	if opt.HTTPClient == nil {
		opt.HTTPClient = &http.Client{}
	}
	o := &observer{opt: opt}
	o.write(ObserverRecord{Kind: RecObserverStarted, Name: opt.Change})

	failures := 0
	first := true
	for {
		connected, err := o.stream(ctx, !first)
		if ctx.Err() != nil {
			o.write(ObserverRecord{Kind: RecObserverStopped})
			return nil
		}
		if connected {
			failures = 0
		}
		first = false
		if err == nil {
			err = io.EOF
		}
		o.write(ObserverRecord{Kind: RecConnectionLost, Note: err.Error()})
		failures++
		if failures > o.opt.MaxReconnects {
			o.write(ObserverRecord{Kind: RecObservationIncomplete, Note: fmt.Sprintf("connection lost after %d attempts: %v", failures, err)})
			return fmt.Errorf("observation incomplète: %w", err)
		}
		select {
		case <-ctx.Done():
			o.write(ObserverRecord{Kind: RecObserverStopped})
			return nil
		case <-time.After(o.opt.Backoff):
		}
	}
}

// stream runs one SSE connection. connected reports whether the stream was
// established (so a later drop restarts the failure budget).
func (o *observer) stream(ctx context.Context, reconnect bool) (connected bool, err error) {
	u := strings.TrimRight(o.opt.BaseURL, "/") + "/api/workspaces/" + url.PathEscape(o.opt.WorkspaceID) + "/events"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := o.opt.HTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("events stream: HTTP %d", resp.StatusCode)
	}
	if reconnect {
		o.write(ObserverRecord{Kind: RecReconnected})
	}
	// Snapshot: the column may have changed while we were not listening (and
	// at startup, to know where the change currently is).
	o.pollColumn(ctx)

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	var evType string
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "event:"):
			evType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if evType != "ping" {
				o.handle(ctx, evType, data)
			}
		case line == "":
			evType = ""
		}
	}
	if err := sc.Err(); err != nil {
		return true, err
	}
	return true, io.EOF
}

func (o *observer) handle(ctx context.Context, evType, data string) {
	var ev struct {
		Type  string `json:"type"`
		Name  string `json:"name"`
		Error string `json:"error"`
	}
	_ = json.Unmarshal([]byte(data), &ev)
	if ev.Type == "" {
		ev.Type = evType
	}
	o.write(ObserverRecord{Kind: RecEvent, Event: ev.Type, Name: ev.Name, Note: ev.Error})
	if (ev.Type == "change_updated" && ev.Name == o.opt.Change) || ev.Type == "ff_done" || ev.Type == "change_created" && ev.Name == o.opt.Change {
		o.pollColumn(ctx)
	}
}

// pollColumn reads the tracked change's kanban column and records a transition
// when it differs from the last known one.
func (o *observer) pollColumn(ctx context.Context) {
	if o.opt.Change == "" {
		return
	}
	u := strings.TrimRight(o.opt.BaseURL, "/") + "/api/workspaces/" + url.PathEscape(o.opt.WorkspaceID) + "/changes/" + url.PathEscape(o.opt.Change)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return
	}
	resp, err := o.opt.HTTPClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return // change not on disk yet (before promotion)
	}
	var detail struct {
		KanbanStatus string `json:"kanban_status"`
	}
	if json.NewDecoder(resp.Body).Decode(&detail) != nil || detail.KanbanStatus == "" {
		return
	}
	o.mu.Lock()
	prev := o.last
	changed := prev != detail.KanbanStatus
	if changed {
		o.last = detail.KanbanStatus
	}
	o.mu.Unlock()
	if changed {
		o.write(ObserverRecord{Kind: RecTransition, Name: o.opt.Change, From: prev, To: detail.KanbanStatus})
	}
}

func (o *observer) write(r ObserverRecord) {
	r.Ts = o.opt.Now()
	data, _ := json.Marshal(r)
	o.mu.Lock()
	defer o.mu.Unlock()
	o.opt.Out.Write(append(data, '\n'))
}

// ReadObserverLog parses an events.jsonl file.
func ReadObserverLog(path string) ([]ObserverRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []ObserverRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var r ObserverRecord
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out = append(out, r)
	}
	return out, sc.Err()
}
