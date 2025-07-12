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

	// suppress is true if we should *not* keep any captures under
	// this frame (nor any frame nested underneath it)
	suppress bool
}

func (f *frame) len() int {
	return len(f.values)
}

func (f *frame) capture(values ...Value) {
	if len(f.values) == 0 {
		f.values = values
	} else {
		f.values = append(f.values, values...)
	}
}

type stack struct {
	frames []frame
	values []Value
}

func (s *stack) push(f frame) {
	s.frames = append(s.frames, f)
}

func (s *stack) pop() frame {
	f := (s.frames)[len(s.frames)-1]
	s.frames = (s.frames)[:len(s.frames)-1]
	return f
}

func (s *stack) top() *frame {
	return s.peek(0)
}

func (s *stack) peek(n int) *frame {
	return &s.frames[len(s.frames)-n-1]
}

func (s *stack) len() int {
	return len(s.frames)
}

func (s *stack) capture(values ...Value) {
	if len(values) == 0 {
		return
	}
	if s.len() > 0 {
		s.top().capture(values...)
		return
	}
	if len(s.values) == 0 {
		s.values = values
	} else {
		s.values = append(s.values, values...)
	}
}

func (s *stack) collectCaptures() {
	n := s.len()
	if n == 0 {
		return
	}
	f := s.top()
	if f.len() > 0 {
		if n == 1 {
			s.values = append(s.values, f.values...)
		} else {
			s.peek(1).capture(f.values...)
		}
	}
}
