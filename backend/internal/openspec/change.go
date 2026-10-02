package openspec

import (
	"bufio"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Tags struct {
	Type                []string `json:"type" yaml:"type"`
	Complexity          int      `json:"complexity" yaml:"complexity"`
	Components          []string `json:"components" yaml:"components"`
	AgentSpecialization []string `json:"agent_specialization" yaml:"agent_specialization"`
	Auto                bool     `json:"auto" yaml:"_auto"`
	TaggedAt            string   `json:"tagged_at" yaml:"_tagged_at"`
}

type Change struct {
	Name              string   `json:"name"`
	KanbanStatus      string   `json:"kanban_status"`
	TasksDone         int      `json:"tasks_done"`
	TasksTotal        int      `json:"tasks_total"`
	Created           string   `json:"created"`
	Schema            string   `json:"schema"`
	DaysSinceActivity int      `json:"days_since_activity"`
	IsStale           bool     `json:"is_stale"`
	Tags              *Tags    `json:"tags,omitempty"`
	Dependencies      []string `json:"dependencies,omitempty"`
	IsGhost           bool     `json:"is_ghost,omitempty"`
	GhostID           string   `json:"ghost_id,omitempty"`
	WorkerActive      bool     `json:"worker_active,omitempty"`
	WorkerPaused      bool     `json:"worker_paused,omitempty"`
	Launched          bool     `json:"launched,omitempty"`
	Order             int      `json:"order,omitempty"`
}

type Task struct {
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type Artifacts struct {
	Proposal string `json:"proposal"`
	Design   string `json:"design"`
}

type ChangeDetail struct {
	Change
	Tasks     []Task    `json:"tasks"`
	Artifacts Artifacts `json:"artifacts"`
}

type openspecMeta struct {
	Schema       string   `yaml:"schema"`
	Created      string   `yaml:"created"`
	Tags         *Tags    `yaml:"tags"`
	Dependencies []string `yaml:"dependencies,omitempty"`
	Launched     *bool    `yaml:"launched,omitempty"`
	Order        *int     `yaml:"order,omitempty"`
}

type openspecProjectConfig struct {
	StaleThresholdDays int `yaml:"stale_threshold_days"`
}

func readStaleThreshold(workspacePath string) int {
	const defaultThreshold = 7
	data, err := os.ReadFile(filepath.Join(workspacePath, "openspec", "config.yaml"))
	if err != nil {
		return defaultThreshold
	}
	var cfg openspecProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil || cfg.StaleThresholdDays <= 0 {
		return defaultThreshold
	}
	return cfg.StaleThresholdDays
}

// ReviewKeySuffix is the suffix of the git config key
// (branch.feature/<change>.<suffix>) marking a change as awaiting review.
const ReviewKeySuffix = "opensp8c-review"

// ReviewKey returns the git config key holding the review marker of a change.
func ReviewKey(changeName string) string {
	return "branch.feature/" + changeName + "." + ReviewKeySuffix
}

// ReviewMarkers returns the set of changes carrying a review marker in the git
// configuration of the repository at workspacePath, in a single git call. The
// set is empty when the folder is not a git repository or git fails: the
// error is never propagated.
func ReviewMarkers(workspacePath string) map[string]bool {
	markers := map[string]bool{}
	cmd := exec.Command("git", "config", "--get-regexp", `^branch\..*\.`+ReviewKeySuffix+`$`)
	cmd.Dir = workspacePath
	out, err := cmd.Output()
	if err != nil {
		return markers
	}
	const prefix, suffix = "branch.feature/", "." + ReviewKeySuffix
	for _, line := range strings.Split(string(out), "\n") {
		key, _, _ := strings.Cut(strings.TrimSpace(line), " ")
		if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) || len(key) <= len(prefix)+len(suffix) {
			continue
		}
		markers[key[len(prefix):len(key)-len(suffix)]] = true
	}
	return markers
}

// branchExists reports whether feature/<change> exists in the repository.
func branchExists(workspacePath, changeName string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/feature/"+changeName)
	cmd.Dir = workspacePath
	return cmd.Run() == nil
}

func ListChanges(workspacePath string) ([]Change, error) {
	threshold := readStaleThreshold(workspacePath)
	changesDir := filepath.Join(workspacePath, "openspec", "changes")
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Change{}, nil
		}
		return nil, err
	}

	markers := ReviewMarkers(workspacePath)
	var changes []Change
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		ch, err := loadChange(changesDir, e.Name(), threshold)
		if err != nil {
			continue
		}
		if markers[ch.Name] && branchExists(workspacePath, ch.Name) {
			markReviewed(ch)
		}
		changes = append(changes, *ch)
	}
	return changes, nil
}

// InReview reports whether the change carries a review marker and its branch
// still exists, i.e. whether it is in the To Review column.
func InReview(workspacePath, changeName string) bool {
	return ReviewMarkers(workspacePath)[changeName] && branchExists(workspacePath, changeName)
}

