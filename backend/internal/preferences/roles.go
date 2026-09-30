package preferences

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/glefebvre/opensp8c/internal/agents"
)

// Role identifies the kind of agent subprocess the platform launches.
type Role string

const (
	RoleExplorer    Role = "explorer"
	RoleFF          Role = "ff"
	RoleImplementer Role = "implementer"
	RoleFixer       Role = "fixer"
	RoleDocumenter  Role = "documenter"
)

// Roles lists every role, in display order.
var Roles = []Role{RoleExplorer, RoleFF, RoleImplementer, RoleFixer, RoleDocumenter}

// Valid reports whether r is one of the defined roles.
func (r Role) Valid() bool {
	for _, k := range Roles {
		if k == r {
			return true
		}
	}
	return false
}

// RoleSetting is one level of agent/model/effort. Empty means "inherit".
type RoleSetting struct {
	Agent  string `json:"agent,omitempty"`
	Model  string `json:"model,omitempty"`
	Effort string `json:"effort,omitempty"`
}

// IsZero reports whether no field is set.
func (s RoleSetting) IsZero() bool { return s == RoleSetting{} }

// AgentSettings holds a global level plus one level per role.
type AgentSettings struct {
	Global RoleSetting          `json:"global"`
	Roles  map[Role]RoleSetting `json:"roles,omitempty"`
}

func (a *AgentSettings) isEmpty() bool {
	return a == nil || (a.Global.IsZero() && len(a.Roles) == 0)
}

func (a *AgentSettings) prune() {
	for r, s := range a.Roles {
		if s.IsZero() {
			delete(a.Roles, r)
		}
	}
	if len(a.Roles) == 0 {
		a.Roles = nil
	}
}

func (a *AgentSettings) clone() *AgentSettings {
	out := &AgentSettings{}
	if a == nil {
		return out
	}
	out.Global = a.Global
	if len(a.Roles) > 0 {
		out.Roles = make(map[Role]RoleSetting, len(a.Roles))
		for r, s := range a.Roles {
			out.Roles[r] = s
		}
	}
	return out
}

// Built-in pool defaults, used when neither the workspace nor Configuration
// define a value.
const (
	DefaultPoolSize           = 3
	DefaultPoolDelegationMode = "hitl-review"
	DefaultPoolMaxAttempts    = 3
	MinPoolSize               = 1
	MaxPoolSize               = 5
	MaxPoolAttempts           = 10
)

// PoolSettings is a fully resolved pool configuration. Stored as
// Configuration defaults, zero fields fall back to the built-in defaults.
type PoolSettings struct {
	Size           int    `json:"size,omitempty"`
	DelegationMode string `json:"delegationMode,omitempty"`
	MaxAttempts    int    `json:"maxAttempts,omitempty"`
	// ValidationCommand is run by `sh -c` at the worktree root after each
	// agent turn; empty means "auto-detect".
	ValidationCommand string `json:"validationCommand,omitempty"`
}

// PoolOverride is a partial per-workspace pool configuration; nil means inherit.
type PoolOverride struct {
	Size              *int    `json:"size,omitempty"`
	DelegationMode    *string `json:"delegationMode,omitempty"`
	MaxAttempts       *int    `json:"maxAttempts,omitempty"`
	ValidationCommand *string `json:"validationCommand,omitempty"`
}

func (o *PoolOverride) isEmpty() bool {
	return o == nil || (o.Size == nil && o.DelegationMode == nil && o.MaxAttempts == nil && o.ValidationCommand == nil)
}

// WorkspacePrefs are the overrides of one workspace, keyed by its stable id.
type WorkspacePrefs struct {
	AgentSettings *AgentSettings               `json:"agentSettings,omitempty"`
	Pool          *PoolOverride                `json:"pool,omitempty"`
	Env           map[string]string            `json:"env,omitempty"`
	AgentEnv      map[string]map[string]string `json:"agentEnv,omitempty"`
}

