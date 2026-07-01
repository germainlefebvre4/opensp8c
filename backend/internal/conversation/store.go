package conversation

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

// exploreNamespace prefixes exploration-scoped storage. It is not a valid
// kebab-case change name, so it can never collide with a real changeName.
const exploreNamespace = "_explore"

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

// exploreDir resolves the storage path for an anonymous exploration session,
// keyed by ghostSessionID instead of a changeName (none exists before promotion).
func (s *Store) exploreDir(wsID, ghostSessionID, kind string) string {
	return filepath.Join(s.basePath, wsID, exploreNamespace, ghostSessionID, kind)
}

// OpenExploreRun creates a new timestamped JSONL file for an exploration run and
// returns its file handle. The caller is responsible for closing the file.
func (s *Store) OpenExploreRun(wsID, ghostSessionID, kind, ts string) (*os.File, error) {
	dir := s.exploreDir(wsID, ghostSessionID, kind)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(dir, ts+".jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

// MoveExplorationLogs moves the entire log tree of an exploration session
// (_explore/<ghostSessionID>/) into the log tree of the change it was promoted
// to. It merges into an existing destination if one is present, and is a no-op
// if the exploration never had any logs written.
func (s *Store) MoveExplorationLogs(wsID, ghostSessionID, changeName string) error {
	src := filepath.Join(s.basePath, wsID, exploreNamespace, ghostSessionID)
	dst := filepath.Join(s.basePath, wsID, changeName)

	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if _, err := os.Stat(dst); err == nil {
		return moveMerge(src, dst)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return renameOrCopy(src, dst)
}

// DeleteExplorationLogs removes all logs for an anonymous exploration session.
// No-op if the exploration never had any logs written.
func (s *Store) DeleteExplorationLogs(wsID, ghostSessionID string) error {
	return os.RemoveAll(filepath.Join(s.basePath, wsID, exploreNamespace, ghostSessionID))
}

// DeleteChangeLogs removes all conversation logs for a change. No-op if none exist.
func (s *Store) DeleteChangeLogs(wsID, changeName string) error {
	return os.RemoveAll(filepath.Join(s.basePath, wsID, changeName))
}

// moveMerge moves the contents of src into dst, recursing into subdirectories
// that already exist at the destination instead of failing outright.
func moveMerge(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if _, err := os.Stat(dstPath); err == nil {
				if err := moveMerge(srcPath, dstPath); err != nil {
					return err
				}
				continue
			}
			if err := renameOrCopy(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		if _, err := os.Stat(dstPath); err == nil {
			// Name collision on a file (e.g. same timestamp): keep both rather than overwrite.
			dstPath += ".dup"
		}
		if err := renameOrCopy(srcPath, dstPath); err != nil {
			return err
		}
	}
	return os.RemoveAll(src)
}

// renameOrCopy moves src to dst, falling back to a recursive copy+delete when
// the two paths are on different filesystems (os.Rename returns EXDEV).
func renameOrCopy(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}
	if !errors.Is(err, syscall.EXDEV) {
		return err
	}
	if err := copyPath(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0644)
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
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
