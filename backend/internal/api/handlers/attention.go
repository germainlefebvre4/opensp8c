package handlers

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/workspace"
)

// signalRank orders signals from the most to the least blocking.
var signalRank = map[string]int{
	workspace.SignalPaused:       0,
	workspace.SignalVerifyFailed: 1,
	workspace.SignalHITL:         2,
	workspace.SignalReview:       3,
}

// ComputeAttention returns the changes waiting for a user action with their
// signals, sorted by their most blocking signal then by name; the signals of a
// change follow the same order. tasksFor supplies the effective tasks of a
// change and is only called for changes that are neither done nor to-explore.
func ComputeAttention(changes []openspec.Change, held map[string]heldWorker, tasksFor func(openspec.Change) []openspec.Task) []workspace.Attention {
	out := []workspace.Attention{}
	for _, ch := range changes {
		var signals []workspace.Signal
		if hw, ok := held[ch.Name]; ok && hw.Paused {
			signals = append(signals, workspace.Signal{Kind: workspace.SignalPaused, Reason: hw.BlockedReason})
		}
		if ch.VerificationState == "failed" {
			signals = append(signals, workspace.Signal{Kind: workspace.SignalVerifyFailed})
		}
		if ch.KanbanStatus != "done" && ch.KanbanStatus != "to-explore" && tasksFor != nil {
			for _, t := range tasksFor(ch) {
				if t.HumanReview && !t.Done {
					signals = append(signals, workspace.Signal{Kind: workspace.SignalHITL, Reason: t.Text})
				}
			}
		}
		if ch.KanbanStatus == "to-review" {
			signals = append(signals, workspace.Signal{Kind: workspace.SignalReview})
		}
		if len(signals) == 0 {
			continue
		}
		sort.SliceStable(signals, func(i, j int) bool { return signalRank[signals[i].Kind] < signalRank[signals[j].Kind] })
		out = append(out, workspace.Attention{Change: ch.Name, Signals: signals})
	}
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := signalRank[out[i].Signals[0].Kind], signalRank[out[j].Signals[0].Kind]
		if ri != rj {
			return ri < rj
		}
		return out[i].Change < out[j].Change
	})
	return out
}

// effectiveTasks resolves the tasks.md of a change as the Kanban does: the
// worker's worktree, else the feature branch, else the repository.
func effectiveTasks(workspaceID, path string, held map[string]heldWorker) func(openspec.Change) []openspec.Task {
	return func(ch openspec.Change) []openspec.Task {
		if hw, ok := held[ch.Name]; ok && openspec.WorktreeHasTasks(hw.WorktreePath, ch.Name) {
			if data, err := os.ReadFile(filepath.Join(hw.WorktreePath, "openspec", "changes", ch.Name, "tasks.md")); err == nil {
				return openspec.ParseTaskListContent(string(data))
			}
		}
		if ch.HasBranch {
			if content, ok := branchTasks(workspaceID, path, ch.Name); ok {
				return openspec.ParseTaskListContent(content)
			}
		}
		if data, err := os.ReadFile(filepath.Join(path, "openspec", "changes", ch.Name, "tasks.md")); err == nil {
			return openspec.ParseTaskListContent(string(data))
		}
		return nil
	}
}
