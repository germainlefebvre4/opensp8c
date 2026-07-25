package pool

import (
	"reflect"
	"testing"

	"github.com/glefebvre/opensp8c/internal/openspec"
)

func TestScheduler_GetRunnableChanges(t *testing.T) {
	changes := []openspec.Change{
		{
			Name:         "change-a",
			KanbanStatus: "done",
		},
		{
			Name:         "change-b",
			KanbanStatus: "todo",
			Dependencies: []string{"change-a"},
		},
		{
			Name:         "change-c",
			KanbanStatus: "in-progress",
		},
		{
			Name:         "change-d",
			KanbanStatus: "todo",
			Dependencies: []string{"change-c"},
		},
		{
			Name:         "change-e",
			KanbanStatus: "todo",
			Dependencies: []string{"non-existent-dep"},
		},
		{
			Name:         "change-f",
			KanbanStatus: "todo",
		},
		{
			Name:         "change-g",
			KanbanStatus: "todo",
			Dependencies: []string{"change-b"},
		},
	}

	scheduler := NewScheduler(changes)
	runnable := scheduler.GetRunnableChanges()

	expected := []string{"change-b", "change-e", "change-f"}
	if !reflect.DeepEqual(runnable, expected) {
		t.Errorf("Expected runnable changes to be %v, got %v", expected, runnable)
	}
}
