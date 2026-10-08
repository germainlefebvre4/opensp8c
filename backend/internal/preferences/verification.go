package preferences

import (
	"encoding/json"
	"strings"

	"github.com/glefebvre/opensp8c/internal/verification"
)

// VerificationSettings are the Configuration-level verification defaults.
// Zero values mean "no setting": at this level off and absent are equivalent.
type VerificationSettings struct {
	Conformity     bool     `json:"conformity,omitempty"`
	UI             bool     `json:"ui,omitempty"`
	UIStartCommand string   `json:"uiStartCommand,omitempty"`
	UIBaseURL      string   `json:"uiBaseUrl,omitempty"`
	UIDriver       string   `json:"uiDriver,omitempty"`
	UIMcpConfig    string   `json:"uiMcpConfig,omitempty"`
	UIAllowedTools []string `json:"uiAllowedTools,omitempty"`
	UIGuidance     string   `json:"uiGuidance,omitempty"`
}

func (v VerificationSettings) isZero() bool {
	return !v.Conformity && !v.UI && v.UIStartCommand == "" && v.UIBaseURL == "" && v.UIDriver == "" &&
		v.UIMcpConfig == "" && len(v.UIAllowedTools) == 0 && v.UIGuidance == ""
}

// VerificationOverride is a partial per-workspace configuration; nil means inherit.
type VerificationOverride struct {
	Conformity     *bool     `json:"conformity,omitempty"`
	UI             *bool     `json:"ui,omitempty"`
	UIStartCommand *string   `json:"uiStartCommand,omitempty"`
	UIBaseURL      *string   `json:"uiBaseUrl,omitempty"`
	UIDriver       *string   `json:"uiDriver,omitempty"`
	UIMcpConfig    *string   `json:"uiMcpConfig,omitempty"`
	UIAllowedTools *[]string `json:"uiAllowedTools,omitempty"`
	UIGuidance     *string   `json:"uiGuidance,omitempty"`
}

func (o *VerificationOverride) isEmpty() bool {
	return o == nil || (o.Conformity == nil && o.UI == nil && o.UIStartCommand == nil && o.UIBaseURL == nil &&
		o.UIDriver == nil && o.UIMcpConfig == nil && o.UIAllowedTools == nil && o.UIGuidance == nil)
}

