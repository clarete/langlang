package diagram

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestASCIIRenderer(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		width  int
		height int
	}{
		{
			name:   "simple terminal",
			input:  `"if"`,
			width:  20,
			height: 10,
		},
		{
			name:   "simple sequence",
			input:  `("if" [expr])`,
			width:  40,
			height: 10,
		},
		{
			name:   "sequence with three elements",
			input:  `("if" [expr] "then")`,
			width:  60,
			height: 10,
		},
		{
			name:   "choice",
			input:  `(+ "red" "blue")`,
			width:  40,
			height: 15,
		},
		{
			name:   "loop",
			input:  `(- [digit] ",")`,
			width:  40,
			height: 15,
		},
		{
			name:   "identifier with optional tail",
			input:  `("[A..Z_a..z]" (- "[0..9A..Z_a..z]" ()))`,
			width:  80,
			height: 20,
		},
		{
			name:   "empty main path",
			input:  `(- () "X")`,
			width:  40,
			height: 10,
		},
		{
			name:   "choice followed by element",
			input:  `((+ "foo" "Bar") "Baz")`,
			width:  60,
			height: 15,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Parse diagram
			d, err := fromBytes([]byte(test.input))
			require.NoError(t, err)

			// Compute metrics
			err = computeVerticalMetrics(d)
			require.NoError(t, err)

		// Wrap to layout (no max width for these tests)
		layout, err := lineWrap(d, 0)
		require.NoError(t, err)

			// Render to ASCII
			ascii := layoutToASCII(layout, test.width, test.height)

			// Print output for visual inspection
			fmt.Printf("\n=== %s ===\n", test.name)
			fmt.Println(ascii)
		})
	}
}

func TestASCIIRendererStack(t *testing.T) {
	// Test that demonstrates stack behavior
	
	// Create a simple HConcat layout
	layout := newHConcat(ltr, []layout{
		newStation(ltr, "A", true),
		newRail(ltr, 16), // 2 chars wide (16/8)
		newStation(ltr, "B", true),
	})
	
	// Render
	ascii := layoutToASCII(layout, 40, 5)
	
	fmt.Println("\n=== Stack Test: HConcat ===")
	fmt.Println(ascii)
	
	// The renderer should have:
	// 1. Pushed dimensions for "A"
	// 2. Pushed dimensions for rail
	// 3. Pushed dimensions for "B"
	// 4. Popped all three when rendering HConcat
	// 5. Pushed HConcat's total dimensions
	
	require.NotEmpty(t, ascii, "ASCII output should not be empty")
}

