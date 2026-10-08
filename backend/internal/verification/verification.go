// Package verification holds the shared model of the automatic verification
// settings (conformity and UI): their three levels, the pure cascade that
// resolves them and the validation of the UI launch parameters.
package verification

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Override holds the on/off values of one level; nil means "inherit".
type Override struct {
	Conformity *bool `json:"conformity,omitempty" yaml:"conformity,omitempty"`
	UI         *bool `json:"ui,omitempty" yaml:"ui,omitempty"`
}

// IsEmpty reports whether the level defines nothing.
func (o *Override) IsEmpty() bool { return o == nil || (o.Conformity == nil && o.UI == nil) }

// Clone returns an independent copy (nil stays nil).
func (o *Override) Clone() *Override {
	if o == nil {
		return nil
	}
	out := &Override{}
	if o.Conformity != nil {
		v := *o.Conformity
		out.Conformity = &v
	}
	if o.UI != nil {
		v := *o.UI
		out.UI = &v
	}
	return out
}

// LaunchParams are the parameters of the UI verification launch.
type LaunchParams struct {
	UIStartCommand string `json:"uiStartCommand,omitempty"`
	UIBaseURL      string `json:"uiBaseUrl,omitempty"`
}

// LaunchOverride is a partial LaunchParams; nil means "inherit".
type LaunchOverride struct {
	UIStartCommand *string `json:"uiStartCommand,omitempty"`
	UIBaseURL      *string `json:"uiBaseUrl,omitempty"`
}

// IsEmpty reports whether no launch parameter is set.
func (l LaunchOverride) IsEmpty() bool { return l.UIStartCommand == nil && l.UIBaseURL == nil }

// Level is one level above the change (platform or workspace): the on/off
// values plus the launch parameters, which a change cannot define.
type Level struct {
	Override
	LaunchOverride
}

// Resolved is the effective value of each step and the launch parameters.
type Resolved struct {
	Conformity bool `json:"conformity"`
	UI         bool `json:"ui"`
	LaunchParams
}

func pickBool(def bool, levels ...*bool) bool {
	for _, v := range levels {
		if v != nil {
			return *v
		}
	}
	return def
}

func pickText(levels ...*string) string {
	out := ""
	for _, v := range levels {
		if v == nil {
			continue
		}
		if t := strings.TrimSpace(*v); t != "" {
			out = t
			break
		}
	}
	return out
}

// Resolve applies the cascade change > workspace > platform > off, step by
// step; launch parameters cascade workspace > platform, field by field. Any
// argument may be nil.
func Resolve(platform, workspace *Level, change *Override) Resolved {
	var p, w Level
	if platform != nil {
		p = *platform
	}
	if workspace != nil {
		w = *workspace
	}
	var c Override
	if change != nil {
		c = *change
	}
	return Resolved{
		Conformity: pickBool(false, c.Conformity, w.Conformity, p.Conformity),
		UI:         pickBool(false, c.UI, w.UI, p.UI),
		LaunchParams: LaunchParams{
			UIStartCommand: pickText(w.UIStartCommand, p.UIStartCommand),
			UIBaseURL:      pickText(w.UIBaseURL, p.UIBaseURL),
		},
	}
}

// BoolPatch distinguishes an absent JSON field (Set false) from a present
// one; JSON null resets the field to inheritance. Any non-boolean value fails
// to decode.
type BoolPatch struct {
	Set   bool
	Reset bool
	Value bool
}

func (b *BoolPatch) UnmarshalJSON(data []byte) error {
	b.Set = true
	if string(data) == "null" {
		b.Reset = true
		return nil
	}
	return json.Unmarshal(data, &b.Value)
}

// Apply returns v updated by the patch.
func (b BoolPatch) Apply(v *bool) *bool {
	if !b.Set {
		return v
	}
	if b.Reset {
		return nil
	}
	n := b.Value
	return &n
}

// Patch is a partial update of an Override. Unknown JSON fields should be
// rejected by the decoder.
type Patch struct {
	Conformity BoolPatch `json:"conformity"`
	UI         BoolPatch `json:"ui"`
}

// Touches reports whether the patch changes anything.
func (p Patch) Touches() bool { return p.Conformity.Set || p.UI.Set }

// Apply returns the Override resulting from the patch; nil when empty.
func (p Patch) Apply(cur *Override) *Override {
	out := Override{}
	if cur != nil {
		out = *cur.Clone()
	}
	out.Conformity = p.Conformity.Apply(out.Conformity)
	out.UI = p.UI.Apply(out.UI)
	if out.IsEmpty() {
		return nil
	}
	return &out
}

// NormalizeText trims the surrounding whitespace of a free-text parameter; an
// empty result means "absent".
func NormalizeText(s string) string { return strings.TrimSpace(s) }

// PortToken is the placeholder of the free port the UI step picks at launch.
const PortToken = "{port}"

// Substitute replaces every PortToken of s with port.
func Substitute(s string, port int) string {
	return strings.ReplaceAll(s, PortToken, strconv.Itoa(port))
}

// ValidateBaseURL checks that a non-empty value is an absolute http(s) URL.
// The port may be the PortToken (":{port}"), which is accepted nowhere else.
func ValidateBaseURL(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	s = strings.Replace(s, ":"+PortToken, ":0", 1)
	u, err := url.Parse(s)
	if err != nil || strings.Contains(s, PortToken) || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("uiBaseUrl must be an absolute http or https URL")
	}
	return nil
}
