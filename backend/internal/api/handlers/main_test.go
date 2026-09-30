package handlers

import (
	"os"
	"testing"
)

// TestMain points the pool's worktrees root to a throwaway directory so tests
// that start workers never write into the real ~/.opensp8c/worktrees.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "opensp8c-worktrees-")
	if err != nil {
		panic(err)
	}
	os.Setenv("OPENSP8C_WORKTREES_DIR", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
