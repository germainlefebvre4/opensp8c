package handlers

import (
	"encoding/json"
	"net/http"
	"os"

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
		for _, e := range h.prefs.ListExplorations(id) {
			changes = append(changes, openspec.Change{
				Name:         e.Name,
				KanbanStatus: "to-explore",
				Created:      e.CreatedAt,
				IsGhost:      true,
				GhostID:      e.ID,
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
