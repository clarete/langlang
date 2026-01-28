package langlang

//go:generate go run ./vmgen -mode oracle -vm-file vm.go -output-path vm_oracle.go

// OracleState represents a snapshot of the PEG VM state that can be
// used for grammar constrained decoding.
type OracleState struct {
	PC     int          // current program counter
	Cursor int          // current input position (prefix length consumed)
	Stack  []StackFrame // call/backtrack stack (uses shared StackFrame type)
}

// Clone creates a deep copy of the oracle state.
func (s OracleState) Clone() OracleState {
	stack := make([]StackFrame, len(s.Stack))
	copy(stack, s.Stack)
	return OracleState{
		PC:     s.PC,
		Cursor: s.Cursor,
		Stack:  stack,
	}
}

// OracleParser is the primary interface for grammar-constrained
// decoding.  It tracks multiple parallel parse paths and provides
// methods to advance through input while computing valid next
// characters.
type OracleParser struct {
	oracle *GrammarOracle
	states []OracleState
}

// NewParser creates a new parser starting at the grammar's entry
// point.  This is the main way to use the oracle for grammar
// constrained decoding.
func (o *GrammarOracle) NewParser() *OracleParser {
	return &OracleParser{
		oracle: o,
		states: []OracleState{o.start()},
	}
}

// NewParserAt creates a new parser starting at a specific rule address.
func (o *GrammarOracle) NewParserAt(ruleAddress int) *OracleParser {
	return &OracleParser{
		oracle: o,
		states: []OracleState{o.startAt(ruleAddress)},
	}
}

// States returns the current states being tracked (for debugging/inspection).
func (p *OracleParser) States() []OracleState {
	return p.states
}

// IsEmpty returns true if no valid parse paths remain.
func (p *OracleParser) IsEmpty() bool {
	return len(p.states) == 0
}

// Advance consumes a rune and returns a new parser with all surviving
// paths.  Returns an empty parser if the character is not valid at
// any current position.
func (p *OracleParser) Advance(ch rune) *OracleParser {
	var next []OracleState
	for _, s := range p.states {
		next = append(next, p.oracle.advance(s, ch)...)
	}
	// Deduplicate states: two states with same (PC, Stack) are equivalent
	next = deduplicateStates(next)
	return &OracleParser{oracle: p.oracle, states: next}
}

// deduplicateStates removes duplicate states based on (PC, Stack).
func deduplicateStates(states []OracleState) []OracleState {
	if len(states) <= 1 {
		return states
	}
	seen := make(map[string]bool)
	result := make([]OracleState, 0, len(states))
	for _, s := range states {
		key := stateKey(s)
		if !seen[key] {
			seen[key] = true
			result = append(result, s)
		}
	}
	return result
}

// stateKey creates a unique string key for state deduplication.
func stateKey(s OracleState) string {
	key := make([]byte, 0, 8+len(s.Stack)*8)
	key = append(key, byte(s.PC), byte(s.PC>>8), byte(s.PC>>16), byte(s.PC>>24))
	for _, f := range s.Stack {
		key = append(key, byte(f.Type), byte(f.PC), byte(f.PC>>8), byte(f.PC>>16))
	}
	return string(key)
}

// AdvanceString consumes a string and returns a new parser with all
// surviving paths.
func (p *OracleParser) AdvanceString(s string) *OracleParser {
	current := p
	for _, ch := range s {
		current = current.Advance(ch)
		if current.IsEmpty() {
			return current
		}
	}
	return current
}

// NextChars returns the set of characters valid at the current
// position.  This is the union of valid chars across all tracked
// parse paths.
func (p *OracleParser) NextChars() *OracleCharSet {
	result := NewOracleCharSet()
	for _, s := range p.states {
		chars := p.oracle.nextChars(s)
		for _, ch := range chars.Runes() {
			result.Add(ch)
		}
		if chars.IsAny() {
			result.SetAny()
			break
		}
	}
	return result
}

// IsAccepting returns true if any tracked path can accept (reach end
// of grammar).
func (p *OracleParser) IsAccepting() bool {
	for _, s := range p.states {
		if p.oracle.isAccepting(s) {
			return true
		}
	}
	return false
}

