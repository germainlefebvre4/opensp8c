// Package language holds the registry of languages supported for agents,
// the resolution of the chat/documentation/code settings, and the directive
// text sent to agents at launch.
package language

// Auto designates the current UI locale (chat and documentation levels only).
const Auto = "auto"

// Default is the language used when nothing else applies.
const Default = "en"

// Language is one entry of the registry.
type Language struct {
	Code        string `json:"code"`
	NativeName  string `json:"nativeName"`  // label displayed in the UI
	EnglishName string `json:"englishName"` // name used in the agent directive
}

// supported is the single source of truth for the languages offered to agents.
// Adding an entry makes it available for the three levels.
var supported = []Language{
	{Code: "en", NativeName: "English", EnglishName: "English"},
	{Code: "fr", NativeName: "Français", EnglishName: "French"},
}

// Supported returns the ordered list of supported languages (a copy).
func Supported() []Language {
	out := make([]Language, len(supported))
	copy(out, supported)
	return out
}

// Lookup finds a language by code.
func Lookup(code string) (Language, bool) {
	for _, l := range supported {
		if l.Code == code {
			return l, true
		}
	}
	return Language{}, false
}

// Levels holds the three raw settings; empty values mean "default".
type Levels struct {
	Chat          string
	Documentation string
	Code          string
}

// Resolved holds the three concrete language codes.
type Resolved struct {
	Chat          string
	Documentation string
	Code          string
}

func resolveAuto(value, uiLocale string) string {
	if value != "" && value != Auto {
		return value
	}
	if _, ok := Lookup(uiLocale); ok {
		return uiLocale
	}
	return Default
}

// Resolve turns the raw settings into concrete languages: explicit value,
// otherwise the UI locale for auto, otherwise English. Code never follows the UI.
func Resolve(langs Levels, uiLocale string) Resolved {
	code := langs.Code
	if code == "" || code == Auto {
		code = Default
	}
	return Resolved{
		Chat:          resolveAuto(langs.Chat, uiLocale),
		Documentation: resolveAuto(langs.Documentation, uiLocale),
		Code:          code,
	}
}
