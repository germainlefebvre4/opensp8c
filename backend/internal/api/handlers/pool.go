package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/glefebvre/opensp8c/internal/pool"
	"github.com/go-chi/chi/v5"
)

type PoolHandler struct {
	ws  *WorkspaceHandler
	reg *pool.Registry
}

func NewPoolHandler(ws *WorkspaceHandler, reg *pool.Registry) *PoolHandler {
	return &PoolHandler{
		ws:  ws,
		reg: reg,
	}
}

func (h *PoolHandler) StartPool(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workspacePath, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}
	workspaceName, _ := h.ws.workspaceName(id)

	var req pool.AgentPoolConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.reg.For(id).Start(req, id, workspaceName, workspacePath); err != nil {
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

	h.reg.For(id).Stop()
	w.WriteHeader(http.StatusOK)
}

func (h *PoolHandler) GetPoolStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.ws.workspacePath(id); !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	cfg, isRunning, workers := h.reg.For(id).Status(id)
	if workers == nil {
		workers = []pool.Worker{}
	}

	resp := map[string]interface{}{
		"is_running": isRunning,
		"config":     cfg,
		"workers":    workers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ListAllPools returns every currently running pool across all workspaces,
// grouped by pool of origin with its configured size and delegation mode,
// each with the detail of its active and paused workers. It is not scoped to
// a single workspace.
func (h *PoolHandler) ListAllPools(w http.ResponseWriter, r *http.Request) {
	pools := h.reg.AllPools()
	if pools == nil {
		pools = []pool.PoolSummary{}
	}
	for i := range pools {
		if pools[i].Workers == nil {
			pools[i].Workers = []pool.Worker{}
		}
	}

	resp := map[string]interface{}{
		"pools": pools,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
