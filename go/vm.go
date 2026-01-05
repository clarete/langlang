package langlang

import (
	"strings"
	"unicode/utf8"
)

type Bytecode struct {
	code []byte
	strs []string
	sets []charset
	sexp [][]expected
	smap map[string]int
	rxps map[int]int
	rxbs bitset512
	srcm *SourceMap
}

func (b *Bytecode) CompileErrorLabels(labels map[string]string) map[int]int {
	if len(labels) == 0 {
		return nil
	}
	result := make(map[int]int, len(labels))
	for label, message := range labels {
		labelID, ok := b.smap[label]
		if !ok {
			continue
		}
		messageID, ok := b.smap[message]
		if !ok {
			messageID = len(b.strs)
			b.strs = append(b.strs, message)
			b.smap[message] = messageID
		}
		result[labelID] = messageID
	}
	return result
}

type bitset512 [8]uint64 // 64 bytes = 1 cache line

func (b *bitset512) Set(id int)      { b[id>>6] |= 1 << (id & 63) }
func (b *bitset512) Has(id int) bool { return b[id>>6]&(1<<(id&63)) != 0 }

type expected struct {
	a, b rune
}

const expectedLimit = 20

type expectedInfo struct {
	cur int
	arr [expectedLimit]expected
}

// add a new `expected` entry to the `expectedInfo` array.  Since the
// statically allocated array's N is ≤20, a linear scan is faster than
// map hashing.
func (e *expectedInfo) add(s expected) {
	if e.cur == expectedLimit {
		return
	}
	if s.b == 0 {
		switch s.a {
		case 0, ' ', '\n', '\r', '\t':
			return
		}
	}
	for i := 0; i < e.cur; i++ {
		if e.arr[i] == s {
			return
		}
	}
	e.arr[e.cur] = s
	e.cur++
}

func (e *expectedInfo) clear() {
	e.cur = 0
}

type virtualMachine struct {
	ffp            int
	ffpPC          int // bytecode PC at furthest failure point
	stack          *stack
	bytecode       *Bytecode
	predicate      bool
	expected       *expectedInfo
	showFails      bool
	errLabels      map[int]int
	capOffsetId    int
	capOffsetStart int
}

// NOTE: changing the order of these variants will break Bytecode ABI
const (
	opHalt byte = iota
	opAny
	opChar
	opRange
	opFail
	opFailTwice
	opChoice
	opChoicePred
	opCapCommit
	opCapPartialCommit
	opCapBackCommit
	opCall
	opCapReturn
	opJump
	opThrow
	opCapBegin
	opCapEnd
	opSet
	opSpan
	opCapTerm
	opCapNonTerm
	opCommit
	opBackCommit
	opPartialCommit
	opReturn
	opCapTermBeginOffset
	opCapNonTermBeginOffset
	opCapEndOffset
	opChar32
	opRange32
)

var opNames = map[byte]string{
	opHalt:                  "halt",
	opAny:                   "any",
	opChar:                  "char",
	opChar32:                "char32",
	opRange:                 "range",
	opRange32:               "range32",
	opSet:                   "set",
	opSpan:                  "span",
	opFail:                  "fail",
	opFailTwice:             "fail_twice",
	opChoice:                "choice",
	opChoicePred:            "choice_pred",
	opCommit:                "commit",
	opPartialCommit:         "partial_commit",
	opBackCommit:            "back_commit",
	opCapCommit:             "cap_commit",
	opCapBackCommit:         "cap_back_commit",
	opCapPartialCommit:      "cap_partial_commit",
	opCapReturn:             "cap_return",
	opCall:                  "call",
	opReturn:                "return",
	opJump:                  "jump",
	opThrow:                 "throw",
	opCapBegin:              "cap_begin",
	opCapEnd:                "cap_end",
	opCapTerm:               "cap_term",
	opCapNonTerm:            "cap_non_term",
	opCapTermBeginOffset:    "cap_term_begin_offset",
	opCapNonTermBeginOffset: "cap_non_term_begin_offset",
	opCapEndOffset:          "cap_end_offset",
}

