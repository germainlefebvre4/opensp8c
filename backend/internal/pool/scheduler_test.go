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

func TestScheduler_GetRunnableChanges_SortedByOrder(t *testing.T) {
	changes := []openspec.Change{
		{
			Name:         "change-a",
			KanbanStatus: "todo",
			Order:        2,
		},
		{
			Name:         "change-b",
			KanbanStatus: "todo",
			Order:        1,
		},
	}

	scheduler := NewScheduler(changes)
	runnable := scheduler.GetRunnableChanges()

	expected := []string{"change-b", "change-a"}
	if !reflect.DeepEqual(runnable, expected) {
		t.Errorf("Expected runnable changes to be sorted by priority order %v, got %v", expected, runnable)
	}
}

func TestScheduler_ToReviewNotRunnableButBlocksDependents(t *testing.T) {
	s := NewScheduler([]openspec.Change{
		{Name: "a", KanbanStatus: "to-review"},
		{Name: "b", KanbanStatus: "todo", Dependencies: []string{"a"}},
		{Name: "c", KanbanStatus: "todo"},
	})
	if got, want := s.GetRunnableChanges(), []string{"c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("runnable = %v, want %v", got, want)
	}
}

func TestScheduler_VerifyingDependencyBlocksTodo(t *testing.T) {
	changes := []openspec.Change{
		{Name: "dep", KanbanStatus: "verifying"},
		{Name: "waiting", KanbanStatus: "todo", Dependencies: []string{"dep"}},
		{Name: "free", KanbanStatus: "todo"},
	}
	got := NewScheduler(changes).GetRunnableChanges()
	if !reflect.DeepEqual(got, []string{"free"}) {
		t.Errorf("runnable = %v, want [free]", got)
	}
}
