package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSyntactic(t *testing.T) {
	t.Run("sequence with literal terminals is always syntactic", func(t *testing.T) {
		// Matches without the spaces in the input
		p := NewParserBasic("abc")
		v, err := p.ParseSyntactic0()
		require.NoError(t, err)
		assert.Equal(t, "abc", v.Text())

		// It doesn't expect spaces between the sequence items
		p = NewParserBasic("a b c")
		_, err = p.ParseSyntactic0()
		require.Error(t, err)
		assert.Equal(t, "Syntactic0: Missing `b` @ 1..2", err.Error())
	})

	t.Run("sequence with grammar nodes that are not terminals are not syntactic", func(t *testing.T) {
		// Optional spaces are introduced between the items
		// within the top-level sequence

		p := NewParserBasic("abcabc!")
		v, err := p.ParseNotSyntactic0()
		require.NoError(t, err)
		assert.Equal(t, "abcabc!", v.Text())

		p = NewParserBasic("abc abc !")
		v, err = p.ParseNotSyntactic0()
		require.NoError(t, err)
		assert.Equal(t, "abc abc !", v.Text())
	})
}
