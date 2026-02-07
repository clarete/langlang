package langlang

type frameType int8

const (
	frameType_Backtracking frameType = iota
	frameType_Call
	frameType_Capture
	frameType_LRCall
)

const lrResultLeftRec = -1 // Initial state - left recursive call in progress

type frame struct {
	// cursor is used in `frameType_{Backtracking,Capture,LRCall}` and
	// stores the position of the parser cursor
	cursor int // 8 bytes, offset 0-7

	// pc is used in both `frameType_{Backtracking,Call,LRCall}` and
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

	// 2 bytes implicit padding (offset 26-27)

	// lrIdx is a 1-based index into stack.lrData for LR frames.
	// 0 means no LR data (non-LR frame).  Only valid when
	// t == frameType_LRCall.
	lrIdx uint32 // 4 bytes, offset 28-31
}

// lrFrameData stores left-recursion state for LR call frames.
// Kept separate from frame to avoid bloating the hot frame struct
// (keeping it at 32 bytes for cache efficiency).
type lrFrameData struct {
	// address is the bytecode address of the left-recursive production
	address int
	// precedence is the precedence level of this LR call
	precedence int
	// result is the cursor position from the previous successful
	// iteration, or lrResultLeftRec if in initial state
	result int
	// committedEnd marks the end of committed captures in nodeArena
	// (captures from successful iterations that survive backtracking)
	committedEnd uint32
}

type stack struct {
	// Hot fields — keep together in the first cache line (offsets 0-71)
	frames    []frame  // offset 0
	nodeArena []NodeID // offset 24 — shared arena for all frame captures
	nodes     []NodeID // offset 48 — top-level captures (when no frames on stack)
	tree      *tree    // offset 72
	// Cold field — only used for left-recursive grammars
	lrData []lrFrameData // offset 80 — side storage for LR frame data (indexed by frame.lrIdx - 1)
}

// lr returns a pointer to the LR data for the given frame.
// Only valid when f.t == frameType_LRCall and f.lrIdx > 0.
func (s *stack) lr(f *frame) *lrFrameData {
	return &s.lrData[f.lrIdx-1]
}

// pushLR appends LR data and returns its 1-based index.
func (s *stack) pushLR(data lrFrameData) uint32 {
	s.lrData = append(s.lrData, data)
	return uint32(len(s.lrData))
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

func (s *stack) popAndCapture() frame {
	idx := len(s.frames) - 1
	f := s.frames[idx]
	s.frames = s.frames[:idx]

	if f.nodesStart != f.nodesEnd {
		if idx > 0 {
			// Extend current parent's range to include
			// child's captures
			s.frames[idx-1].nodesEnd = f.nodesEnd
		} else {
			// Stack is empty, copy to top-level nodes
			s.nodes = append(s.nodes, s.nodeArena[f.nodesStart:f.nodesEnd]...)
		}
	}
	return f
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
	if s.lrData != nil {
		s.lrData = s.lrData[:0]
	}
}
