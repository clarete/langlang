package langlang

import (
	"fmt"
	"os"
	"path/filepath"
)

type RelativeImportLoader struct{}

func NewRelativeImportLoader() *RelativeImportLoader {
	return &RelativeImportLoader{}
}

func (ril *RelativeImportLoader) GetPath(importPath, parentPath string) (string, error) {
	return getRelativePath(importPath, parentPath)
}

func (ril *RelativeImportLoader) GetContent(path string) ([]byte, error) {
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
	b, ok := l.files[path]
	if !ok {
		return nil, fmt.Errorf("import not found: %s", path)
	}
	return b, nil
}

func getRelativePath(importPath, parentPath string) (string, error) {
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
