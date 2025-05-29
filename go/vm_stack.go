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

	predicate bool

	// capId is used in `frameType_Capture` and stores the string
	// ID of the capture that is being created
	capId int

	// values keeps a slice of the values currently captured under
	// a capId
	values []Value
}

type stack struct {
	frames []frame
	values []Value
}

func (s *stack) capture(values ...Value) {
	if len(s.values) == 0 {
		s.values = values
	} else {
		s.values = append(s.values, values...)
	}
}

func (s *stack) push(f frame) {
	s.frames = append(s.frames, f)
}

func (s *stack) pop() frame {
	f := s.frames[len(s.frames)-1]
	s.frames = s.frames[:len(s.frames)-1]
	return f
}

func (s *stack) top() *frame {
	return &s.frames[len(s.frames)-1]
}

func (s *stack) len() int {
	return len(s.frames)
}

func (s *stack) pushBacktrack(pc, cursor int) {
	var predicate bool
	if s.len() > 0 {
		predicate = s.top().predicate
	}
	s.push(frame{
		t:         frameType_Backtracking,
		pc:        pc,
		cursor:    cursor,
		predicate: predicate,
	})
}

func (s *stack) pushBacktrackPred(pc, cursor int) {
	s.push(frame{
		t:         frameType_Backtracking,
		pc:        pc,
		cursor:    cursor,
		predicate: true,
	})
}

func (s *stack) pushCall(pc int) {
	s.push(frame{t: frameType_Call, pc: pc})
}
