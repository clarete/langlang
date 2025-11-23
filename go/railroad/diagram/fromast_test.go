package diagram

import (
	"testing"

	"github.com/clarete/langlang/go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiagramFromAstNode(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		{
			name:     "lone literal",
			expr:     `'literal'`,
			expected: `"literal"`,
		},
		{
			name:     "non terminal",
			expr:     `G`,
			expected: `[G]`,
		},
		{
			name:     "sequence",
			expr:     `'lit1' nt1 'lit2' nt2 `,
			expected: `("lit1" [nt1] "lit2" [nt2])`,
		},
		{
			name:     "choice",
			expr:     `'a' / 'b'`,
			expected: `(+ "a" "b")`,
		},
		{
			name:     "zero or more",
			expr:     `'a'*`,
			expected: `(- "a" ())`,
		},
		{
			name:     "one or more",
			expr:     `'a'+`,
			expected: `(- "a" ())`,
		},
		{
			name:     "optional",
			expr:     `'a'?`,
			expected: `(+ "a" ())`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			i := "G <- " + test.expr
			p := langlang.NewGrammarParser([]byte(i))

			ast, err := p.Parse()
			require.NoError(t, err)

			grammar, ok := ast.(*langlang.GrammarNode)
			require.True(t, ok)

			df := grammar.DefsByName["G"]
			dg := fromAstNode(df.Expr)
			st := dg.String()
			assert.Equal(t, test.expected, st)
		})
	}
}
