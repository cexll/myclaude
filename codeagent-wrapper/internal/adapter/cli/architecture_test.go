package cli

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestMainImportsBootstrapDirectly(t *testing.T) {
	root := projectRoot(t)
	path := filepath.Join(root, "cmd", "codeagent-wrapper", "main.go")
	imports := parseImports(t, path)
	if len(imports) != 1 || !slices.Contains(imports, "codeagent-wrapper/internal/bootstrap") {
		t.Fatalf("%s imports = %v, want [codeagent-wrapper/internal/bootstrap]", path, imports)
	}
}

func TestBootstrapComposesAppAndCLIAdapter(t *testing.T) {
	root := projectRoot(t)
	path := filepath.Join(root, "internal", "bootstrap", "bootstrap.go")
	imports := parseImports(t, path)
	if len(imports) != 3 ||
		!slices.Contains(imports, "os") ||
		!slices.Contains(imports, "codeagent-wrapper/internal/adapter/cli") ||
		!slices.Contains(imports, "codeagent-wrapper/internal/app") {
		t.Fatalf("%s imports = %v, want os + internal/adapter/cli + internal/app", path, imports)
	}
}

func TestCLIAdapterDoesNotImportAppOrBootstrap(t *testing.T) {
	root := projectRoot(t)
	files, err := filepath.Glob(filepath.Join(root, "internal", "adapter", "cli", "*.go"))
	if err != nil {
		t.Fatalf("Glob(): %v", err)
	}

	for _, path := range files {
		base := filepath.Base(path)
		if strings.HasSuffix(base, "_test.go") {
			continue
		}

		imports := parseImports(t, path)
		for _, imp := range imports {
			switch imp {
			case "codeagent-wrapper/internal/app", "codeagent-wrapper/internal/bootstrap":
				t.Fatalf("%s imports forbidden entry package %q", base, imp)
			}
		}
	}
}

func parseImports(t *testing.T, path string) []string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile(%q): %v", path, err)
	}

	imports := make([]string, 0, len(file.Imports))
	for _, imp := range file.Imports {
		imports = append(imports, strings.Trim(imp.Path.Value, `"`))
	}
	return imports
}

func projectRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("Abs(): %v", err)
	}
	return root
}
