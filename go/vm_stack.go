package langlang

type frameType int

const (
	frameType_Backtracking frameType = iota
	frameType_Call
	frameType_Capture
)

type frame struct {
	t frameType

	// pc is used in both `frameType_{Backtracking,Call}` and
	// stores the program counter index
	pc int

	// cursor is used in `frameType_{Backtracking,Capture}` and
	// stores the position of the parser cursor
	cursor int

	// line is a zero-based counter that increments when
	// `updatePos` sees a `\n` character.
	line int

	// column is a zero-based counter that increments when
	// `updatePos` sees a new character.  It gets reset when `\n`
	// is under the cursor.
	column int

	// predicate is a flag that marks the current stack frame as a
	// frame created within the predicate Not.
	predicate bool

	// capId is used in `frameType_Capture` and stores the string
	// ID of the capture that is being created
	capId int

	// values keeps a slice of the values currently captured under
	// a capId
	values []Value

	// captured contains how many values have been captured
	captured int

	// suppress is true if we should *not* keep any captures under
	// this frame (nor any frame nested underneath it)
	suppress bool
}

type stack []frame

func (s *stack) push(f frame) {
	*s = append(*s, f)
}

func (s *stack) pop() frame {
	f := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return f
}

func (s *stack) top() *frame {
	return &(*s)[len(*s)-1]
}

func (s *stack) len() int {
	return len(*s)
}

func (s *stack) dropUncommittedValues(captured int) {
	if top, ok := s.findCaptureFrame(); ok {
		top.values = top.values[:captured]
	}
}

func (s *stack) findCaptureFrame() (*frame, bool) {
	for i := s.len() - 1; i >= 0; i-- {
		if (*s)[i].t == frameType_Capture {
			return &(*s)[i], true
		}
	}
	return nil, false
}
