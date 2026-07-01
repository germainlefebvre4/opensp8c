package conversation

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// SessionLog serializes stdin/stdout/stderr traffic for a single session
// lifetime into one JSONL file. Multiple goroutines (stdin writer, stdout
// fan-out, stderr reader) may write concurrently; a mutex guarantees lines
// are never interleaved or truncated.
type SessionLog struct {
	mu   sync.Mutex
	file *os.File
}

func NewSessionLog(f *os.File) *SessionLog {
	return &SessionLog{file: f}
}

type sessionLogLine struct {
	Ts   string          `json:"ts"`
	Dir  string          `json:"dir"`
	Data json.RawMessage `json:"data"`
}

// WriteLine appends a timestamped line for a "in" or "out" message, where data
// is the raw JSON payload as exchanged with the subprocess.
func (l *SessionLog) WriteLine(dir string, data []byte) error {
	if l == nil {
		return nil
	}
	return l.write(dir, json.RawMessage(data))
}

// WriteErr appends a timestamped stderr line, encoding text as a JSON string.
func (l *SessionLog) WriteErr(text string) error {
	if l == nil {
		return nil
	}
	encoded, err := json.Marshal(text)
	if err != nil {
		return err
	}
	return l.write("err", json.RawMessage(encoded))
}

func (l *SessionLog) write(dir string, data json.RawMessage) error {
	line, err := json.Marshal(sessionLogLine{
		Ts:   time.Now().UTC().Format(time.RFC3339Nano),
		Dir:  dir,
		Data: data,
	})
	if err != nil {
		return err
	}
	line = append(line, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()
	_, err = l.file.Write(line)
	return err
}

func (l *SessionLog) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}
