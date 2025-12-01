package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate go run ../../cmd/langlang -grammar ./basic.peg -output-language go -output-path ./basic.go -disable-inline-defs=false

func TestIsSyntactic(t *testing.T) {
	t.Run("sequence with literal terminals is always syntactic", func(t *testing.T) {
		// Matches without the spaces in the input
		p := newBasicParser("abc")
		v, err := p.ParseSyntactic0()
		require.NoError(t, err)
		assert.Equal(t, `Syntactic0 (1..4)
└── "abc" (1..4)`, v.Pretty(v.Root()))

		// It doesn't expect spaces between the sequence items
		p = newBasicParser("a b c")
		_, err = p.ParseSyntactic0()
		require.Error(t, err)
		assert.Equal(t, "Expected 'b' but got ' ' @ 2", err.Error())
	})

	t.Run("sequence with grammar nodes that are not terminals are not syntactic", func(t *testing.T) {
		// Optional spaces are introduced between the items
		// within the top-level sequence

		p := newBasicParser("abcabc!")
		v, err := p.ParseNotSyntactic0()
		require.NoError(t, err)
		assert.Equal(t, `NotSyntactic0 (1..8)
└── Sequence<3> (1..8)
    ├── Syntactic0 (1..4)
    │   └── "abc" (1..4)
    ├── Syntactic0 (4..7)
    │   └── "abc" (4..7)
    └── "!" (7..8)`, v.Pretty(v.Root()))

		p = newBasicParser("abc abc !")
		v, err = p.ParseNotSyntactic0()
		require.NoError(t, err)
		assert.Equal(t, `NotSyntactic0 (1..10)
└── Sequence<5> (1..10)
    ├── Syntactic0 (1..4)
    │   └── "abc" (1..4)
    ├── Spacing (4..5)
    │   └── " " (4..5)
    ├── Syntactic0 (5..8)
    │   └── "abc" (5..8)
    ├── Spacing (8..9)
    │   └── " " (8..9)
    └── "!" (9..10)`, v.Pretty(v.Root()))
	})

	t.Run("Lexification operator on a single item within a syntactic rule", func(t *testing.T) {
		p := newBasicParser("1st")
		v, err := p.ParseOrdinal()
		require.NoError(t, err)
		assert.Equal(t, `Ordinal (1..4)
└── Sequence<2> (1..4)
    ├── Decimal (1..2)
    │   └── "1" (1..2)
    └── "st" (2..4)`, v.Pretty(v.Root()))

		p = newBasicParser("1 st")
		_, err = p.ParseOrdinal()
		require.Error(t, err)
		assert.Equal(t, "[ord] Expected 's', 'n', 'r', 't' but got ' ' @ 2", err.Error())
	})

	t.Run("Lexification operator on a sequence within a sequence", func(t *testing.T) {
		for _, test := range [][]string{
			{"a9:30", `SPC0 (1..6)
└── Sequence<5> (1..6)
    ├── Letter (1..2)
    │   └── "a" (1..2)
    ├── Alnum (2..3)
    │   └── "9" (2..3)
    ├── ":" (3..4)
    ├── Digit (4..5)
    │   └── "3" (4..5)
    └── Digit (5..6)
        └── "0" (5..6)`},
			{"a999:99", `SPC0 (1..8)
└── Sequence<7> (1..8)
    ├── Letter (1..2)
    │   └── "a" (1..2)
    ├── Alnum (2..3)
    │   └── "9" (2..3)
    ├── Alnum (3..4)
    │   └── "9" (3..4)
    ├── Alnum (4..5)
    │   └── "9" (4..5)
    ├── ":" (5..6)
    ├── Digit (6..7)
    │   └── "9" (6..7)
    └── Digit (7..8)
        └── "9" (7..8)`},
			{"bb :12", `SPC0 (1..7)
└── Sequence<6> (1..7)
    ├── Letter (1..2)
    │   └── "b" (1..2)
    ├── Alnum (2..3)
    │   └── "b" (2..3)
    ├── Spacing (3..4)
    │   └── " " (3..4)
    ├── ":" (4..5)
    ├── Digit (5..6)
    │   └── "1" (5..6)
    └── Digit (6..7)
        └── "2" (6..7)`},
		} {
			p := newBasicParser(test[0])
			v, err := p.ParseSPC0()
			require.NoError(t, err)
			assert.Equal(t, test[1], v.Pretty(v.Root()))
		}

		for test, errMsg := range map[string]string{
			" a9:30":   "Expected 'A-Z', 'a-z' but got ' ' @ 1",
			"a 999:99": "Expected '0-9', 'A-Z', 'a-z' but got ' ' @ 2",
			"a9: 30":   "Expected '0-9' but got ' ' @ 4",
		} {
			p := newBasicParser(test)
			_, err := p.ParseSPC0()
			require.Error(t, err, test)
			assert.Equal(t, errMsg, err.Error())
		}
	})

	t.Run("Variation of lexification operator on a sequence within a sequence", func(t *testing.T) {
		for _, test := range [][]string{
			{"a9:30", `SPC1 (1..6)
└── Sequence<5> (1..6)
    ├── Letter (1..2)
    │   └── "a" (1..2)
    ├── Alnum (2..3)
    │   └── "9" (2..3)
    ├── ":" (3..4)
    ├── Digit (4..5)
    │   └── "3" (4..5)
    └── Digit (5..6)
        └── "0" (5..6)`},
			{"a 999:99", `SPC1 (1..9)
└── Sequence<8> (1..9)
    ├── Letter (1..2)
    │   └── "a" (1..2)
    ├── Spacing (2..3)
    │   └── " " (2..3)
    ├── Alnum (3..4)
    │   └── "9" (3..4)
    ├── Alnum (4..5)
    │   └── "9" (4..5)
    ├── Alnum (5..6)
    │   └── "9" (5..6)
    ├── ":" (6..7)
    ├── Digit (7..8)
    │   └── "9" (7..8)
    └── Digit (8..9)
        └── "9" (8..9)`},
			{"a 999: 99", `SPC1 (1..10)
└── Sequence<9> (1..10)
    ├── Letter (1..2)
    │   └── "a" (1..2)
    ├── Spacing (2..3)
    │   └── " " (2..3)
    ├── Alnum (3..4)
    │   └── "9" (3..4)
    ├── Alnum (4..5)
    │   └── "9" (4..5)
    ├── Alnum (5..6)
    │   └── "9" (5..6)
    ├── ":" (6..7)
    ├── Spacing (7..8)
    │   └── " " (7..8)
    ├── Digit (8..9)
    │   └── "9" (8..9)
    └── Digit (9..10)
        └── "9" (9..10)`},
		} {
			p := newBasicParser(test[0])
			v, err := p.ParseSPC1()
			require.NoError(t, err)
			assert.Equal(t, test[1], v.Pretty(v.Root()))
		}

		for test, errMsg := range map[string]string{
			" a9:30":    "Expected 'A-Z', 'a-z' but got ' ' @ 1",
			"a 999 :99": "Expected '0-9', 'A-Z', 'a-z' but got ' ' @ 6..7",
		} {
			p := newBasicParser(test)
			_, err := p.ParseSPC1()
			require.Error(t, err, test)
			assert.Equal(t, errMsg, err.Error())
		}
	})
}

