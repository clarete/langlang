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

func NewSpan(start, end Location) Span {
	return Span{Start: start, End: end}
}

func (s Span) String() string {
	startLoc := s.Start
	endLoc := s.End
	startLine, startCol := int(startLoc.Line), int(startLoc.Column)
	endLine, endCol := int(endLoc.Line), int(endLoc.Column)
	if startLine == endLine && endLine == startCol && endCol == endLine && startLine == 0 {
		startLine++
		endLine++
		startCol++
		endCol++
	}
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

// ---- Position index ----

// posIndex provides Location/Span (rune columns) and CursorU16
// (UTF-16 code-unit offsets) derived from the same underlying UTF-8
// input.
type posIndex struct {
	input []byte

	// lineStart holds byte 0-based offsets of each line start
	lineStart []int

	// lazily built checkpointed units indexes
	runeUnits, u16Units *unitsIndex
}

func newPosIndex(input []byte) *posIndex {
	// Always include line 1 starting at offset 0.
	lineStart := make([]int, 1, 64)
	lineStart[0] = 0
	for i, b := range input {
		if b == '\n' {
			// next line starts after '\n'
			lineStart = append(lineStart, i+1)
		}
	}
	return &posIndex{input: input, lineStart: lineStart}
}

func (pi *posIndex) Span(r Range) Span {
	return Span{
		Start: pi.LocationAt(r.Start),
		End:   pi.LocationAt(r.End),
	}
}

func (pi *posIndex) LocationAt(cursor int) Location {
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(pi.input) {
		cursor = len(pi.input)
	}

	// Find first lineStart > cursor, then step back one.
	lineIdx := sort.Search(len(pi.lineStart), func(i int) bool {
		return pi.lineStart[i] > cursor
	}) - 1
	if lineIdx < 0 {
		lineIdx = 0
	}

	lineStart := pi.lineStart[lineIdx]

	pi.ensureRuneUnits()

	// Column is rune-based and 1-indexed
	col := int32(pi.runeUnits.UnitsAt(cursor)-pi.runeUnits.UnitsAt(lineStart)) + 1

	return Location{
		Line:   int32(lineIdx + 1),
		Column: col,
		Cursor: cursor,
	}
}

func (pi *posIndex) CursorU16(cursor int) int {
	pi.ensureU16Units()
	return pi.u16Units.UnitsAt(cursor)
}

func (pi *posIndex) ensureRuneUnits() {
	if pi.runeUnits == nil {
		pi.runeUnits = newUnitsIndex(pi.input, unitsModeRune)
	}
}

func (pi *posIndex) ensureU16Units() {
	if pi.u16Units == nil {
		pi.u16Units = newUnitsIndex(pi.input, unitsModeUTF16)
	}
}

// ---- checkpointed units index (runes or UTF-16 code units) ----

type unitsMode uint8

const (
	unitsModeRune unitsMode = iota
	unitsModeUTF16
)

// unitsIndex maps UTF-8 byte offsets (Go cursors) to absolute "units"
// offsets.  The meaning of "units" depends on the builder:
//
//   - UTF8:  +1 per decoded rune
//   - UTF16: +1 per BMP rune, +2 for surrogate-pairs (r > 0xFFFF)
//
// It is built lazily and uses sparse checkpoints to avoid O(n)
// memory.  UnitsAt is O(log checkpoints + stride) per call after the
// initial build.
type unitsIndex struct {
	input []byte
	// checkpoints are byte offsets at UTF-8 rune boundaries.
	byteOffsets []int
	// unitOffsets are absolute unit offsets at the corresponding
	// byteOffsets.
	unitOffsets []int
	// mode is either rune or utf16
	mode unitsMode
}

func newUnitsIndex(input []byte, mode unitsMode) *unitsIndex {
	// Chosen to keep per-call scanning bounded while keeping the
	// index small
	const strideBytes = 64

	var (
		unitCount            = 0
		bytesSinceCheckpoint = 0
		index                = &unitsIndex{
			input:       input,
			byteOffsets: make([]int, 0, 128),
			unitOffsets: make([]int, 0, 128),
			mode:        mode,
		}
	)

	index.byteOffsets = append(index.byteOffsets, 0)
	index.unitOffsets = append(index.unitOffsets, 0)

	for i := 0; i < len(input); {
		r, size := utf8.DecodeRune(input[i:])
		if size <= 0 {
			size = 1
			r = utf8.RuneError
		}

		// Advance by this rune
		i += size
		unitCount += index.unitsForRune(r)
		bytesSinceCheckpoint += size

		// Emit a checkpoint at the *current* rune boundary:
		if bytesSinceCheckpoint >= strideBytes {
			index.byteOffsets = append(index.byteOffsets, i)
			index.unitOffsets = append(index.unitOffsets, unitCount)
			bytesSinceCheckpoint = 0
		}
	}

	// Ensure checkpoint at EOF
	if last := index.byteOffsets[len(index.byteOffsets)-1]; last != len(input) {
		index.byteOffsets = append(index.byteOffsets, len(input))
		index.unitOffsets = append(index.unitOffsets, unitCount)
	}
	return index
}

func (ix *unitsIndex) UnitsAt(cursor int) int {
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(ix.input) {
		cursor = len(ix.input)
	}

	// Find last checkpoint byteOffset <= cursor.
	i := sort.Search(len(ix.byteOffsets), func(i int) bool {
		return ix.byteOffsets[i] > cursor
	}) - 1
	if i < 0 {
		i = 0
	}

	bytePos := ix.byteOffsets[i]
	unitPos := ix.unitOffsets[i]

	// Scan forward from checkpoint to cursor
	for bytePos < cursor {
		r, size := utf8.DecodeRune(ix.input[bytePos:])
		if size <= 0 {
			size = 1
			r = utf8.RuneError
		}
		if bytePos+size > cursor {
			break
		}
		unitPos += ix.unitsForRune(r)
		bytePos += size
	}
	return unitPos
}

func (ix *unitsIndex) unitsForRune(r rune) int {
	switch ix.mode {
	case unitsModeRune:
		return 1
	case unitsModeUTF16:
		if r > 0xFFFF {
			return 2
		}
		return 1
	default:
		return 1
	}
}