// Analyze computes both IsAccepting and NextChars in a single
// operation.  This is more efficient when you need both results.
func (p *OracleParser) Analyze() (accepting bool, chars *OracleCharSet) {
	chars = NewOracleCharSet()
	for _, s := range p.states {
		stateAccepting, stateChars := p.oracle.analyze(s)
		if stateAccepting {
			accepting = true
		}
		for _, ch := range stateChars.Runes() {
			chars.Add(ch)
		}
		if stateChars.IsAny() {
			chars.SetAny()
			break
		}
	}
	return accepting, chars
}

// GrammarOracle provides the interface for grammar constrained
// decoding.
type GrammarOracle struct {
	bytecode *Bytecode
	vm       *virtualMachine
}

// NewGrammarOracle creates a new oracle for the given compiled grammar.
func NewGrammarOracle(bytecode *Bytecode) *GrammarOracle {
	return &GrammarOracle{
		bytecode: bytecode,
		vm:       NewVirtualMachine(bytecode),
	}
}

// start returns the initial oracle state for parsing.
func (o *GrammarOracle) start() OracleState {
	return OracleState{
		PC:     0,
		Cursor: 0,
		Stack:  nil,
	}
}

// startAt returns an oracle state starting at a specific rule.
func (o *GrammarOracle) startAt(ruleAddress int) OracleState {
	if ruleAddress <= 0 {
		return o.start()
	}
	return OracleState{
		PC:     ruleAddress,
		Cursor: 0,
		Stack: []StackFrame{{
			Type: StackFrameCall,
			PC:   opCallSizeInBytes, // return to after the implicit call
		}},
	}
}

// advance attempts to consume a single rune from the given state.
// Returns all possible resulting states (one per surviving path).
func (o *GrammarOracle) advance(state OracleState, ch rune) []OracleState {
	// Find all positions that could consume input
	consumePoints := o.findConsumePoints(state)

	var results []OracleState
	for _, cp := range consumePoints {
		if o.matchesCharAt(cp, ch) {
			newState := o.advancePastConsumePoint(cp, state.Cursor)
			results = append(results, newState)
		}
	}

	return results
}

// advanceString attempts to consume a string from the given state.
func (o *GrammarOracle) advanceString(state OracleState, s string) []OracleState {
	current := []OracleState{state}
	for _, ch := range s {
		var next []OracleState
		for _, st := range current {
			next = append(next, o.advance(st, ch)...)
		}
		if len(next) == 0 {
			return nil
		}
		current = next
	}
	return current
}

