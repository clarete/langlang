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

	line int

	column int

	predicate bool

	// capId is used in `frameType_Capture` and stores the string
	// ID of the capture that is being created
	capId int

	// values keeps a slice of the values currently captured under
	// a capId
	values []Value
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

func (s *stack) findCaptureFrame() (*frame, bool) {
	for i := s.len() - 1; i >= 0; i-- {
		if (*s)[i].t == frameType_Capture {
			return &(*s)[i], true
		}
	}
	return nil, false
}
