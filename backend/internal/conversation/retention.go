package conversation

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/workspace"
)

// StartRetentionLoop runs an immediate sweep and then repeats on the given
// interval for the lifetime of the process. Intended to be launched with `go`.
func StartRetentionLoop(cfg *config.Config, prefs *preferences.Service, convStore *Store, interval time.Duration) {
	RunRetentionSweep(cfg, prefs, convStore)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		RunRetentionSweep(cfg, prefs, convStore)
	}
}

// RunRetentionSweep applies the two conversation log retention policies
// (archived changes, inactive unpromoted explorations) to every configured
// workspace, deleting logs past their configured TTL.
func RunRetentionSweep(cfg *config.Config, prefs *preferences.Service, convStore *Store) {
	now := time.Now().UTC()
	changeTTL := time.Duration(cfg.ChangeLogRetentionDaysOrDefault()) * 24 * time.Hour
	exploreTTL := time.Duration(cfg.ExploreLogRetentionDaysOrDefault()) * 24 * time.Hour

	for _, wc := range cfg.Workspaces {
		absPath, err := filepath.Abs(wc.Path)
		if err != nil {
			continue
		}
		wsID := workspace.StableID(absPath)
		sweepArchivedChanges(convStore, wsID, absPath, now, changeTTL)
		sweepInactiveExplorations(convStore, prefs, wsID, now, exploreTTL)
	}
}

// sweepArchivedChanges deletes conversation logs for changes archived longer
// ago than ttl, based on the date encoded in the archive folder name.
func sweepArchivedChanges(convStore *Store, wsID, workspacePath string, now time.Time, ttl time.Duration) {
	archiveDir := filepath.Join(workspacePath, "openspec", "changes", "archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		archivedAt, name, ok := parseArchiveDirName(e.Name())
		if !ok || now.Sub(archivedAt) < ttl {
			continue
		}
		if err := convStore.DeleteChangeLogs(wsID, name); err != nil {
			log.Printf("[retention] failed to delete change logs %s/%s: %v", wsID, name, err)
			continue
		}
		log.Printf("[retention] deleted change logs %s/%s (archived %s)", wsID, name, archivedAt.Format("2006-01-02"))
	}
}

// parseArchiveDirName parses "<YYYY-MM-DD>-<name>" archive folder names.
func parseArchiveDirName(dirName string) (time.Time, string, bool) {
	if len(dirName) < 12 || dirName[10] != '-' {
		return time.Time{}, "", false
	}
	datePart, name := dirName[:10], dirName[11:]
	t, err := time.Parse("2006-01-02", datePart)
	if err != nil || name == "" {
		return time.Time{}, "", false
	}
	return t, name, true
}

// sweepInactiveExplorations deletes conversation logs for anonymous
// explorations whose ghost record has been inactive longer than ttl.
// Explorations that have been promoted no longer appear in ListExplorations
// (their record is deleted at promotion time), so this only ever targets
// abandoned, never-promoted explorations.
func sweepInactiveExplorations(convStore *Store, prefs *preferences.Service, wsID string, now time.Time, ttl time.Duration) {
	if prefs == nil {
		return
	}
	for _, e := range prefs.ListExplorations(wsID) {
		lastActivity, err := time.Parse(time.RFC3339, e.LastActivityAt)
		if err != nil || now.Sub(lastActivity) < ttl {
			continue
		}
		if err := convStore.DeleteExplorationLogs(wsID, e.ID); err != nil {
			log.Printf("[retention] failed to delete exploration logs %s/%s: %v", wsID, e.ID, err)
			continue
		}
		log.Printf("[retention] deleted exploration logs %s/%s (inactive since %s)", wsID, e.ID, lastActivity.Format("2006-01-02"))
	}
}
