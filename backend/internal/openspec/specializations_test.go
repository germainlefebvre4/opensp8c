package openspec

import (
	"reflect"
	"testing"
)

func TestCombineSpecializationVocabularyDeduplicatesBaseValues(t *testing.T) {
	custom := []string{"frontend", "ml-ops", "backend"}
	got := CombineSpecializationVocabulary(custom)

	want := append(append([]string{}, BaseAgentSpecializations...), "ml-ops")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}

	count := 0
	for _, v := range got {
		if v == "frontend" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 'frontend' to appear exactly once, got %d times", count)
	}
}

func TestCombineSpecializationVocabularyNoCustom(t *testing.T) {
	got := CombineSpecializationVocabulary(nil)
	if !reflect.DeepEqual(got, BaseAgentSpecializations) {
		t.Errorf("expected base vocabulary unchanged, got %v", got)
	}
}

func TestSanitizeCustomSpecializationsStripsBaseDuplicates(t *testing.T) {
	got := SanitizeCustomSpecializations([]string{"frontend", "ml-ops", "backend"})
	want := []string{"ml-ops"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestSanitizeCustomSpecializationsDedupesWithinCustom(t *testing.T) {
	got := SanitizeCustomSpecializations([]string{"ml-ops", "embedded", "ml-ops"})
	want := []string{"ml-ops", "embedded"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestSanitizeCustomSpecializationsEmpty(t *testing.T) {
	got := SanitizeCustomSpecializations(nil)
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}
