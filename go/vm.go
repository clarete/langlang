package langlang

import (
	"encoding/binary"
	"io"
	"strings"
)

type Input interface {
	io.Seeker
	io.RuneReader
	io.ReaderAt
}

type Bytecode struct {
	code []byte
	strs []string
	smap map[string]int
	rxps map[int]int
}

func (b *Bytecode) Match(input Input) (Value, int, error) {
	return b.MatchE(input, nil, nil)
}

func (b *Bytecode) MatchE(
	input Input,
	errLabels map[string]string,
	suppress map[int]struct{},
) (Value, int, error) {
	vm := newVirtualMachine(b, errLabels, suppress)
	return vm.Match(input)
}

type expected struct {
	a, b rune
}

const expectedLimit = 20

type expectedInfo struct {
	cur int
	arr [expectedLimit]expected
	set map[expected]struct{}
}

func newExpectedInfo() expectedInfo {
	return expectedInfo{
		set: make(map[expected]struct{}, expectedLimit),
	}
}

func (e *expectedInfo) add(s expected) {
	if e.cur == expectedLimit {
		return
	}
	e.set[s] = struct{}{}
	e.arr[e.cur] = s
	e.cur++
}

func (e *expectedInfo) clear() {
	e.cur = 0
	clear(e.set)
}

type virtualMachine struct {
	pc        int
	ffp       int
	cursor    int
	line      int
	column    int
	stack     *stack
	bytecode  *Bytecode
	predicate bool
	values    []Value
	expected  expectedInfo
	errLabels map[string]string
	supprset  map[int]struct{}
	suppress  int
}

// NOTE: changing the order of these variants will break Bytecode ABI
const (
	opHalt byte = iota
	opAny
	opChar
	opSpan
	opFail
	opFailTwice
	opChoice
	opChoicePred
	opCommit
	opPartialCommit
	opBackCommit
	opCall
	opReturn
	opJump
	opThrow
	opCapBegin
	opCapEnd
)

var opNames = map[byte]string{
	opHalt:          "halt",
	opAny:           "any",
	opChar:          "char",
	opSpan:          "span",
	opFail:          "fail",
	opFailTwice:     "fail_twice",
	opChoice:        "choice",
	opChoicePred:    "choice_pred",
	opCommit:        "commit",
	opPartialCommit: "partial_commit",
	opBackCommit:    "back_commit",
	opCall:          "call",
	opReturn:        "return",
	opJump:          "jump",
	opThrow:         "throw",
	opCapBegin:      "cap_begin",
	opCapEnd:        "cap_end",
}

var (
	// opAnySizeInBytes: 1 because `Any` has no params
	opAnySizeInBytes = 1
	// opCharSizeInBytes: 3, 1 for the opcode and 2 for the
	// literal char.  TODO: This is too small for certain chars.
	opCharSizeInBytes = 3
	// opSpanSizeInBytes: 1 for the opcode followed by two runes,
	// each one 2 bytes long.  TODO: This is too small for certain
	// chars.
	opSpanSizeInBytes = 5
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
	opReturnSizeInBytes   = 1
	opJumpSizeInBytes     = 3
	opThrowSizeInBytes    = 3
	opHaltSizeInBytes     = 1
	opCapBeginSizeInBytes = 3
	opCapEndSizeInBytes   = 1
)

func newVirtualMachine(
	bytecode *Bytecode,
	errLabels map[string]string,
	suppressSet map[int]struct{},
) *virtualMachine {
	return &virtualMachine{
		stack:     &stack{},
		bytecode:  bytecode,
		errLabels: errLabels,
		expected:  newExpectedInfo(),
		supprset:  suppressSet,
		ffp:       -1,
	}
}

