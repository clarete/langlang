package python

import (
	"fmt"
	"testing"
	"github.com/clarete/langlang/go"
	"github.com/stretchr/testify/require"
)

func TestPyDebugSpacing(t *testing.T) {
	// Test 1: Custom Spacing that only matches horizontal whitespace
	grammar1 := `
FileInput <- (NL / Statement)* EOF
EOF <- !.
Statement <- SimpleStmts
SimpleStmts <- SimpleStmt NEWLINE
SimpleStmt <- ReturnStmt / ExprStmt
ReturnStmt <- RETURN Expr?
ExprStmt <- Expr
Expr <- NAME
NAME <- !Keyword Ident Spacing
Ident <- [a-zA-Z_] [a-zA-Z_0-9]*
Keyword <- ('return') ![a-zA-Z_0-9]
RETURN <- 'return' ![a-zA-Z_0-9] Spacing
NEWLINE <- #([ \t]* '\r'? '\n') / #([ \t]* !.)
NL <- #([ \t]* '\r'? '\n')
Spacing <- #([ \t])*
`
	// Test 2: No custom Spacing (use built-in)
	grammar2 := `
FileInput <- (NL / Statement)* EOF
EOF <- !.
Statement <- SimpleStmts
SimpleStmts <- SimpleStmt NEWLINE
SimpleStmt <- ReturnStmt / ExprStmt
ReturnStmt <- RETURN Expr?
ExprStmt <- Expr
Expr <- NAME
NAME <- !Keyword Ident
Ident <- #([a-zA-Z_] [a-zA-Z_0-9]*)
Keyword <- #('return' ![a-zA-Z_0-9])
RETURN <- #('return' ![a-zA-Z_0-9])
NEWLINE <- #([ \t]* '\r'? '\n') / #([ \t]* !.)
NL <- #([ \t]* '\r'? '\n')
`
	for i, grammar := range []string{grammar1, grammar2} {
		fmt.Printf("=== Grammar %d ===\n", i+1)
		matcher, err := langlang.MatcherFromString(grammar)
		require.NoError(t, err)
		inputs := []string{
			"return\n",
			"x\n",
			"return x\n",
		}
		for _, input := range inputs {
			_, n, err := matcher.Match([]byte(input))
			status := "OK"
			if err != nil {
				status = fmt.Sprintf("FAIL(%v)", err)
			}
			fmt.Printf("  %s %q: n=%d len=%d\n", status, input, n, len(input))
		}
	}
}
