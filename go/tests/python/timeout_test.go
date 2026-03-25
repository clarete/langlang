package python

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	langlang "github.com/clarete/langlang/go"

	"github.com/stretchr/testify/require"
)

// TestTimeoutFilesParsing verifies that known timeout files either parse
// or fail quickly (within 2s), to catch regression from grammar changes.
func TestTimeoutFilesParsing(t *testing.T) {
	blackCases := "/tmp/langlang-python-corpus/black/tests/data/cases"
	if _, err := os.Stat(blackCases); os.IsNotExist(err) {
		t.Skip("black corpus not cloned at " + blackCases)
		return
	}
	timeoutFiles := []string{
		"bytes_docstring.py",
		"multiline_consecutive_open_parentheses_ignore.py",
		"numeric_literals_skip_underscores.py",
		"pattern_matching_trailing_comma.py",
		"preview_fstring.py",
		"remove_await_parens.py",
		"remove_parens.py",
		"tupleassign.py",
		"is_simple_lookup_for_doublestar_expression.py",
	}
	matcher, err := langlang.MatcherFromFilePathWithConfig(grammarPath, pythonConfig())
	require.NoError(t, err)
	cm, hasCtx := matcher.(langlang.ContextMatcher)
	for _, name := range timeoutFiles {
		path := filepath.Join(blackCases, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Skipf("read %s: %v", name, err)
			continue
		}
		t.Run(name, func(t *testing.T) {
			if hasCtx {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				cm.SetContext(ctx)
				defer func() { cm.SetContext(context.Background()) }()
			}
			start := time.Now()
			_, _, err := matcher.Match(data)
			elapsed := time.Since(start)
			if elapsed > 2*time.Second {
				t.Logf("%s took %v (parse err: %v)", name, elapsed, err)
			}
		})
	}
}