// findConsumePoints explores all paths from a state to find positions
// where input-consuming instructions are reached.
// Returns exploreItem structs representing consume points (reuses the type).
func (o *GrammarOracle) findConsumePoints(state OracleState) []exploreItem {
	var (
		code     = o.bytecode.code
		results  = make([]exploreItem, 0, 100)
		visited  = make(map[visitKey]bool)
		worklist = []exploreItem{{
			pc:        state.PC,
			stack:     append([]StackFrame{}, state.Stack...),
			predicate: false,
		}}
	)
	for len(worklist) > 0 {
		item := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]

		var (
			key       = visitKey{pc: item.pc, stackLen: len(item.stack)}
			pc        = item.pc
			stack     = item.stack
			predicate = item.predicate
		)
		if visited[key] {
			continue
		}
		visited[key] = true

		push := func(f StackFrame) []StackFrame {
			return append(append([]StackFrame{}, stack...), f)
		}
		pop := func() (StackFrame, []StackFrame) {
			if len(stack) == 0 {
				return StackFrame{}, nil
			}
			idx := len(stack) - 1
			return stack[idx], stack[:idx]
		}
		addItem := func(newPC int, newStack []StackFrame, newPredicate bool) {
			worklist = append(worklist, exploreItem{
				pc:        newPC,
				stack:     newStack,
				predicate: newPredicate,
			})
		}
		tryBacktrack := func() {
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].Type == StackFrameBacktrack {
					addItem(stack[i].PC, append([]StackFrame{}, stack[:i]...), stack[i].Predicate)
					break
				}
			}
		}
		op := code[pc]

		switch op {
		case opHalt:
			continue

		case opAny, opChar, opChar32, opRange, opRange32, opSet:
			// Record this as a consume point (don't explore backtrack -
			// that's for failure, not alternative exploration)
			results = append(results, exploreItem{
				pc:    pc,
				stack: append([]StackFrame{}, stack...),
			})

		case opSpan:
			results = append(results, exploreItem{
				pc:    pc,
				stack: append([]StackFrame{}, stack...),
			})
			// Span can match zero chars, so also continue past it
			addItem(pc+opSpanSizeInBytes, append([]StackFrame{}, stack...), predicate)

		case opFail:
			for len(stack) > 0 {
				var f StackFrame
				f, stack = pop()
				if f.Type == StackFrameBacktrack {
					addItem(f.PC, append([]StackFrame{}, stack...), f.Predicate)
					break
				}
			}

		case opFailTwice:
			_, stack = pop()
			for len(stack) > 0 {
				var f StackFrame
				f, stack = pop()
				if f.Type == StackFrameBacktrack {
					addItem(f.PC, append([]StackFrame{}, stack...), f.Predicate)
					break
				}
			}

		case opChoice:
			lb := int(decodeU16(code, pc+1))
			addItem(pc+opChoiceSizeInBytes, push(StackFrame{
				Type:      StackFrameBacktrack,
				PC:        lb,
				Predicate: predicate,
			}), predicate)
			addItem(lb, append([]StackFrame{}, stack...), predicate)

		case opChoicePred:
			lb := int(decodeU16(code, pc+1))
			addItem(pc+opChoiceSizeInBytes, push(StackFrame{
				Type:      StackFrameBacktrack,
				PC:        lb,
				Predicate: true,
			}), true)
			addItem(lb, append([]StackFrame{}, stack...), predicate)

		case opCommit:
			_, newStack := pop()
			addItem(int(decodeU16(code, pc+1)), newStack, predicate)

		case opBackCommit:
			_, newStack := pop()
			addItem(int(decodeU16(code, pc+1)), newStack, predicate)

		case opPartialCommit:
			// Path 1: Continue the loop (jump to target)
			addItem(int(decodeU16(code, pc+1)), append([]StackFrame{}, stack...), predicate)
			// Path 2: Exit the loop (use backtrack frame if present)
			tryBacktrack()

		case opCall:
			addItem(int(decodeU16(code, pc+1)), push(StackFrame{
				Type: StackFrameCall,
				PC:   pc + opCallSizeInBytes,
			}), predicate)

		case opReturn, opCapReturn:
			f, newStack := pop()
			if f.Type == StackFrameCall {
				addItem(f.PC, newStack, predicate)
			}

		case opJump:
			addItem(int(decodeU16(code, pc+1)), append([]StackFrame{}, stack...), predicate)

		case opThrow:
			continue

		// Capture instructions, just skip past them
		case opCapBegin:
			addItem(pc+opCapBeginSizeInBytes, append([]StackFrame{}, stack...), predicate)
		case opCapEnd:
			addItem(pc+opCapEndSizeInBytes, append([]StackFrame{}, stack...), predicate)
		case opCapTerm:
			addItem(pc+opCapTermSizeInBytes, append([]StackFrame{}, stack...), predicate)
		case opCapNonTerm:
			addItem(pc+opCapNonTermSizeInBytes, append([]StackFrame{}, stack...), predicate)
		case opCapTermBeginOffset:
			addItem(pc+opCapTermBeginOffsetSizeInBytes, append([]StackFrame{}, stack...), predicate)
		case opCapNonTermBeginOffset:
			addItem(pc+opCapNonTermBeginOffsetSizeInBytes, append([]StackFrame{}, stack...), predicate)
		case opCapEndOffset:
			addItem(pc+opCapEndOffsetSizeInBytes, append([]StackFrame{}, stack...), predicate)
		case opCapCommit:
			_, newStack := pop()
			addItem(int(decodeU16(code, pc+1)), newStack, predicate)
		case opCapBackCommit:
			_, newStack := pop()
			addItem(int(decodeU16(code, pc+1)), newStack, predicate)
		case opCapPartialCommit:
			// Same as opPartialCommit - explore both loop continuation and exit
			addItem(int(decodeU16(code, pc+1)), append([]StackFrame{}, stack...), predicate)
			tryBacktrack()
		}
	}

	return results
}

