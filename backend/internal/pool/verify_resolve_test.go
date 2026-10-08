package pool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glefebvre/opensp8c/internal/preferences"
)

func TestResolveVerification(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "openspec", "changes", "c"), 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("no step by default", func(t *testing.T) {
		m := &Manager{workspacePath: repo}
		if r := m.resolveVerification("ws", "c"); r.Conformity || r.UI || anyStepEnabled(r) {
			t.Errorf("got %+v", r)
		}
	})
	t.Run("configuration level", func(t *testing.T) {
		m := &Manager{workspacePath: repo, prefs: newVerifyPrefs(t, true)}
		if r := m.resolveVerification("ws", "c"); !r.Conformity || !anyStepEnabled(r) {
			t.Errorf("got %+v", r)
		}
	})
	t.Run("workspace level wins over configuration", func(t *testing.T) {
		prefs := newVerifyPrefs(t, true)
		if err := prefs.PatchWorkspace("ws", preferences.WorkspaceSettingsPatch{Verification: &preferences.VerificationPatch{Conformity: boolPatch(false)}}); err != nil {
			t.Fatal(err)
		}
		m := &Manager{workspacePath: repo, prefs: prefs}
		if r := m.resolveVerification("ws", "c"); r.Conformity {
			t.Errorf("workspace override ignored: %+v", r)
		}
		if r := m.resolveVerification("other", "c"); !r.Conformity {
			t.Errorf("other workspace should inherit: %+v", r)
		}
	})
	t.Run("workspace enables, change disables", func(t *testing.T) {
		prefs := newVerifyPrefs(t, false)
		if err := prefs.PatchWorkspace("ws", preferences.WorkspaceSettingsPatch{Verification: &preferences.VerificationPatch{Conformity: boolPatch(true)}}); err != nil {
			t.Fatal(err)
		}
		m := &Manager{workspacePath: repo, prefs: prefs}
		if r := m.resolveVerification("ws", "c"); !r.Conformity {
			t.Fatalf("got %+v", r)
		}
		writeChangeMeta(t, repo, "c", "schema: spec-driven\nverification:\n  conformity: false\n")
		if r := m.resolveVerification("ws", "c"); r.Conformity {
			t.Errorf("change override ignored: %+v", r)
		}
	})
	t.Run("change enables over configuration off", func(t *testing.T) {
		writeChangeMeta(t, repo, "c", "schema: spec-driven\nverification:\n  conformity: true\n")
		m := &Manager{workspacePath: repo, prefs: newVerifyPrefs(t, false)}
		if r := m.resolveVerification("ws", "c"); !r.Conformity {
			t.Errorf("got %+v", r)
		}
	})
}
