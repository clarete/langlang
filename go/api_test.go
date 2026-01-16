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
            ├── Capture (7..8)
            │   └── Charset[[a]] (7..8)
            ├── Identifier[Spacing] (1)
            ├── Optional (10..12)
            │   └── Identifier[G] (10..11)
            ├── Identifier[Spacing] (1)
            └── Capture (14..15)
                └── Charset[[b]] (14..15)`,
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
            ├── Charset[[\"]] (7..8)
            ├── ZeroOrMore (10..19)
            │   └── Charset[[` + "\x00" + `..!#..ÿ]] (11..17)
            └── Charset[[\"]] (21..22)`,
		},
		{
			Name: "Span from Not X with label",
			Grammar: `G <- '"' (!'"' .)* '"'^DQ
`,
			ExpectedAST: `Grammar (1:1..2:1)
└── Definition[G] (1..26)
    └── Capture[G] (1..26)
        └── Sequence (6..26)
            ├── Capture (7..8)
            │   └── Charset[[\"]] (7..8)
            ├── Capture (11..17)
            │   └── ZeroOrMore (10..19)
            │       └── Charset[[` + "\x00" + `..!#..ÿ]] (11..17)
            └── Throw[DQ] (20..26)
                └── Capture (21..22)
                    └── Charset[[\"]] (21..22)`,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			cfg := NewConfig()
			cfg.SetBool("grammar.add_builtins", false)
			// defaults to true, but we're just making sure
			cfg.SetBool("grammar.add_charsets", true)

			ast, err := grammarFromBytes([]byte(test.Grammar), cfg)
			require.NoError(t, err)
			require.NotNil(t, ast)

			assert.Equal(t, test.ExpectedAST, ast.PrettyString())
		})
	}
}

func TestGrammarTransformationsWithErrors(t *testing.T) {
	t.Run("Range where end > start", func(t *testing.T) {
		_, err := grammarFromBytes([]byte(`
                           Letter <- [z-a]
		`), NewConfig())
		require.Error(t, err)
		require.Equal(t, "range out of bounds, did you mean `a-z`?", err.Error())
	})
}

func TestMatcherFromLoader(t *testing.T) {
	t.Run("Success - resolves @import via in-memory loader", func(t *testing.T) {
		loader := NewInMemoryImportLoader()
		loader.Add("expr.peg", []byte(`@import Value from "./value.peg"

Expr <- Value EOF^eof
`))
		loader.Add("value.peg", []byte(`@import Number from "./number.peg"

Value <- Number
`))
		loader.Add("number.peg", []byte(`Number <- [0-9]+`))

		cfg := NewConfig()
		resolver := NewImportResolver(loader)
		matcher, err := resolver.MatcherFor("expr.peg", cfg)
		require.NoError(t, err)
		require.NotNil(t, matcher)

		_, pos, err := matcher.Match([]byte("123"))
		require.NoError(t, err)
		assert.Equal(t, 3, pos)
	})
}

func grammarFromBytes(input []byte, cfg *Config) (AstNode, error) {
	name := "grammar.peg"
	loader := NewInMemoryImportLoader()
	loader.Add(name, input)
	return NewImportResolver(loader).Resolve(name, cfg)
}
