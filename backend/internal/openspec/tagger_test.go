package openspec

import "testing"

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
