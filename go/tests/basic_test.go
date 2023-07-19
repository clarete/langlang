package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate go run ../cmd -language go -grammar ./basic.peg -go-struct-suffix Basic -output ./basic.go

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

	t.Run("Lexification operator on a single item within a syntactic rule", func(t *testing.T) {
		p := NewParserBasic("1st")
		v, err := p.ParseOrdinal()
		require.NoError(t, err)
		assert.Equal(t, "1st", v.Text())

		p = NewParserBasic("1 st")
		_, err = p.ParseOrdinal()
		require.Error(t, err)
		assert.Equal(t, "Ordinal: ord @ 1", err.Error())
	})

	t.Run("Lexification operator on a sequence within a sequence", func(t *testing.T) {
		for _, test := range []string{
			"a9:30",
			"a999:99",
			"bb :12",
		} {
			p := NewParserBasic(test)
			v, err := p.ParseSPC0()
			require.NoError(t, err)
			assert.Equal(t, test, v.Text())
		}

		for test, errMsg := range map[string]string{
			" a9:30":   "Letter: Expected char between `a' and `z', got ` ' @ 0",
			"a 999:99": "Alnum: Expected char between `a' and `z', got ` ' @ 1",
			"a9: 30":   "Digit: Expected char between `0' and `9', got ` ' @ 3",
		} {
			p := NewParserBasic(test)
			_, err := p.ParseSPC0()
			require.Error(t, err, test)
			assert.Equal(t, errMsg, err.Error())
		}
	})

	t.Run("Variation of lexification operator on a sequence within a sequence", func(t *testing.T) {
		for _, test := range []string{
			"a9:30",
			"a 999:99",
			"a 999: 99",
		} {
			p := NewParserBasic(test)
			v, err := p.ParseSPC1()
			require.NoError(t, err)
			assert.Equal(t, test, v.Text())
		}

		for test, errMsg := range map[string]string{
			" a9:30":    "Letter: Expected char between `a' and `z', got ` ' @ 0",
			"a 999 :99": "SPC1: Missing `:` @ 5..6",
		} {
			p := NewParserBasic(test)
			_, err := p.ParseSPC1()
			require.Error(t, err, test)
			assert.Equal(t, errMsg, err.Error())
		}
	})
}

func TestAnd(t *testing.T) {
	t.Run("all and uses match", func(t *testing.T) {
		for _, test := range []string{
			"#",
			"#*",
			"#***",
		} {
			p := NewParserBasic(test)
			v, err := p.ParseHashWithAnAnd()
			require.NoError(t, err)
			assert.Equal(t, test, v.Text())
		}
	})

	t.Run("all and uses do not match", func(t *testing.T) {
		for test, errMsg := range map[string]string{
			"x": "HashWithAnAnd: Missing `#` @ 0..1",
			// these ones error because the rule ends on EOF
			"##":   "HashWithAnAnd: Missing `*` @ 1..2",
			"#**!": "HashWithAnAnd: Missing `*` @ 3..4",
		} {
			p := NewParserBasic(test)
			_, err := p.ParseHashWithAnAnd()
			require.Error(t, err)
			assert.Equal(t, errMsg, err.Error())
		}
	})
}

func TestNullable(t *testing.T) {
	t.Run("matching will succeed but no input will be consumed", func(t *testing.T) {
		p := NewParserBasic("c")
		v, err := p.ParseMaybeNull()
		require.NoError(t, err)
		assert.Nil(t, v)
	})
}