// markReviewed gives ch the to-review status, which takes precedence over the
// status derived from its tasks. Staleness does not apply to a change in review.
func markReviewed(ch *Change) {
	ch.KanbanStatus = "to-review"
	ch.IsStale = false
}

func ListArchivedChanges(workspacePath string) ([]Change, error) {
	archiveDir := filepath.Join(workspacePath, "openspec", "changes", "archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Change{}, nil
		}
		return nil, err
	}

	var changes []Change
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		ch, err := loadChange(archiveDir, e.Name(), math.MaxInt)
		if err != nil {
			continue
		}
		ch.KanbanStatus = "archived"
		ch.IsStale = false
		changes = append(changes, *ch)
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Created > changes[j].Created
	})

	return changes, nil
}

func deriveStatus(done, total int, launched bool) string {
	switch {
	case total == 0:
		return "to-explore"
	case !launched:
		return "ready"
	case done == 0:
		return "todo"
	case done < total:
		return "in-progress"
	default:
		return "done"
	}
}

func loadChange(changesDir, name string, threshold int) (*Change, error) {
	changeDir := filepath.Join(changesDir, name)
	metaPath := filepath.Join(changeDir, ".openspec.yaml")

	meta := openspecMeta{}
	data, err := os.ReadFile(metaPath)
	if err == nil {
		_ = yaml.Unmarshal(data, &meta)
	}
	if meta.Tags != nil && meta.Tags.Type == nil {
		meta.Tags.Type = []string{}
	}
	if meta.Tags != nil && meta.Tags.AgentSpecialization == nil {
		meta.Tags.AgentSpecialization = []string{}
	}

	tasksPath := filepath.Join(changeDir, "tasks.md")
	done, total := ParseTaskProgress(tasksPath)
	effectiveLaunched := meta.Launched == nil || *meta.Launched
	status := deriveStatus(done, total, effectiveLaunched)

	daysSince := -1
	isStale := false
	if stat, statErr := os.Stat(tasksPath); statErr == nil {
		daysSince = int(time.Since(stat.ModTime()).Hours() / 24)
		if (status == "in-progress" || status == "done") && daysSince >= threshold {
			isStale = true
		}
	}

	order := 0
	if meta.Order != nil {
		order = *meta.Order
	}

	return &Change{
		Name:              name,
		KanbanStatus:      status,
		TasksDone:         done,
		TasksTotal:        total,
		Created:           meta.Created,
		Schema:            meta.Schema,
		DaysSinceActivity: daysSince,
		IsStale:           isStale,
		Tags:              meta.Tags,
		Dependencies:      meta.Dependencies,
		Launched:          effectiveLaunched,
		Order:             order,
	}, nil
}

// ParseTaskProgress counts checked ("- [x]") vs. total ("- [") checklist
// items in the tasks.md at tasksPath. Exported so callers outside this
// package (e.g. pool.Manager, which needs to read a worktree's tasks.md
// rather than the original change's) can reuse the same parsing behavior.
func ParseTaskProgress(tasksPath string) (done, total int) {
	f, err := os.Open(tasksPath)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "- [") {
			total++
			if strings.HasPrefix(line, "- [x]") || strings.HasPrefix(line, "- [X]") {
				done++
			}
		}
	}
	return done, total
}

// worktreeTasksPath returns the tasks.md of changeName inside a pool worker's
// worktree.
func worktreeTasksPath(worktreePath, changeName string) string {
	return filepath.Join(worktreePath, "openspec", "changes", changeName, "tasks.md")
}

// ApplyWorktreeProgress overlays on ch the progress of the tasks.md found in
// the worktree of the pool worker holding the change: task counts, kanban
// column (capped at in-progress until the merge releases the worker) and
// staleness. It is a no-op when worktreePath is empty or the worktree's
// tasks.md is missing or has no task. Returns whether the overlay applied.
func ApplyWorktreeProgress(ch *Change, workspacePath, worktreePath string) bool {
	if worktreePath == "" {
		return false
	}
	tasksPath := worktreeTasksPath(worktreePath, ch.Name)
	done, total := ParseTaskProgress(tasksPath)
	if total == 0 {
		return false
	}
	status := deriveStatus(done, total, true)
	if status == "done" {
		status = "in-progress"
	}
	inReview := ch.KanbanStatus == "to-review"
	ch.TasksDone = done
	ch.TasksTotal = total
	if !inReview {
		ch.KanbanStatus = status
	}
	ch.DaysSinceActivity = -1
	ch.IsStale = false
	if stat, err := os.Stat(tasksPath); err == nil {
		ch.DaysSinceActivity = int(time.Since(stat.ModTime()).Hours() / 24)
		ch.IsStale = !inReview && status == "in-progress" && ch.DaysSinceActivity >= readStaleThreshold(workspacePath)
	}
	return true
}

