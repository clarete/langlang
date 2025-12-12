package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate go run ../../cmd/langlang -grammar ./recovery.peg -output-language go -output-path ./recovery.go -disable-capture-spaces

func TestRecoverySuccess(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Input    string
		Expected string
	}{
		{
			Name:  "Basic Check Assignment",
			Input: "var = 1;",
			Expected: `P (1..9)
└── Stm (1..9)
    └── AssignStm (1..9)
        └── Sequence<4> (1..9)
            ├── Identifier (1..4)
            │   └── "var" (1..4)
            ├── "=" (5..6)
            ├── Expr (7..8)
            │   └── Number (7..8)
            │       └── "1" (7..8)
            └── ";" (8..9)`,
		},
		{
			Name:  "Basic Check If Statement",
			Input: "if (false) {}",
			Expected: `P (1..14)
└── Stm (1..14)
    └── IfStm (1..14)
        └── Sequence<5> (1..14)
            ├── "if" (1..3)
            ├── "(" (4..5)
            ├── Expr (5..10)
            │   └── Bool (5..10)
            │       └── "false" (5..10)
            ├── ")" (10..11)
            └── Body (12..14)
                └── Sequence<2> (12..14)
                    ├── "{" (12..13)
                    └── "}" (13..14)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			p := NewParser()
			p.SetInput([]byte(test.Input))
			v, err := p.ParseP()
			require.NoError(t, err)
			root, hasRoot := v.Root()
			require.True(t, hasRoot)
			assert.Equal(t, test.Expected, v.Pretty(root))
		})
	}
}

func TestRecovery(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Input    string
		Expected string
	}{
		{
			Name:  "Missing Semi Colon in Assignment",
			Input: "var = 1",
			Expected: `P (1..8)
└── Stm (1..8)
    └── AssignStm (1..8)
        └── Sequence<4> (1..8)
            ├── Identifier (1..4)
            │   └── "var" (1..4)
            ├── "=" (5..6)
            ├── Expr (7..8)
            │   └── Number (7..8)
            │       └── "1" (7..8)
            └── Error<assignsemi> (8)`,
		},
		{
			Name:  "Missing Left Parenthesis in If Expression",
			Input: "if false) {}",
			Expected: `P (1..13)
└── Stm (1..13)
    └── IfStm (1..13)
        └── Sequence<5> (1..13)
            ├── "if" (1..3)
            ├── Error<iflpar> (4)
            ├── Expr (4..9)
            │   └── Bool (4..9)
            │       └── "false" (4..9)
            ├── ")" (9..10)
            └── Body (11..13)
                └── Sequence<2> (11..13)
                    ├── "{" (11..12)
                    └── "}" (12..13)`,
		},
		{
			Name:  "Missing If Expression",
			Input: "if () {}",
			Expected: `P (1..9)
└── Stm (1..9)
    └── IfStm (1..9)
        └── Sequence<5> (1..9)
            ├── "if" (1..3)
            ├── "(" (4..5)
            ├── Error<ifexpr> (5)
            ├── ")" (5..6)
            └── Body (7..9)
                └── Sequence<2> (7..9)
                    ├── "{" (7..8)
                    └── "}" (8..9)`,
		},
		{
			Name:  "Garbage in If Expression",
			Input: "if ($%^) {}",
			Expected: `P (1..12)
└── Stm (1..12)
    └── IfStm (1..12)
        └── Sequence<5> (1..12)
            ├── "if" (1..3)
            ├── "(" (4..5)
            ├── Error<ifexpr> (5..8)
            │   └── "$%^" (5..8)
            ├── ")" (8..9)
            └── Body (10..12)
                └── Sequence<2> (10..12)
                    ├── "{" (10..11)
                    └── "}" (11..12)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			p := NewParser()
			p.SetInput([]byte(test.Input))
			v, err := p.ParseP()
			require.NoError(t, err)
			require.NotNil(t, v)
			root, hasRoot := v.Root()
			require.True(t, hasRoot)
			assert.Equal(t, test.Expected, v.Pretty(root))
		})
	}
}