var (
	// opAnySizeInBytes: 1 because `Any` has no params
	opAnySizeInBytes = 1
	// opCharSizeInBytes: 3, 1 for the opcode and 2 for the
	// literal char.
	opCharSizeInBytes = 3
	// opChar32SizeInBytes: 1 for opcode, 4 for uint32 rune
	opChar32SizeInBytes = 5
	// opRangeSizeInBytes: 1 for the opcode followed by two runes,
	// each one 2 bytes long.
	opRangeSizeInBytes = 5
	// opRange32SizeInBytes: 1 for opcode, 8 for two uint32 runes
	opRange32SizeInBytes = 9
	// opSetSizeInBytes: 1 for the opcode, followed by the 16bit
	// set address
	opSetSizeInBytes  = 3
	opSpanSizeInBytes = 3
	// opChoiceSizeInBytes is 3, 1 for the opcode, and 2 for the
	// label that the VM should go when it backtracks
	opChoiceSizeInBytes = 3
	opCommitSizeInBytes = 3
	opFailSizeInBytes   = 1
	// opCallSizeInBytes contains the following bytes
	//  1. operator
	//  2. low nib of 16bit uint label address
	//  3. high nib of 16bit uint label address
	//  4. uint8 precedence level
	opCallSizeInBytes = 4
	// opReturnSizeInBytes contains just one byte for the operator
	opReturnSizeInBytes                = 1
	opJumpSizeInBytes                  = 3
	opThrowSizeInBytes                 = 3
	opHaltSizeInBytes                  = 1
	opCapBeginSizeInBytes              = 3
	opCapEndSizeInBytes                = 1
	opCapTermSizeInBytes               = 3
	opCapNonTermSizeInBytes            = 5
	opCapTermBeginOffsetSizeInBytes    = 1
	opCapNonTermBeginOffsetSizeInBytes = 3
	opCapEndOffsetSizeInBytes          = 1
)

func NewVirtualMachine(bytecode *Bytecode) *virtualMachine {
	tr := &tree{
		nodes:       make([]node, 0, 256),
		children:    make([]NodeID, 0, 512),
		childRanges: make([]struct{ start, end int32 }, 0, 256),
	}
	stk := &stack{
		frames:    make([]frame, 0, 256),
		nodes:     make([]NodeID, 0, 256),
		nodeArena: make([]NodeID, 0, 256),
		tree:      tr,
	}
	stk.tree.bindStrings(bytecode.strs)
	vm := &virtualMachine{stack: stk, bytecode: bytecode, ffp: -1}
	return vm
}

func (vm *virtualMachine) SetShowFails(showFails bool) {
	if showFails {
		vm.expected = &expectedInfo{}
		vm.showFails = true
		return
	}
	vm.showFails = false
}

func (vm *virtualMachine) SetLabelMessages(labels map[int]int) {
	vm.errLabels = labels
}

func (vm *virtualMachine) SourceMap() *SourceMap {
	return vm.bytecode.srcm
}

func (vm *virtualMachine) Match(data []byte) (Tree, int, error) {
	return vm.MatchRule(data, 0)
}

