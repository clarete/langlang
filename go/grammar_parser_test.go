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
			ExpectedOutput: `Grammar (0..6)
└── Definition[A] (0..6)
    └── Sequence (5..6)
        └── Any (5..6)`,
		},

		{
			Name:    "Choice",
			Grammar: "A <- 'a' / 'b'",
			ExpectedOutput: `Grammar (0..14)
└── Definition[A] (0..14)
    └── Choice (5..14)
        ├── Sequence (5..9)
        │   └── Literal[a] (6..7)
        └── Sequence (11..14)
            └── Literal[b] (12..13)`,
		},
		{
			Name:    "Comment",
			Grammar: "A <- . // something something",
			ExpectedOutput: `Grammar (0..29)
└── Definition[A] (0..29)
    └── Sequence (5..29)
        └── Any (5..6)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser([]byte(test.Grammar))
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
			ExpectedOutput: `Grammar (0..6)
└── Definition[G] (0..6)
    └── Sequence (5..6)
        └── Any (5..6)`,
		},
		{
			Name:    "Single Choice",
			Grammar: "G <- 'a' / 'b' / 'c'",
			ExpectedOutput: `Grammar (0..20)
└── Definition[G] (0..20)
    └── Choice (5..20)
        ├── Sequence (5..9)
        │   └── Literal[a] (6..7)
        └── Choice (11..20)
            ├── Sequence (11..15)
            │   └── Literal[b] (12..13)
            └── Sequence (17..20)
                └── Literal[c] (18..19)`,
		},
		{
			Name:    "More Items",
			Grammar: "G <- A B C 'D'",
			ExpectedOutput: `Grammar (0..14)
└── Definition[G] (0..14)
    └── Sequence (5..14)
        ├── Identifier[A] (5..6)
        ├── Identifier[B] (7..8)
        ├── Identifier[C] (9..10)
        └── Literal[D] (12..13)`,
		},
		{
			Name:    "Sequence with Optional followed by ID",
			Grammar: "G <- .? x",
			ExpectedOutput: `Grammar (0..9)
└── Definition[G] (0..9)
    └── Sequence (5..9)
        ├── Optional (5..7)
        │   └── Any (5..6)
        └── Identifier[x] (8..9)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser([]byte(test.Grammar))
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
			ExpectedOutput: `Grammar (0..7)
└── Definition[G] (0..7)
    └── Sequence (5..7)
        └── And (5..7)
            └── Any (6..7)`,
		},
		{
			Name:    "Not",
			Grammar: "G <- !.",
			ExpectedOutput: `Grammar (0..7)
└── Definition[G] (0..7)
    └── Sequence (5..7)
        └── Not (5..7)
            └── Any (6..7)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser([]byte(test.Grammar))
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
			ExpectedOutput: `Grammar (0..7)
└── Definition[G] (0..7)
    └── Sequence (5..7)
        └── Optional (5..7)
            └── Any (5..6)`,
		},
		{
			Name:    "Zero or More",
			Grammar: "G <- .*",
			ExpectedOutput: `Grammar (0..7)
└── Definition[G] (0..7)
    └── Sequence (5..7)
        └── ZeroOrMore (5..7)
            └── Any (5..6)`,
		},
		{
			Name:    "One or More",
			Grammar: "G <- .+",
			ExpectedOutput: `Grammar (0..7)
└── Definition[G] (0..7)
    └── Sequence (5..7)
        └── OneOrMore (5..7)
            └── Any (5..6)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser([]byte(test.Grammar))
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
			ExpectedOutput: `Grammar (0..6)
└── Definition[G] (0..6)
    └── Sequence (5..6)
        └── Any (5..6)`,
		},
		{
			Name:    "Single Quote Literal",
			Grammar: "G <- 'abcd'",
			ExpectedOutput: `Grammar (0..11)
└── Definition[G] (0..11)
    └── Sequence (5..11)
        └── Literal[abcd] (6..10)`,
		},
		{
			Name:    "Double Quote Literal",
			Grammar: `G <- "abcd"`,
			ExpectedOutput: `Grammar (0..11)
└── Definition[G] (0..11)
    └── Sequence (5..11)
        └── Literal[abcd] (6..10)`,
		},
		{
			Name:    "Identifier",
			Grammar: "G <- FooBarBaz",
			ExpectedOutput: `Grammar (0..14)
└── Definition[G] (0..14)
    └── Sequence (5..14)
        └── Identifier[FooBarBaz] (5..14)`,
		},
		{
			Name:    "Class with single Range",
			Grammar: "G <- [0-9]",
			ExpectedOutput: `Grammar (0..10)
└── Definition[G] (0..10)
    └── Sequence (5..10)
        └── Class (5..10)
            └── Range[0, 9] (6..9)`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser([]byte(test.Grammar))
			output, err := parser.Parse()
			require.NoError(t, err)
			assert.Equal(t, test.ExpectedOutput, output.PrettyString())
		})
	}
}
