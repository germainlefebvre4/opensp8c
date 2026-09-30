package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/go-chi/chi/v5"
)

// WorkspaceSettingsHandler serves the per-workspace overrides of the
// platform configuration (agent/model/effort per role, pool, environment).
type WorkspaceSettingsHandler struct {
	ws    *WorkspaceHandler
	prefs *preferences.Service
}

func NewWorkspaceSettingsHandler(ws *WorkspaceHandler, prefs *preferences.Service) *WorkspaceSettingsHandler {
	return &WorkspaceSettingsHandler{ws: ws, prefs: prefs}
}

type agentSettingsView struct {
	Global preferences.RoleSetting                      `json:"global"`
	Roles  map[preferences.Role]preferences.RoleSetting `json:"roles"`
}

type resolvedView struct {
	Global preferences.Resolved                      `json:"global"`
	Roles  map[preferences.Role]preferences.Resolved `json:"roles"`
}

func storedView(p *preferences.Preferences, workspaceID string) agentSettingsView {
	st := p.StoredSettings(workspaceID)
	return agentSettingsView{Global: st.Global, Roles: st.Roles}
}

func resolveView(p *preferences.Preferences, workspaceID string) resolvedView {
	v := resolvedView{Global: p.ResolveGlobalRow(workspaceID), Roles: map[preferences.Role]preferences.Resolved{}}
	for _, r := range preferences.Roles {
		v.Roles[r] = p.ResolveRole(workspaceID, r, "")
	}
	return v
}

func agentEnvView(src map[string]map[string]string) map[string]map[string]string {
	out := make(map[string]map[string]string, len(agents.SupportedAgents))
	for _, a := range agents.SupportedAgents {
		e := src[a.ID]
		if e == nil {
			e = map[string]string{}
		}
		out[a.ID] = e
	}
	return out
}

func (h *WorkspaceSettingsHandler) write(w http.ResponseWriter, workspaceID string) {
	p, err := h.prefs.Load()
	if err != nil {
		http.Error(w, "failed to load preferences", http.StatusInternalServerError)
		return
	}
	ov := p.WorkspaceOverrides(workspaceID)
	pool := preferences.PoolOverride{}
	if ov.Pool != nil {
		pool = *ov.Pool
	}
	env := ov.Env
	if env == nil {
		env = map[string]string{}
	}
	globalEnv := p.Env
	if globalEnv == nil {
		globalEnv = map[string]string{}
	}
	json.NewEncoder(w).Encode(map[string]any{
		"overrides": map[string]any{
			"agentSettings": storedView(p, workspaceID),
			"pool":          pool,
			"env":           env,
			"agentEnv":      agentEnvView(ov.AgentEnv),
		},
		"inherited": map[string]any{
			"agentSettings": resolveView(p, ""),
			"pool":          p.ResolvePool(""),
			"env":           globalEnv,
			"agentEnv":      agentEnvView(p.AgentEnv),
		},
		"resolved": map[string]any{
			"agentSettings": resolveView(p, workspaceID),
			"pool":          p.ResolvePool(workspaceID),
		},
	})
}

// Get returns { overrides, inherited, resolved } for the workspace.
func (h *WorkspaceSettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.ws.workspacePath(id); !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}
	h.write(w, id)
}

// Patch merges a partial update; null resets a field to inheritance.
func (h *WorkspaceSettingsHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.ws.workspacePath(id); !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}
	var body preferences.WorkspaceSettingsPatch
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.prefs.PatchWorkspace(id, body); err != nil {
		var ve *preferences.ValidationError
		if errors.As(err, &ve) {
			http.Error(w, ve.Msg, http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to save preferences", http.StatusInternalServerError)
		return
	}
	h.write(w, id)
}