func (vm *virtualMachine) MatchRule(data []byte, ruleAddress int) (Tree, int, error) {
	vm.reset()
	vm.stack.tree.bindInput(data)

	var (
		pc     = 0
		cursor = 0
		stack  = vm.stack
		code   = vm.bytecode.code
		sets   = vm.bytecode.sets
		ilen   = len(data)
	)

	if ruleAddress > 0 {
		stack.push(vm.mkCallFrame(opCallSizeInBytes))
		pc = ruleAddress
	}
code:
	for {
		op := code[pc]

		switch op {
		case opHalt:
			if len(stack.nodes) > 0 {
				idx := len(stack.nodes) - 1
				nid := stack.nodes[idx]
				stack.tree.SetRoot(nid)
			}
			return stack.tree, cursor, nil

		case opAny:
			if cursor >= ilen {
				goto fail
			}
			_, s := decodeRune(data, cursor)
			cursor += s
			pc += opAnySizeInBytes

		case opChar:
			e := rune(decodeU16(code, pc+1))
			if cursor >= ilen {
				goto fail
			}
			c, s := decodeRune(data, cursor)
			if c != e {
				if vm.showFails {
					vm.updateExpected(cursor, expected{a: e})
				}
				goto fail
			}
			cursor += s
			pc += opCharSizeInBytes

		case opChar32:
			e := rune(decodeU32(code, pc+1))
			if cursor >= ilen {
				goto fail
			}
			c, s := decodeRune(data, cursor)
			if c != e {
				if vm.showFails {
					vm.updateExpected(cursor, expected{a: e})
				}
				goto fail
			}
			cursor += s
			pc += opChar32SizeInBytes

		case opRange:
			if cursor >= ilen {
				goto fail
			}
			c, s := decodeRune(data, cursor)
			a := rune(decodeU16(code, pc+1))
			b := rune(decodeU16(code, pc+3))
			if c < a || c > b {
				if vm.showFails {
					vm.updateExpected(cursor, expected{a: a, b: b})
				}
				goto fail
			}
			cursor += s
			pc += opRangeSizeInBytes

		case opRange32:
			if cursor >= ilen {
				goto fail
			}
			c, s := decodeRune(data, cursor)
			a := rune(decodeU32(code, pc+1))
			b := rune(decodeU32(code, pc+5))
			if c < a || c > b {
				if vm.showFails {
					vm.updateExpected(cursor, expected{a: a, b: b})
				}
				goto fail
			}
			cursor += s
			pc += opRange32SizeInBytes

		case opSet:
			if cursor >= ilen {
				goto fail
			}
			c := data[cursor]
			i := decodeU16(code, pc+1)
			if !sets[i].hasByte(c) {
				if vm.showFails {
					vm.updateSetExpected(cursor, i)
				}
				goto fail
			}
			cursor++
			pc += opSetSizeInBytes

		case opSpan:
			sid := decodeU16(code, pc+1)
			set := sets[sid]
			for cursor < ilen {
				c := data[cursor]
				if set.hasByte(c) {
					cursor++
					continue
				}
				break
			}
			pc += opSetSizeInBytes

		case opFail:
			goto fail

		case opFailTwice:
			stack.pop()
			goto fail

		case opChoice:
			lb := int(decodeU16(code, pc+1))
			stack.push(mkBacktrackFrame(lb, cursor))
			pc += opChoiceSizeInBytes

		case opChoicePred:
			lb := int(decodeU16(code, pc+1))
			stack.push(mkBacktrackPredFrame(lb, cursor))
			pc += opChoiceSizeInBytes
			vm.predicate = true

		case opCommit:
			stack.pop()
			pc = int(decodeU16(code, pc+1))

		case opBackCommit:
			cursor = stack.pop().cursor
			pc = int(decodeU16(code, pc+1))

		case opPartialCommit:
			pc = int(decodeU16(code, pc+1))
			stack.top().cursor = cursor

		case opCall:
			stack.push(vm.mkCallFrame(pc + opCallSizeInBytes))
			pc = int(decodeU16(code, pc+1))

		case opReturn:
			f := stack.pop()
			pc = int(f.pc)

		case opJump:
			pc = int(decodeU16(code, pc+1))

		case opThrow:
			if vm.predicate {
				pc += opThrowSizeInBytes
				goto fail
			}
			lb := int(decodeU16(code, pc+1))
			if addr, ok := vm.bytecode.rxps[lb]; ok {
				stack.push(vm.mkCallFrame(pc + opThrowSizeInBytes))
				pc = addr
				continue
			}
			return nil, cursor, vm.mkErr(data, lb, cursor, vm.ffp)

		case opCapBegin:
			id := int(decodeU16(code, pc+1))
			stack.push(vm.mkCaptureFrame(id, cursor))
			pc += opCapBeginSizeInBytes

		case opCapEnd:
			f := stack.pop()
			nodes := stack.frameNodes(&f)
			stack.truncateArena(f.nodesStart)
			vm.newNode(cursor, f, nodes)
			pc += opCapEndSizeInBytes

		case opCapTerm:
			vm.newTermNode(cursor, int(decodeU16(code, pc+1)))
			pc += opCapTermSizeInBytes

		case opCapNonTerm:
			id := int(decodeU16(code, pc+1))
			offset := int(decodeU16(code, pc+3))
			vm.newNonTermNode(id, cursor, offset)
			pc += opCapNonTermSizeInBytes

		case opCapTermBeginOffset:
			vm.capOffsetId = -1
			vm.capOffsetStart = cursor
			pc += opCapTermBeginOffsetSizeInBytes

		case opCapNonTermBeginOffset:
			vm.capOffsetId = int(decodeU16(code, pc+1))
			vm.capOffsetStart = cursor
			pc += opCapNonTermBeginOffsetSizeInBytes

		case opCapEndOffset:
			offset := cursor - vm.capOffsetStart
			pc += opCapEndOffsetSizeInBytes
			if vm.capOffsetId < 0 {
				vm.newTermNode(cursor, offset)
				continue
			}
			vm.newNonTermNode(vm.capOffsetId, cursor, offset)

		case opCapCommit:
			stack.popAndCapture()
			pc = int(decodeU16(code, pc+1))

		case opCapBackCommit:
			f := stack.popAndCapture()
			cursor = f.cursor
			pc = int(decodeU16(code, pc+1))

		case opCapPartialCommit:
			pc = int(decodeU16(code, pc+1))
			top := stack.top()
			top.cursor = cursor
			stack.collectCaptures()
			// Reset this frame's capture range for the
			// next iteration
			top.nodesStart = uint32(len(stack.nodeArena))
			top.nodesEnd = top.nodesStart

		case opCapReturn:
			f := stack.popAndCapture()
			pc = int(f.pc)

		default:
			panic("NO ENTIENDO SENOR")
		}
	}

fail:
	if cursor > vm.ffp {
		vm.ffp = cursor
		vm.ffpPC = pc
	}

	for stack.len() > 0 {
		f := stack.pop()
		stack.truncateArena(f.nodesStart)
		if f.t == frameType_Backtracking {
			pc = int(f.pc)
			vm.predicate = f.predicate
			cursor = f.cursor
			goto code
		}
	}

	if len(stack.nodes) > 0 {
		idx := len(stack.nodes) - 1
		nid := stack.nodes[idx]
		stack.tree.SetRoot(nid)
	}
	return stack.tree, cursor, vm.mkErr(data, 0, cursor, vm.ffp)
}

