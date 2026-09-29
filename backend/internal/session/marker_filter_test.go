package session

import "testing"

func runFilter(deltas []string) string {
	f := NewMarkerFilter("ghost_named")
	out := ""
	for _, d := range deltas {
		out += f.Feed(d)
	}
	return out + f.Flush()
}

func TestMarkerFilter(t *testing.T) {
	tests := []struct {
		name   string
		deltas []string
		want   string
	}{
		{"single block", []string{"{\"event\":\"ghost_named\",\"name\":\"a-b\"}\n\nHello"}, "Hello"},
		{"token by token (real stream)", []string{"{\"", "event\":\"ghost_named", "\",\"name\":\"rethink-", "application-ergonomics\"", "}\n\nLet me look", " at what"}, "Let me look at what"},
		{"mixed with text before", []string{"Intro.\n", "{\"event\":\"ghost_named\",\"name\":\"x\"}", "\nSuite"}, "Intro.\nSuite"},
		{"legit brace invalidated", []string{"{\"", "foo\": 1}"}, "{\"foo\": 1}"},
		{"legit brace in one chunk", []string{"{ code }"}, "{ code }"},
		{"other ghost event untouched", []string{"{\"event\":\"ghost_question\",\"question\":\"Q?\"}"}, "{\"event\":\"ghost_question\",\"question\":\"Q?\"}"},
		{"never completed is restored", []string{"{\"event\":\"ghost_named\",\"na"}, "{\"event\":\"ghost_named\",\"na"},
		{"partial prefix flushed", []string{"{\"eve"}, "{\"eve"},
		{"brace inside name string", []string{"{\"event\":\"ghost_named\",\"name\":\"a}b\"}", "Text"}, "Text"},
		{"utf8 text preserved", []string{"{\"event\":\"ghost_named\",\"name\":\"x\"}\n", "Périmètre é"}, "Périmètre é"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runFilter(tt.deltas); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMarkerFilterNeverEmitsMarkerFragments(t *testing.T) {
	full := "{\"event\":\"ghost_named\",\"name\":\"abc\"}\n\nAfter"
	// Split at every possible boundary pair.
	for i := 0; i <= len(full); i++ {
		for j := i; j <= len(full); j++ {
			f := NewMarkerFilter("ghost_named")
			got := f.Feed(full[:i]) + f.Feed(full[i:j]) + f.Feed(full[j:]) + f.Flush()
			if got != "After" {
				t.Fatalf("split %d/%d: got %q", i, j, got)
			}
		}
	}
}
