package handlers

import (
	"encoding/json"
	"net/http"
	"os"

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
	env := p.Env
	if env == nil {
		env = map[string]string{}
	}
	systemEnv := map[string]string{
		"GOOGLE_CLOUD_PROJECT": os.Getenv("GOOGLE_CLOUD_PROJECT"),
		"GEMINI_MODEL":         os.Getenv("GEMINI_MODEL"),
		"GEMINI_SANDBOX":       os.Getenv("GEMINI_SANDBOX"),
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"defaultAgent": p.DefaultAgent,
		"env":          env,
		"systemEnv":    systemEnv,
	})
}

func (h *PreferencesHandler) PatchPreferences(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DefaultAgent string            `json:"defaultAgent"`
		Env          map[string]string `json:"env"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.DefaultAgent != "" {
		if _, ok := agents.ByID(body.DefaultAgent); !ok {
			http.Error(w, "unknown agent id", http.StatusBadRequest)
			return
		}
		if err := h.prefs.SetDefaultAgent(body.DefaultAgent); err != nil {
			http.Error(w, "failed to save preferences", http.StatusInternalServerError)
			return
		}
	}

	if body.Env != nil {
		if err := h.prefs.SetEnv(body.Env); err != nil {
			http.Error(w, "failed to save preferences", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