// Helpers

func (vm *virtualMachine) reset() {
	vm.stack.reset()
	vm.stack.tree.reset()
	vm.ffp = -1
	vm.ffpPC = 0

	if vm.showFails {
		vm.expected.clear()
	}
}

// Stack Management Helpers

func mkBacktrackFrame(pc, cursor int) frame {
	return frame{
		t:      frameType_Backtracking,
		pc:     uint32(pc),
		cursor: cursor,
	}
}

func mkBacktrackPredFrame(pc, cursor int) frame {
	f := mkBacktrackFrame(pc, cursor)
	f.predicate = true
	return f
}

func (vm *virtualMachine) mkCaptureFrame(id, cursor int) frame {
	return frame{
		t:      frameType_Capture,
		capId:  uint32(id),
		cursor: cursor,
	}
}

func (vm *virtualMachine) mkCallFrame(pc int) frame {
	return frame{t: frameType_Call, pc: uint32(pc)}
}

// Node Capture Helpers

func (vm *virtualMachine) newTermNode(cursor, offset int) {
	if offset > 0 {
		begin := cursor - offset
		nodeID := vm.stack.tree.AddString(begin, cursor)
		vm.stack.capture(nodeID)
	}
}

func (vm *virtualMachine) newNonTermNode(capId, cursor, offset int) {
	if offset > 0 {
		begin := cursor - offset
		named := vm.stack.tree.AddNamedString(int32(capId), begin, cursor)
		vm.stack.capture(named)
	}
}

func (vm *virtualMachine) newNode(cursor int, f frame, nodes []NodeID) {
	var (
		nodeID  NodeID
		hasNode = false
		isrxp   = vm.bytecode.rxbs.Has(int(f.capId))
		capId   = int32(f.capId)
		start   = f.cursor
		end     = cursor
	)
	switch len(nodes) {
	case 0:
		if cursor-f.cursor > 0 {
			nodeID = vm.stack.tree.AddString(start, end)
			hasNode = true
		} else if !isrxp {
			// Only return early if this is NOT an error recovery expression
			// Error recovery expressions need to create Error nodes even when empty
			return
		}
		// If isrxp and nothing captured, hasNode remains false
	case 1:
		nodeID = nodes[0]
		hasNode = true
	default:
		nodeID = vm.stack.tree.AddSequence(nodes, start, end)
		hasNode = true
	}

	// This is a capture of an error recovery expression, so we
	// need to wrap the captured node (even if it is invalid) around
	// an Error.
	if isrxp {
		msgID, ok := vm.errLabels[int(f.capId)]
		if !ok {
			msgID = int(f.capId)
		}
		var errNode NodeID
		if hasNode {
			errNode = vm.stack.tree.AddErrorWithChild(capId, int32(msgID), nodeID, start, end)
		} else {
			errNode = vm.stack.tree.AddError(capId, int32(msgID), start, end)
		}
		vm.stack.capture(errNode)
		return
	}

	// If nothing has been captured up until now, it means that
	// it's a leaf node in the syntax tree, and the cursor didn't
	// move, so we can bail earlier.
	if !hasNode {
		return
	}

	// if the capture ID is empty, it means that it is an inner
	// expression capture.
	if f.capId == 0 {
		vm.stack.capture(nodeID)
		return
	}

	// This is a named capture.  The `AddCaptures` step of the
	// Grammar Compiler wraps the expression within `Definition`
	// nodes with `Capture` nodes named after the definition.
	named := vm.stack.tree.AddNode(capId, nodeID, start, end)
	vm.stack.capture(named)
}

