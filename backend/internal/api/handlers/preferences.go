package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/language"
	"github.com/glefebvre/opensp8c/internal/openspec"
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
	agentEnv := make(map[string]map[string]string, len(agents.SupportedAgents))
	for _, a := range agents.SupportedAgents {
		e := p.AgentEnv[a.ID]
		if e == nil {
			e = map[string]string{}
		}
		agentEnv[a.ID] = e
	}
	customAgentSpecializations := p.CustomAgentSpecializations
	if customAgentSpecializations == nil {
		customAgentSpecializations = []string{}
	}
	levels := p.LanguageLevels()
	if levels.Chat == "" {
		levels.Chat = language.Auto
	}
	if levels.Documentation == "" {
		levels.Documentation = language.Auto
	}
	if levels.Code == "" {
		levels.Code = language.Default
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agentSettings":         storedView(p, ""),
		"resolvedAgentSettings": resolveView(p, ""),
		"poolDefaults":          p.ResolvePool(""),
		"agentLanguages": map[string]string{
			"chat":          levels.Chat,
			"documentation": levels.Documentation,
			"code":          levels.Code,
		},
		"uiLocale":                   p.UILocale,
		"supportedLanguages":         language.Supported(),
		"defaultAgent":               p.DefaultAgent,
		"env":                        env,
		"agentEnv":                   agentEnv,
		"systemEnv":                  systemEnv,
		"nativeQuestionMode":         p.NativeQuestionMode,
		"customAgentSpecializations": customAgentSpecializations,
	})
}

func (h *PreferencesHandler) PatchPreferences(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DefaultAgent               string                       `json:"defaultAgent"`
		Env                        map[string]string            `json:"env"`
		AgentEnv                   map[string]map[string]string `json:"agentEnv"`
		NativeQuestionMode         *bool                        `json:"nativeQuestionMode"`
		CustomAgentSpecializations []string                     `json:"customAgentSpecializations"`
		AgentLanguages             *struct {
			Chat          *string `json:"chat"`
			Documentation *string `json:"documentation"`
			Code          *string `json:"code"`
		} `json:"agentLanguages"`
		UILocale      *string                         `json:"uiLocale"`
		AgentSettings *preferences.AgentSettingsPatch `json:"agentSettings"`
		PoolDefaults  *preferences.PoolPatch          `json:"poolDefaults"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	for id := range body.AgentEnv {
		if _, ok := agents.ByID(id); !ok {
			http.Error(w, "unknown agent id", http.StatusBadRequest)
			return
		}
	}

	// Validate language settings before any write so a rejection changes nothing.
	if body.AgentLanguages != nil {
		validLevel := func(v *string, allowAuto bool) bool {
			if v == nil {
				return true
			}
			if *v == language.Auto {
				return allowAuto
			}
			_, ok := language.Lookup(*v)
			return ok
		}
		l := body.AgentLanguages
		if !validLevel(l.Chat, true) || !validLevel(l.Documentation, true) || !validLevel(l.Code, false) {
			http.Error(w, "invalid agent language", http.StatusBadRequest)
			return
		}
	}
	if body.UILocale != nil {
		if _, ok := language.Lookup(*body.UILocale); !ok {
			http.Error(w, "invalid ui locale", http.StatusBadRequest)
			return
		}
	}

	if err := h.prefs.ValidateGlobalUpdate(body.AgentSettings, body.PoolDefaults); err != nil {
		var ve *preferences.ValidationError
		if errors.As(err, &ve) {
			http.Error(w, ve.Msg, http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to load preferences", http.StatusInternalServerError)
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

	if body.AgentEnv != nil {
		if err := h.prefs.SetAgentEnv(body.AgentEnv); err != nil {
			http.Error(w, "failed to save preferences", http.StatusInternalServerError)
			return
		}
	}

	if body.NativeQuestionMode != nil {
		if err := h.prefs.SetNativeQuestionMode(*body.NativeQuestionMode); err != nil {
			http.Error(w, "failed to save preferences", http.StatusInternalServerError)
			return
		}
	}

	if body.CustomAgentSpecializations != nil {
		sanitized := openspec.SanitizeCustomSpecializations(body.CustomAgentSpecializations)
		if err := h.prefs.SetCustomAgentSpecializations(sanitized); err != nil {
			http.Error(w, "failed to save preferences", http.StatusInternalServerError)
			return
		}
	}

	if body.AgentLanguages != nil {
		update := preferences.AgentLanguagesUpdate{
			Chat:          body.AgentLanguages.Chat,
			Documentation: body.AgentLanguages.Documentation,
			Code:          body.AgentLanguages.Code,
		}
		if err := h.prefs.SetAgentLanguages(update); err != nil {
			http.Error(w, "failed to save preferences", http.StatusInternalServerError)
			return
		}
	}

	if body.AgentSettings != nil {
		if err := h.prefs.SetAgentSettings(*body.AgentSettings); err != nil {
			http.Error(w, "failed to save preferences", http.StatusInternalServerError)
			return
		}
	}

	if body.PoolDefaults != nil {
		if err := h.prefs.SetPoolDefaults(*body.PoolDefaults); err != nil {
			http.Error(w, "failed to save preferences", http.StatusInternalServerError)
			return
		}
	}

	if body.UILocale != nil {
		if err := h.prefs.SetUILocale(*body.UILocale); err != nil {
			http.Error(w, "failed to save preferences", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

type agentModelsView struct {
	Models         []agents.Model `json:"models"`
	EffortLevels   []string       `json:"effortLevels"`
	SupportsModel  bool           `json:"supportsModel"`
	SupportsEffort bool           `json:"supportsEffort"`
}

// ListAgentModels returns the model catalog and effort levels of each agent,
// keyed by agent id. Discovery failures silently fall back to the seed list.
func (h *PreferencesHandler) ListAgentModels(w http.ResponseWriter, r *http.Request) {
	out := make(map[string]agentModelsView, len(agents.SupportedAgents))
	for _, a := range agents.SupportedAgents {
		models := a.Models(r.Context())
		if models == nil {
			models = []agents.Model{}
		}
		levels := a.EffortLevels
		if levels == nil || !a.SupportsEffort() {
			levels = []string{}
		}
		out[a.ID] = agentModelsView{
			Models:         models,
			EffortLevels:   levels,
			SupportsModel:  a.SupportsModel(),
			SupportsEffort: a.SupportsEffort(),
		}
	}
	json.NewEncoder(w).Encode(out)
}
