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
			ExpectedOutput: `Definition[A] (1..7)
└── Sequence (6..7)
    └── Any (6..7)`,
		},

		{
			Name:    "Choice",
			Grammar: "A <- 'a' / 'b'",
			ExpectedOutput: `Definition[A] (1..15)
└── Choice (6..15)
    ├── Sequence (6..10)
    │   └── Literal[a] (6..9)
    └── Sequence (12..15)
        └── Literal[b] (12..15)`,
		},
		{
			Name:    "Comment",
			Grammar: "A <- . // something something",
			ExpectedOutput: `Definition[A] (1..30)
└── Sequence (6..30)
    └── Any (6..7)`,
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
			ExpectedOutput: `Sequence (1..2)
└── Any (1..2)`,
		},
		{
			Name:    "Single Choice",
			Grammar: "'a' / 'b' / 'c'",
			ExpectedOutput: `Choice (1..16)
├── Sequence (1..5)
│   └── Literal[a] (1..4)
└── Choice (7..16)
    ├── Sequence (7..11)
    │   └── Literal[b] (7..10)
    └── Sequence (13..16)
        └── Literal[c] (13..16)`,
		},
		{
			Name:    "More Items",
			Grammar: "A B C 'D'",
			ExpectedOutput: `Sequence (1..10)
├── Identifier[A] (1..2)
├── Identifier[B] (3..4)
├── Identifier[C] (5..6)
└── Literal[D] (7..10)`,
		},
		{
			Name:    "Sequence with Optional followed by ID",
			Grammar: ".? x",
			ExpectedOutput: `Sequence (1..5)
├── Optional (1..3)
│   └── Any (1..2)
└── Identifier[x] (4..5)`,
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
			ExpectedOutput: `And (1..3)
└── Any (2..3)`,
		},
		{
			Name:    "Not",
			Grammar: "!.",
			ExpectedOutput: `Not (1..3)
└── Any (2..3)`,
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
			ExpectedOutput: `Optional (1..3)
└── Any (1..2)`,
		},
		{
			Name:    "Zero or More",
			Grammar: ".*",
			ExpectedOutput: `ZeroOrMore (1..3)
└── Any (1..2)`,
		},
		{
			Name:    "One or More",
			Grammar: ".+",
			ExpectedOutput: `OneOrMore (1..3)
└── Any (1..2)`,
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
			ExpectedOutput: "Any (1..2)",
		},
		{
			Name:           "Single Quote Literal",
			Grammar:        "'abcd'",
			ExpectedOutput: "Literal[abcd] (1..7)",
		},
		{
			Name:           "Double Quote Literal",
			Grammar:        `"abcd"`,
			ExpectedOutput: "Literal[abcd] (1..7)",
		},
		{
			Name:           "Identifier",
			Grammar:        "FooBarBaz",
			ExpectedOutput: "Identifier[FooBarBaz] (1..10)",
		},
		{
			Name:    "Class with single Range",
			Grammar: "[0-9]",
			ExpectedOutput: `Class (1..6)
└── Range[0, 9] (2..5)`,
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
