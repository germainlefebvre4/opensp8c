package pool

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/verification"
)

func boolPatch(v bool) verification.BoolPatch {
	return verification.BoolPatch{Set: true, Value: v}
}

// newVerifyPrefs returns preferences whose Configuration level enables the
// conformity verification.
func newVerifyPrefs(t *testing.T, conformity bool) *preferences.Service {
	t.Helper()
	svc := preferences.NewService(filepath.Join(t.TempDir(), "preferences.json"))
	if err := svc.SetVerificationDefaults(preferences.VerificationPatch{Conformity: boolPatch(conformity)}); err != nil {
		t.Fatal(err)
	}
	return svc
}

// verdictResult is the final event of a verifier turn answering with text.
func verdictResult(text string) string {
	b, _ := json.Marshal(map[string]any{"type": "result", "subtype": "success", "result": text})
	return string(b)
}

func writeChangeMeta(t *testing.T, repo, change, yaml string) {
	t.Helper()
	p := filepath.Join(repo, "openspec", "changes", change, ".openspec.yaml")
	if err := os.WriteFile(p, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
}

func vpatchConformity(v bool) preferences.VerificationPatch {
	return preferences.VerificationPatch{Conformity: boolPatch(v)}
}
