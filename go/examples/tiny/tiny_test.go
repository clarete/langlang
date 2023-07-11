package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

//go:generate go run ../../cmd -language go -grammar ./tiny.peg -output ./tiny.go

func TestTiny(t *testing.T) {
	t.Run("Expr", func(t *testing.T) {
		assertExpr(t, "an_id", "an_id")
		assertExpr(t, "12345", "12345")
		assertExpr(t, "(2+34)", "(2+34)")
		assertExpr(t, "Call()", "Call()")
	})

	t.Run("Call", func(t *testing.T) {
		assertCall(t, "NoParams()", "NoParams()")
		assertCall(t, "OneParam(10)", "OneParam(10)")
		assertCall(t, "TwoParams(a, b)", "TwoParams(a, b)")
		assertCall(t, "ThreeParamsRecurse(a, Rec(x, y), c)", "ThreeParamsRecurse(a, Rec(x, y), c)")
	})

	t.Run("Num", func(t *testing.T) {
		assertNum(t, "12340", "12340")
		assertNum(t, "43210", "43210")
	})

	t.Run("Id", func(t *testing.T) {
		assertId(t, "just_an_id", "just_an_id")
		assertId(t, "_start_with_underscore", "_start_with_underscore")
		assertId(t, "can_have_numbers_1234", "can_have_numbers_1234")
	})
}

func assertExpr(t *testing.T, expected, input string) {
	v, err := NewParser(input).ParsePrimary()
	require.NoError(t, err)
	require.Equal(t, expected, v.Text())
}

func assertCall(t *testing.T, expected, input string) {
	v, err := NewParser(input).ParseCall()
	require.NoError(t, err)
	require.Equal(t, expected, v.Text())
}

func assertNum(t *testing.T, expected, input string) {
	v, err := NewParser(input).ParseNum()
	require.NoError(t, err)
	require.Equal(t, expected, v.Text())
}

func assertId(t *testing.T, expected, input string) {
	v, err := NewParser(input).ParseId()
	require.NoError(t, err)
	require.Equal(t, expected, v.Text())
}