// matchesCharAt checks if a consume point can match the given character.
func (o *GrammarOracle) matchesCharAt(cp exploreItem, ch rune) bool {
	var (
		code = o.bytecode.code
		sets = o.bytecode.sets
		op   = code[cp.pc]
	)
	switch op {
	case opAny:
		return true
	case opChar:
		return ch == rune(decodeU16(code, cp.pc+1))
	case opChar32:
		return ch == rune(decodeU32(code, cp.pc+1))
	case opRange:
		a := rune(decodeU16(code, cp.pc+1))
		b := rune(decodeU16(code, cp.pc+3))
		return ch >= a && ch <= b
	case opRange32:
		a := rune(decodeU32(code, cp.pc+1))
		b := rune(decodeU32(code, cp.pc+5))
		return ch >= a && ch <= b
	case opSet, opSpan:
		sid := decodeU16(code, cp.pc+1)
		cs := &sets[sid]
		if fitcs(ch) {
			return cs.hasByte(byte(ch))
		}
		return false
	default:
		return false
	}
}

// advancePastConsumePoint creates a new state after consuming at a point.
func (o *GrammarOracle) advancePastConsumePoint(cp exploreItem, baseCursor int) OracleState {
	var (
		code   = o.bytecode.code
		op     = code[cp.pc]
		nextPC int
	)
	switch op {
	case opAny:
		nextPC = cp.pc + opAnySizeInBytes
	case opChar:
		nextPC = cp.pc + opCharSizeInBytes
	case opChar32:
		nextPC = cp.pc + opChar32SizeInBytes
	case opRange:
		nextPC = cp.pc + opRangeSizeInBytes
	case opRange32:
		nextPC = cp.pc + opRange32SizeInBytes
	case opSet:
		nextPC = cp.pc + opSetSizeInBytes
	case opSpan:
		// After matching in a span, stay at the span instruction
		nextPC = cp.pc
	default:
		nextPC = cp.pc + 1
	}
	return OracleState{
		PC:     nextPC,
		Cursor: baseCursor + 1,
		Stack:  append([]StackFrame{}, cp.stack...),
	}
}

// isAccepting checks if the given state can reach an accepting state
// (opHalt) without consuming any more input.
func (o *GrammarOracle) isAccepting(state OracleState) bool {
	accepting, _ := o.exploreState(state, false)
	return accepting
}

// nextChars computes the set of characters that would be valid to
// consume next from the given state.
func (o *GrammarOracle) nextChars(state OracleState) *OracleCharSet {
	_, chars := o.exploreState(state, true)
	return chars
}

// analyze computes both isAccepting and nextChars in a single pass.
func (o *GrammarOracle) analyze(state OracleState) (accepting bool, chars *OracleCharSet) {
	return o.exploreState(state, true)
}

// visitKey is used for cycle detection within a worklist
type visitKey struct {
	pc, stackLen int
}

// exploreItem represents a state to explore (like a choice branch)
type exploreItem struct {
	pc        int
	stack     []StackFrame
	predicate bool
}

