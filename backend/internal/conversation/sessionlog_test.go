package conversation

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestSessionLog_WriteLineAndWriteErr(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "session.jsonl"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	l := NewSessionLog(f)

	if err := l.WriteLine("in", []byte(`{"a":1}`)); err != nil {
		t.Fatalf("WriteLine in: %v", err)
	}
	if err := l.WriteLine("out", []byte(`{"b":2}`)); err != nil {
		t.Fatalf("WriteLine out: %v", err)
	}
	if err := l.WriteErr("boom"); err != nil {
		t.Fatalf("WriteErr: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "session.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var lines []sessionLogLine
	for scanner.Scan() {
		var l sessionLogLine
		if err := json.Unmarshal(scanner.Bytes(), &l); err != nil {
			t.Fatalf("unmarshal line %q: %v", scanner.Text(), err)
		}
		lines = append(lines, l)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].Dir != "in" || string(lines[0].Data) != `{"a":1}` {
		t.Errorf("unexpected line 0: %+v", lines[0])
	}
	if lines[1].Dir != "out" || string(lines[1].Data) != `{"b":2}` {
		t.Errorf("unexpected line 1: %+v", lines[1])
	}
	if lines[2].Dir != "err" || string(lines[2].Data) != `"boom"` {
		t.Errorf("unexpected line 2: %+v", lines[2])
	}
	for _, l := range lines {
		if l.Ts == "" {
			t.Errorf("expected non-empty ts, got %+v", l)
		}
	}
}

// Concurrent writers (mirroring stdin/stdout/stderr goroutines in production)
// must never interleave or truncate each other's lines.
func TestSessionLog_ConcurrentWritesAreSerialized(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "session.jsonl"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	l := NewSessionLog(f)

	const writersPerDir = 20
	var wg sync.WaitGroup
	for i := 0; i < writersPerDir; i++ {
		wg.Add(3)
		go func(i int) {
			defer wg.Done()
			_ = l.WriteLine("in", []byte(fmt.Sprintf(`{"n":%d}`, i)))
		}(i)
		go func(i int) {
			defer wg.Done()
			_ = l.WriteLine("out", []byte(fmt.Sprintf(`{"n":%d}`, i)))
		}(i)
		go func(i int) {
			defer wg.Done()
			_ = l.WriteErr(fmt.Sprintf("err-%d", i))
		}(i)
	}
	wg.Wait()
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "session.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	count := 0
	for scanner.Scan() {
		var l sessionLogLine
		if err := json.Unmarshal(scanner.Bytes(), &l); err != nil {
			t.Fatalf("corrupted/interleaved line %q: %v", scanner.Text(), err)
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
	want := writersPerDir * 3
	if count != want {
		t.Fatalf("expected %d well-formed lines, got %d", want, count)
	}
}

func TestSessionLog_NilSafe(t *testing.T) {
	var l *SessionLog
	if err := l.WriteLine("in", []byte(`{}`)); err != nil {
		t.Errorf("expected nil-safe WriteLine, got: %v", err)
	}
	if err := l.WriteErr("x"); err != nil {
		t.Errorf("expected nil-safe WriteErr, got: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Errorf("expected nil-safe Close, got: %v", err)
	}
}
