package diagram

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLineWrap(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "terminal",
			input:    `"if"`,
			expected: `(station ltr "if" true)`,
		},
		{
			name:     "non-terminal",
			input:    `[expr]`,
			expected: `(station ltr [expr] false)`,
		},
		{
			name:     "simple sequence",
			input:    `("if" [expr])`,
			expected: `(hconcat ltr (station ltr "if" true) (station ltr [expr] false))`,
		},
		{
			name:     "three element sequence",
			input:    `("if" [expr] "then")`,
			expected: `(hconcat ltr (station ltr "if" true) (station ltr [expr] false) (station ltr "then" true))`,
		},
		{
			name:     "choice with two terminals",
			input:    `(+ "red" "blue")`,
			expected: `(vconcat-inline ltr vertical vertical "choice" (station ltr "red" true) (station ltr "blue" true))`,
		},
		{
			name:     "choice with non-terminals",
			input:    `(+ [stmt1] [stmt2])`,
			expected: `(vconcat-inline ltr vertical vertical "choice" (station ltr [stmt1] false) (station ltr [stmt2] false))`,
		},
		{
			name:     "optional (choice with empty)",
			input:    `(+ "optional" ())`,
			expected: `(vconcat-inline ltr vertical vertical "choice" (station ltr "optional" true) (rail ltr 0))`,
		},
		{
			name:     "loop with separator",
			input:    `(- [digit] ",")`,
			expected: `(vconcat-block ltr vertical vertical - (station ltr [digit] false) (station ltr "," true))`,
		},
		{
			name:     "zero-or-more (loop with empty)",
			input:    `(- "a" ())`,
			expected: `(vconcat-block ltr vertical vertical - (station ltr "a" true) (rail ltr 0))`,
		},
		{
			name:     "sequence with choice",
			input:    `("if" (+ "a" "b") "then")`,
			expected: `(hconcat ltr (station ltr "if" true) (vconcat-inline ltr vertical vertical "choice" (station ltr "a" true) (station ltr "b" true)) (station ltr "then" true))`,
		},
		{
			name:     "nested sequence",
			input:    `(("if" [expr]) "then")`,
			expected: `(hconcat ltr (hconcat ltr (station ltr "if" true) (station ltr [expr] false)) (station ltr "then" true))`,
		},
		{
			name:     "choice of sequences",
			input:    `(+ ("if" [expr]) ("while" [expr]))`,
			expected: `(vconcat-inline ltr vertical vertical "choice" (hconcat ltr (station ltr "if" true) (station ltr [expr] false)) (hconcat ltr (station ltr "while" true) (station ltr [expr] false)))`,
		},
		{
			name:     "loop of sequence",
			input:    `(- ([digit] [digit]) ",")`,
			expected: `(vconcat-block ltr vertical vertical - (hconcat ltr (station ltr [digit] false) (station ltr [digit] false)) (station ltr "," true))`,
		},
		{
			name:     "nested loops",
			input:    `(- (- "a" ()) "b")`,
			expected: `(vconcat-block ltr vertical vertical - (vconcat-block ltr vertical vertical - (station ltr "a" true) (rail ltr 0)) (station ltr "b" true))`,
		},
		{
			name:     "nested choices",
			input:    `(+ (+ "a" "b") "c")`,
			expected: `(vconcat-inline ltr vertical vertical "choice" (vconcat-inline ltr vertical vertical "choice" (station ltr "a" true) (station ltr "b" true)) (station ltr "c" true))`,
		},
		{
			name:     "complex: if-then-else",
			input:    `("if" [expr] "then" [stmt] (+ ("else" [stmt]) ()))`,
			expected: `(hconcat ltr (station ltr "if" true) (station ltr [expr] false) (station ltr "then" true) (station ltr [stmt] false) (vconcat-inline ltr vertical vertical "choice" (hconcat ltr (station ltr "else" true) (station ltr [stmt] false)) (rail ltr 0)))`,
		},
		{
			name:     "complex: while loop",
			input:    `("while" [expr] "do" [stmt])`,
			expected: `(hconcat ltr (station ltr "while" true) (station ltr [expr] false) (station ltr "do" true) (station ltr [stmt] false))`,
		},
		{
			name:     "complex: comma-separated list",
			input:    `([item] (- ("," [item]) ()))`,
			expected: `(hconcat ltr (station ltr [item] false) (vconcat-block ltr vertical vertical - (hconcat ltr (station ltr "," true) (station ltr [item] false)) (rail ltr 0)))`,
		},
		{
			name:     "choice with three alternatives",
			input:    `(+ (+ "a" "b") "c")`,
			expected: `(vconcat-inline ltr vertical vertical "choice" (vconcat-inline ltr vertical vertical "choice" (station ltr "a" true) (station ltr "b" true)) (station ltr "c" true))`,
		},
		{
			name:     "empty sequence element",
			input:    `("start" () "end")`,
			expected: `(hconcat ltr (station ltr "start" true) (rail ltr 0) (station ltr "end" true))`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d, err := fromBytes([]byte(test.input))
			require.NoError(t, err, "failed to parse diagram")

			err = computeVerticalMetrics(d)
			require.NoError(t, err, "failed to compute vertical metrics")

			layout, err := lineWrap(d, 0) // maxWidth=0 means no wrapping
			require.NoError(t, err, "failed to wrap diagram")
			assert.Equal(t, test.expected, layout.String())
		})
	}
}

func TestLineWrapWithWidth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth float64
		check    func(*testing.T, layout)
	}{
		{
			name:     "fits on one line",
			input:    `("A" "B" "C")`,
			maxWidth: 1000.0,
			check: func(t *testing.T, l layout) {
				// Should be a simple hconcat
				hc, ok := l.(*hconcat)
				require.True(t, ok, "expected hconcat")
				assert.Len(t, hc.children, 3)
			},
		},
		{
			name:     "needs wrapping",
			input:    `("AAA" "BBB" "CCC" "DDD")`,
			maxWidth: 50.0, // Small width to force wrapping
			check: func(t *testing.T, l layout) {
				// Should be wrapped into a vconcat-block
				_, ok := l.(*vconcatBlock)
				require.True(t, ok, "expected vconcatBlock for wrapped sequence, got %T", l)
				t.Logf("Wrapped layout: %s", l.String())
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d, err := fromBytes([]byte(test.input))
			require.NoError(t, err, "failed to parse diagram")

			err = computeVerticalMetrics(d)
			require.NoError(t, err, "failed to compute vertical metrics")

			layout, err := lineWrap(d, test.maxWidth)
			require.NoError(t, err, "failed to wrap diagram")

			test.check(t, layout)
		})
	}
}