func (vm *virtualMachine) Match(input Input) (Value, int, error) {
	// dbg := func(m string) {}
	// dbg = func(m string) { fmt.Print(m) }

code:
	for {
		op := vm.bytecode.code[vm.pc]
		// dbg(fmt.Sprintf("in[c=%02d, pc=%02d]: 0x%x=%s\n", vm.cursor, vm.pc, op, opNames[op]))

		switch op {
		case opHalt:
			// dbg(fmt.Sprintf("vals: %#v\n", vm.values))
			var top Value
			if len(vm.values) > 0 {
				top = vm.values[len(vm.values)-1]
			}
			return top, vm.cursor, nil

		case opAny:
			c, s, err := input.ReadRune()
			if err != nil {
				if err == io.EOF {
					vm.updateFFP(expected{})
					goto fail
				}
				return nil, vm.cursor, err
			}
			vm.updatePos(c, s)
			vm.pc += opAnySizeInBytes

		case opChar:
			e := rune(decodeU16(vm.bytecode.code[vm.pc+1:]))
			c, s, err := input.ReadRune()
			if err != nil {
				if err == io.EOF {
					goto fail
				}
				return nil, vm.cursor, err
			}
			if c != e {
				vm.updateFFP(expected{a: e})
				goto fail
			}
			vm.updatePos(c, s)
			vm.pc += opCharSizeInBytes

		case opSpan:
			c, s, err := input.ReadRune()
			if err != nil {
				if err == io.EOF {
					goto fail
				}
				return nil, vm.cursor, err
			}
			a := rune(decodeU16(vm.bytecode.code[vm.pc+1:]))
			b := rune(decodeU16(vm.bytecode.code[vm.pc+3:]))
			if c < a || c > b {
				vm.updateFFP(expected{a: a, b: b})
				goto fail
			}
			vm.updatePos(c, s)
			vm.pc += opSpanSizeInBytes

		case opFail:
			goto fail

		case opFailTwice:
			vm.stack.pop()
			goto fail

		case opChoice:
			lb := int(decodeU16(vm.bytecode.code[vm.pc+1:]))
			vm.stack.push(vm.mkBacktrackFrame(lb))
			vm.pc += opChoiceSizeInBytes

		case opChoicePred:
			lb := int(decodeU16(vm.bytecode.code[vm.pc+1:]))
			vm.stack.push(vm.mkBacktrackPredFrame(lb))
			vm.pc += opChoiceSizeInBytes
			vm.predicate = true

		case opCommit:
			vm.stack.pop()
			vm.pc = int(decodeU16(vm.bytecode.code[vm.pc+1:]))

		case opPartialCommit:
			vm.pc = int(decodeU16(vm.bytecode.code[vm.pc+1:]))
			top := vm.stack.top()
			top.cursor = vm.cursor
			top.line = vm.line
			top.column = vm.column
			top.captured = vm.numCapturedValues()

		case opBackCommit:
			vm.backtrackToFrame(input, vm.stack.pop())
			vm.pc = int(decodeU16(vm.bytecode.code[vm.pc+1:]))

		case opCall:
			vm.stack.push(vm.mkCallFrame(vm.pc + opCallSizeInBytes))
			vm.pc = int(decodeU16(vm.bytecode.code[vm.pc+1:]))

		case opReturn:
			vm.pc = vm.stack.pop().pc

		case opThrow:
			if vm.predicate {
				vm.pc += opThrowSizeInBytes
				goto fail
			}
			lb := int(decodeU16(vm.bytecode.code[vm.pc+1:]))
			id := vm.bytecode.strs[lb]
			if addr, ok := vm.bytecode.rxps[lb]; ok {
				vm.stack.push(vm.mkCallFrame(vm.pc + opThrowSizeInBytes))
				vm.pc = addr
				continue
			}
			return nil, vm.cursor, vm.mkErr(input, id)

		case opCapBegin:
			id := int(decodeU16(vm.bytecode.code[vm.pc+1:]))
			vm.stack.push(vm.mkCaptureFrame(id))
			vm.pc += opCapBeginSizeInBytes

		case opCapEnd:
			vm.newNode(input, vm.stack.pop())
			vm.pc += opCapEndSizeInBytes

		default:
			panic("NO ENTIENDO SENOR")
		}
	}

fail:
	// dbg(fmt.Sprintf("fl[c=%02d, pc=%02d]", vm.cursor, vm.pc))

	for vm.stack.len() > 0 {
		f := vm.stack.pop()
		if f.t == frameType_Backtracking {
			vm.pc = f.pc
			vm.predicate = f.predicate
			vm.backtrackToFrame(input, f)
			// dbg(fmt.Sprintf(" -> [c=%02d, pc=%02d]\n", vm.cursor, vm.pc))
			goto code
		}
	}
	// dbg(fmt.Sprintf(" -> boom: %d, %d\n", vm.cursor, vm.ffp))

	return nil, vm.cursor, vm.mkErr(input, "")
}

// Cursor/Line/Column Helpers

func (vm *virtualMachine) updatePos(c rune, s int) {
	vm.cursor += s
	vm.column++
	if c == '\n' {
		vm.column = 0
		vm.line++
	}
}

// Stack Management Helpers

func (vm *virtualMachine) backtrackToFrame(input Input, f frame) {
	vm.cursor = f.cursor
	vm.line = f.line
	vm.column = f.column
	vm.stack.dropUncommittedValues(f.captured)
	input.Seek(int64(vm.cursor), io.SeekStart)
}

func (vm *virtualMachine) mkBacktrackFrame(pc int) frame {
	return frame{
		t:        frameType_Backtracking,
		pc:       pc,
		cursor:   vm.cursor,
		line:     vm.line,
		column:   vm.column,
		captured: vm.numCapturedValues(),
	}
}

func (vm *virtualMachine) numCapturedValues() int {
	captured := 0
	if f, ok := vm.stack.findCaptureFrame(); ok {
		captured = len(f.values)
	}
	return captured
}

func (vm *virtualMachine) mkBacktrackPredFrame(pc int) frame {
	f := vm.mkBacktrackFrame(pc)
	f.predicate = true
	return f
}

func (vm *virtualMachine) mkCaptureFrame(id int) frame {
	// if either the capture ID has been disabled, or a capture ID
	// wrapping it is suppressed, so we put this flag here for
	// `opCapEnd` to pick it up and skip capturing the node.
	_, shouldSuppress := vm.supprset[id]
	if shouldSuppress {
		vm.suppress++
	}
	return frame{
		t:        frameType_Capture,
		capId:    id,
		cursor:   vm.cursor,
		line:     vm.line,
		column:   vm.column,
		suppress: shouldSuppress,
	}
}