func TestAnd(t *testing.T) {
	t.Run("Succeeds", func(t *testing.T) {
		for _, test := range [][]string{
			{"#", `HashWithAnAnd (1..2)
└── "#" (1..2)`},
			{"#*", `HashWithAnAnd (1..3)
└── Sequence<2> (1..3)
    ├── "#" (1..2)
    └── "*" (2..3)`},
			{"#***", `HashWithAnAnd (1..5)
└── Sequence<4> (1..5)
    ├── "#" (1..2)
    ├── "*" (2..3)
    ├── "*" (3..4)
    └── "*" (4..5)`},
		} {
			p := newBasicParser(test[0])
			v, err := p.ParseHashWithAnAnd()
			require.NoError(t, err)
			assert.Equal(t, test[1], v.Pretty(v.Root()))
		}
	})

	t.Run("Fails", func(t *testing.T) {
		for test, errMsg := range map[string]string{
			"x":    "[missingdot] Expected '#' but got 'x' @ 1",
			"##":   "Expected EOF @ 2..3",
			"#**!": "Expected EOF @ 4..5",
		} {
			p := newBasicParser(test)
			p.SetLabelMessages(map[string]string{"eof": "Expected EOF"})
			_, err := p.ParseHashWithAnAnd()
			require.Error(t, err)
			assert.Equal(t, errMsg, err.Error())
		}
	})
}

func TestNot(t *testing.T) {
	t.Run("Succeeds", func(t *testing.T) {
		for _, test := range [][]string{
			{"*", `HashWithNot (1..2)
└── "*" (1..2)`},
			{"*#", `HashWithNot (1..3)
└── Sequence<2> (1..3)
    ├── "*" (1..2)
    └── "#" (2..3)`},
			{"*###", `HashWithNot (1..5)
└── Sequence<2> (1..5)
    ├── "*" (1..2)
    └── "###" (2..5)`},
		} {
			p := newBasicParser(test[0])
			v, err := p.ParseHashWithNot()
			require.NoError(t, err)
			assert.Equal(t, test[1], v.Pretty(v.Root()))
		}
	})

	t.Run("Fails", func(t *testing.T) {
		for test, errMsg := range map[string]string{
			"#":    "[missingdotnot] Unexpected '#' @ 1",
			"**":   "Expected EOF @ 2..3",
			"*##*": "Expected EOF @ 4..5",
		} {
			p := newBasicParser(test)
			p.SetLabelMessages(map[string]string{"eofn": "Expected EOF"})
			_, err := p.ParseHashWithNot()
			require.Error(t, err)
			assert.Equal(t, errMsg, err.Error())
		}
	})
}

func TestNullable(t *testing.T) {
	t.Run("matching will succeed but no input will be consumed", func(t *testing.T) {
		p := newBasicParser("c")
		_, err := p.ParseMaybeNull()
		require.NoError(t, err)
	})
}

func newBasicParser(input string) *Parser {
	p := NewParser()
	p.SetInput([]byte(input))
	p.SetShowFails(true)
	return p
}
