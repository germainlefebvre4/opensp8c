package conversation

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type RunMeta struct {
	Ts           string `json:"ts"`
	MessageCount int    `json:"messageCount"`
}

type Store struct {
	basePath string
}

func NewStore(basePath string) *Store {
	return &Store{basePath: basePath}
}

func (s *Store) dir(wsID, changeName, kind string) string {
	return filepath.Join(s.basePath, wsID, changeName, kind)
}

// OpenRun creates a new timestamped JSONL file for a run and returns its file handle.
// The caller is responsible for closing the file.
func (s *Store) OpenRun(wsID, changeName, kind, ts string) (*os.File, error) {
	dir := s.dir(wsID, changeName, kind)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(dir, ts+".jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

func (s *Store) List(wsID, changeName, kind string) ([]RunMeta, error) {
	dir := s.dir(wsID, changeName, kind)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []RunMeta{}, nil
		}
		return nil, err
	}

	var runs []RunMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		ts := strings.TrimSuffix(e.Name(), ".jsonl")
		count, _ := countLines(filepath.Join(dir, e.Name()))
		runs = append(runs, RunMeta{Ts: ts, MessageCount: count})
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].Ts > runs[j].Ts
	})
	return runs, nil
}

func (s *Store) Load(wsID, changeName, kind, ts string) ([][]byte, error) {
	path := filepath.Join(s.dir(wsID, changeName, kind), ts+".jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return [][]byte{}, nil
	}
	var lines [][]byte
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if line != "" {
			lines = append(lines, []byte(line))
		}
	}
	return lines, nil
}

func countLines(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, nil
	}
	count := 0
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if line != "" {
			count++
		}
	}
	return count, nil
}