func (w *WorkspacePrefs) isEmpty() bool {
	return w == nil || (w.AgentSettings.isEmpty() && w.Pool.isEmpty() && len(w.Env) == 0 && len(w.AgentEnv) == 0)
}

func (w *WorkspacePrefs) clone() *WorkspacePrefs {
	out := &WorkspacePrefs{}
	if w == nil {
		return out
	}
	if w.AgentSettings != nil {
		out.AgentSettings = w.AgentSettings.clone()
	}
	if w.Pool != nil {
		c := *w.Pool
		out.Pool = &c
	}
	out.Env = copyMap(w.Env)
	if w.AgentEnv != nil {
		out.AgentEnv = make(map[string]map[string]string, len(w.AgentEnv))
		for k, v := range w.AgentEnv {
			out.AgentEnv[k] = copyMap(v)
		}
	}
	return out
}

func copyMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// Resolved is the effective agent/model/effort for one launch.
type Resolved struct {
	Agent  string `json:"agent"`
	Model  string `json:"model"`
	Effort string `json:"effort"`
}

// Preset returns the built-in setting of a role for an agent. Only Claude has
// presets; other agents fall back to the CLI default.
func Preset(role Role, agentID string) RoleSetting {
	if agentID != "claude" {
		return RoleSetting{}
	}
	switch role {
	case RoleExplorer:
		return RoleSetting{Model: "opus", Effort: "high"}
	case RoleFF, RoleImplementer, RoleFixer:
		return RoleSetting{Model: "sonnet", Effort: "medium"}
	case RoleDocumenter:
		return RoleSetting{Model: "haiku", Effort: "low"}
	}
	return RoleSetting{}
}

// levels returns the settings to inspect, most specific first: workspace role,
// workspace global, Configuration role, Configuration global.
func (p *Preferences) levels(workspaceID string, role Role) []RoleSetting {
	var out []RoleSetting
	if p != nil && workspaceID != "" {
		if ws := p.Workspaces[workspaceID]; ws != nil && ws.AgentSettings != nil {
			out = append(out, ws.AgentSettings.Roles[role], ws.AgentSettings.Global)
		} else {
			out = append(out, RoleSetting{}, RoleSetting{})
		}
	} else {
		out = append(out, RoleSetting{}, RoleSetting{})
	}
	if p != nil && p.AgentSettings != nil {
		out = append(out, p.AgentSettings.Roles[role], p.AgentSettings.Global)
	} else {
		out = append(out, RoleSetting{}, RoleSetting{})
	}
	return out
}

func (p *Preferences) defaultAgent() string {
	if p == nil || p.DefaultAgent == "" {
		return "claude"
	}
	return p.DefaultAgent
}

// ResolveRole resolves agent, model and effort for a role in a workspace
// (empty workspaceID resolves Configuration only). It is pure and safe on a
// nil receiver. lockedAgent, when non-empty, wins over any configured agent;
// model and effort are then resolved for that agent.
func (p *Preferences) ResolveRole(workspaceID string, role Role, lockedAgent string) Resolved {
	levels := p.levels(workspaceID, role)
	def := p.defaultAgent()

	agent := lockedAgent
	if agent == "" {
		agent = def
		for _, l := range levels {
			if l.Agent != "" {
				agent = l.Agent
				break
			}
		}
	}

	// effective agent of level i: its own, else the nearest less specific one.
	effective := func(i int) string {
		for j := i; j < len(levels); j++ {
			if levels[j].Agent != "" {
				return levels[j].Agent
			}
		}
		return def
	}

	pick := func(get func(RoleSetting) string) string {
		for i, l := range levels {
			if v := get(l); v != "" && effective(i) == agent {
				return v
			}
		}
		return ""
	}

	preset := Preset(role, agent)
	res := Resolved{Agent: agent}
	res.Model = pick(func(s RoleSetting) string { return s.Model })
	if res.Model == "" {
		res.Model = preset.Model
	}
	res.Effort = pick(func(s RoleSetting) string { return s.Effort })
	if res.Effort == "" {
		res.Effort = preset.Effort
	}
	if res.Effort != "" {
		cfg, ok := agents.ByID(agent)
		if !ok || !cfg.ValidEffort(res.Effort) {
			res.Effort = ""
		}
	}
	return res
}

