package langlang

import (
	"encoding/hex"
	"fmt"
	"math/bits"
	"strings"
)

// charset is a bitmap that makes `opSet` and `opSpan` more efficient
// by providing the `has` method with an O(1) with fast invariant
// implementation (using shifts instead of div and mod.)
type charset struct {
	bits [32]byte
}

func newCharSet() *charset { return &charset{} }

func newCharsetForRune(r rune) *charset {
	cs := newCharSet()
	cs.add(r)
	return cs
}

func newCharsetForRange(a, b rune) *charset {
	cs := newCharSet()
	cs.addRange(a, b)
	return cs
}

func fitcs(r rune) bool { return int(r) < 0x80 && int(r)>>7 != 1 }

func (cs *charset) complement() *charset {
	newCs := charset{}
	for i, item := range cs.bits {
		newCs.bits[i] = ^item
	}
	return &newCs
}

func (cs *charset) begin() int         { return 0 }
func (cs *charset) end() int           { return 256 }
func (cs *charset) eq(o *charset) bool { return cs.encoded() == o.encoded() }
func (cs *charset) encoded() string    { return hex.EncodeToString(cs.bits[:]) }

func (cs *charset) add(r rune) {
	if !fitcs(r) {
		panic(fmt.Sprintf("code point `U+%X` (%c) is out of bounds", r, r))
	}
	i := int(r)
	cs.bits[i>>3] |= 1 << (i & 7)
}

func (cs *charset) addRange(start, end rune) {
	if !fitcs(start) || !fitcs(end) || start > end {
		panic(fmt.Sprintf("range out of charset bounds: %c-%c", start, end))
	}
	for r := start; r <= end; r++ {
		cs.add(r)
	}
}

func (cs *charset) hasByte(i byte) bool {
	// writing `i/8` as `i>>3` and `i%8` as `i&7` because division
	// is usually slower than bit shifting operators.
	return cs.bits[i>>3]&(1<<(i&7)) != 0
}

func (cs *charset) popcount() int {
	var total int
	for _, oneByte := range cs.bits {
		total += bits.OnesCount16(uint16(oneByte))
	}
	return total
}

func charsetMerge(a, b *charset) *charset {
	out := newCharSet()
	for i := 0; i < 32; i++ {
		out.bits[i] = a.bits[i] | b.bits[i]
	}
	return out
}

func (cs *charset) precomputeExpectedSet() []expected {
	var (
		ex []expected
		rg bool
		st int
		pr int = -2
	)
	// If we've got too many entries on the set, it means that
	// it's likely a `complement` set, and it won't look good as
	// debug info anyway, so we just ommit it here.
	if cs.popcount() > 100 {
		return ex
	}
	for r := cs.begin(); r < cs.end(); r++ {
		has := cs.hasByte(byte(r))
		if has {
			if !rg {
				rg = true
				st = r
			}
			pr = r
		} else if rg {
			rg = false
			addRangeToSlice(&ex, rune(st), rune(pr))
		}
	}
	if rg {
		addRangeToSlice(&ex, rune(st), rune(pr))
	}

	return ex
}

func addRangeToSlice(ex *[]expected, start, end rune) {
	if start == end {
		*ex = append(*ex, expected{a: start})
	} else if end == start+1 {
		*ex = append(*ex, expected{a: start})
		*ex = append(*ex, expected{a: end})
	} else {
		*ex = append(*ex, expected{a: start, b: end})
	}
}

func (cs *charset) String() string {
	var (
		s  strings.Builder
		rg bool
		st int
		pr int = -2
	)
	s.WriteString("[")

	for r := cs.begin(); r < cs.end(); r++ {
		has := cs.hasByte(byte(r))
		if has {
			if !rg {
				rg = true
				st = r
			}
			pr = r
		} else if rg {
			rg = false
			addRangeToStr(&s, rune(st), rune(pr))
		}
	}
	if rg {
		addRangeToStr(&s, rune(st), rune(pr))
	}

	s.WriteString("]")
	return s.String()
}

func addRangeToStr(s *strings.Builder, start, end rune) {
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
