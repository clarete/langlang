package langlang

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// BuiltinsPath is the special path used to identify the builtins
// grammar.  This path is recognized by all ImportLoaders and resolves
// to the embedded builtins.peg content. This allows builtins to be
// tracked with proper FileIDs for LSP navigation (Go to Definition,
// etc.).
const BuiltinsPath = "langlang:builtins.peg"

//go:embed builtins.peg
var builtinsText []byte

type RelativeImportLoader struct{}

func NewRelativeImportLoader() *RelativeImportLoader {
	return &RelativeImportLoader{}
}

func (ril *RelativeImportLoader) GetPath(importPath, parentPath string) (string, error) {
	return getRelativePath(importPath, parentPath)
}

func (ril *RelativeImportLoader) GetContent(path string) ([]byte, error) {
	// Return embedded builtins content for the special path
	if isBuiltinsPath(path) {
		return builtinsText, nil
	}
	return os.ReadFile(path)
}

type InMemoryImportLoader struct{ files map[string][]byte }

func NewInMemoryImportLoader() *InMemoryImportLoader {
	return &InMemoryImportLoader{files: map[string][]byte{}}
}

func (l *InMemoryImportLoader) Add(path string, content []byte) {
	l.files[path] = content
}

func (l *InMemoryImportLoader) GetPath(importPath, parentPath string) (string, error) {
	return getRelativePath(importPath, parentPath)
}

func (l *InMemoryImportLoader) GetContent(path string) ([]byte, error) {
	if isBuiltinsPath(path) {
		return builtinsText, nil
	}
	b, ok := l.files[path]
	if !ok {
		return nil, fmt.Errorf("import not found: %s", path)
	}
	return b, nil
}

func getRelativePath(importPath, parentPath string) (string, error) {
	// Handle builtins path specially
	if isBuiltinsPath(importPath) {
		return importPath, nil
	}
	// Root node handling
	if importPath == parentPath {
		return importPath, nil
	}
	var contents string
	if len(importPath) < 4 {
		return contents, fmt.Errorf("path too short, it should start with ./: %s", importPath)
	}
	if importPath[:2] != "./" {
		return contents, fmt.Errorf("path isn't relative to the import site: %s", importPath)
	}
	modulePath := importPath[2:]
	return filepath.Join(filepath.Dir(parentPath), modulePath), nil
}

// isBuiltinsPath checks if a path refers to the builtins grammar
func isBuiltinsPath(path string) bool {
	return path == BuiltinsPath
}
