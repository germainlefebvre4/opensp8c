package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/preferences"
)

type KanbanHandler struct {
	ws    *WorkspaceHandler
	prefs *preferences.Service
}

func NewKanbanHandler(ws *WorkspaceHandler, prefs *preferences.Service) *KanbanHandler {
	return &KanbanHandler{ws: ws, prefs: prefs}
}

func (h *KanbanHandler) ListChanges(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	changes, err := openspec.ListChanges(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if changes == nil {
		changes = []openspec.Change{}
	}

	// Merge ghost records (app-level explorations) into the changes list.
	if h.prefs != nil {
		draftsDir := filepath.Join(filepath.Dir(h.prefs.Path()), "drafts")
		for _, e := range h.prefs.ListExplorations(id) {
			tasksDone := 0
			tasksTotal := 0

			draftPath := filepath.Join(draftsDir, e.ID+".json")
			if data, err := os.ReadFile(draftPath); err == nil {
				var draft struct {
					Tasks []struct {
						Done bool `json:"done"`
					} `json:"tasks"`
				}
				if err := json.Unmarshal(data, &draft); err == nil {
					tasksTotal = len(draft.Tasks)
					for _, t := range draft.Tasks {
						if t.Done {
							tasksDone++
						}
					}
				}
			}

			changes = append(changes, openspec.Change{
				Name:         e.Name,
				KanbanStatus: "to-explore",
				Created:      e.CreatedAt,
				IsGhost:      true,
				GhostID:      e.ID,
				TasksDone:    tasksDone,
				TasksTotal:   tasksTotal,
			})
		}
	}

	json.NewEncoder(w).Encode(changes)
}

func (h *KanbanHandler) ListArchivedChanges(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	changes, err := openspec.ListArchivedChanges(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if changes == nil {
		changes = []openspec.Change{}
	}
	json.NewEncoder(w).Encode(changes)
}

func (h *KanbanHandler) GetChange(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := chi.URLParam(r, "name")

	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	detail, err := openspec.GetChangeDetail(path, name)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "change not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(detail)
}
