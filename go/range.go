package langlang

import "fmt"

const eof = -1

// Range takes as little as possible (8 bytes in 64bit systems) to
// represent a position within the input.
type Range struct{ Start, End int }

func NewRange(start, end int) Range {
	return Range{Start: start, End: end}
}

func (r Range) String() string {
	if r.Start == r.End {
		return fmt.Sprintf("%d", r.Start)
	}
	return fmt.Sprintf("%d..%d", r.Start, r.End)
}

func (r Range) Str(v []byte) string {
	return string(v[r.Start:r.End])
}

func (r Range) Contains(other Range) bool {
	return other.Start >= r.Start && other.End <= r.End
}