// exploreState is the unified worklist algorithm that computes both:
//
// - Whether any path can reach Halt without consuming input (accepting)
// - The set of characters that would be valid to consume next
//
// If collectChars is false, character collection is skipped (slightly
// faster when only IsAccepting is needed).
func (o *GrammarOracle) exploreState(
	state OracleState,
	collectChars bool,
) (accepting bool, chars *OracleCharSet) {
	code := o.bytecode.code
	sets := o.bytecode.sets
	visited := make(map[visitKey]bool)

	if collectChars {
		chars = NewOracleCharSet()
	}

	worklist := []exploreItem{{
		pc:        state.PC,
		stack:     append([]StackFrame{}, state.Stack...),
		predicate: false,
	}}

	for len(worklist) > 0 {
		// Pop from worklist
		item := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]

		pc := item.pc
		stack := item.stack
		predicate := item.predicate
		key := visitKey{pc: pc, stackLen: len(stack)}
		if visited[key] {
			continue
		}
		visited[key] = true

		push := func(f StackFrame) []StackFrame { return append(stack, f) }

		pop := func() (StackFrame, []StackFrame) {
			if len(stack) == 0 {
				return StackFrame{}, nil
			}
			idx := len(stack) - 1
			return stack[idx], stack[:idx]
		}
		addItem := func(newPC int, newStack []StackFrame, newPredicate bool) {
			worklist = append(worklist, exploreItem{
				pc:        newPC,
				stack:     newStack,
				predicate: newPredicate,
			})
		}
		tryBacktrack := func() {
			for len(stack) > 0 {
				var f StackFrame
				f, stack = pop()
				if f.Type == StackFrameBacktrack {
					addItem(f.PC, append([]StackFrame{}, stack...), f.Predicate)
					break
				}
			}
		}

		op := code[pc]

		switch op {
		case opHalt:
			accepting = true
			continue

		case opAny:
			if collectChars {
				chars.SetAny()
				return accepting, chars
			}

		case opChar:
			if collectChars {
				chars.Add(rune(decodeU16(code, pc+1)))
			}

		case opChar32:
			if collectChars {
				chars.Add(rune(decodeU32(code, pc+1)))
			}

		case opRange:
			if collectChars {
				a := rune(decodeU16(code, pc+1))
				b := rune(decodeU16(code, pc+3))
				chars.AddRange(a, b)
			}

		case opRange32:
			if collectChars {
				a := rune(decodeU32(code, pc+1))
				b := rune(decodeU32(code, pc+5))
				chars.AddRange(a, b)
			}

		case opSet:
			if collectChars {
				sid := decodeU16(code, pc+1)
				chars.AddCharset(&sets[sid])
			}

		case opSpan:
			if collectChars {
				sid := decodeU16(code, pc+1)
				chars.AddCharset(&sets[sid])
			}
			addItem(pc+opSpanSizeInBytes, append([]StackFrame{}, stack...), predicate)

		case opFail:
			tryBacktrack()

		case opFailTwice:
			_, stack = pop()
			tryBacktrack()

		case opChoice:
			lb := int(decodeU16(code, pc+1))
			// Explore both alternatives
			addItem(pc+opChoiceSizeInBytes, push(StackFrame{
				Type:      StackFrameBacktrack,
				PC:        lb,
				Predicate: predicate,
			}), predicate)
			addItem(lb, append([]StackFrame{}, stack...), predicate)

		case opChoicePred:
			lb := int(decodeU16(code, pc+1))
			addItem(pc+opChoiceSizeInBytes, push(StackFrame{
				Type:      StackFrameBacktrack,
				PC:        lb,
				Predicate: true,
			}), true)
			addItem(lb, append([]StackFrame{}, stack...), predicate)

		case opCommit:
			_, newStack := pop()
			addItem(int(decodeU16(code, pc+1)), newStack, predicate)

		case opBackCommit:
			_, newStack := pop()
			addItem(int(decodeU16(code, pc+1)), newStack, predicate)

		case opPartialCommit:
			addItem(int(decodeU16(code, pc+1)), append([]StackFrame{}, stack...), predicate)

		case opCall:
			addItem(int(decodeU16(code, pc+1)), push(StackFrame{
				Type: StackFrameCall,
				PC:   pc + opCallSizeInBytes,
			}), predicate)

		case opReturn, opCapReturn:
			f, newStack := pop()
			if f.Type == StackFrameCall {
				addItem(f.PC, newStack, predicate)
			}

		case opJump:
			addItem(int(decodeU16(code, pc+1)), append([]StackFrame{}, stack...), predicate)

		case opThrow:
			continue

		case opCapBegin:
			addItem(pc+opCapBeginSizeInBytes, append([]StackFrame{}, stack...), predicate)
		case opCapEnd:
			addItem(pc+opCapEndSizeInBytes, append([]StackFrame{}, stack...), predicate)
		case opCapTerm:
			addItem(pc+opCapTermSizeInBytes, append([]StackFrame{}, stack...), predicate)
		case opCapNonTerm:
			addItem(pc+opCapNonTermSizeInBytes, append([]StackFrame{}, stack...), predicate)
		case opCapTermBeginOffset:
			addItem(pc+opCapTermBeginOffsetSizeInBytes, append([]StackFrame{}, stack...), predicate)
		case opCapNonTermBeginOffset:
			addItem(pc+opCapNonTermBeginOffsetSizeInBytes, append([]StackFrame{}, stack...), predicate)
		case opCapEndOffset:
			addItem(pc+opCapEndOffsetSizeInBytes, append([]StackFrame{}, stack...), predicate)
		case opCapCommit:
			_, newStack := pop()
			addItem(int(decodeU16(code, pc+1)), newStack, predicate)
		case opCapBackCommit:
			_, newStack := pop()
			addItem(int(decodeU16(code, pc+1)), newStack, predicate)
		case opCapPartialCommit:
			// Same as opPartialCommit - explore both loop continuation and exit
			addItem(int(decodeU16(code, pc+1)), append([]StackFrame{}, stack...), predicate)
			tryBacktrack()
		}
	}
	return accepting, chars
}
