package langlang

import (
	"fmt"
	"sort"
	"unicode/utf8"
)

const eof = -1

//  ---- Range ----

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

//  ---- Span ----

func (s Span) String() string {
	startLoc := s.Start
	endLoc := s.End
	startLine, startCol := int(startLoc.Line), int(startLoc.Column)
	endLine, endCol := int(endLoc.Line), int(endLoc.Column)
	if startLine == endLine && startLine == 1 {
		if startCol == endCol {
			return fmt.Sprintf("%d", startCol)
		}
		return fmt.Sprintf("%d..%d", startCol, endCol)
	}
	if startLine == endLine && startCol == endCol {
		return fmt.Sprintf("%d:%d", startLine, startCol)
	}
	return fmt.Sprintf("%d:%d..%d:%d", startLine, startCol, endLine, endCol)
}

// LineIndex allows fast conversion from byte cursor offsets to line/column.
//
// It stores the start byte offset of each line (0-based). Given a
// cursor, it finds the line by binary searching line starts (O(log
// lines)) and computes the column as (runes since lineStart + 1).
//
// Construction is O(n) over the input and is intended to be cached
// per input.
type LineIndex struct {
	input     []byte
	lineStart []int
}

func NewLineIndex(input []byte) *LineIndex {
	// Always include line 1 starting at offset 0.
	lineStart := make([]int, 1, 64)
	lineStart[0] = 0
	for i, b := range input {
		if b == '\n' {
			// next line starts after '\n'
			lineStart = append(lineStart, i+1)
		}
	}
	return &LineIndex{input: input, lineStart: lineStart}
}

func (li *LineIndex) Span(r Range) Span {
	return Span{
		Start: li.LocationAt(r.Start),
		End:   li.LocationAt(r.End),
	}
}

func (li *LineIndex) LocationAt(cursor int) Location {
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(li.input) {
		cursor = len(li.input)
	}

	// Find first lineStart > cursor, then step back one.
	lineIdx := sort.Search(len(li.lineStart), func(i int) bool {
		return li.lineStart[i] > cursor
	}) - 1
	if lineIdx < 0 {
		lineIdx = 0
	}

	lineStart := li.lineStart[lineIdx]
	// Column is rune-based and 1-indexed.
	col := int32(utf8.RuneCount(li.input[lineStart:cursor])) + 1

	return Location{
		Line:   int32(lineIdx + 1),
		Column: col,
		Cursor: cursor,
	}
}
