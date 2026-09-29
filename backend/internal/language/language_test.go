package language

import (
	"strings"
	"testing"
)

func TestLookupAndOrder(t *testing.T) {
	if l, ok := Lookup("fr"); !ok || l.EnglishName != "French" {
		t.Fatalf("fr not found: %+v", l)
	}
	if _, ok := Lookup("xx"); ok {
		t.Fatal("unknown code must not be found")
	}
	got := Supported()
	if len(got) < 2 || got[0].Code != "en" || got[1].Code != "fr" {
		t.Fatalf("unstable order: %+v", got)
	}
	got[0].Code = "zz"
	if Supported()[0].Code != "en" {
		t.Fatal("Supported must return a copy")
	}
}

func TestResolve(t *testing.T) {
	cases := []struct {
		name string
		in   Levels
		ui   string
		want Resolved
	}{
		{"chat auto follows ui", Levels{Chat: "auto"}, "fr", Resolved{"fr", "fr", "en"}},
		{"explicit doc wins", Levels{Documentation: "en"}, "fr", Resolved{"fr", "en", "en"}},
		{"no ui locale", Levels{}, "", Resolved{"en", "en", "en"}},
		{"unknown ui locale", Levels{Chat: "auto"}, "de", Resolved{"en", "en", "en"}},
		{"explicit code", Levels{Code: "fr"}, "en", Resolved{"en", "en", "fr"}},
		{"code auto ignored", Levels{Code: "auto"}, "fr", Resolved{"fr", "fr", "en"}},
	}
	for _, c := range cases {
		if got := Resolve(c.in, c.ui); got != c.want {
			t.Errorf("%s: got %+v want %+v", c.name, got, c.want)
		}
	}
}

func TestDirective(t *testing.T) {
	r := Resolved{Chat: "fr", Documentation: "fr", Code: "en"}
	chat := Directive(Chat, r)
	if !strings.Contains(chat, "French") || !strings.Contains(chat, "Unless the user explicitly asks otherwise") {
		t.Errorf("chat: %s", chat)
	}
	docs := Directive(Docs, r)
	for _, s := range []string{"French", "SHALL/MUST", "WHEN/THEN", "file names", "i18n"} {
		if !strings.Contains(docs, s) {
			t.Errorf("docs missing %q: %s", s, docs)
		}
	}
	worker := Directive(Worker, r)
	for _, s := range []string{"English", "French", "identifiers", "comments", "commit messages", "error messages", "user-visible text", "SHALL/MUST"} {
		if !strings.Contains(worker, s) {
			t.Errorf("worker missing %q: %s", s, worker)
		}
	}
	en := Directive(Chat, Resolved{Chat: "en"})
	if !strings.Contains(en, "English") || strings.Contains(en, "French") {
		t.Errorf("en chat: %s", en)
	}
}
