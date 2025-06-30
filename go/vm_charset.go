package langlang

import (
	"bytes"
	"fmt"
	"strings"
)

// A charset is a bitmap that uses one bit per character within a
// given range.  The operation opSpan uses the `Has` operation to
// check if the rune under the cursor belongs to the charset in a
// single operation.
//
// Name        | Range         | bits      | bytes
// ------------+---------------+-----------+-------
// ASCII       | U+0000–007F   |       128 | 16
// Latin1      | U+0000–00FF   |       256 | 32
// BMP         | U+0000–FFFF   |    65_536 | 8192b (8k)
// Unicode     | U+0000–10FFFF | 1_114_112 | 139264 (136 Kb)
type charsetSize int

const (
	charsetSize_ASCII   charsetSize = 16
	charsetSize_Latin1              = 32
	charsetSize_BMP                 = 8_192
	charsetSize_Unicode             = 139_264
)

var charsetSizeName = map[charsetSize]string{
	charsetSize_ASCII:   "ascii",
	charsetSize_Latin1:  "latin1",
	charsetSize_BMP:     "bmp",
	charsetSize_Unicode: "unicode",
}

// charset is a bitmap that makes `opSpan` more efficient by providing
// the `has` method with an O(1) and very fast invariant
// implementation (using shifts instead of div and mod.)
type charset struct {
	// mcp holds the `maxCodePoint`.  This is only used during
	// compilation, not during match time, so generators don't
	// need to populate it.
	mcp charsetSize

	// bits hold all the codepoints of this charset up to `mcp`
	bits []byte
}

func newCharSet(mcp charsetSize) *charset {
	return &charset{mcp: mcp, bits: make([]byte, mcp)}
}

func newCharsetFromString(s string) *charset {
	for _, c := range s {
		cs := newCharseForRune(c)
		cs.add(c)
		return cs
	}
	return nil
}

func newCharseForRune(r rune) *charset {
	cs := newCharSet(charsetSizeForRune(r))
	cs.add(r)
	return cs
}

func newCharsetForRange(a, b rune) *charset {
	sa := charsetSizeForRune(a)
	sb := charsetSizeForRune(b)
	cs := newCharSet(max(sa, sb))
	cs.addRange(a, b)
	return cs
}

func charsetSizeForRune(r rune) charsetSize {
	rpos := int(r) >> 3
	switch {
	case rpos < int(charsetSize_ASCII):
		return charsetSize_ASCII
	case rpos < int(charsetSize_Latin1):
		return charsetSize_Latin1
	case rpos < int(charsetSize_BMP):
		return charsetSize_BMP
	default:
		return charsetSize_Unicode
	}
}

func (cs *charset) begin() int              { return 0 }
func (cs *charset) end() int                { return int(cs.mcp) << 3 }
func (cs *charset) outOfBounds(r rune) bool { return int(r) < cs.begin() || int(r) > cs.end() }

func (cs *charset) add(r rune) {
	if cs.outOfBounds(r) {
		panic(fmt.Sprintf("code point `U+%X` (%c) is out of bounds `%s`", r, r, charsetSizeName[cs.mcp]))
	}
	i := int(r)
	cs.bits[i>>3] |= 1 << (i & 7)
}

func (cs *charset) addRange(start, end rune) {
	if cs.outOfBounds(start) || cs.outOfBounds(end) || start > end {
		panic(fmt.Sprintf("range out of charset bounds `%s`", charsetSizeName[cs.mcp]))
	}
	for r := start; r <= end; r++ {
		cs.add(r)
	}
}

func (cs *charset) has(r rune) bool {
	// bounds check skipped because the compiler should have
	// created the appropriate size of a set for us anyway.
	i := int(r)

	// writing `i/8` as `i>>3` and and `i%8` as `i&7` because
	// division is usually slower than bit shifting operators.
	x := i >> 3

	// accounts for different charset sizes.
	if len(cs.bits) < x {
		return false
	}
	return cs.bits[x]&(1<<(i&7)) != 0
}

func (cs *charset) eq(o *charset) bool {
	return cs.mcp == o.mcp && bytes.Equal(cs.bits, o.bits)
}

func (cs *charset) String() string {
	var (
		s  strings.Builder
		rg bool
		st rune
		pr rune = -2
	)
	s.WriteString("[")

	for i := cs.begin(); i < cs.end(); i++ {
		r := rune(i)
		has := cs.has(r)
		if has {
			if !rg {
				rg = true
				st = r
			}
			pr = r
		} else if rg {
			rg = false
			addRange(&s, st, pr)
		}
	}
	if rg {
		addRange(&s, st, pr)
	}

	s.WriteString("]")
	return s.String()
}

func addRange(s *strings.Builder, start, end rune) {
	if start == end {
		s.WriteString(escapeLiteral(string(start)))
	} else if end == start+1 {
		s.WriteString(escapeLiteral(string(start)))
		s.WriteString(escapeLiteral(string(end)))
	} else {
		s.WriteString(escapeLiteral(string(start)))
		s.WriteString("..")
		s.WriteString(escapeLiteral(string(end)))
	}
}

func charsetMerge(a, b *charset) *charset {
	out := newCharSet(max(a.mcp, b.mcp))
	for i := a.begin(); i < a.end(); i++ {
		r := rune(i)
		if a.has(r) {
			out.add(r)
		}
	}
	for i := b.begin(); i < b.end(); i++ {
		r := rune(i)
		if b.has(r) {
			out.add(r)
		}
	}
	return out
}
