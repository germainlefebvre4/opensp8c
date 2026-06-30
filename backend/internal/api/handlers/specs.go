package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/glefebvre/opensp8c/internal/openspec"
)

type SpecsHandler struct {
	ws *WorkspaceHandler
}

func NewSpecsHandler(ws *WorkspaceHandler) *SpecsHandler {
	return &SpecsHandler{ws: ws}
}

func (h *SpecsHandler) ListSpecs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	specs, err := openspec.ListSpecs(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if specs == nil {
		specs = []openspec.Spec{}
	}
	json.NewEncoder(w).Encode(specs)
}

func (h *SpecsHandler) GetSpec(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := chi.URLParam(r, "name")

	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	spec, err := openspec.ReadSpec(path, name)
	if err != nil {
		http.Error(w, "spec not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(spec)
}

func (h *SpecsHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	overview, err := openspec.ListSpecsWithChanges(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(overview)
}

func (h *SpecsHandler) UpdateSpec(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := chi.URLParam(r, "name")

	path, ok := h.ws.workspacePath(id)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := openspec.WriteSpec(path, name, body.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
