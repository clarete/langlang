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

	// predicate is a flag that marks the current stack frame as a
	// frame created within the predicate Not.
	predicate bool

	// capId is used in `frameType_Capture` and stores the string
	// ID of the capture that is being created
	capId int

	// nodes keeps a slice of the node IDs currently captured under
	// a capId
	nodes []NodeID

	// suppress is true if we should *not* keep any captures under
	// this frame (nor any frame nested underneath it)
	suppress bool
}

func (f *frame) len() int {
	return len(f.nodes)
}

func (f *frame) capture(nodes ...NodeID) {
	if len(f.nodes) == 0 {
		f.nodes = nodes
	} else {
		f.nodes = append(f.nodes, nodes...)
	}
}

type stack struct {
	frames []frame
	nodes  []NodeID
	tree   *tree
}

func (s *stack) push(f frame) {
	s.frames = append(s.frames, f)
}

func (s *stack) pop() frame {
	idx := len(s.frames) - 1
	f := s.frames[idx]
	// Clear nodes slice in the frame that's still in the stack
	// to help GC and potentially reuse capacity
	s.frames[idx].nodes = s.frames[idx].nodes[:0]
	s.frames = s.frames[:idx]
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

func (s *stack) capture(nodes ...NodeID) {
	if len(nodes) == 0 {
		return
	}
	if s.len() > 0 {
		s.top().capture(nodes...)
		return
	}
	if len(s.nodes) == 0 {
		s.nodes = nodes
	} else {
		s.nodes = append(s.nodes, nodes...)
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
			s.nodes = append(s.nodes, f.nodes...)
		} else {
			s.peek(1).capture(f.nodes...)
		}
	}
}
