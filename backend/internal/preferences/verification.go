package preferences

import (
	"strings"

	"github.com/glefebvre/opensp8c/internal/verification"
)

// VerificationSettings are the Configuration-level verification defaults.
// Zero values mean "no setting": at this level off and absent are equivalent.
type VerificationSettings struct {
	Conformity     bool   `json:"conformity,omitempty"`
	UI             bool   `json:"ui,omitempty"`
	UIStartCommand string `json:"uiStartCommand,omitempty"`
	UIBaseURL      string `json:"uiBaseUrl,omitempty"`
}

// VerificationOverride is a partial per-workspace configuration; nil means inherit.
type VerificationOverride struct {
	Conformity     *bool   `json:"conformity,omitempty"`
	UI             *bool   `json:"ui,omitempty"`
	UIStartCommand *string `json:"uiStartCommand,omitempty"`
	UIBaseURL      *string `json:"uiBaseUrl,omitempty"`
}

func (o *VerificationOverride) isEmpty() bool {
	return o == nil || (o.Conformity == nil && o.UI == nil && o.UIStartCommand == nil && o.UIBaseURL == nil)
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
	return out
}

// level converts to the shared cascade type.
func (o *VerificationOverride) level() *verification.Level {
	if o == nil {
		return nil
	}
	return &verification.Level{
		Override:       verification.Override{Conformity: o.Conformity, UI: o.UI},
		LaunchOverride: verification.LaunchOverride{UIStartCommand: o.UIStartCommand, UIBaseURL: o.UIBaseURL},
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
	return out
}

// VerificationPatch is a partial update of the verification settings of a
// level above the change. null resets a field to inheritance.
type VerificationPatch struct {
	Conformity     verification.BoolPatch `json:"conformity"`
	UI             verification.BoolPatch `json:"ui"`
	UIStartCommand StringPatch            `json:"uiStartCommand"`
	UIBaseURL      StringPatch            `json:"uiBaseUrl"`
}

// applyVerificationPatch merges a patch into a VerificationOverride (nil = no
// override). The result is nil when nothing remains.
func applyVerificationPatch(cur *VerificationOverride, patch VerificationPatch) (*VerificationOverride, error) {
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
	if out.isEmpty() {
		return nil, nil
	}
	return out, nil
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
	next, err := applyVerificationPatch(p.VerificationDefaults.override(), patch)
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
	if *d == (VerificationSettings{}) {
		d = nil
	}
	p.VerificationDefaults = d
	return s.save(p)
}
