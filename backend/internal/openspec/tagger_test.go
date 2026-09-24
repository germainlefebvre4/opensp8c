package openspec

import (
	"reflect"
	"testing"
)

func TestDeriveTypeNoRecognizedPath(t *testing.T) {
	got := DeriveType("- [ ] Update README.md")
	if !reflect.DeepEqual(got, []string{}) {
		t.Errorf("expected empty list, got %v", got)
	}
}

func TestDeriveTypeFrontendOnly(t *testing.T) {
	got := DeriveType("- [ ] Edit frontend/src/App.tsx")
	if !reflect.DeepEqual(got, []string{"frontend"}) {
		t.Errorf("expected [frontend], got %v", got)
	}
}

func TestDeriveTypeFrontendAndBackend(t *testing.T) {
	got := DeriveType("- [ ] Edit frontend/src/App.tsx\n- [ ] Edit backend/internal/api/handlers.go")
	if !reflect.DeepEqual(got, []string{"frontend", "backend"}) {
		t.Errorf("expected [frontend backend], got %v", got)
	}
}

func TestDeriveTypeAllThreeCategories(t *testing.T) {
	got := DeriveType("- [ ] Edit frontend/src/App.tsx\n- [ ] Edit backend/internal/api/handlers.go\n- [ ] Edit scripts/migrate.sh")
	if !reflect.DeepEqual(got, []string{"frontend", "backend", "batch"}) {
		t.Errorf("expected [frontend backend batch], got %v", got)
	}
}

func TestFilterToVocabulary(t *testing.T) {
	vocabulary := []string{"frontend", "backend", "database"}

	got := FilterToVocabulary([]string{"frontend", "made-up-value", "database"}, vocabulary)
	if len(got) != 2 || got[0] != "frontend" || got[1] != "database" {
		t.Errorf("expected out-of-vocabulary value to be filtered, got %v", got)
	}
}

func TestFilterToVocabularyEmptyVocabulary(t *testing.T) {
	got := FilterToVocabulary([]string{"frontend", "backend"}, nil)
	if len(got) != 0 {
		t.Errorf("expected empty result for empty vocabulary, got %v", got)
	}
}

func TestFilterToVocabularyKeepsAllValidValues(t *testing.T) {
	vocabulary := []string{"frontend", "backend"}
	got := FilterToVocabulary([]string{"frontend", "backend"}, vocabulary)
	if len(got) != 2 {
		t.Errorf("expected both valid values kept, got %v", got)
	}
}
