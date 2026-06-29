package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/preferences"
)

type PreferencesHandler struct {
	prefs *preferences.Service
}

func NewPreferencesHandler(prefs *preferences.Service) *PreferencesHandler {
	return &PreferencesHandler{prefs: prefs}
}

func (h *PreferencesHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	statuses := agents.DetectAll()
	json.NewEncoder(w).Encode(statuses)
}

func (h *PreferencesHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	p, err := h.prefs.Load()
	if err != nil {
		http.Error(w, "failed to load preferences", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"defaultAgent": p.DefaultAgent})
}

func (h *PreferencesHandler) PatchPreferences(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DefaultAgent string `json:"defaultAgent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.DefaultAgent == "" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if _, ok := agents.ByID(body.DefaultAgent); !ok {
		http.Error(w, "unknown agent id", http.StatusBadRequest)
		return
	}
	if err := h.prefs.SetDefaultAgent(body.DefaultAgent); err != nil {
		http.Error(w, "failed to save preferences", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
