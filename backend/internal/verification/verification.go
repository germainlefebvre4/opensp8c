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
	"unicode/utf8"
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

// UI drivers: how the verifier agent gets a browser.
const (
	DriverAuto       = "auto"
	DriverPlaywright = "playwright"
	DriverChrome     = "chrome"
	DriverCustom     = "custom"
)

// Limits of the free-form driver settings.
const (
	MaxGuidanceLen = 4000
	MaxToolEntries = 50
)

// LaunchParams are the parameters of the UI verification launch.
type LaunchParams struct {
	UIStartCommand string   `json:"uiStartCommand,omitempty"`
	UIBaseURL      string   `json:"uiBaseUrl,omitempty"`
	UIDriver       string   `json:"uiDriver"`
	UIMcpConfig    string   `json:"uiMcpConfig,omitempty"`
	UIAllowedTools []string `json:"uiAllowedTools,omitempty"`
	// UIGuidance is the platform guidance followed by the workspace one.
	UIGuidance string `json:"uiGuidance,omitempty"`
}

// LaunchOverride is a partial LaunchParams; nil means "inherit".
type LaunchOverride struct {
	UIStartCommand *string   `json:"uiStartCommand,omitempty"`
	UIBaseURL      *string   `json:"uiBaseUrl,omitempty"`
	UIDriver       *string   `json:"uiDriver,omitempty"`
	UIMcpConfig    *string   `json:"uiMcpConfig,omitempty"`
	UIAllowedTools *[]string `json:"uiAllowedTools,omitempty"`
	UIGuidance     *string   `json:"uiGuidance,omitempty"`
}

// IsEmpty reports whether no launch parameter is set.
func (l LaunchOverride) IsEmpty() bool {
	return l.UIStartCommand == nil && l.UIBaseURL == nil && l.UIDriver == nil &&
		l.UIMcpConfig == nil && l.UIAllowedTools == nil && l.UIGuidance == nil
}

// ParseDriver validates a driver name; the empty string (after trimming)
// means "absent" and is returned as is.
func ParseDriver(s string) (string, error) {
	s = strings.TrimSpace(s)
	switch s {
	case "", DriverAuto, DriverPlaywright, DriverChrome, DriverCustom:
		return s, nil
	}
	return "", fmt.Errorf("uiDriver must be one of auto, playwright, chrome, custom")
}

// ValidateGuidance checks the length of the (trimmed) guidance text.
func ValidateGuidance(s string) error {
	if utf8.RuneCountInString(strings.TrimSpace(s)) > MaxGuidanceLen {
		return fmt.Errorf("uiGuidance must be at most %d characters", MaxGuidanceLen)
	}
	return nil
}

// NormalizeTools trims every entry and drops the empty ones; nil when none
// remains. More than MaxToolEntries non-empty entries is an error.
func NormalizeTools(in []string) ([]string, error) {
	var out []string
	for _, t := range in {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	if len(out) > MaxToolEntries {
		return nil, fmt.Errorf("uiAllowedTools must have at most %d entries", MaxToolEntries)
	}
	return out, nil
}

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

// pickDriver returns the first valid driver among levels (unknown values of a
// hand-edited file are ignored), or auto.
func pickDriver(levels ...*string) string {
	for _, v := range levels {
		if v == nil {
			continue
		}
		if d, err := ParseDriver(*v); err == nil && d != "" {
			return d
		}
	}
	return DriverAuto
}

func pickTools(levels ...*[]string) []string {
	for _, v := range levels {
		if v == nil {
			continue
		}
		var out []string
		for _, t := range *v {
			if t = strings.TrimSpace(t); t != "" {
				out = append(out, t)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

// joinGuidance accumulates the platform then the workspace text.
func joinGuidance(levels ...*string) string {
	var parts []string
	for _, v := range levels {
		if v == nil {
			continue
		}
		if t := strings.TrimSpace(*v); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "\n\n")
}

// Resolve applies the cascade change > workspace > platform > off, step by
// step; launch parameters cascade workspace > platform, field by field, except
// the guidance which accumulates. A platform driver "chrome" is ignored: that
// driver can only be chosen by a workspace. Any argument may be nil.
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
	if p.UIDriver != nil && strings.TrimSpace(*p.UIDriver) == DriverChrome {
		p.UIDriver = nil
	}
	return Resolved{
		Conformity: pickBool(false, c.Conformity, w.Conformity, p.Conformity),
		UI:         pickBool(false, c.UI, w.UI, p.UI),
		LaunchParams: LaunchParams{
			UIStartCommand: pickText(w.UIStartCommand, p.UIStartCommand),
			UIBaseURL:      pickText(w.UIBaseURL, p.UIBaseURL),
			UIDriver:       pickDriver(w.UIDriver, p.UIDriver),
			UIMcpConfig:    pickText(w.UIMcpConfig, p.UIMcpConfig),
			UIAllowedTools: pickTools(w.UIAllowedTools, p.UIAllowedTools),
			UIGuidance:     joinGuidance(p.UIGuidance, w.UIGuidance),
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