func cloneStr(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func (o *VerificationOverride) clone() *VerificationOverride {
	if o == nil {
		return nil
	}
	out := &VerificationOverride{}
	if o.Conformity != nil {
		v := *o.Conformity
		out.Conformity = &v
	}
	if o.UI != nil {
		v := *o.UI
		out.UI = &v
	}
	if o.UIStartCommand != nil {
		v := *o.UIStartCommand
		out.UIStartCommand = &v
	}
	if o.UIBaseURL != nil {
		v := *o.UIBaseURL
		out.UIBaseURL = &v
	}
	out.UIDriver = cloneStr(o.UIDriver)
	out.UIMcpConfig = cloneStr(o.UIMcpConfig)
	out.UIGuidance = cloneStr(o.UIGuidance)
	if o.UIAllowedTools != nil {
		v := append([]string(nil), *o.UIAllowedTools...)
		out.UIAllowedTools = &v
	}
	return out
}

// level converts to the shared cascade type.
func (o *VerificationOverride) level() *verification.Level {
	if o == nil {
		return nil
	}
	return &verification.Level{
		Override: verification.Override{Conformity: o.Conformity, UI: o.UI},
		LaunchOverride: verification.LaunchOverride{
			UIStartCommand: o.UIStartCommand, UIBaseURL: o.UIBaseURL,
			UIDriver: o.UIDriver, UIMcpConfig: o.UIMcpConfig, UIAllowedTools: o.UIAllowedTools, UIGuidance: o.UIGuidance,
		},
	}
}

// level converts the Configuration defaults: only enabled steps and non-blank
// parameters are set, since off equals absent at this level.
func (v *VerificationSettings) level() *verification.Level {
	if v == nil {
		return nil
	}
	out := &verification.Level{}
	if v.Conformity {
		t := true
		out.Conformity = &t
	}
	if v.UI {
		t := true
		out.UI = &t
	}
	if c := strings.TrimSpace(v.UIStartCommand); c != "" {
		out.UIStartCommand = &c
	}
	if u := strings.TrimSpace(v.UIBaseURL); u != "" {
		out.UIBaseURL = &u
	}
	if d := strings.TrimSpace(v.UIDriver); d != "" {
		out.UIDriver = &d
	}
	if c := strings.TrimSpace(v.UIMcpConfig); c != "" {
		out.UIMcpConfig = &c
	}
	if tools := cleanTools(v.UIAllowedTools); len(tools) > 0 {
		out.UIAllowedTools = &tools
	}
	if g := strings.TrimSpace(v.UIGuidance); g != "" {
		out.UIGuidance = &g
	}
	return out
}

// cleanTools trims entries and drops the blank ones, without the size limit
// (a hand-edited file is not rejected at load time).
func cleanTools(in []string) []string {
	var out []string
	for _, t := range in {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func (v *VerificationSettings) override() *VerificationOverride {
	if v == nil {
		return &VerificationOverride{}
	}
	out := &VerificationOverride{}
	if v.Conformity {
		t := true
		out.Conformity = &t
	}
	if v.UI {
		t := true
		out.UI = &t
	}
	if v.UIStartCommand != "" {
		c := v.UIStartCommand
		out.UIStartCommand = &c
	}
	if v.UIBaseURL != "" {
		u := v.UIBaseURL
		out.UIBaseURL = &u
	}
	if v.UIDriver != "" {
		d := v.UIDriver
		out.UIDriver = &d
	}
	if v.UIMcpConfig != "" {
		c := v.UIMcpConfig
		out.UIMcpConfig = &c
	}
	if len(v.UIAllowedTools) > 0 {
		t := append([]string(nil), v.UIAllowedTools...)
		out.UIAllowedTools = &t
	}
	if v.UIGuidance != "" {
		g := v.UIGuidance
		out.UIGuidance = &g
	}
	return out
}

// StringListPatch distinguishes an absent JSON field (Set false) from a
// present one; null resets the field, as does a list without a non-blank entry.
type StringListPatch struct {
	Set   bool
	Value []string
}

func (s *StringListPatch) UnmarshalJSON(b []byte) error {
	s.Set = true
	if string(b) == "null" {
		s.Value = nil
		return nil
	}
	return json.Unmarshal(b, &s.Value)
}

// VerificationPatch is a partial update of the verification settings of a
// level above the change. null resets a field to inheritance.
type VerificationPatch struct {
	Conformity     verification.BoolPatch `json:"conformity"`
	UI             verification.BoolPatch `json:"ui"`
	UIStartCommand StringPatch            `json:"uiStartCommand"`
	UIBaseURL      StringPatch            `json:"uiBaseUrl"`
	UIDriver       StringPatch            `json:"uiDriver"`
	UIMcpConfig    StringPatch            `json:"uiMcpConfig"`
	UIAllowedTools StringListPatch        `json:"uiAllowedTools"`
	UIGuidance     StringPatch            `json:"uiGuidance"`
}

// applyVerificationPatch merges a patch into a VerificationOverride (nil = no
// override). The result is nil when nothing remains. allowChrome is false for
// the Configuration level: the chrome driver can only be chosen per workspace.
func applyVerificationPatch(cur *VerificationOverride, patch VerificationPatch, allowChrome bool) (*VerificationOverride, error) {
	out := cur.clone()
	if out == nil {
		out = &VerificationOverride{}
	}
	out.Conformity = patch.Conformity.Apply(out.Conformity)
	out.UI = patch.UI.Apply(out.UI)
	if patch.UIStartCommand.Set {
		if v := verification.NormalizeText(patch.UIStartCommand.Value); v == "" {
			out.UIStartCommand = nil
		} else {
			out.UIStartCommand = &v
		}
	}
	if patch.UIBaseURL.Set {
		v := verification.NormalizeText(patch.UIBaseURL.Value)
		if err := verification.ValidateBaseURL(v); err != nil {
			return nil, invalid("%s", err.Error())
		}
		if v == "" {
			out.UIBaseURL = nil
		} else {
			out.UIBaseURL = &v
		}
	}
	if patch.UIDriver.Set {
		d, err := verification.ParseDriver(patch.UIDriver.Value)
		if err != nil {
			return nil, invalid("%s", err.Error())
		}
		if d == verification.DriverChrome && !allowChrome {
			return nil, invalid("uiDriver chrome can only be set for a workspace")
		}
		out.UIDriver = nilIfEmpty(d)
	}
	if patch.UIMcpConfig.Set {
		out.UIMcpConfig = nilIfEmpty(verification.NormalizeText(patch.UIMcpConfig.Value))
	}
	if patch.UIAllowedTools.Set {
		tools, err := verification.NormalizeTools(patch.UIAllowedTools.Value)
		if err != nil {
			return nil, invalid("%s", err.Error())
		}
		if len(tools) == 0 {
			out.UIAllowedTools = nil
		} else {
			out.UIAllowedTools = &tools
		}
	}
	if patch.UIGuidance.Set {
		if err := verification.ValidateGuidance(patch.UIGuidance.Value); err != nil {
			return nil, invalid("%s", err.Error())
		}
		out.UIGuidance = nilIfEmpty(verification.NormalizeText(patch.UIGuidance.Value))
	}
	if out.isEmpty() {
		return nil, nil
	}
	return out, nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ResolveVerification resolves the verification settings of a workspace
// (empty id resolves Configuration only) given the change-level override read
// from the main repository. Safe on a nil receiver.
func (p *Preferences) ResolveVerification(workspaceID string, change *verification.Override) verification.Resolved {
	if p == nil {
		return verification.Resolve(nil, nil, change)
	}
	var ws *VerificationOverride
	if workspaceID != "" {
		if w := p.Workspaces[workspaceID]; w != nil {
			ws = w.Verification
		}
	}
	return verification.Resolve(p.VerificationDefaults.level(), ws.level(), change)
}

// SetVerificationDefaults applies a partial update of the Configuration
// verification defaults.
func (s *Service) SetVerificationDefaults(patch VerificationPatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.load()
	if err != nil {
		return err
	}
	next, err := applyVerificationPatch(p.VerificationDefaults.override(), patch, false)
	if err != nil {
		return err
	}
	if next == nil {
		p.VerificationDefaults = nil
		return s.save(p)
	}
	d := &VerificationSettings{}
	if next.Conformity != nil {
		d.Conformity = *next.Conformity
	}
	if next.UI != nil {
		d.UI = *next.UI
	}
	if next.UIStartCommand != nil {
		d.UIStartCommand = *next.UIStartCommand
	}
	if next.UIBaseURL != nil {
		d.UIBaseURL = *next.UIBaseURL
	}
	if next.UIDriver != nil {
		d.UIDriver = *next.UIDriver
	}
	if next.UIMcpConfig != nil {
		d.UIMcpConfig = *next.UIMcpConfig
	}
	if next.UIAllowedTools != nil {
		d.UIAllowedTools = *next.UIAllowedTools
	}
	if next.UIGuidance != nil {
		d.UIGuidance = *next.UIGuidance
	}
	if d.isZero() {
		d = nil
	}
	p.VerificationDefaults = d
	return s.save(p)
}
