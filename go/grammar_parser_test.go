package langlang

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUnicodeEscapes(t *testing.T) {
	for _, test := range []struct {
		Name        string
		Escape      string
		FixedEscape string
		Value       string
		WantError   bool
	}{
		{
			Name:        "before surrogates",
			Escape:      `\u{D7FF}`,
			FixedEscape: `\uD7FF`,
			Value:       "\uD7FF",
		},
		{
			Name:        "first high surrogate",
			Escape:      `\u{D800}`,
			FixedEscape: `\uD800`,
			WantError:   true,
		},
		{
			Name:        "last high surrogate",
			Escape:      `\u{DBFF}`,
			FixedEscape: `\uDBFF`,
			WantError:   true,
		},
		{
			Name:        "first low surrogate",
			Escape:      `\u{DC00}`,
			FixedEscape: `\uDC00`,
			WantError:   true,
		},
		{
			Name:        "last low surrogate",
			Escape:      `\u{DFFF}`,
			FixedEscape: `\uDFFF`,
			WantError:   true,
		},
		{
			Name:        "after surrogates",
			Escape:      `\u{E000}`,
			FixedEscape: `\uE000`,
			Value:       "\uE000",
		},
		{
			Name:        "replacement character",
			Escape:      `\u{FFFD}`,
			FixedEscape: `\uFFFD`,
			Value:       "\uFFFD",
		},
		{
			Name:   "supplementary character",
			Escape: `\u{1F600}`,
			Value:  "😀",
		},
		{
			Name:   "maximum scalar value",
			Escape: `\u{10FFFF}`,
			Value:  "\U0010FFFF",
		},
		{
			Name:      "above maximum scalar value",
			Escape:    `\u{110000}`,
			WantError: true,
		},
		{
			Name:        "surrogate pair",
			Escape:      `\u{D83D}\u{DE00}`,
			FixedEscape: `\uD83D\uDE00`,
			WantError:   true,
		},
	} {
		for _, form := range []struct {
			Name   string
			Escape string
		}{
			{
				Name:   "braced decoder",
				Escape: test.Escape,
			},
			{
				Name:   "four digit decoder",
				Escape: test.FixedEscape,
			},
		} {
			if form.Escape == "" {
				continue
			}
			t.Run(test.Name+"/"+form.Name, func(t *testing.T) {
				value, err := unescape(form.Escape)
				if test.WantError {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, test.Value, value)
			})
		}
		for _, form := range []struct {
			Name    string
			Grammar string
		}{
			{
				Name:    "single quoted literal",
				Grammar: "G <- '%s'",
			},
			{
				Name:    "double quoted literal",
				Grammar: `G <- "%s"`,
			},
			{
				Name:    "character class",
				Grammar: "G <- [%s]",
			},
		} {
			t.Run(test.Name+"/"+form.Name, func(t *testing.T) {
				parser := NewGrammarParser([]byte(fmt.Sprintf(form.Grammar, test.Escape)))
				output, err := parser.Parse()
				require.NoError(t, err)
				grammar := output.(*GrammarNode)
				if test.WantError {
					require.NotEmpty(t, grammar.Errors)
					return
				}
				require.Empty(t, grammar.Errors)
				require.Len(t, grammar.Definitions, 1)
				seq := grammar.Definitions[0].Expr.(*SequenceNode)
				require.Len(t, seq.Items, 1)
				value := seq.Items[0]
				if class, ok := value.(*ClassNode); ok {
					require.Len(t, class.Items, 1)
					value = class.Items[0]
				}
				assert.Equal(t, test.Value, value.(*LiteralNode).Value)
			})
		}
	}
}

func TestParseImportInvalidEscape(t *testing.T) {
	for _, test := range []struct {
		Name    string
		Grammar string
	}{
		{
			Name:    "escape sequence instead of source file",
			Grammar: `@import X from '\q'`,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			parser := NewGrammarParser([]byte(test.Grammar))
			output, err := parser.Parse()
			require.NoError(t, err)
			grammar := output.(*GrammarNode)
			require.Len(t, grammar.Errors, 1)
			assert.Contains(t, grammar.Errors[0].Message, "unknown unescape sequence: q")
			assert.Empty(t, grammar.Imports)
		})
	}
}

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
        ├── Sequence (6..9)
        │   └── Literal[a] (7..8)
        └── Sequence (12..15)
            └── Literal[b] (13..14)`,
		},
		{
			Name:    "Comment",
			Grammar: "A <- . // something something",
			ExpectedOutput: `Grammar (1..30)
└── Definition[A] (1..7)
    └── Sequence (6..7)
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
        ├── Sequence (6..9)
        │   └── Literal[a] (7..8)
        └── Choice (12..21)
            ├── Sequence (12..15)
            │   └── Literal[b] (13..14)
            └── Sequence (18..21)
                └── Literal[c] (19..20)`,
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
        └── Literal[D] (13..14)`,
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
        └── Literal[abcd] (7..11)`,
		},
		{
			Name:    "Double Quote Literal",
			Grammar: `G <- "abcd"`,
			ExpectedOutput: `Grammar (1..12)
└── Definition[G] (1..12)
    └── Sequence (6..12)
        └── Literal[abcd] (7..11)`,
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
			parser := NewGrammarParser([]byte(test.Grammar))
			output, err := parser.Parse()
			require.NoError(t, err)
			assert.Equal(t, test.ExpectedOutput, output.PrettyString())
		})
	}
}
