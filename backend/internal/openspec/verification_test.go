package openspec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glefebvre/opensp8c/internal/verification"
)

const verifMetaYAML = `schema: spec-driven
created: "2024-01-01"
verification:
    conformity: true
    ui: false
`

func vp(t *testing.T, j string) verification.Patch {
	t.Helper()
	var p verification.Patch
	if err := json.Unmarshal([]byte(j), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func verifFixture(t *testing.T, yamlContent string) (workspace, changeRoot string) {
	t.Helper()
	workspace = t.TempDir()
	changesDir := filepath.Join(workspace, "openspec", "changes")
	writeChangeFixture(t, changesDir, "c", yamlContent, "- [ ] item\n")
	return workspace, filepath.Join(changesDir, "c")
}

func readMeta(t *testing.T, changeRoot string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(changeRoot, ".openspec.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSetVerification(t *testing.T) {
	ws, root := verifFixture(t, "schema: spec-driven\ncreated: \"2024-01-01\"\n")

	got, err := SetVerification(root, vp(t, `{"conformity":true}`))
	if err != nil || got == nil || got.Conformity == nil || !*got.Conformity {
		t.Fatalf("write: %v %+v", err, got)
	}
	if ov, _ := ReadVerification(ws, "c"); ov == nil || ov.UI != nil || !*ov.Conformity {
		t.Fatalf("read: %+v", ov)
	}
	if _, err := SetVerification(root, vp(t, `{"ui":false}`)); err != nil {
		t.Fatal(err)
	}
	if ov, _ := ReadVerification(ws, "c"); ov == nil || ov.UI == nil || *ov.UI || !*ov.Conformity {
		t.Fatalf("both: %+v", ov)
	}
	// Removing one value keeps the other.
	if _, err := SetVerification(root, vp(t, `{"conformity":null}`)); err != nil {
		t.Fatal(err)
	}
	if ov, _ := ReadVerification(ws, "c"); ov == nil || ov.Conformity != nil || ov.UI == nil {
		t.Fatalf("one removed: %+v", ov)
	}
	// Removing the last one leaves no residual section.
	if got, err := SetVerification(root, vp(t, `{"ui":null}`)); err != nil || got != nil {
		t.Fatalf("clear: %v %+v", err, got)
	}
	if strings.Contains(readMeta(t, root), "verification") {
		t.Errorf("residual section:\n%s", readMeta(t, root))
	}
}

func TestSetVerificationCreatesMissingFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "openspec", "changes", "new")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := SetVerification(root, vp(t, `{"ui":true}`)); err != nil {
		t.Fatal(err)
	}
	if m := readMeta(t, root); !strings.Contains(m, "schema: spec-driven") || !strings.Contains(m, "ui: true") {
		t.Errorf("unexpected file:\n%s", m)
	}
}

func TestReadVerificationAbsent(t *testing.T) {
	ws, _ := verifFixture(t, "schema: spec-driven\n")
	if ov, err := ReadVerification(ws, "c"); err != nil || ov != nil {
		t.Errorf("%v %+v", err, ov)
	}
	if ov, err := ReadVerification(ws, "missing"); err != nil || ov != nil {
		t.Errorf("missing: %v %+v", err, ov)
	}
}

func TestWritersPreserveVerification(t *testing.T) {
	check := func(t *testing.T, root string) {
		t.Helper()
		if m := readMeta(t, root); !strings.Contains(m, "conformity: true") || !strings.Contains(m, "ui: false") {
			t.Errorf("verification lost:\n%s", m)
		}
	}
	t.Run("SetLaunched", func(t *testing.T) {
		_, root := verifFixture(t, verifMetaYAML)
		if err := SetLaunched(root, true); err != nil {
			t.Fatal(err)
		}
		if err := SetLaunched(root, false); err != nil {
			t.Fatal(err)
		}
		check(t, root)
	})
	t.Run("ClearKanbanState", func(t *testing.T) {
		_, root := verifFixture(t, verifMetaYAML+"launched: true\norder: 2\n")
		if err := ClearKanbanState(root); err != nil {
			t.Fatal(err)
		}
		check(t, root)
	})
	t.Run("ReorderReady", func(t *testing.T) {
		ws, root := verifFixture(t, verifMetaYAML)
		if err := ReorderReady(filepath.Join(ws, "openspec", "changes"), []string{"c"}); err != nil {
			t.Fatal(err)
		}
		check(t, root)
	})
	t.Run("TagChange", func(t *testing.T) {
		ws, root := verifFixture(t, verifMetaYAML)
		if err := TagChange(root, ws, true, nil); err != nil {
			t.Fatal(err)
		}
		check(t, root)
	})
}
