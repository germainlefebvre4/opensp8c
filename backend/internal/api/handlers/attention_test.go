package handlers

import (
	"reflect"
	"testing"

	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/workspace"
)

func TestComputeAttention(t *testing.T) {
	tasks := map[string][]openspec.Task{
		"hitl-two":   {{Text: "a", HumanReview: true}, {Text: "b", HumanReview: true}, {Text: "c", HumanReview: true, Done: true}},
		"hitl-done":  {{Text: "a", HumanReview: true, Done: true}},
		"paused-mix": {{Text: "walk", HumanReview: true}},
		"done-one":   {{Text: "x", HumanReview: true}},
	}
	tasksFor := func(ch openspec.Change) []openspec.Task { return tasks[ch.Name] }
	changes := []openspec.Change{
		{Name: "z-review", KanbanStatus: "to-review"},
		{Name: "hitl-two", KanbanStatus: "in-progress"},
		{Name: "hitl-done", KanbanStatus: "in-progress"},
		{Name: "paused-mix", KanbanStatus: "in-progress"},
		{Name: "vfail", KanbanStatus: "verifying", VerificationState: "failed"},
		{Name: "vrun", KanbanStatus: "verifying", VerificationState: "running"},
		{Name: "vq", KanbanStatus: "verifying", VerificationState: "queued"},
		{Name: "vw", KanbanStatus: "verifying", VerificationState: "waiting"},
		{Name: "queued", KanbanStatus: "todo"},
		{Name: "running", KanbanStatus: "in-progress"},
		{Name: "done-one", KanbanStatus: "done"},
		{Name: "a-review", KanbanStatus: "to-review"},
	}
	held := map[string]heldWorker{
		"paused-mix": {Paused: true, BlockedReason: "tests en échec"},
		"running":    {},
	}
	got := ComputeAttention(changes, held, tasksFor)
	want := []workspace.Attention{
		{Change: "paused-mix", Signals: []workspace.Signal{{Kind: "paused", Reason: "tests en échec"}, {Kind: "hitl", Reason: "walk"}}},
		{Change: "vfail", Signals: []workspace.Signal{{Kind: "verify-failed"}}},
		{Change: "hitl-two", Signals: []workspace.Signal{{Kind: "hitl", Reason: "a"}, {Kind: "hitl", Reason: "b"}}},
		{Change: "a-review", Signals: []workspace.Signal{{Kind: "review"}}},
		{Change: "z-review", Signals: []workspace.Signal{{Kind: "review"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
}

func TestComputeAttention_EmptyIsNotNil(t *testing.T) {
	if got := ComputeAttention(nil, nil, nil); got == nil || len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}
