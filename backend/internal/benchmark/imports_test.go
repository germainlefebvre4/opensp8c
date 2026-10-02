package benchmark

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// The calculation and the report must only see normalized phase events and
// result structs, so a backend metrics module can replace ObserverSource
// without touching them. Only ObserverSource (and the collection plumbing)
// reads files.
func TestPureFilesImportNoFileReaders(t *testing.T) {
	pure := []string{"calc.go", "report.go", "source.go", "result.go"}
	forbidden := map[string]bool{
		"os": true, "io/fs": true, "io/ioutil": true, "bufio": true, "path/filepath": true,
		"net/http": true, "os/exec": true,
	}
	for _, file := range pure {
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if forbidden[path] || strings.HasPrefix(path, "github.com/glefebvre/opensp8c/internal/") {
				t.Errorf("%s imports %q: calculation and report must stay source-agnostic", file, path)
			}
		}
	}
}

// The benchmark never depends on platform packages (it only reads their
// on-disk formats), so a platform refactor cannot silently change a
// measurement.
func TestPackageImportsNoPlatformPackage(t *testing.T) {
	files := []string{
		"baseline.go", "calc.go", "collect.go", "config.go", "observer.go",
		"observersource.go", "prepare.go", "report.go", "result.go", "source.go",
	}
	for _, file := range files {
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			if strings.Contains(imp.Path.Value, "opensp8c/internal/") {
				t.Errorf("%s imports platform package %s", file, imp.Path.Value)
			}
		}
	}
}
