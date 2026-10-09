package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/glefebvre/opensp8c/internal/workspace"
	"github.com/go-chi/chi/v5"
)

type WorkspaceHandler struct {
	cfg     *config.Config
	cfgPath string
	poolReg *pool.Registry
}

func NewWorkspaceHandler(cfg *config.Config, cfgPath string, poolReg *pool.Registry) *WorkspaceHandler {
	return &WorkspaceHandler{cfg: cfg, cfgPath: cfgPath, poolReg: poolReg}
}

func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	var workspaces []workspace.Workspace
	for _, wc := range h.cfg.Workspaces {
		absPath, _ := filepath.Abs(wc.Path)
		counts := map[string]int{
			"to-explore":  0,
			"ready":       0,
			"todo":        0,
			"in-progress": 0,
			"verifying":   0,
			"to-review":   0,
			"done":        0,
		}
		id := workspace.StableID(absPath)
		attention := []workspace.Attention{}
		if changes, err := openspec.ListChanges(absPath); err == nil {
			held := applyLiveState(h.poolReg, id, absPath, changes)
			for _, ch := range changes {
				counts[ch.KanbanStatus]++
			}
			attention = ComputeAttention(changes, held, effectiveTasks(id, absPath, held))
		}
		workspaces = append(workspaces, workspace.Workspace{
			ID:         id,
			Name:       wc.Name,
			Path:       absPath,
			TaskCounts: counts,
			Attention:  attention,
		})
	}
	if workspaces == nil {
		workspaces = []workspace.Workspace{}
	}
	json.NewEncoder(w).Encode(workspaces)
}

func (h *WorkspaceHandler) Add(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	absPath, err := filepath.Abs(body.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := workspace.Validate(absPath); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// check duplicate
	id := workspace.StableID(absPath)
	for _, wc := range h.cfg.Workspaces {
		existing, _ := filepath.Abs(wc.Path)
		if workspace.StableID(existing) == id {
			http.Error(w, "workspace already exists", http.StatusConflict)
			return
		}
	}

	name := body.Name
	if name == "" {
		name = filepath.Base(absPath)
	}

	h.cfg.Workspaces = append(h.cfg.Workspaces, config.WorkspaceConfig{Name: name, Path: absPath})
	if err := h.cfg.Save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ws := workspace.Workspace{ID: id, Name: name, Path: absPath, Attention: []workspace.Attention{}}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ws)
}

func (h *WorkspaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	newList := h.cfg.Workspaces[:0]
	found := false
	for _, wc := range h.cfg.Workspaces {
		absPath, _ := filepath.Abs(wc.Path)
		if workspace.StableID(absPath) == id {
			found = true
			continue
		}
		newList = append(newList, wc)
	}
	if !found {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	h.cfg.Workspaces = newList
	if err := h.cfg.Save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.poolReg != nil {
		h.poolReg.Remove(id)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkspaceHandler) workspacePath(id string) (string, bool) {
	for _, wc := range h.cfg.Workspaces {
		absPath, _ := filepath.Abs(wc.Path)
		if workspace.StableID(absPath) == id {
			return absPath, true
		}
	}
	return "", false
}

func (h *WorkspaceHandler) workspaceName(id string) (string, bool) {
	for _, wc := range h.cfg.Workspaces {
		absPath, _ := filepath.Abs(wc.Path)
		if workspace.StableID(absPath) == id {
			return wc.Name, true
		}
	}
	return "", false
}
