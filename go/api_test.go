package langlang

import (
	"os"
	"path/filepath"
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
└── Definition[G] (0..25)
    └── Capture[G] (0..25)
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

func TestMatcherFromBytes(t *testing.T) {
	t.Run("Success - simple grammar", func(t *testing.T) {
		grammar := []byte(`G <- "hello"`)
		cfg := NewConfig()

		matcher, err := MatcherFromBytes(grammar, cfg)
		require.NoError(t, err)
		require.NotNil(t, matcher)

		// Test that the matcher can match valid input
		val, pos, err := matcher.Match([]byte("hello"))
		require.NoError(t, err)
		assert.Equal(t, 5, pos)
		assert.NotNil(t, val)
	})

	t.Run("Success - arithmetic grammar", func(t *testing.T) {
		grammar := []byte(`Num <- [0-9]+`)
		cfg := NewConfig()

		matcher, err := MatcherFromBytes(grammar, cfg)
		require.NoError(t, err)
		require.NotNil(t, matcher)

		// Test that the matcher can match valid input
		val, pos, err := matcher.Match([]byte("123"))
		require.NoError(t, err)
		assert.Equal(t, 3, pos)
		assert.NotNil(t, val)
	})

	t.Run("Success - with configuration options", func(t *testing.T) {
		grammar := []byte(`G <- "test"`)
		cfg := NewConfig()
		cfg.SetBool("grammar.add_builtins", true)
		cfg.SetBool("grammar.add_charsets", true)
		cfg.SetBool("grammar.captures", true)

		matcher, err := MatcherFromBytes(grammar, cfg)
		require.NoError(t, err)
		require.NotNil(t, matcher)

		// Test that the matcher works with transformations
		val, pos, err := matcher.Match([]byte("test"))
		require.NoError(t, err)
		assert.Equal(t, 4, pos)
		assert.NotNil(t, val)
	})

	t.Run("Success - complex grammar with choices", func(t *testing.T) {
		grammar := []byte(`G <- 'a' / 'b' / 'c'`)
		cfg := NewConfig()

		matcher, err := MatcherFromBytes(grammar, cfg)
		require.NoError(t, err)
		require.NotNil(t, matcher)

		// Test matching each choice
		val, pos, err := matcher.Match([]byte("a"))
		require.NoError(t, err)
		assert.Equal(t, 1, pos)
		assert.NotNil(t, val)

		val, pos, err = matcher.Match([]byte("b"))
		require.NoError(t, err)
		assert.Equal(t, 1, pos)
		assert.NotNil(t, val)

		val, pos, err = matcher.Match([]byte("c"))
		require.NoError(t, err)
		assert.Equal(t, 1, pos)
		assert.NotNil(t, val)
	})

	t.Run("Matcher fails on non-matching input", func(t *testing.T) {
		grammar := []byte(`G <- "hello"`)
		cfg := NewConfig()

		matcher, err := MatcherFromBytes(grammar, cfg)
		require.NoError(t, err)

		// Test that the matcher fails on invalid input
		_, _, err = matcher.Match([]byte("goodbye"))
		require.Error(t, err)
	})
}

func TestMatcherFromFile(t *testing.T) {
	t.Run("Success - from existing file", func(t *testing.T) {
		cfg := NewConfig()

		matcher, err := MatcherFromFile("examples/tiny/tiny.peg", cfg)
		require.NoError(t, err)
		require.NotNil(t, matcher)

		// Test that the matcher can match valid expressions
		val, pos, err := matcher.Match([]byte("42"))
		require.NoError(t, err)
		assert.Greater(t, pos, 0)
		assert.NotNil(t, val)

		// Test arithmetic expression
		val, pos, err = matcher.Match([]byte("2+3"))
		require.NoError(t, err)
		assert.Equal(t, 3, pos)
		assert.NotNil(t, val)
	})

	t.Run("Success - from temporary file", func(t *testing.T) {
		// Create a temporary grammar file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.peg")
		grammar := []byte(`G <- [a-z]+`)
		err := os.WriteFile(tmpFile, grammar, 0644)
		require.NoError(t, err)

		cfg := NewConfig()
		matcher, err := MatcherFromFile(tmpFile, cfg)
		require.NoError(t, err)
		require.NotNil(t, matcher)

		// Test that the matcher works
		val, pos, err := matcher.Match([]byte("hello"))
		require.NoError(t, err)
		assert.Equal(t, 5, pos)
		assert.NotNil(t, val)
	})

	t.Run("Success - with configuration options", func(t *testing.T) {
		cfg := NewConfig()
		cfg.SetBool("grammar.add_builtins", true)
		cfg.SetBool("grammar.add_charsets", true)
		cfg.SetBool("grammar.captures", true)

		matcher, err := MatcherFromFile("examples/tiny/tiny.peg", cfg)
		require.NoError(t, err)
		require.NotNil(t, matcher)
	})

	t.Run("Error - file does not exist", func(t *testing.T) {
		cfg := NewConfig()

		matcher, err := MatcherFromFile("non_existent_file.peg", cfg)
		require.Error(t, err)
		assert.Nil(t, matcher)
	})

	t.Run("Matcher fails on non-matching input", func(t *testing.T) {
		cfg := NewConfig()

		matcher, err := MatcherFromFile("examples/tiny/tiny.peg", cfg)
		require.NoError(t, err)

		// Test that the matcher fails on invalid input
		_, _, err = matcher.Match([]byte("!@#$%"))
		require.Error(t, err)
	})
}
