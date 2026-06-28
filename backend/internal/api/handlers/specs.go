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