// ApplyRole returns cfg with Model and Effort set from the resolved role
// settings; the agent itself is chosen by the caller (see ResolveRole).
func ApplyRole(cfg agents.AgentConfig, r Resolved) agents.AgentConfig {
	cfg.Model = r.Model
	cfg.Effort = r.Effort
	return cfg
}

// ResolvePool resolves the pool configuration for a workspace: workspace
// override, then Configuration defaults, then built-in defaults. Safe on nil.
func (p *Preferences) ResolvePool(workspaceID string) PoolSettings {
	out := PoolSettings{Size: DefaultPoolSize, DelegationMode: DefaultPoolDelegationMode, MaxAttempts: DefaultPoolMaxAttempts}
	if p == nil {
		return out
	}
	if d := p.PoolDefaults; d != nil {
		if d.Size != 0 {
			out.Size = d.Size
		}
		if d.DelegationMode != "" {
			out.DelegationMode = d.DelegationMode
		}
		if d.MaxAttempts != 0 {
			out.MaxAttempts = d.MaxAttempts
		}
		out.ValidationCommand = strings.TrimSpace(d.ValidationCommand)
	}
	if ws := p.Workspaces[workspaceID]; ws != nil && ws.Pool != nil {
		if ws.Pool.Size != nil {
			out.Size = *ws.Pool.Size
		}
		if ws.Pool.DelegationMode != nil {
			out.DelegationMode = *ws.Pool.DelegationMode
		}
		if ws.Pool.MaxAttempts != nil {
			out.MaxAttempts = *ws.Pool.MaxAttempts
		}
		if ws.Pool.ValidationCommand != nil {
			out.ValidationCommand = strings.TrimSpace(*ws.Pool.ValidationCommand)
		}
	}
	return out
}

// EnvForWorkspace returns, in increasing priority: global env, agent env,
// workspace env, workspace agent env. Safe on a nil receiver.
func (p *Preferences) EnvForWorkspace(workspaceID, agentID string) map[string]string {
	out := map[string]string{}
	if p == nil {
		return out
	}
	for k, v := range p.Env {
		out[k] = v
	}
	for k, v := range p.AgentEnv[agentID] {
		out[k] = v
	}
	if ws := p.Workspaces[workspaceID]; ws != nil {
		for k, v := range ws.Env {
			out[k] = v
		}
		for k, v := range ws.AgentEnv[agentID] {
			out[k] = v
		}
	}
	return out
}

// ---- Patches and validation ----

// ValidationError marks a rejected update (HTTP 400), as opposed to an I/O failure.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

func invalid(format string, a ...any) error {
	return &ValidationError{Msg: fmt.Sprintf(format, a...)}
}

// StringPatch distinguishes an absent JSON field (Set false) from a present
// one; JSON null or "" resets the field to inheritance.
type StringPatch struct {
	Set   bool
	Value string
}

func (s *StringPatch) UnmarshalJSON(b []byte) error {
	s.Set = true
	if string(b) == "null" {
		s.Value = ""
		return nil
	}
	return json.Unmarshal(b, &s.Value)
}

// IntPatch is the integer counterpart of StringPatch; null resets.
type IntPatch struct {
	Set   bool
	Reset bool
	Value int
}

func (i *IntPatch) UnmarshalJSON(b []byte) error {
	i.Set = true
	if string(b) == "null" {
		i.Reset = true
		return nil
	}
	return json.Unmarshal(b, &i.Value)
}

// RoleSettingPatch is a partial update of one level.
type RoleSettingPatch struct {
	Agent  StringPatch `json:"agent"`
	Model  StringPatch `json:"model"`
	Effort StringPatch `json:"effort"`
}

