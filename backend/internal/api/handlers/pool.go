package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/glefebvre/opensp8c/internal/pool"
)

type PoolHandler struct {
	ws  *WorkspaceHandler
	mgr *pool.Manager
}

func NewPoolHandler(ws *WorkspaceHandler, mgr *pool.Manager) *PoolHandler {
	return &PoolHandler{
		ws:  ws,
		mgr: mgr,
	}
}

func (h *PoolHandler) StartPool(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workspacePath, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	var req pool.AgentPoolConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.mgr.Start(req, workspacePath); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *PoolHandler) StopPool(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.ws.workspacePath(id); !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	h.mgr.Stop()
	w.WriteHeader(http.StatusOK)
}

func (h *PoolHandler) GetPoolStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.ws.workspacePath(id); !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	cfg, isRunning, workers := h.mgr.Status()
	
	resp := map[string]interface{}{
		"is_running": isRunning,
		"config":     cfg,
		"workers":    workers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
