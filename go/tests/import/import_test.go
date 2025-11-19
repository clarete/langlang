package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate go run ../../cmd/langlang -grammar ./import_gr_expr.peg -output-language go -output-path ./import.go
//go:generate go run ../../cmd/langlang -grammar ./import_gr_expr.peg -output-language go -output-path ./import.nocap.go -disable-captures -go-parser NoCapParser -go-remove-lib

func TestImport(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Match    string
		Expected string
	}{
		{
			Name:  "First Level",
			Match: "true",
			Expected: `Expr (1..5)
└── Term (1..5)
    └── Multi (1..5)
        └── Primary (1..5)
            └── Value (1..5)
                └── Boolean (1..5)
                    └── "true" (1..5)`,
		},
		{
			Name:  "Second Level: Single Quote",
			Match: "'foo bar'",
			Expected: `Expr (1..10)
└── Term (1..10)
    └── Multi (1..10)
        └── Primary (1..10)
            └── Value (1..10)
                └── String (1..10)
                    └── SingleQuote (1..10)
                        └── "'foo bar'" (1..10)`,
		},
		{
			Name:  "Second Level: Number",
			Match: "0.55",
			Expected: `Expr (1..5)
└── Term (1..5)
    └── Multi (1..5)
        └── Primary (1..5)
            └── Value (1..5)
                └── Number (1..5)
                    └── Float (1..5)
                        └── Sequence<3> (1..5)
                            ├── Decimal (1..2)
                            │   └── "0" (1..2)
                            ├── "." (2..3)
                            └── Decimal (3..5)
                                └── "55" (3..5)`,
		},
		{
			Name:  "Third Level",
			Match: "3 + 2",
			Expected: `Expr (1..6)
└── Term (1..6)
    └── Sequence<4> (1..6)
        ├── Multi (1..3)
        │   └── Sequence<2> (1..3)
        │       ├── Primary (1..2)
        │       │   └── Value (1..2)
        │       │       └── Number (1..2)
        │       │           └── Decimal (1..2)
        │       │               └── "3" (1..2)
        │       └── Spacing (2..3)
        │           └── " " (2..3)
        ├── "+" (3..4)
        ├── Spacing (4..5)
        │   └── " " (4..5)
        └── Multi (5..6)
            └── Primary (5..6)
                └── Value (5..6)
                    └── Number (5..6)
                        └── Decimal (5..6)
                            └── "2" (5..6)`,
		},
		{
			Name:  "Override",
			Match: "0xC0FFEE",
			Expected: `Expr (1..9)
└── Term (1..9)
    └── Multi (1..9)
        └── Primary (1..9)
            └── Value (1..9)
                └── Number (1..9)
                    └── Hexadecimal (1..9)
                        └── Sequence<7> (1..9)
                            ├── "0x" (1..3)
                            ├── "C" (3..4)
                            ├── "0" (4..5)
                            ├── "F" (5..6)
                            ├── "F" (6..7)
                            ├── "E" (7..8)
                            └── "E" (8..9)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			p := NewParser()
			p.SetInput([]byte(test.Match))
			v, err := p.ParseExpr()
			require.NoError(t, err)
			assert.Equal(t, test.Expected, PrettyString(p.GetInput(), v))
		})
	}

	for _, test := range []struct {
		Name  string
		Input string
		Error string
	}{
		{
			Name:  "Dont accept overriden",
			Input: "3+#xC0FFEE",
			Error: `[TermRightOperand] Unexpected '#' @ 3`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			p := NewParser()
			p.SetInput([]byte(test.Input))
			p.SetShowFails(false)
			_, err := p.ParseExpr()
			require.Error(t, err)
			assert.Equal(t, test.Error, err.Error())
		})
	}

	for _, test := range []struct {
		Name     string
		Input    string
		Expected string
	}{
		{
			Name:  "label dependency",
			Input: "0xG + 2",
			Expected: `Expr (1..8)
└── Term (1..8)
    └── Sequence<4> (1..8)
        ├── Multi (1..5)
        │   └── Sequence<2> (1..5)
        │       ├── Primary (1..4)
        │       │   └── Value (1..4)
        │       │       └── Number (1..4)
        │       │           └── Hexadecimal (1..4)
        │       │               └── Sequence<2> (1..4)
        │       │                   ├── "0x" (1..3)
        │       │                   └── Error<LabelHex> (3..4)
        │       │                       └── "G" (3..4)
        │       └── Spacing (4..5)
        │           └── " " (4..5)
        ├── "+" (5..6)
        ├── Spacing (6..7)
        │   └── " " (6..7)
        └── Multi (7..8)
            └── Primary (7..8)
                └── Value (7..8)
                    └── Number (7..8)
                        └── Decimal (7..8)
                            └── "2" (7..8)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			p := NewParser()
			p.SetInput([]byte(test.Input))
			v, err := p.ParseExpr()
			require.NoError(t, err)
			require.NotNil(t, v)
			assert.Equal(t, test.Expected, PrettyString(p.GetInput(), v))
		})
	}
}

type scenario struct {
	Name  string
	Query []byte
}

var tests = []scenario{
	{
		Name:  "Single Digit",
		Query: []byte("1"),
	},
	{
		Name:  "Term",
		Query: []byte("41 + 22"),
	},
	{
		Name:  "Multi",
		Query: []byte("33 * 44"),
	},
}

func BenchmarkParser(b *testing.B) {
	p := NewParser()
	p.SetShowFails(false)

	for _, scenario := range tests {
		b.Run(scenario.Name, func(b *testing.B) {
			b.SetBytes(int64(len(scenario.Query)))
			p.SetInput(scenario.Query)

			for b.Loop() {
				p.ParseExpr()
			}
		})
	}
}

func BenchmarkNoCapParser(b *testing.B) {
	p := NewNoCapParser()
	p.SetShowFails(false)

	for _, scenario := range tests {
		b.Run(scenario.Name, func(b *testing.B) {
			b.SetBytes(int64(len(scenario.Query)))
			p.SetInput(scenario.Query)

			for b.Loop() {
				p.ParseExpr()
			}
		})
	}
}