// AgentSettingsPatch is a partial update of an AgentSettings. A null role
// value resets the whole role.
type AgentSettingsPatch struct {
	Global *RoleSettingPatch            `json:"global"`
	Roles  map[string]*RoleSettingPatch `json:"roles"`
}

// PoolPatch is a partial update of the pool configuration.
type PoolPatch struct {
	Size           IntPatch    `json:"size"`
	DelegationMode StringPatch `json:"delegationMode"`
	MaxAttempts    IntPatch    `json:"maxAttempts"`
	// ValidationCommand: null or blank resets to inheritance.
	ValidationCommand StringPatch `json:"validationCommand"`
}

func (r RoleSettingPatch) apply(s RoleSetting) RoleSetting {
	if r.Agent.Set {
		s.Agent = r.Agent.Value
	}
	if r.Model.Set {
		s.Model = r.Model.Value
	}
	if r.Effort.Set {
		s.Effort = r.Effort.Value
	}
	return s
}

func (r RoleSettingPatch) touches() bool { return r.Agent.Set || r.Model.Set || r.Effort.Set }

// ValidateModel checks the shape of a model identifier: it must never be
// interpretable as a CLI flag or split into several arguments.
func ValidateModel(model string) error {
	if strings.TrimSpace(model) == "" {
		return invalid("model must not be empty")
	}
	if strings.HasPrefix(model, "-") {
		return invalid("model must not start with a dash")
	}
	if len(model) > 100 {
		return invalid("model is too long")
	}
	for _, c := range model {
		if unicode.IsSpace(c) || unicode.IsControl(c) {
			return invalid("model must not contain spaces or control characters")
		}
	}
	return nil
}

// ValidateRoleSetting validates one resulting level.
func ValidateRoleSetting(s RoleSetting) error {
	if s.Agent != "" {
		if _, ok := agents.ByID(s.Agent); !ok {
			return invalid("unknown agent %q", s.Agent)
		}
	}
	if s.Model != "" {
		if err := ValidateModel(s.Model); err != nil {
			return err
		}
	}
	if s.Effort != "" {
		if s.Agent != "" {
			cfg, _ := agents.ByID(s.Agent)
			if !cfg.ValidEffort(s.Effort) {
				return invalid("effort %q is not supported by agent %q", s.Effort, s.Agent)
			}
		} else {
			ok := false
			for _, a := range agents.SupportedAgents {
				if a.ValidEffort(s.Effort) {
					ok = true
					break
				}
			}
			if !ok {
				return invalid("unknown effort %q", s.Effort)
			}
		}
	}
	return nil
}

// applyAgentSettingsPatch returns the settings resulting from the patch, or a
// ValidationError. The input is not modified.
func applyAgentSettingsPatch(cur *AgentSettings, patch AgentSettingsPatch) (*AgentSettings, error) {
	out := cur.clone()
	if patch.Global != nil && patch.Global.touches() {
		out.Global = patch.Global.apply(out.Global)
		if err := ValidateRoleSetting(out.Global); err != nil {
			return nil, err
		}
	}
	for name, rp := range patch.Roles {
		role := Role(name)
		if !role.Valid() {
			return nil, invalid("unknown role %q", name)
		}
		if rp == nil {
			delete(out.Roles, role)
			continue
		}
		if !rp.touches() {
			continue
		}
		if out.Roles == nil {
			out.Roles = map[Role]RoleSetting{}
		}
		next := rp.apply(out.Roles[role])
		if err := ValidateRoleSetting(next); err != nil {
			return nil, err
		}
		out.Roles[role] = next
	}
	out.prune()
	return out, nil
}

func validPoolSize(n int) bool { return n >= MinPoolSize && n <= MaxPoolSize }
func validPoolMode(m string) bool {
	return m == "full-autonomy" || m == "hitl-review"
}
func validPoolAttempts(n int) bool { return n >= 1 && n <= MaxPoolAttempts }

