package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate go run ../../cmd/langlang -grammar ./charsets.peg -output-language goeval -output-path ./charsets.go
//go:generate go run ../../cmd/langlang -grammar ./charsets.peg -output-language goeval -output-path ./charsets.nocap.go -disable-captures -go-parser NoCapParser -go-remove-lib

type P interface {
	ParseIdentifier() (Value, error)
	ParseDigits() (Value, error)
	ParseHiragana() (Value, error)
}

type test struct {
	Name     string
	Input    string
	Expected string
	Func     func(P) (Value, error)
}

var (
	identifierFn = func(p P) (Value, error) { return p.ParseIdentifier() }
	digitsFn     = func(p P) (Value, error) { return p.ParseDigits() }
	hiraganaFn   = func(p P) (Value, error) { return p.ParseHiragana() }
)

var tests = []test{
	{
		Name:  "Identifier Letter",
		Input: "A",
		Func:  identifierFn,
		Expected: `Identifier (1..2)
└── "A" (1..2)`,
	},
	{
		Name:  "Identifier Word",
		Input: "Abacate",
		Func:  identifierFn,
		Expected: `Identifier (1..8)
└── "Abacate" (1..8)`,
	},
	{
		Name:  "Digit Single",
		Input: "0",
		Func:  digitsFn,
		Expected: `Digits (1..2)
└── "0" (1..2)`,
	},
	{
		Name:  "Digit Multiple",
		Input: "1234",
		Func:  digitsFn,
		Expected: `Digits (1..5)
└── "1234" (1..5)`,
	},
	{
		Name:  "Hiragana Single",
		Input: "あ",
		Func:  hiraganaFn,
		Expected: `Hiragana (1..2)
└── "あ" (1..2)`,
	},
	{
		Name:  "Hiragana Many",
		Input: "こんにちは",
		Func:  hiraganaFn,
		Expected: `Hiragana (1..6)
└── "こんにちは" (1..6)`,
	},
}

func TestCharset(t *testing.T) {
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			p := NewParser()
			p.SetInput(test.Input)
			p.SetCaptureSpaces(false)
			v, err := test.Func(p)
			require.NoError(t, err)
			assert.Equal(t, test.Expected, v.PrettyString())
		})
	}
}

func BenchmarkParser(b *testing.B) {

	b.ResetTimer()

	for _, test := range tests {
		b.Run(test.Name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				p := NewNoCapParser()
				p.SetInput(test.Input)
				test.Func(p)
			}
		})
	}
}

func BenchmarkNoCapParser(b *testing.B) {

	b.ResetTimer()

	for _, test := range tests {
		b.Run(test.Name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				p := NewNoCapParser()
				p.SetInput(test.Input)
				test.Func(p)
			}
		})
	}
}