func (vm *virtualMachine) mkCallFrame(pc int) frame {
	return frame{t: frameType_Call, pc: pc}
}

// Node Capture Helpers

func (vm *virtualMachine) newNode(input Input, f frame) {
	if f.suppress {
		vm.suppress--
		return
	}
	var (
		node     Value
		_, isrxp = vm.bytecode.rxps[f.capId]
		capId    = vm.bytecode.strs[f.capId]
		begin    = NewLocation(f.line, f.column, f.cursor)
		end      = NewLocation(vm.line, vm.column, vm.cursor)
		span     = NewSpan(begin, end)
	)
	switch len(f.values) {
	case 0:
		if vm.cursor-f.cursor > 0 {
			buff := make([]byte, vm.cursor-f.cursor)
			if _, err := input.ReadAt(buff, int64(f.cursor)); err != nil {
				panic(err.Error())
			}
			text := string(buff)
			node = NewString(text, span)
		}
	case 1:
		node = f.values[0]
	default:
		node = NewSequence(f.values, span)
	}

	// This is a capture of an error recovery expression, so we
	// need to wrap the captured node (even if it is nil) around
	// an Error.
	if isrxp {
		msg, ok := vm.errLabels[capId]
		if !ok {
			msg = capId
		}
		vm.capture(NewError(capId, msg, node, span))
		return
	}

	// If nothing has been captured up until now, it means that
	// it's a leaf node in the syntax tree, and the cursor didn't
	// move, so we can bail earlier.
	if node == nil {
		return
	}

	// if the capture ID is empty, it means that it is an inner
	// expression capture.
	if capId == "" {
		vm.capture(node)
		return
	}

	// This is a named capture.  The `AddCaptures` step of the
	// Grammar Compiler wraps the expression within `Definition`
	// nodes with `Capture` nodes named after the definition.
	vm.capture(NewNode(capId, node, span))
}

func (vm *virtualMachine) capture(values ...Value) {
	if capFrame, ok := vm.stack.findCaptureFrame(); ok {
		if len(capFrame.values) == 0 {
			capFrame.values = values
		} else {
			capFrame.values = append(capFrame.values, values...)
		}
		return
	}
	if len(vm.values) == 0 {
		vm.values = values
	} else {
		vm.values = append(vm.values, values...)
	}
}

// Error Handling Helpers

func (vm *virtualMachine) updateFFP(s expected) {
	if vm.cursor > vm.ffp {
		vm.ffp = vm.cursor
		vm.expected.clear()
		if _, ok := skipFromFFPUpdate[s]; ok || vm.predicate {
			return
		}
		vm.expected.add(s)
	} else if vm.cursor == vm.ffp {
		if _, ok := skipFromFFPUpdate[s]; ok || vm.predicate {
			return
		}
		if _, ok := vm.expected.set[s]; ok {
			return
		}
		vm.expected.add(s)
	}
}

var skipFromFFPUpdate = map[expected]struct{}{
	expected{}:        struct{}{},
	expected{a: ' '}:  struct{}{},
	expected{a: '\n'}: struct{}{},
	expected{a: '\r'}: struct{}{},
	expected{a: '\t'}: struct{}{},
}

func (vm *virtualMachine) mkErr(input Input, errLabel string) error {
	// First we seek back to where the cursor backtracked to, and
	// increment the information about line and column.
	input.Seek(int64(vm.cursor), io.SeekStart)
	line, column, cursor := vm.line, vm.column, vm.cursor

	for cursor < vm.ffp {
		c, s, err := input.ReadRune()
		if err != nil {
			break
		}

		cursor += s
		column++
		if c == '\n' {
			column = 0
			line++
		}
	}

	// at this point, the input cursor should be at vm.ffp, so we
	// try to read the unexpected value to add it to the err
	// message.  Right now, we read just a single char from ffp's
	// location but it would be nice to read a full "word" at some
	// point.
	var (
		isEof   bool
		message strings.Builder
		pos     = NewLocation(line, column, vm.ffp)
		span    = NewSpan(pos, pos)
	)
	c, _, err := input.ReadRune()
	if err != nil {
		if err == io.EOF {
			isEof = true
		} else {
			return err
		}
	}
	if m, ok := vm.errLabels[errLabel]; ok {
		// If an error message has been associated with the
		// error label, we just use the message.
		message.WriteString(m)
	} else {
		// Prefix message with the error label if available
		if errLabel != "" {
			message.WriteRune('[')
			message.WriteString(errLabel)
			message.WriteRune(']')
			message.WriteRune(' ')
		}
		// Use information automatically collected by
		// `opChar`, `opSpan`, and `opAny` fail and they're
		// not within predicates.
		if len(vm.expected.set) > 0 {
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
	return ParsingError{Message: message.String(), Label: errLabel, Span: span}
}

var decodeU16 = binary.LittleEndian.Uint16
var writeU16 = binary.LittleEndian.PutUint16
