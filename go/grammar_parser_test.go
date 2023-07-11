package langlang

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDefinition(t *testing.T) {
	t.Run("Simplest", func(t *testing.T) {
		parser := NewGrammarParser("A <- .")
		output, err := parser.ParseDefinition()
		require.NoError(t, err)
		assert.Equal(t, `Definition[A](Sequence(Any @ 5..6) @ 5..6) @ 0..6`, output.String())
	})

	t.Run("Less simple", func(t *testing.T) {
		parser := NewGrammarParser("A <- 'a' / 'b'")
		output, err := parser.ParseDefinition()
		require.NoError(t, err)
		assert.Equal(t, "Definition[A](Choice(Sequence(Literal(a) @ 5..8) @ 5..9, "+
			"Sequence(Literal(b) @ 11..14) @ 11..14) @ 5..14) @ 0..14", output.String())
	})

	t.Run("With comment", func(t *testing.T) {
		parser := NewGrammarParser("A <- . // something something")
		output, err := parser.ParseDefinition()
		require.NoError(t, err)
		assert.Equal(t, `Definition[A](Sequence(Any @ 5..6) @ 5..29) @ 0..29`, output.String())
	})
}

func TestParseExpression(t *testing.T) {
	t.Run("Single item", func(t *testing.T) {
		parser := NewGrammarParser(".")
		output, err := parser.ParseExpression()
		require.NoError(t, err)
		assert.Equal(t, `Sequence(Any @ 0..1) @ 0..1`, output.String())
	})

	t.Run("More items", func(t *testing.T) {
		parser := NewGrammarParser("'a' / 'b' / 'c'")
		output, err := parser.ParseExpression()
		require.NoError(t, err)
		assert.Equal(t, "Choice(Sequence(Literal(a) @ 0..3) @ 0..4, "+
			"Sequence(Literal(b) @ 6..9) @ 6..10, "+
			"Sequence(Literal(c) @ 12..15) @ 12..15) @ 0..15", output.String())
	})
}

func TestParseSequence(t *testing.T) {
	t.Run("Single item", func(t *testing.T) {
		parser := NewGrammarParser(".")
		output, err := parser.ParseSequence()
		require.NoError(t, err)
		assert.Equal(t, `Sequence(Any @ 0..1) @ 0..1`, output.String())
	})

	t.Run("More items", func(t *testing.T) {
		parser := NewGrammarParser("a 'a' .")
		output, err := parser.ParseSequence()
		require.NoError(t, err)
		assert.Equal(t, `Sequence(Identifier(a) @ 0..1, Literal(a) @ 2..5, Any @ 6..7) @ 0..7`, output.String())
	})
}

func TestParsePrefix(t *testing.T) {
	t.Run("No Prefix", func(t *testing.T) {
		parser := NewGrammarParser(".")
		output, err := parser.ParsePrefix()
		require.NoError(t, err)
		assert.Equal(t, `Any @ 0..1`, output.String())
	})

	t.Run("And", func(t *testing.T) {
		parser := NewGrammarParser("&.")
		output, err := parser.ParsePrefix()
		require.NoError(t, err)
		assert.Equal(t, `And(Any @ 1..2) @ 0..2`, output.String())
	})

	t.Run("Not", func(t *testing.T) {
		parser := NewGrammarParser("!.")
		output, err := parser.ParsePrefix()
		require.NoError(t, err)
		assert.Equal(t, `Not(Any @ 1..2) @ 0..2`, output.String())
	})
}

func TestParseSuffix(t *testing.T) {
	t.Run("No Suffix", func(t *testing.T) {
		parser := NewGrammarParser(".")
		output, err := parser.ParseSuffix()
		require.NoError(t, err)
		assert.Equal(t, `Any @ 0..1`, output.String())
	})

	t.Run("Optional", func(t *testing.T) {
		parser := NewGrammarParser(".?")
		output, err := parser.ParseSuffix()
		require.NoError(t, err)
		assert.Equal(t, `Optional(Any @ 0..1) @ 0..2`, output.String())
	})

	t.Run("Space after optional", func(t *testing.T) {
		parser := NewGrammarParser(".? x")
		output, err := parser.ParseSequence()
		require.NoError(t, err)
		assert.Equal(t, `Sequence(Optional(Any @ 0..1) @ 0..2, Identifier(x) @ 3..4) @ 0..4`, output.String())
	})

	t.Run("Zero Or More", func(t *testing.T) {
		parser := NewGrammarParser(".*")
		output, err := parser.ParseSuffix()
		require.NoError(t, err)
		assert.Equal(t, `ZeroOrMore(Any @ 0..1) @ 0..2`, output.String())
	})

	t.Run("One Or More", func(t *testing.T) {
		parser := NewGrammarParser(".+")
		output, err := parser.ParseSuffix()
		require.NoError(t, err)
		assert.Equal(t, `OneOrMore(Any @ 0..1) @ 0..2`, output.String())
	})
}

func TestParsePrimary(t *testing.T) {
	t.Run("Any", func(t *testing.T) {
		parser := NewGrammarParser(".")
		output, err := parser.ParsePrimary()
		require.NoError(t, err)
		assert.Equal(t, `Any @ 0..1`, output.String())
	})

	t.Run("Single Quote Literal", func(t *testing.T) {
		parser := NewGrammarParser("'abcd'")
		output, err := parser.ParsePrimary()
		require.NoError(t, err)
		assert.Equal(t, `Literal(abcd) @ 0..6`, output.String())
	})

	t.Run("Identifier", func(t *testing.T) {
		parser := NewGrammarParser("foobarbaz")
		output, err := parser.ParsePrimary()
		require.NoError(t, err)
		assert.Equal(t, `Identifier(foobarbaz) @ 0..9`, output.String())
	})

	t.Run("Class with single range", func(t *testing.T) {
		parser := NewGrammarParser("[0-9]")
		output, err := parser.ParsePrimary()
		require.NoError(t, err)
		assert.Equal(t, `Class(Range(0, 9) @ 1..4) @ 0..5`, output.String())
	})
}