// applyPoolPatch merges a patch into a PoolOverride (nil = no override).
func applyPoolPatch(cur *PoolOverride, patch PoolPatch) (*PoolOverride, error) {
	out := PoolOverride{}
	if cur != nil {
		out = *cur
	}
	if patch.Size.Set {
		if patch.Size.Reset {
			out.Size = nil
		} else {
			if !validPoolSize(patch.Size.Value) {
				return nil, invalid("pool size must be between %d and %d", MinPoolSize, MaxPoolSize)
			}
			v := patch.Size.Value
			out.Size = &v
		}
	}
	if patch.DelegationMode.Set {
		if patch.DelegationMode.Value == "" {
			out.DelegationMode = nil
		} else {
			if !validPoolMode(patch.DelegationMode.Value) {
				return nil, invalid("invalid delegation mode %q", patch.DelegationMode.Value)
			}
			v := patch.DelegationMode.Value
			out.DelegationMode = &v
		}
	}
	if patch.MaxAttempts.Set {
		if patch.MaxAttempts.Reset {
			out.MaxAttempts = nil
		} else {
			if !validPoolAttempts(patch.MaxAttempts.Value) {
				return nil, invalid("max attempts must be between 1 and %d", MaxPoolAttempts)
			}
			v := patch.MaxAttempts.Value
			out.MaxAttempts = &v
		}
	}
	if patch.ValidationCommand.Set {
		if v := strings.TrimSpace(patch.ValidationCommand.Value); v == "" {
			out.ValidationCommand = nil
		} else {
			out.ValidationCommand = &v
		}
	}
	if out.isEmpty() {
		return nil, nil
	}
	return &out, nil
}

// ---- Service methods ----

// SetAgentSettings applies a partial update of the Configuration-level settings.
func (s *Service) SetAgentSettings(patch AgentSettingsPatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	next, err := applyAgentSettingsPatch(p.AgentSettings, patch)
	if err != nil {
		return err
	}
	if next.isEmpty() {
		next = nil
	}
	p.AgentSettings = next
	return s.save(p)
}

// SetPoolDefaults applies a partial update of the Configuration pool defaults.
func (s *Service) SetPoolDefaults(patch PoolPatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	cur := &PoolOverride{}
	if d := p.PoolDefaults; d != nil {
		if d.Size != 0 {
			v := d.Size
			cur.Size = &v
		}
		if d.DelegationMode != "" {
			v := d.DelegationMode
			cur.DelegationMode = &v
		}
		if d.MaxAttempts != 0 {
			v := d.MaxAttempts
			cur.MaxAttempts = &v
		}
		if d.ValidationCommand != "" {
			v := d.ValidationCommand
			cur.ValidationCommand = &v
		}
	}
	next, err := applyPoolPatch(cur, patch)
	if err != nil {
		return err
	}
	if next == nil {
		p.PoolDefaults = nil
	} else {
		d := &PoolSettings{}
		if next.Size != nil {
			d.Size = *next.Size
		}
		if next.DelegationMode != nil {
			d.DelegationMode = *next.DelegationMode
		}
		if next.MaxAttempts != nil {
			d.MaxAttempts = *next.MaxAttempts
		}
		if next.ValidationCommand != nil {
			d.ValidationCommand = *next.ValidationCommand
		}
		p.PoolDefaults = d
	}
	return s.save(p)
}

// WorkspaceSettingsPatch is a partial update of one workspace's overrides.
type WorkspaceSettingsPatch struct {
	AgentSettings *AgentSettingsPatch          `json:"agentSettings"`
	Pool          *PoolPatch                   `json:"pool"`
	Env           map[string]string            `json:"env"`
	AgentEnv      map[string]map[string]string `json:"agentEnv"`
}

