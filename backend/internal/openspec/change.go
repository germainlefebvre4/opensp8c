package openspec

import (
	"bufio"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Tags struct {
	Type       string   `json:"type" yaml:"type"`
	Complexity int      `json:"complexity" yaml:"complexity"`
	Components []string `json:"components" yaml:"components"`
	Auto       bool     `json:"auto" yaml:"_auto"`
	TaggedAt   string   `json:"tagged_at" yaml:"_tagged_at"`
}

type Change struct {
	Name              string `json:"name"`
	KanbanStatus      string `json:"kanban_status"`
	TasksDone         int    `json:"tasks_done"`
	TasksTotal        int    `json:"tasks_total"`
	Created           string `json:"created"`
	Schema            string `json:"schema"`
	DaysSinceActivity int    `json:"days_since_activity"`
	IsStale           bool   `json:"is_stale"`
	Tags              *Tags  `json:"tags,omitempty"`
	IsGhost           bool   `json:"is_ghost,omitempty"`
	GhostID           string `json:"ghost_id,omitempty"`
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
	Schema  string `yaml:"schema"`
	Created string `yaml:"created"`
	Tags    *Tags  `yaml:"tags"`
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

	var changes []Change
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		ch, err := loadChange(changesDir, e.Name(), threshold)
		if err != nil {
			continue
		}
		changes = append(changes, *ch)
	}
	return changes, nil
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

func deriveStatus(done, total int) string {
	switch {
	case total == 0:
		return "to-explore"
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

	tasksPath := filepath.Join(changeDir, "tasks.md")
	done, total := parseTaskProgress(tasksPath)
	status := deriveStatus(done, total)

	daysSince := -1
	isStale := false
	if stat, statErr := os.Stat(tasksPath); statErr == nil {
		daysSince = int(time.Since(stat.ModTime()).Hours() / 24)
		if (status == "in-progress" || status == "done") && daysSince >= threshold {
			isStale = true
		}
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
	}, nil
}

func parseTaskProgress(tasksPath string) (done, total int) {
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

func GetChangeDetail(workspacePath, changeName string) (*ChangeDetail, error) {
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
	}

	tasks := parseTaskList(filepath.Join(changeDir, "tasks.md"))
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

func ToggleTask(workspacePath, changeName string, index int) error {
	tasksPath := filepath.Join(workspacePath, "openspec", "changes", changeName, "tasks.md")
	data, err := os.ReadFile(tasksPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	taskIdx := 0
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- [") {
			continue
		}
		if taskIdx == index {
			if strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]") {
				lines[i] = strings.Replace(line, "- [x]", "- [ ]", 1)
				lines[i] = strings.Replace(lines[i], "- [X]", "- [ ]", 1)
			} else {
				lines[i] = strings.Replace(line, "- [ ]", "- [x]", 1)
			}
			found = true
			break
		}
		taskIdx++
	}

	if !found {
		return os.ErrNotExist
	}
	return os.WriteFile(tasksPath, []byte(strings.Join(lines, "\n")), 0644)
}

