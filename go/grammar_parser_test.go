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
			ExpectedOutput: `Grammar (1..7)
└── Definition[A] (1..7)
    └── Sequence (6..7)
        └── Any (6..7)`,
		},

		{
			Name:    "Choice",
			Grammar: "A <- 'a' / 'b'",
			ExpectedOutput: `Grammar (1..15)
└── Definition[A] (1..15)
    └── Choice (6..15)
        ├── Sequence (6..10)
        │   └── Literal[a] (6..9)
        └── Sequence (12..15)
            └── Literal[b] (12..15)`,
		},
		{
			Name:    "Comment",
			Grammar: "A <- . // something something",
			ExpectedOutput: `Grammar (1..30)
└── Definition[A] (1..30)
    └── Sequence (6..30)
        └── Any (6..7)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser(test.Grammar)
			output, err := parser.Parse()
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
			Grammar: "G <- .",
			ExpectedOutput: `Grammar (1..7)
└── Definition[G] (1..7)
    └── Sequence (6..7)
        └── Any (6..7)`,
		},
		{
			Name:    "Single Choice",
			Grammar: "G <- 'a' / 'b' / 'c'",
			ExpectedOutput: `Grammar (1..21)
└── Definition[G] (1..21)
    └── Choice (6..21)
        ├── Sequence (6..10)
        │   └── Literal[a] (6..9)
        └── Choice (12..21)
            ├── Sequence (12..16)
            │   └── Literal[b] (12..15)
            └── Sequence (18..21)
                └── Literal[c] (18..21)`,
		},
		{
			Name:    "More Items",
			Grammar: "G <- A B C 'D'",
			ExpectedOutput: `Grammar (1..15)
└── Definition[G] (1..15)
    └── Sequence (6..15)
        ├── Identifier[A] (6..7)
        ├── Identifier[B] (8..9)
        ├── Identifier[C] (10..11)
        └── Literal[D] (12..15)`,
		},
		{
			Name:    "Sequence with Optional followed by ID",
			Grammar: "G <- .? x",
			ExpectedOutput: `Grammar (1..10)
└── Definition[G] (1..10)
    └── Sequence (6..10)
        ├── Optional (6..8)
        │   └── Any (6..7)
        └── Identifier[x] (9..10)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser(test.Grammar)
			output, err := parser.Parse()
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
			Grammar: "G <- &.",
			ExpectedOutput: `Grammar (1..8)
└── Definition[G] (1..8)
    └── Sequence (6..8)
        └── And (6..8)
            └── Any (7..8)`,
		},
		{
			Name:    "Not",
			Grammar: "G <- !.",
			ExpectedOutput: `Grammar (1..8)
└── Definition[G] (1..8)
    └── Sequence (6..8)
        └── Not (6..8)
            └── Any (7..8)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser(test.Grammar)
			output, err := parser.Parse()
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
			Grammar: "G <- .?",
			ExpectedOutput: `Grammar (1..8)
└── Definition[G] (1..8)
    └── Sequence (6..8)
        └── Optional (6..8)
            └── Any (6..7)`,
		},
		{
			Name:    "Zero or More",
			Grammar: "G <- .*",
			ExpectedOutput: `Grammar (1..8)
└── Definition[G] (1..8)
    └── Sequence (6..8)
        └── ZeroOrMore (6..8)
            └── Any (6..7)`,
		},
		{
			Name:    "One or More",
			Grammar: "G <- .+",
			ExpectedOutput: `Grammar (1..8)
└── Definition[G] (1..8)
    └── Sequence (6..8)
        └── OneOrMore (6..8)
            └── Any (6..7)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser(test.Grammar)
			output, err := parser.Parse()
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
			Name:    "Any",
			Grammar: "G <- .",
			ExpectedOutput: `Grammar (1..7)
└── Definition[G] (1..7)
    └── Sequence (6..7)
        └── Any (6..7)`,
		},
		{
			Name:    "Single Quote Literal",
			Grammar: "G <- 'abcd'",
			ExpectedOutput: `Grammar (1..12)
└── Definition[G] (1..12)
    └── Sequence (6..12)
        └── Literal[abcd] (6..12)`,
		},
		{
			Name:    "Double Quote Literal",
			Grammar: `G <- "abcd"`,
			ExpectedOutput: `Grammar (1..12)
└── Definition[G] (1..12)
    └── Sequence (6..12)
        └── Literal[abcd] (6..12)`,
		},
		{
			Name:    "Identifier",
			Grammar: "G <- FooBarBaz",
			ExpectedOutput: `Grammar (1..15)
└── Definition[G] (1..15)
    └── Sequence (6..15)
        └── Identifier[FooBarBaz] (6..15)`,
		},
		{
			Name:    "Class with single Range",
			Grammar: "G <- [0-9]",
			ExpectedOutput: `Grammar (1..11)
└── Definition[G] (1..11)
    └── Sequence (6..11)
        └── Class (6..11)
            └── Range[0, 9] (7..10)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser(test.Grammar)
			output, err := parser.Parse()
			require.NoError(t, err)
			assert.Equal(t, test.ExpectedOutput, output.PrettyString())
		})
	}
}
