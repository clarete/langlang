package diagram

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// tests the round trip of parsing a diagram from string and back to string
func TestDiagramFromBytes(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{
			name: "terminal",
			expr: `"if"`,
		},
		{
			name: "non-terminal",
			expr: `[expr]`,
		},
		{
			name: "sequence",
			expr: `("if" [expr])`,
		},
		{
			name: "three element sequence",
			expr: `("if" [expr] "then")`,
		},
		{
			name: "choice with two terminals",
			expr: `(+ "red" "blue")`,
		},
		{
			name: "choice with non-terminals",
			expr: `(+ [stmt1] [stmt2])`,
		},
		{
			name: "optional, or choice with empty",
			expr: `(+ "optional" ())`,
		},
		{
			name: "loop with separator",
			expr: `(- [digit] ",")`,
		},
		{
			name: "zero-or-more (loop with empty)",
			expr: `(- "a" ())`,
		},
		{
			name: "sequence with choice",
			expr: `("if" (+ "a" "b") "then")`,
		},
		{
			name: "sequence with choice",
			expr: `("if" (+ "a" "b") "then")`,
		},
		{
			name: "nested sequence",
			expr: `(("if" [expr]) "then")`,
		},
		{
			name: "choice of sequences",
			expr: `(+ ("if" [expr]) ("while" [expr]))`,
		},
		{
			name: "loop of sequence",
			expr: `(- ([digit] [digit]) ",")`,
		},
		{
			name: "nested loops",
			expr: `(- (- "a" ()) "b")`,
		},
		{
			name: "nested choices",
			expr: `(+ (+ "a" "b") "c")`,
		},
		{
			name: "complex: if-then-else",
			expr: `("if" [expr] "then" [stmt] (+ ("else" [stmt]) ()))`,
		},
		{
			name: "complex: while loop",
			expr: `("while" [expr] "do" [stmt])`,
		},
		{
			name: "complex: comma-separated list",
			expr: `([item] (- ("," [item]) ()))`,
		},
		{
			name: "choice with three alternatives",
			expr: `(+ (+ "a" "b") "c")`,
		},
		{
			name: "empty sequence element",
			expr: `("start" () "end")`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d, err := fromBytes([]byte(test.expr))
			assert.NoError(t, err)
			assert.Equal(t, test.expr, d.String())
		})
	}
}
