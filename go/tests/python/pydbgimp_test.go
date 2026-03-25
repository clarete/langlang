package python

import (
	"fmt"
	"testing"
	"github.com/clarete/langlang/go"
	"github.com/stretchr/testify/require"
)

func TestPyDebugImport(t *testing.T) {
	matcher, err := langlang.MatcherFromFilePathWithConfig(grammarPath, pythonConfig())
	require.NoError(t, err)
	inputs := []string{
		"from . import sibling\n",
		"from .. import parent\n",
		"from ...deep import module\n",
	}
	for _, input := range inputs {
		_, n, err := matcher.Match([]byte(input))
		if err != nil {
			if pe, ok := err.(langlang.ParsingError); ok {
				fmt.Printf("FAIL %q: byte %d: %s\n", input, pe.End, pe.Message[:80])
			}
		} else {
			fmt.Printf("OK   %q: n=%d\n", input, n)
		}
	}
}
