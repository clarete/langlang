package python

import (
	"fmt"
	"testing"
	"github.com/clarete/langlang/go"
	"github.com/stretchr/testify/require"
)

func TestPyDebugLC(t *testing.T) {
	matcher, err := langlang.MatcherFromFilePathWithConfig(grammarPath, pythonConfig())
	require.NoError(t, err)
	inputs := []string{
		"x = [i for i in range(10)]\n",
		"s = f\"val={x}\"\n",
		"s = r\"no\\\\escape\"\n",
		"b = b\"data\"\n",
	}
	for _, input := range inputs {
		_, n, err := matcher.Match([]byte(input))
		status := "OK"
		if err != nil {
			if pe, ok := err.(langlang.ParsingError); ok {
				status = fmt.Sprintf("FAIL at byte %d: %s", pe.End, pe.Message[:min(60, len(pe.Message))])
			} else {
				status = fmt.Sprintf("FAIL: %v", err)
			}
		}
		prefix := status
		if len(prefix) > 4 {
			prefix = prefix[:4]
		}
		fmt.Printf("%-4s %q: n=%d len=%d\n", prefix, input, n, len(input))
	}
}

func min(a, b int) int {
	if a < b { return a }
	return b
}
