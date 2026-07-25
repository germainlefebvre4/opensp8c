package pool

import (
	"github.com/glefebvre/opensp8c/internal/openspec"
)

// Scheduler builds a DAG from changes and identifies which ones can be executed.
type Scheduler struct {
	changes []openspec.Change
	changeMap map[string]openspec.Change
}

// NewScheduler creates a new scheduler with the current snapshot of changes.
func NewScheduler(changes []openspec.Change) *Scheduler {
	s := &Scheduler{
		changes:   changes,
		changeMap: make(map[string]openspec.Change),
	}
	for _, c := range changes {
		s.changeMap[c.Name] = c
	}
	return s
}

// GetRunnableChanges returns a list of change names from the "todo" column
// that have no pending dependencies. A dependency is considered pending if
// it is currently in "todo", "in-progress", or "to-review".
func (s *Scheduler) GetRunnableChanges() []string {
	var runnable []string

	for _, c := range s.changes {
		if c.KanbanStatus != "todo" {
			continue
		}

		canRun := true
		for _, depName := range c.Dependencies {
			dep, exists := s.changeMap[depName]
			// If it exists and is not completely done/archived, we can't run this change.
			// If it doesn't exist in the active changes, we assume it's merged/archived/fulfilled.
			if exists && (dep.KanbanStatus == "todo" || dep.KanbanStatus == "in-progress" || dep.KanbanStatus == "to-review") {
				canRun = false
				break
			}
		}

		if canRun {
			runnable = append(runnable, c.Name)
		}
	}

	return runnable
}
