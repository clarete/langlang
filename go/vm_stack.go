package langlang

type frameType int8

const (
	frameType_Backtracking frameType = iota
	frameType_Call
	frameType_Capture
)

type frame struct {
	// cursor is used in `frameType_{Backtracking,Capture}` and
	// stores the position of the parser cursor
	cursor int // 8 bytes, offset 0-7

	// pc is used in both `frameType_{Backtracking,Call}` and
	// stores the program counter index
	pc uint32 // 4 bytes, offset 8-11

	// capId is used in `frameType_Capture` and stores the string
	// ID of the capture that is being created
	capId uint32 // 4 bytes, offset 12-15

	// nodesStart and nodesEnd are indices into the stack's
	// nodeArena.  This avoids per-frame slice allocations.
	nodesStart uint32 // 4 bytes, offset 16-19
	nodesEnd   uint32 // 4 bytes, offset 20-23

	// t is the type of the frame
	t frameType // 1 byte, offset 24

	// predicate is a flag that marks the current stack frame as a
	// frame created within the predicate Not.
	predicate bool // 1 byte, offset 25

	// implicitly padding: 6 bytes, offset 26-32
}

type stack struct {
	frames    []frame
	nodeArena []NodeID // shared arena for all frame captures
	nodes     []NodeID // top-level captures (when no frames on stack)
	tree      *tree
}

// push adds a frame to the stack. The frame's node range starts at
// the current arena position.
func (s *stack) push(f frame) {
	f.nodesStart = uint32(len(s.nodeArena))
	f.nodesEnd = f.nodesStart
	s.frames = append(s.frames, f)
}

func (s *stack) pop() frame {
	idx := len(s.frames) - 1
	f := s.frames[idx]
	s.frames = s.frames[:idx]
	return f
}

func (s *stack) top() *frame {
	return &s.frames[len(s.frames)-1]
}

func (s *stack) len() int {
	return len(s.frames)
}

// frameNodes returns the captured nodes for a frame as a slice.
func (s *stack) frameNodes(f *frame) []NodeID {
	return s.nodeArena[f.nodesStart:f.nodesEnd]
}

// capture adds nodes to the top frame, or to s.nodes if no frames exist.
func (s *stack) capture(nodes ...NodeID) {
	if len(nodes) == 0 {
		return
	}
	n := len(s.frames)
	if n > 0 {
		// Append to arena and update top frame's end index
		s.nodeArena = append(s.nodeArena, nodes...)
		s.frames[n-1].nodesEnd = uint32(len(s.nodeArena))
		return
	}
	s.nodes = append(s.nodes, nodes...)
}

// commitCapturesToParent transfers the popped child frame's captures
// to its parent by extending the parent's range.  Called AFTER pop,
// so current top is the parent.  `childStart` and `childEnd` are the
// arena indices from the popped frame.
func (s *stack) commitCapturesToParent(childStart, childEnd uint32) {
	if childStart == childEnd {
		// No captures to commit
		return
	}
	n := len(s.frames)
	if n == 0 {
		// Stack is empty, copy to top-level nodes
		s.nodes = append(s.nodes, s.nodeArena[childStart:childEnd]...)
	} else {
		// Extend current parent's range to include child's
		// captures
		s.frames[n-1].nodesEnd = childEnd
	}
}

// collectCaptures moves captures from the top frame to its parent (or
// to `s.nodes` if it's the only frame). Used for partial commits.
func (s *stack) collectCaptures() {
	n := len(s.frames)
	if n == 0 {
		return
	}
	f := &s.frames[n-1]
	if f.nodesEnd > f.nodesStart {
		if n == 1 {
			s.nodes = append(s.nodes, s.nodeArena[f.nodesStart:f.nodesEnd]...)
		} else {
			// Extend parent's range to include this frame's captures
			s.frames[n-2].nodesEnd = f.nodesEnd
		}
	}
}

// truncateArena resets the arena to a given position, used when
// discarding captures on backtrack or fail.
func (s *stack) truncateArena(pos uint32) {
	s.nodeArena = s.nodeArena[:pos]
}

// reset clears the stack for reuse.
func (s *stack) reset() {
	s.frames = s.frames[:0]
	s.nodeArena = s.nodeArena[:0]
	s.nodes = s.nodes[:0]
}
