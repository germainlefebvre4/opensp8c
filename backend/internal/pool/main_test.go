package pool

import (
	"context"
	"os"
	"testing"
)

// TestMain points the worktrees root to a throwaway directory so no test can
// ever write into the real ~/.opensp8c/worktrees.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "opensp8c-worktrees-")
	if err != nil {
		panic(err)
	}
	os.Setenv("OPENSP8C_WORKTREES_DIR", dir)
	// Commits made by the pool need an identity independent of the machine.
	for k, v := range map[string]string{
		"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@t",
		"GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@t",
		"GIT_CONFIG_COUNT": "1", "GIT_CONFIG_KEY_0": "commit.gpgsign", "GIT_CONFIG_VALUE_0": "false",
	} {
		os.Setenv(k, v)
	}
	// Never run the real claude CLI to probe its flags.
	permissionPromptsNone = func(context.Context) bool { return true }
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
