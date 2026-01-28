package langlang

// OracleCharSet represents a set of valid next characters.  It
// supports both ASCII ranges (via bitmap) and Unicode ranges.
type OracleCharSet struct {
	ascii   charset   // bitmap for ASCII chars 0-127
	ranges  [][2]rune // non-ASCII ranges [lo, hi]
	any     bool      // true if ANY character is valid
	isEmpty bool      // cached emptiness check
}

func NewOracleCharSet() *OracleCharSet      { return &OracleCharSet{isEmpty: true} }
func (cs *OracleCharSet) IsAny() bool       { return cs.any }
func (cs *OracleCharSet) IsEmpty() bool     { return cs.isEmpty }
func (cs *OracleCharSet) Ranges() [][2]rune { return cs.ranges }

func (cs *OracleCharSet) Add(r rune) {
	cs.isEmpty = false
	if fitcs(r) {
		cs.ascii.add(r)
	} else {
		cs.ranges = append(cs.ranges, [2]rune{r, r})
	}
}

func (cs *OracleCharSet) AddRange(lo, hi rune) {
	cs.isEmpty = false
	// Handle ASCII portion
	if fitcs(lo) && fitcs(hi) {
		cs.ascii.addRange(lo, hi)
		return
	}
	// Handle mixed or pure Unicode ranges
	if fitcs(lo) {
		cs.ascii.addRange(lo, 0x7F)
		lo = 0x80
	}
	if lo <= hi {
		cs.ranges = append(cs.ranges, [2]rune{lo, hi})
	}
}

func (cs *OracleCharSet) AddCharset(other *charset) {
	if other == nil {
		return
	}
	cs.isEmpty = false
	for i := 0; i < 32; i++ {
		cs.ascii.bits[i] |= other.bits[i]
	}
}

func (cs *OracleCharSet) SetAny() {
	cs.any = true
	cs.isEmpty = false
}

func (cs *OracleCharSet) Contains(r rune) bool {
	if cs.any {
		return true
	}
	if fitcs(r) {
		return cs.ascii.hasByte(byte(r))
	}
	for _, rng := range cs.ranges {
		if r >= rng[0] && r <= rng[1] {
			return true
		}
	}
	return false
}

func (cs *OracleCharSet) Union(other *OracleCharSet) {
	if other.any {
		cs.any = true
		cs.isEmpty = false
		return
	}
	if other.isEmpty {
		return
	}
	cs.isEmpty = false
	for i := 0; i < 32; i++ {
		cs.ascii.bits[i] |= other.ascii.bits[i]
	}
	cs.ranges = append(cs.ranges, other.ranges...)
}

func (cs *OracleCharSet) Runes() []rune {
	if cs.any {
		return nil // too many to enumerate
	}
	var runes []rune
	for r := 0; r < 128; r++ {
		if cs.ascii.hasByte(byte(r)) {
			runes = append(runes, rune(r))
		}
	}
	return runes
}

func (cs *OracleCharSet) Equal(other *OracleCharSet) bool {
	if cs.any != other.any {
		return false
	}
	if cs.any {
		return true
	}
	if cs.isEmpty != other.isEmpty {
		return false
	}
	// Compare ASCII bitmaps
	for i := 0; i < 32; i++ {
		if cs.ascii.bits[i] != other.ascii.bits[i] {
			return false
		}
	}
	// Compare ranges (simplified - just length for now)
	if len(cs.ranges) != len(other.ranges) {
		return false
	}
	for i, r := range cs.ranges {
		if r != other.ranges[i] {
			return false
		}
	}
	return true
}
