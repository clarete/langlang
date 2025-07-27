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
			ExpectedAST: `Grammar (1..7)
└── Definition[G] (1..7)
    └── Capture[G] (1..7)
        └── Any (6..7)`,
		},
		{
			Name:    "Charset from Classes",
			Grammar: "G <- [a-zA-Z0-9_.]",
			ExpectedAST: `Grammar (1..19)
└── Definition[G] (1..19)
    └── Capture[G] (1..19)
        └── Charset[[.0..9A..Z_a..z]] (6..19)`,
		},
		{
			Name:    "Charset from single char literals",
			Grammar: `G <- "a" G? "b"`,
			ExpectedAST: `Grammar (1..16)
└── Definition[G] (1..16)
    └── Capture[G] (1..16)
        └── Sequence (6..16)
            ├── Identifier[Spacing] (1)
            ├── Capture (6..9)
            │   └── Charset[[a]] (6..9)
            ├── Identifier[Spacing] (1)
            ├── Optional (10..12)
            │   └── Identifier[G] (10..11)
            ├── Identifier[Spacing] (1)
            └── Capture (13..16)
                └── Charset[[b]] (13..16)`,
		},
		{
			Name:    "Complement from Not Any",
			Grammar: "G <- !['] .",
			ExpectedAST: `Grammar (1..12)
└── Definition[G] (1..12)
    └── Capture[G] (1..12)
        └── Charset[[` + "\x00" + `..&(..ÿ]] (6..12)`,
		},

		{
			Name:    "Span from Not X Any",
			Grammar: `G <- '"' (!'"' .)* '"'`,
			ExpectedAST: `Grammar (1..23)
└── Definition[G] (1..23)
    └── Capture[G] (1..23)
        └── Sequence (6..23)
            ├── Charset[[\"]] (6..9)
            ├── ZeroOrMore (10..19)
            │   └── Charset[[` + "\x00" + `..!#..ÿ]] (11..17)
            └── Charset[[\"]] (20..23)`,
		},
		{
			Name: "Span from Not X with label",
			Grammar: `G <- '"' (!'"' .)* '"'^DQ
`,
			ExpectedAST: `Grammar (1..2:1)
└── Definition[G] (1..2:1)
    └── Capture[G] (1..2:1)
        └── Sequence (6..26)
            ├── Capture (6..9)
            │   └── Charset[[\"]] (6..9)
            ├── Capture (11..17)
            │   └── ZeroOrMore (10..19)
            │       └── Charset[[` + "\x00" + `..!#..ÿ]] (11..17)
            └── Throw[DQ] (20..26)
                └── Capture (20..23)
                    └── Charset[[\"]] (20..23)`,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			cfg := NewConfig()
			cfg.SetBool("grammar.add_builtins", false)
			// defaults to true, but we're just making sure
			cfg.SetBool("grammar.add_charsets", true)

			ast, err := GrammarFromString(test.Grammar, cfg)
			require.NoError(t, err)
			require.NotNil(t, ast)

			assert.Equal(t, test.ExpectedAST, ast.PrettyString())
		})
	}
}