// PatchWorkspace applies a partial update to a workspace section. Env and
// per-agent env replace the previous dictionaries when present. Empty
// sections are removed from the file.
func (s *Service) PatchWorkspace(workspaceID string, patch WorkspaceSettingsPatch) error {
	if workspaceID == "" {
		return invalid("workspace id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	ws := p.Workspaces[workspaceID].clone()
	if patch.AgentSettings != nil {
		next, err := applyAgentSettingsPatch(ws.AgentSettings, *patch.AgentSettings)
		if err != nil {
			return err
		}
		ws.AgentSettings = next
		if next.isEmpty() {
			ws.AgentSettings = nil
		}
	}
	if patch.Pool != nil {
		next, err := applyPoolPatch(ws.Pool, *patch.Pool)
		if err != nil {
			return err
		}
		ws.Pool = next
	}
	if patch.Env != nil {
		ws.Env = patch.Env
	}
	for id, env := range patch.AgentEnv {
		if _, ok := agents.ByID(id); !ok {
			return invalid("unknown agent %q", id)
		}
		if ws.AgentEnv == nil {
			ws.AgentEnv = map[string]map[string]string{}
		}
		if len(env) == 0 {
			delete(ws.AgentEnv, id)
		} else {
			ws.AgentEnv[id] = env
		}
	}
	if ws.isEmpty() {
		delete(p.Workspaces, workspaceID)
		if len(p.Workspaces) == 0 {
			p.Workspaces = nil
		}
	} else {
		if p.Workspaces == nil {
			p.Workspaces = map[string]*WorkspacePrefs{}
		}
		p.Workspaces[workspaceID] = ws
	}
	return s.save(p)
}

// ValidateGlobalUpdate dry-runs a Configuration-level settings/pool update
// without writing, so a caller can validate everything before its first write.
func (s *Service) ValidateGlobalUpdate(a *AgentSettingsPatch, pool *PoolPatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	if a != nil {
		if _, err := applyAgentSettingsPatch(p.AgentSettings, *a); err != nil {
			return err
		}
	}
	if pool != nil {
		if _, err := applyPoolPatch(&PoolOverride{}, *pool); err != nil {
			return err
		}
	}
	return nil
}

// StoredSettings returns the raw agent settings of a level with every role
// present (possibly empty). ws == "" selects the Configuration level.
func (p *Preferences) StoredSettings(workspaceID string) AgentSettings {
	out := AgentSettings{Roles: map[Role]RoleSetting{}}
	var src *AgentSettings
	if workspaceID == "" {
		if p != nil {
			src = p.AgentSettings
		}
	} else if p != nil {
		if ws := p.Workspaces[workspaceID]; ws != nil {
			src = ws.AgentSettings
		}
	}
	if src != nil {
		out.Global = src.Global
	}
	for _, r := range Roles {
		if src != nil {
			out.Roles[r] = src.Roles[r]
		} else {
			out.Roles[r] = RoleSetting{}
		}
	}
	return out
}

// WorkspaceOverrides returns the raw section of a workspace (never nil).
func (p *Preferences) WorkspaceOverrides(workspaceID string) *WorkspacePrefs {
	if p == nil {
		return &WorkspacePrefs{}
	}
	return p.Workspaces[workspaceID].clone()
}

// ResolveGlobalRow resolves the "global" row of a level: its agent defaults to
// the platform default agent, model and effort are the stored values.
func (p *Preferences) ResolveGlobalRow(workspaceID string) Resolved {
	st := p.StoredSettings(workspaceID)
	if workspaceID != "" {
		// A workspace inherits the Configuration global for unset fields.
		cfg := p.StoredSettings("")
		g := st.Global
		if g.Agent == "" {
			g.Agent = cfg.Global.Agent
		}
		if g.Agent == cfg.Global.Agent || cfg.Global.Agent == "" {
			if g.Model == "" {
				g.Model = cfg.Global.Model
			}
			if g.Effort == "" {
				g.Effort = cfg.Global.Effort
			}
		}
		st.Global = g
	}
	agent := st.Global.Agent
	if agent == "" {
		agent = p.defaultAgent()
	}
	return Resolved{Agent: agent, Model: st.Global.Model, Effort: st.Global.Effort}
}