// Error Handling Helpers

func (vm *virtualMachine) updateExpected(cursor int, s expected) {
	shouldClear := cursor > vm.ffp
	shouldAdd := cursor >= vm.ffp

	if shouldClear {
		vm.expected.clear()
	}
	if shouldAdd {
		vm.expected.add(s)
	}
}

func (vm *virtualMachine) updateSetExpected(cursor int, sid uint16) {
	shouldClear := cursor > vm.ffp
	shouldAdd := cursor >= vm.ffp

	if shouldClear {
		vm.expected.clear()
	}
	if shouldAdd {
		for i, item := range vm.bytecode.sexp[sid] {
			vm.expected.add(item)
			if i > expectedLimit-1 {
				break
			}
		}
	}
}

func (vm *virtualMachine) mkErr(data []byte, errLabelID int, cursor, errCursor int) error {
	// at this point, the input cursor should be at vm.ffp, so we
	// try to read the unexpected value to add it to the err
	// message.  Right now, we read just a single char from ffp's
	// location but it would be nice to read a full "word" at some
	// point.
	var (
		isEof   bool
		message strings.Builder
		c       rune
	)
	if cursor >= len(data) {
		isEof = true
	} else {
		c, _ = decodeRune(data, cursor)
	}
	if msgID, ok := vm.errLabels[errLabelID]; ok {
		// If an error message has been associated with the
		// error label, we just use the message.
		message.WriteString(vm.bytecode.strs[msgID])
	} else {
		// Prefix message with the error label if available
		if errLabelID > 0 {
			errLabel := vm.bytecode.strs[errLabelID]
			message.WriteRune('[')
			message.WriteString(errLabel)
			message.WriteRune(']')
			message.WriteRune(' ')
		}
		// Use information automatically collected by
		// `opChar`, `opRange`, and `opAny` fail and they're not
		// within predicates.
		if vm.showFails && vm.expected.cur > 0 {
			message.WriteString("Expected ")
			for i := 0; i < vm.expected.cur; i++ {
				e := vm.expected.arr[i]
				message.WriteRune('\'')
				message.WriteRune(e.a)
				if e.b != 0 {
					message.WriteRune('-')
					message.WriteRune(e.b)
				}
				message.WriteRune('\'')
				if i < vm.expected.cur-1 {
					message.WriteString(", ")
				}
			}
			message.WriteString(" but got ")
		} else {
			message.WriteString("Unexpected ")
		}
		if isEof {
			message.WriteString("EOF")
		} else {
			message.WriteRune('\'')
			message.WriteRune(c)
			message.WriteRune('\'')
		}
	}
	errLabel := ""
	if errLabelID > 0 {
		errLabel = vm.bytecode.strs[errLabelID]
	}
	var expected []ErrHint
	if vm.expected != nil && vm.expected.cur > 0 {
		expected = make([]ErrHint, vm.expected.cur)
		for i := 0; i < vm.expected.cur; i++ {
			ex := vm.expected.arr[i]
			switch {
			case ex.a == 0:
				expected[i] = ErrHint{Type: ErrHintType_EOF}
			case ex.a != 0 && ex.b == 0:
				expected[i] = ErrHint{
					Type: ErrHintType_Char,
					Char: ex.a,
				}
			case ex.a != 0 && ex.b != 0:
				expected[i] = ErrHint{
					Type:  ErrHintType_Range,
					Range: [2]rune{ex.a, ex.b},
				}
			default:
				expected[i] = ErrHint{Type: ErrHintType_Unknown}
			}
		}
	}
	return ParsingError{
		Message:  message.String(),
		Label:    errLabel,
		Start:    cursor,
		End:      errCursor,
		Expected: expected,
		FFPPC:    vm.ffpPC,
	}
}

// decodeU16 decodes a uint16 from byte array `b`. See
// https://github.com/golang/go/issues/14808
func decodeU16(code []byte, offset int) uint16 {
	return uint16(code[offset]) | uint16(code[offset+1])<<8
}

func decodeU32(code []byte, offset int) uint32 {
	return uint32(code[offset]) |
		uint32(code[offset+1])<<8 |
		uint32(code[offset+2])<<16 |
		uint32(code[offset+3])<<24
}

func decodeRune(data []byte, offset int) (rune, int) {
	if r := data[offset]; r < utf8.RuneSelf {
		return rune(r), 1
	}
	return utf8.DecodeRune(data[offset:])
}
