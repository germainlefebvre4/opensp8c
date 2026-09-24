package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/preferences"
)

type SpecializationsHandler struct {
	prefs *preferences.Service
}

func NewSpecializationsHandler(prefs *preferences.Service) *SpecializationsHandler {
	return &SpecializationsHandler{prefs: prefs}
}

func (h *SpecializationsHandler) GetAgentSpecializations(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"base":   openspec.BaseAgentSpecializations,
		"custom": customAgentSpecializationsFor(h.prefs),
	})
}

// customAgentSpecializationsFor loads the user-defined specialization extensions,
// defaulting to an empty slice if preferences fail to load or none are set.
func customAgentSpecializationsFor(prefs *preferences.Service) []string {
	if p, err := prefs.Load(); err == nil && p.CustomAgentSpecializations != nil {
		return p.CustomAgentSpecializations
	}
	return []string{}
}

// specializationVocabularyFor builds the closed vocabulary (base + user extensions)
// used to constrain the LLM tagger.
func specializationVocabularyFor(prefs *preferences.Service) []string {
	return openspec.CombineSpecializationVocabulary(customAgentSpecializationsFor(prefs))
}
