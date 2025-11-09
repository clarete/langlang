package langlang

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrammarTransformations(t *testing.T) {
	tests := []vmTest{
		{
			Name:    "Capture Any",
			Grammar: "G <- .",
			ExpectedAST: `Grammar (0..6)
└── Definition[G] (0..6)
    └── Capture[G] (0..6)
        └── Any (5..6)`,
		},
		{
			Name:    "Charset from Classes",
			Grammar: "G <- [a-zA-Z0-9_.]",
			ExpectedAST: `Grammar (0..18)
└── Definition[G] (0..18)
    └── Capture[G] (0..18)
        └── Charset[[.0..9A..Z_a..z]] (5..18)`,
		},
		{
			Name:    "Charset from single char literals",
			Grammar: `G <- "a" G? "b"`,
			ExpectedAST: `Grammar (0..15)
└── Definition[G] (0..15)
    └── Capture[G] (0..15)
        └── Sequence (5..15)
            ├── Identifier[Spacing] (0)
            ├── Capture (6..7)
            │   └── Charset[[a]] (6..7)
            ├── Identifier[Spacing] (0)
            ├── Optional (9..11)
            │   └── Identifier[G] (9..10)
            ├── Identifier[Spacing] (0)
            └── Capture (13..14)
                └── Charset[[b]] (13..14)`,
		},
		{
			Name:    "Complement from Not Any",
			Grammar: "G <- !['] .",
			ExpectedAST: `Grammar (0..11)
└── Definition[G] (0..11)
    └── Capture[G] (0..11)
        └── Charset[[` + "\x00" + `..&(..ÿ]] (5..11)`,
		},

		{
			Name:    "Span from Not X Any",
			Grammar: `G <- '"' (!'"' .)* '"'`,
			ExpectedAST: `Grammar (0..22)
└── Definition[G] (0..22)
    └── Capture[G] (0..22)
        └── Sequence (5..22)
            ├── Charset[[\"]] (6..7)
            ├── ZeroOrMore (9..18)
            │   └── Charset[[` + "\x00" + `..!#..ÿ]] (10..16)
            └── Charset[[\"]] (20..21)`,
		},
		{
			Name: "Span from Not X with label",
			Grammar: `G <- '"' (!'"' .)* '"'^DQ
`,
			ExpectedAST: `Grammar (0..26)
└── Definition[G] (0..26)
    └── Capture[G] (0..26)
        └── Sequence (5..25)
            ├── Capture (6..7)
            │   └── Charset[[\"]] (6..7)
            ├── Capture (10..16)
            │   └── ZeroOrMore (9..18)
            │       └── Charset[[` + "\x00" + `..!#..ÿ]] (10..16)
            └── Throw[DQ] (19..25)
                └── Capture (20..21)
                    └── Charset[[\"]] (20..21)`,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			cfg := NewConfig()
			cfg.SetBool("grammar.add_builtins", false)
			// defaults to true, but we're just making sure
			cfg.SetBool("grammar.add_charsets", true)

			ast, err := GrammarFromBytes([]byte(test.Grammar), cfg)
			require.NoError(t, err)
			require.NotNil(t, ast)

			assert.Equal(t, test.ExpectedAST, ast.PrettyString())
		})
	}
}