// GetChangeDetail loads a change with its tasks and artifacts. When
// worktreePath is not empty and holds a usable tasks.md for the change, the
// task list and progress come from that worktree.
func GetChangeDetail(workspacePath, changeName, worktreePath string) (*ChangeDetail, error) {
	changesDir := filepath.Join(workspacePath, "openspec", "changes")
	changeDir := filepath.Join(changesDir, changeName)

	isArchived := false
	actualChangesDir := changesDir
	if _, err := os.Stat(changeDir); err != nil {
		archiveDir := filepath.Join(changesDir, "archive")
		archivedDir := filepath.Join(archiveDir, changeName)
		if _, err2 := os.Stat(archivedDir); err2 != nil {
			return nil, err
		}
		actualChangesDir = archiveDir
		changeDir = archivedDir
		isArchived = true
	}

	threshold := readStaleThreshold(workspacePath)
	ch, err := loadChange(actualChangesDir, changeName, threshold)
	if err != nil {
		return nil, err
	}
	if isArchived {
		ch.KanbanStatus = "archived"
		ch.IsStale = false
	} else if ReviewMarkers(workspacePath)[changeName] && branchExists(workspacePath, changeName) {
		markReviewed(ch)
	}

	tasks := parseTaskList(filepath.Join(changeDir, "tasks.md"))
	if !isArchived && ApplyWorktreeProgress(ch, workspacePath, worktreePath) {
		tasks = parseTaskList(worktreeTasksPath(worktreePath, changeName))
	}
	proposal := readFileContent(filepath.Join(changeDir, "proposal.md"))
	design := readFileContent(filepath.Join(changeDir, "design.md"))

	return &ChangeDetail{
		Change:    *ch,
		Tasks:     tasks,
		Artifacts: Artifacts{Proposal: proposal, Design: design},
	}, nil
}

func parseTaskList(tasksPath string) []Task {
	f, err := os.Open(tasksPath)
	if err != nil {
		return []Task{}
	}
	defer f.Close()

	var tasks []Task
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "- [") {
			continue
		}
		done := strings.HasPrefix(line, "- [x]") || strings.HasPrefix(line, "- [X]")
		text := strings.TrimSpace(line[5:])
		tasks = append(tasks, Task{Text: text, Done: done})
	}
	return tasks
}

func readFileContent(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// SetLaunched marks a change's persistent "launched" state, promoting it
// between the Ready and To Do kanban columns without touching tasks.md.
// If .openspec.yaml does not exist, it initializes one with schema "spec-driven".
func SetLaunched(changeRoot string, launched bool) error {
	metaPath := filepath.Join(changeRoot, ".openspec.yaml")

	var meta openspecMeta
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			meta = openspecMeta{Schema: "spec-driven"}
		} else {
			return err
		}
	} else {
		if err := yaml.Unmarshal(data, &meta); err != nil {
			return err
		}
	}

	meta.Launched = &launched

	out, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, out, 0644)
}

// ClearKanbanState removes the persistent "launched"/"order" fields from a
// change's .openspec.yaml, resetting it to a clean pre-Ready state.
func ClearKanbanState(changeRoot string) error {
	metaPath := filepath.Join(changeRoot, ".openspec.yaml")

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return err
	}

	var meta openspecMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return err
	}

	meta.Launched = nil
	meta.Order = nil

	out, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, out, 0644)
}

// ReorderReady assigns sequential priority ranks (1..N) to the changes named
// in orderedNames, in that order, persisting each one's .openspec.yaml.
func ReorderReady(changesDir string, orderedNames []string) error {
	for i, name := range orderedNames {
		metaPath := filepath.Join(changesDir, name, ".openspec.yaml")

		data, err := os.ReadFile(metaPath)
		if err != nil {
			return err
		}

		var meta openspecMeta
		if err := yaml.Unmarshal(data, &meta); err != nil {
			return err
		}

		order := i + 1
		meta.Order = &order

		out, err := yaml.Marshal(meta)
		if err != nil {
			return err
		}
		if err := os.WriteFile(metaPath, out, 0644); err != nil {
			return err
		}
	}
	return nil
}

func ToggleTask(workspacePath, changeName string, index int) (string, bool, error) {
	tasksPath := filepath.Join(workspacePath, "openspec", "changes", changeName, "tasks.md")
	data, err := os.ReadFile(tasksPath)
	if err != nil {
		return "", false, err
	}

	lines := strings.Split(string(data), "\n")
	taskIdx := 0
	found := false
	var taskText string
	var done bool
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- [") {
			continue
		}
		if taskIdx == index {
			if strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]") {
				lines[i] = strings.Replace(line, "- [x]", "- [ ]", 1)
				lines[i] = strings.Replace(lines[i], "- [X]", "- [ ]", 1)
				done = false
				taskText = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, "- [x]"), "- [X]"))
			} else {
				lines[i] = strings.Replace(line, "- [ ]", "- [x]", 1)
				done = true
				taskText = strings.TrimSpace(strings.TrimPrefix(trimmed, "- [ ]"))
			}
			found = true
			break
		}
		taskIdx++
	}

	if !found {
		return "", false, os.ErrNotExist
	}
	if err := os.WriteFile(tasksPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return "", false, err
	}
	return taskText, done, nil
}
