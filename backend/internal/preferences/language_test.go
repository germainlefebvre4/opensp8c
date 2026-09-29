package preferences

import (
	"os"
	"strings"
	"testing"

	"github.com/glefebvre/opensp8c/internal/language"
)

func strp(s string) *string { return &s }

func TestLegacyFileWithoutLanguageFields(t *testing.T) {
	svc := newTestService(t)
	if err := os.WriteFile(svc.Path(), []byte(`{"defaultAgent":"claude"}`), 0644); err != nil {
		t.Fatal(err)
	}
	p, err := svc.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := p.ResolvedLanguages(); got != (language.Resolved{Chat: "en", Documentation: "en", Code: "en"}) {
		t.Errorf("defaults: %+v", got)
	}
}

func TestSetAgentLanguagesPartial(t *testing.T) {
	svc := newTestService(t)
	if err := svc.SetAgentLanguages(AgentLanguagesUpdate{Chat: strp("fr"), Code: strp("fr")}); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetAgentLanguages(AgentLanguagesUpdate{Documentation: strp("en")}); err != nil {
		t.Fatal(err)
	}
	p, _ := svc.Load()
	want := AgentLanguages{Chat: "fr", Documentation: "en", Code: "fr"}
	if *p.AgentLanguages != want {
		t.Errorf("got %+v want %+v", *p.AgentLanguages, want)
	}
}

func TestSetUILocale(t *testing.T) {
	svc := newTestService(t)
	if err := svc.SetUILocale("fr"); err != nil {
		t.Fatal(err)
	}
	p, _ := svc.Load()
	if p.UILocale != "fr" {
		t.Errorf("uiLocale = %q", p.UILocale)
	}
}

func TestLanguageDirective(t *testing.T) {
	var nilPrefs *Preferences
	d := nilPrefs.LanguageDirective(language.Chat)
	if !strings.Contains(d, "English") {
		t.Errorf("nil receiver directive: %s", d)
	}

	p := &Preferences{UILocale: "fr", AgentLanguages: &AgentLanguages{Chat: "auto", Documentation: "en"}}
	if d := p.LanguageDirective(language.Chat); !strings.Contains(d, "French") {
		t.Errorf("chat: %s", d)
	}
	if d := p.LanguageDirective(language.Docs); !strings.Contains(d, "English") {
		t.Errorf("docs: %s", d)
	}
}
