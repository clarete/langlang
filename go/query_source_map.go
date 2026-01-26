package langlang

// SourceMapQuery provides bytecode to grammar source mappings for
// debugging and visualization without requiring VM execution.  This
// computes the same mapping as encoding but without producing the
// actual bytecode bytes.
var SourceMapQuery = &Query[FilePath, *SourceMap]{
	Name:    "SourceMap",
	Compute: computeSourceMap,
}

func computeSourceMap(db *Database, key FilePath) (*SourceMap, error) {
	// Get the compiled program (not bytecode) - we compute offsets
	// by walking instructions and summing SizeInBytes()
	program, err := Get(db, CompiledProgramQuery, key)
	if err != nil {
		return nil, err
	}
	return BuildSourceMapFromProgram(program), nil
}

// BuildSourceMapFromProgram creates a SourceMap by walking the
// program's instructions and computing bytecode offsets without
// actually encoding the bytecode. This is useful for debugging and
// visualization tools that need source mappings.
func BuildSourceMapFromProgram(p *Program) *SourceMap {
	var (
		cursor  int
		entries []srcMapEntry
		lastSrc SourceLocation
	)

	for _, instruction := range p.code {
		// Labels don't emit bytes
		if _, ok := instruction.(ILabel); ok {
			continue
		}

		src := instruction.SourceLocation()
		if src != lastSrc {
			lastSrc = src
			entries = append(entries, srcMapEntry{
				offset:      cursor,
				fileID:      int(src.FileID),
				startLine:   src.Span.Start.Line,
				startCol:    src.Span.Start.Column,
				startCursor: src.Span.Start.Cursor,
				endLine:     src.Span.End.Line,
				endCol:      src.Span.End.Column,
				endCursor:   src.Span.End.Cursor,
			})
		}
		cursor += instruction.SizeInBytes()
	}

	return &SourceMap{
		Data:  srcMapDeltaEncode(entries),
		Files: p.sourceFiles,
	}
}

// ---- SourceMap methods ----

// FileAt returns the file path for a given FileID.
func (sm *SourceMap) FileAt(fileID FileID) string {
	return sm.Files[fileID]
}

// Len returns the number of entries in the source map.
func (sm *SourceMap) Len() int {
	return len(sm.getEntries())
}

