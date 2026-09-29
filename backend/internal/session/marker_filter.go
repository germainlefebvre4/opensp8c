package session

import "strings"

// MarkerFilter removes a `{"event":"<name>", ...}` marker from a stream of
// text deltas, however the stream fragments it. It is fed deltas in order and
// returns the text that is safe to display: text that could still turn out to
// be the start of a marker is held back until it is either confirmed (the
// marker is dropped once its JSON object closes) or invalidated (the held text
// is released unchanged, in order). Flush releases whatever is still held, so
// a marker that never completes is never lost as text.
//
// The filter is generic over the event name so other `ghost_*` markers can
// adopt it later.
type MarkerFilter struct {
	prefix string // `{"event":"<name>"`

	held     string // text that may still become a marker prefix
	inMarker bool   // prefix confirmed, scanning for the closing brace
	marker   string // raw marker text seen so far, restored by Flush if never closed
	depth    int
	inString bool
	escaped  bool
	skipWS   bool // drop whitespace directly following a removed marker
}

// NewMarkerFilter returns a filter that removes markers whose "event" field is eventName.
func NewMarkerFilter(eventName string) *MarkerFilter {
	return &MarkerFilter{prefix: `{"event":"` + eventName + `"`}
}

// Feed consumes the next delta and returns the text that can be displayed now.
func (f *MarkerFilter) Feed(s string) string {
	work := f.held + s
	f.held = ""
	var out strings.Builder
	i := 0
	for i < len(work) {
		c := work[i]
		if f.inMarker {
			i++
			f.marker += string(c)
			if f.inString {
				switch {
				case f.escaped:
					f.escaped = false
				case c == '\\':
					f.escaped = true
				case c == '"':
					f.inString = false
				}
				continue
			}
			switch c {
			case '"':
				f.inString = true
			case '{':
				f.depth++
			case '}':
				f.depth--
				if f.depth == 0 {
					f.inMarker = false
					f.marker = ""
					f.skipWS = true
				}
			}
			continue
		}
		if f.skipWS {
			if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
				i++
				continue
			}
			f.skipWS = false
		}
		if c != '{' {
			out.WriteByte(c)
			i++
			continue
		}
		rest := work[i:]
		switch {
		case strings.HasPrefix(rest, f.prefix):
			f.inMarker = true
			f.depth, f.inString, f.escaped = 0, false, false
			// The loop's marker branch consumes the opening brace next.
		case len(rest) < len(f.prefix) && strings.HasPrefix(f.prefix, rest):
			f.held = rest
			return out.String()
		default:
			out.WriteByte(c)
			i++
		}
	}
	return out.String()
}

// Flush returns any text still held back and resets the filter for the next block.
func (f *MarkerFilter) Flush() string {
	rest := f.held + f.marker
	f.held, f.marker = "", ""
	f.inMarker, f.skipWS = false, false
	f.depth, f.inString, f.escaped = 0, false, false
	return rest
}
