package langlang

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRelativePath(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
		parentPath string
		expected   string
		expectErr  bool
	}{
		// URI-style paths (inmemory://)
		{
			name:       "inmemory URI with relative import",
			importPath: "./foo.peg",
			parentPath: "inmemory://project/dir/bar.peg",
			expected:   "inmemory://project/dir/foo.peg",
		},
		{
			name:       "inmemory URI with relative import in nested dir",
			importPath: "./sub/other.peg",
			parentPath: "inmemory://project/dir/bar.peg",
			expected:   "inmemory://project/dir/sub/other.peg",
		},
		{
			name:       "inmemory URI with parent directory traversal",
			importPath: "../sibling/other.peg",
			parentPath: "inmemory://project/dir/bar.peg",
			expected:   "inmemory://project/sibling/other.peg",
		},
		{
			name:       "inmemory URI at root level",
			importPath: "./foo.peg",
			parentPath: "inmemory://project/bar.peg",
			expected:   "inmemory://project/foo.peg",
		},

		// file:// URIs
		{
			name:       "file URI with relative import",
			importPath: "./foo.peg",
			parentPath: "file:///home/user/project/bar.peg",
			expected:   "file:///home/user/project/foo.peg",
		},

		// Builtins path (should pass through unchanged)
		{
			name:       "builtins path",
			importPath: BuiltinsPath,
			parentPath: "inmemory://project/bar.peg",
			expected:   BuiltinsPath,
		},
		{
			name:       "builtins path from file",
			importPath: BuiltinsPath,
			parentPath: "/home/user/project/bar.peg",
			expected:   BuiltinsPath,
		},

		// Root node handling (importPath == parentPath)
		{
			name:       "root node same path",
			importPath: "inmemory://project/foo.peg",
			parentPath: "inmemory://project/foo.peg",
			expected:   "inmemory://project/foo.peg",
		},

		// Regular file paths (no scheme)
		{
			name:       "file path with relative import",
			importPath: "./foo.peg",
			parentPath: "/home/user/project/bar.peg",
			expected:   "/home/user/project/foo.peg",
		},
		{
			name:       "file path with nested relative import",
			importPath: "./sub/other.peg",
			parentPath: "/home/user/project/bar.peg",
			expected:   "/home/user/project/sub/other.peg",
		},

		// Error cases
		{
			name:       "file path without ./ prefix",
			importPath: "foo.peg",
			parentPath: "/home/user/project/bar.peg",
			expectErr:  true,
		},
		{
			name:       "file path too short",
			importPath: "x",
			parentPath: "/home/user/project/bar.peg",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getRelativePath(tt.importPath, tt.parentPath)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