// LocationAt returns the grammar source location for a given bytecode
// offset.  On the first call, it decodes the delta+varint data and
// caches the result.
func (sm *SourceMap) LocationAt(pc int) (SourceLocation, bool) {
	entries := sm.getEntries()
	lo, hi := 0, len(entries)
	for lo < hi {
		mid := (lo + hi) / 2
		if entries[mid].offset <= pc {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo == 0 {
		return SourceLocation{}, false
	}
	e := entries[lo-1]
	return SourceLocation{
		FileID: FileID(e.fileID),
		Span: Span{
			Start: Location{
				Line:   e.startLine,
				Column: e.startCol,
				Cursor: e.startCursor,
			},
			End: Location{
				Line:   e.endLine,
				Column: e.endCol,
				Cursor: e.endCursor,
			},
		},
	}, true
}

// getEntries lazily decodes the delta+varint data on first access.
func (sm *SourceMap) getEntries() []srcMapEntry {
	if sm.entries == nil {
		sm.entries = srcMapDeltaDecode(sm.Data)
	}
	return sm.entries
}

// ---- Delta + Varint encoding/decoding for source map data ----

// Format: First entry uses unsigned varints for absolute values.
// Subsequent entries use signed varints (zigzag) for deltas from
// previous.
//
// Entry fields (in order):
//
//   - BytecodeOffset (always increasing, so unsigned delta)
//
//   - FileID, StartLine, StartCol, StartCursor, EndLine, EndCol,
//     EndCursor (signed deltas)
//
// Varint format (unsigned): 7 bits per byte, MSB = continuation flag
//
// zigzag encoding for signed: (n << 1) ^ (n >> 31) maps small
// negatives to small positives

// srcMapDeltaEncode compresses entries using delta + varint encoding.
func srcMapDeltaEncode(entries []srcMapEntry) []byte {
	var (
		// estimate ~8 bytes per entry
		out  = make([]byte, 0, len(entries)*8)
		prev srcMapEntry
	)
	for i, e := range entries {
		if i == 0 {
			// First entry: absolute values (unsigned varints)
			out = appendUvarint(out, uint64(e.offset))
			out = appendUvarint(out, uint64(e.fileID))
			out = appendUvarint(out, uint64(e.startLine))
			out = appendUvarint(out, uint64(e.startCol))
			out = appendUvarint(out, uint64(e.startCursor))
			out = appendUvarint(out, uint64(e.endLine))
			out = appendUvarint(out, uint64(e.endCol))
			out = appendUvarint(out, uint64(e.endCursor))
		} else {
			// Subsequent entries: deltas (offset is unsigned, rest are signed)
			out = appendUvarint(out, uint64(e.offset-prev.offset))
			out = appendSvarint(out, int64(e.fileID-prev.fileID))
			out = appendSvarint(out, int64(e.startLine-prev.startLine))
			out = appendSvarint(out, int64(e.startCol-prev.startCol))
			out = appendSvarint(out, int64(e.startCursor-prev.startCursor))
			out = appendSvarint(out, int64(e.endLine-prev.endLine))
			out = appendSvarint(out, int64(e.endCol-prev.endCol))
			out = appendSvarint(out, int64(e.endCursor-prev.endCursor))
		}
		prev = e
	}
	return out
}

// srcMapDeltaDecode decompresses delta + varint encoded data.
func srcMapDeltaDecode(data []byte) []srcMapEntry {
	var (
		prev    srcMapEntry
		pos     = 0
		entries = make([]srcMapEntry, 0, len(data)/8) // estimate
	)
	for pos < len(data) {
		var (
			e srcMapEntry
			v uint64
			s int64
		)
		if len(entries) == 0 {
			// First entry: absolute values
			v, pos = readUvarint(data, pos)
			e.offset = int(v)
			v, pos = readUvarint(data, pos)
			e.fileID = int(v)
			v, pos = readUvarint(data, pos)
			e.startLine = int(v)
			v, pos = readUvarint(data, pos)
			e.startCol = int(v)
			v, pos = readUvarint(data, pos)
			e.startCursor = int(v)
			v, pos = readUvarint(data, pos)
			e.endLine = int(v)
			v, pos = readUvarint(data, pos)
			e.endCol = int(v)
			v, pos = readUvarint(data, pos)
			e.endCursor = int(v)
		} else {
			// Subsequent entries: deltas
			v, pos = readUvarint(data, pos)
			e.offset = prev.offset + int(v)
			s, pos = readSvarint(data, pos)
			e.fileID = prev.fileID + int(s)
			s, pos = readSvarint(data, pos)
			e.startLine = prev.startLine + int(s)
			s, pos = readSvarint(data, pos)
			e.startCol = prev.startCol + int(s)
			s, pos = readSvarint(data, pos)
			e.startCursor = prev.startCursor + int(s)
			s, pos = readSvarint(data, pos)
			e.endLine = prev.endLine + int(s)
			s, pos = readSvarint(data, pos)
			e.endCol = prev.endCol + int(s)
			s, pos = readSvarint(data, pos)
			e.endCursor = prev.endCursor + int(s)
		}
		entries = append(entries, e)
		prev = e
	}
	return entries
}

// appendUvarint appends an unsigned varint to the buffer.
func appendUvarint(buf []byte, v uint64) []byte {
	for v >= 0x80 {
		buf = append(buf, byte(v)|0x80)
		v >>= 7
	}
	return append(buf, byte(v))
}

// appendSvarint appends a signed varint using zigzag encoding.
func appendSvarint(buf []byte, v int64) []byte {
	return appendUvarint(buf, uint64((v<<1)^(v>>63)))
}

// readUvarint reads an unsigned varint from data at pos, returns
// value and new pos.
func readUvarint(data []byte, pos int) (uint64, int) {
	var (
		v     uint64
		shift uint
	)
	for pos < len(data) {
		b := data[pos]
		pos++
		v |= uint64(b&0x7F) << shift
		if b < 0x80 {
			break
		}
		shift += 7
	}
	return v, pos
}

// readSvarint reads a signed varint from data at pos (zigzag decoded)
func readSvarint(data []byte, pos int) (int64, int) {
	v, pos := readUvarint(data, pos)
	return int64((v >> 1) ^ -(v & 1)), pos
}
