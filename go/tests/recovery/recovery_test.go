package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate go run ../../cmd/langlang -language go -grammar ./recovery.peg -output ./recovery.go

func TestRecovery(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Input    string
		Expected string
	}{
		{
			Name:     "Basic Check Assignment",
			Input:    "var = 1;",
			Expected: `P(Stm(AssignStm(Sequence(Identifier(Sequence("v" @ 0..1, Sequence("a" @ 1..2, "r" @ 2..3) @ 1..3) @ 0..3) @ 0..3, "=" @ 4..5, Expr(Number("1" @ 6..7) @ 6..7) @ 6..7, ";" @ 7..8) @ 0..8) @ 0..8) @ 0..8) @ 0..8`,
		},
		{
			Name:     "Basic Check If Statement",
			Input:    "if (false) {}",
			Expected: `P(Stm(IfStm(Sequence("if" @ 0..2, "(" @ 3..4, Expr(Bool("false" @ 4..9) @ 4..9) @ 4..9, ")" @ 9..10, Body(Sequence("{" @ 11..12, "}" @ 12..13) @ 11..13) @ 11..13) @ 0..13) @ 0..13) @ 0..13) @ 0..13`,
		},
		{
			Name:     "Missing Semi Colon in Assignment",
			Input:    "var = 1",
			Expected: `P(Stm(AssignStm(Sequence(Identifier(Sequence("v" @ 0..1, Sequence("a" @ 1..2, "r" @ 2..3) @ 1..3) @ 0..3) @ 0..3, "=" @ 4..5, Expr(Number("1" @ 6..7) @ 6..7) @ 6..7, Error("assignsemi") @ 7) @ 0..7) @ 0..7) @ 0..7) @ 0..7`,
		},
		{
			Name:     "Missing Left Parenthesis in If Expression",
			Input:    "if false) {}",
			Expected: `P(Stm(IfStm(Sequence("if" @ 0..2, Error("iflpar") @ 3, Expr(Bool("false" @ 3..8) @ 3..8) @ 3..8, ")" @ 8..9, Body(Sequence("{" @ 10..11, "}" @ 11..12) @ 10..12) @ 10..12) @ 0..12) @ 0..12) @ 0..12) @ 0..12`,
		},
		{
			Name:     "Missing If Expression",
			Input:    "if () {}",
			Expected: `P(Stm(IfStm(Sequence("if" @ 0..2, "(" @ 3..4, Error("ifexpr") @ 4, ")" @ 4..5, Body(Sequence("{" @ 6..7, "}" @ 7..8) @ 6..8) @ 6..8) @ 0..8) @ 0..8) @ 0..8) @ 0..8`,
		},
		{
			Name:     "Garbage in If Expression",
			Input:    "if ($%^) {}",
			Expected: `P(Stm(IfStm(Sequence("if" @ 0..2, "(" @ 3..4, Error("ifexpr", ifexpr(Sequence("$" @ 4..5, "%" @ 5..6, "^" @ 6..7) @ 4..7) @ 4..7) @ 4..7, ")" @ 7..8, Body(Sequence("{" @ 9..10, "}" @ 10..11) @ 9..11) @ 9..11) @ 0..11) @ 0..11) @ 0..11) @ 0..11`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			p := NewParser()
			p.SetInput(test.Input)
			p.SetCaptureSpaces(false)
			v, err := p.ParseP()
			require.NoError(t, err)
			assert.Equal(t, test.Expected, v.String())
		})
	}
}
