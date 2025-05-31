package langlang

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDefinition(t *testing.T) {
	for _, test := range []struct {
		Name           string
		Grammar        string
		ExpectedOutput string
	}{
		{
			Name:    "Any",
			Grammar: "A <- .",
			ExpectedOutput: `Definition[A] (0..6)
└── Sequence (5..6)
    └── Any (5..6)`,
		},

		{
			Name:    "Choice",
			Grammar: "A <- 'a' / 'b'",
			ExpectedOutput: `Definition[A] (0..14)
└── Choice (5..14)
    ├── Sequence (5..9)
    │   └── Literal[a] (5..8)
    └── Sequence (11..14)
        └── Literal[b] (11..14)`,
		},
		{
			Name:    "Comment",
			Grammar: "A <- . // something something",
			ExpectedOutput: `Definition[A] (0..29)
└── Sequence (5..29)
    └── Any (5..6)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser(test.Grammar)
			output, err := parser.ParseDefinition()
			require.NoError(t, err)
			assert.Equal(t, test.ExpectedOutput, output.PrettyString())
		})
	}
}

func TestParseExpression(t *testing.T) {
	for _, test := range []struct {
		Name           string
		Grammar        string
		ExpectedOutput string
	}{
		{
			Name:    "Single Item",
			Grammar: ".",
			ExpectedOutput: `Sequence (0..1)
└── Any (0..1)`,
		},
		{
			Name:    "Single Choice",
			Grammar: "'a' / 'b' / 'c'",
			ExpectedOutput: `Choice (0..15)
├── Sequence (0..4)
│   └── Literal[a] (0..3)
└── Choice (6..15)
    ├── Sequence (6..10)
    │   └── Literal[b] (6..9)
    └── Sequence (12..15)
        └── Literal[c] (12..15)`,
		},
		{
			Name:    "More Items",
			Grammar: "A B C 'D'",
			ExpectedOutput: `Sequence (0..9)
├── Identifier[A] (0..1)
├── Identifier[B] (2..3)
├── Identifier[C] (4..5)
└── Literal[D] (6..9)`,
		},
		{
			Name:    "Sequence with Optional followed by ID",
			Grammar: ".? x",
			ExpectedOutput: `Sequence (0..4)
├── Optional (0..2)
│   └── Any (0..1)
└── Identifier[x] (3..4)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser(test.Grammar)
			output, err := parser.ParseExpression()
			require.NoError(t, err)
			assert.Equal(t, test.ExpectedOutput, output.PrettyString())
		})
	}
}

func TestParsePrefix(t *testing.T) {
	for _, test := range []struct {
		Name           string
		Grammar        string
		ExpectedOutput string
	}{
		{
			Name:    "And",
			Grammar: "&.",
			ExpectedOutput: `And (0..2)
└── Any (1..2)`,
		},
		{
			Name:    "Not",
			Grammar: "!.",
			ExpectedOutput: `Not (0..2)
└── Any (1..2)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser(test.Grammar)
			output, err := parser.ParsePrefix()
			require.NoError(t, err)
			assert.Equal(t, test.ExpectedOutput, output.PrettyString())
		})
	}
}

func TestParseSuffix(t *testing.T) {
	for _, test := range []struct {
		Name           string
		Grammar        string
		ExpectedOutput string
	}{
		{
			Name:    "Optional",
			Grammar: ".?",
			ExpectedOutput: `Optional (0..2)
└── Any (0..1)`,
		},
		{
			Name:    "Zero or More",
			Grammar: ".*",
			ExpectedOutput: `ZeroOrMore (0..2)
└── Any (0..1)`,
		},
		{
			Name:    "One or More",
			Grammar: ".+",
			ExpectedOutput: `OneOrMore (0..2)
└── Any (0..1)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser(test.Grammar)
			output, err := parser.ParseSuffix()
			require.NoError(t, err)
			assert.Equal(t, test.ExpectedOutput, output.PrettyString())
		})
	}
}

func TestParsePrimary(t *testing.T) {
	for _, test := range []struct {
		Name           string
		Grammar        string
		ExpectedOutput string
	}{
		{
			Name:           "Any",
			Grammar:        ".",
			ExpectedOutput: "Any (0..1)",
		},
		{
			Name:           "Single Quote Literal",
			Grammar:        "'abcd'",
			ExpectedOutput: "Literal[abcd] (0..6)",
		},
		{
			Name:           "Double Quote Literal",
			Grammar:        `"abcd"`,
			ExpectedOutput: "Literal[abcd] (0..6)",
		},
		{
			Name:           "Identifier",
			Grammar:        "FooBarBaz",
			ExpectedOutput: "Identifier[FooBarBaz] (0..9)",
		},
		{
			Name:    "Class with single Range",
			Grammar: "[0-9]",
			ExpectedOutput: `Class (0..5)
└── Range[0, 9] (1..4)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser(test.Grammar)
			output, err := parser.ParsePrimary()
			require.NoError(t, err)
			assert.Equal(t, test.ExpectedOutput, output.PrettyString())
		})
	}
}
