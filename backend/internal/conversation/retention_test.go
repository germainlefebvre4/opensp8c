package conversation

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/workspace"
)

func setupRetentionWorkspace(t *testing.T, archivedDirNames []string) (wsPath, wsID string) {
	t.Helper()
	wsPath = t.TempDir()
	for _, dirName := range archivedDirNames {
		archDir := filepath.Join(wsPath, "openspec", "changes", "archive", dirName)
		if err := os.MkdirAll(archDir, 0755); err != nil {
			t.Fatalf("mkdir archive: %v", err)
		}
	}
	absPath, err := filepath.Abs(wsPath)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	return wsPath, workspace.StableID(absPath)
}

func TestRunRetentionSweep_ChangeArchivedExpiredVsFresh(t *testing.T) {
	old := time.Now().UTC().AddDate(0, 0, -20).Format("2006-01-02")
	recent := time.Now().UTC().AddDate(0, 0, -2).Format("2006-01-02")

	wsPath, wsID := setupRetentionWorkspace(t, []string{
		old + "-old-change",
		recent + "-recent-change",
	})

	convDir := t.TempDir()
	store := NewStore(convDir)
	f1, _ := store.OpenRun(wsID, "old-change", "chat", "2026-01-01T00-00-00Z")
	f1.Close()
	f2, _ := store.OpenRun(wsID, "recent-change", "chat", "2026-01-01T00-00-00Z")
	f2.Close()

	cfg := &config.Config{
		Workspaces:             []config.WorkspaceConfig{{Name: "ws", Path: wsPath}},
		ChangeLogRetentionDays: 15,
	}
	prefs := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))

	RunRetentionSweep(cfg, prefs, store)

	if _, err := os.Stat(filepath.Join(convDir, wsID, "old-change")); !os.IsNotExist(err) {
		t.Errorf("expected old-change logs purged (archived %d days ago), got err=%v", 20, err)
	}
	if _, err := os.Stat(filepath.Join(convDir, wsID, "recent-change")); err != nil {
		t.Errorf("expected recent-change logs kept, got err=%v", err)
	}
}

func TestRunRetentionSweep_ExplorationExpiredVsFresh(t *testing.T) {
	wsPath, wsID := setupRetentionWorkspace(t, nil)

	convDir := t.TempDir()
	store := NewStore(convDir)
	f1, _ := store.OpenExploreRun(wsID, "ghost-old", "chat", "2026-01-01T00-00-00Z")
	f1.Close()
	f2, _ := store.OpenExploreRun(wsID, "ghost-fresh", "chat", "2026-01-01T00-00-00Z")
	f2.Close()

	prefsPath := filepath.Join(t.TempDir(), "preferences.json")
	prefs := preferences.NewService(prefsPath)
	now := time.Now().UTC()
	_ = prefs.AddExploration(preferences.ExplorationRecord{
		ID: "ghost-old", WorkspaceID: wsID, Name: "explore-ghostold",
		LastActivityAt: now.AddDate(0, 0, -20).Format(time.RFC3339),
	})
	_ = prefs.AddExploration(preferences.ExplorationRecord{
		ID: "ghost-fresh", WorkspaceID: wsID, Name: "explore-ghostfresh",
		LastActivityAt: now.AddDate(0, 0, -2).Format(time.RFC3339),
	})

	draftsDir := filepath.Join(filepath.Dir(prefsPath), "drafts")
	_ = os.MkdirAll(draftsDir, 0755)
	_ = os.WriteFile(filepath.Join(draftsDir, "ghost-old.json"), []byte("{}"), 0644)
	_ = os.WriteFile(filepath.Join(draftsDir, "ghost-fresh.json"), []byte("{}"), 0644)

	cfg := &config.Config{
		Workspaces:              []config.WorkspaceConfig{{Name: "ws", Path: wsPath}},
		ExploreLogRetentionDays: 15,
	}

	RunRetentionSweep(cfg, prefs, store)

	if _, err := os.Stat(filepath.Join(convDir, wsID, "_explore", "ghost-old")); !os.IsNotExist(err) {
		t.Errorf("expected ghost-old logs purged, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(convDir, wsID, "_explore", "ghost-fresh")); err != nil {
		t.Errorf("expected ghost-fresh logs kept, got err=%v", err)
	}

	if _, err := os.Stat(filepath.Join(draftsDir, "ghost-old.json")); !os.IsNotExist(err) {
		t.Errorf("expected ghost-old draft file deleted, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(draftsDir, "ghost-fresh.json")); err != nil {
		t.Errorf("expected ghost-fresh draft file kept, got err=%v", err)
	}
}

// A promoted exploration has its ExplorationRecord deleted at promotion time,
// so it must never be reachable through the explore rule even if its logs
// (now living under the change's own directory) look old by the same clock.
func TestRunRetentionSweep_PromotedExplorationExcludedFromExploreRule(t *testing.T) {
	wsPath, wsID := setupRetentionWorkspace(t, nil)

	convDir := t.TempDir()
	store := NewStore(convDir)
	f, _ := store.OpenRun(wsID, "promoted-change", "chat", "2026-01-01T00-00-00Z")
	f.Close()

	prefs := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	// No ExplorationRecord for "promoted-change": deleted by runPromoteFF on success.

	cfg := &config.Config{
		Workspaces:              []config.WorkspaceConfig{{Name: "ws", Path: wsPath}},
		ExploreLogRetentionDays: 15,
	}

	RunRetentionSweep(cfg, prefs, store)

	if _, err := os.Stat(filepath.Join(convDir, wsID, "promoted-change")); err != nil {
		t.Errorf("expected promoted change logs untouched by the explore rule, got err=%v", err)
	}
}
