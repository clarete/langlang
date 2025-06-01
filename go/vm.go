package langlang

import (
	"fmt"
	"io"
)

type Input interface {
	io.Seeker
	io.RuneReader
	io.ReaderAt
}

type Bytecode struct {
	code []byte
	strs []string
}

func (b *Bytecode) Match(input Input) (Value, int, error) {
	vm := NewVirtualMachine(b)
	return vm.Match(input)
}

type VirtualMachine struct {
	pc        int
	cursor    int
	line      int
	column    int
	stack     *stack
	bytecode  *Bytecode
	predicate bool
	values    []Value
}

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
	opThrow:         "throw",
	opCapBegin:      "cap_begin",
	opCapEnd:        "cap_end",
}

func NewVirtualMachine(bytecode *Bytecode) *VirtualMachine {
	return &VirtualMachine{stack: &stack{}, bytecode: bytecode}
}

func (vm *VirtualMachine) Match(input Input) (Value, int, error) {
	dbg := func(m string) {}
	dbg = func(m string) { fmt.Print(m) }

code:
	for {
		op := vm.bytecode.code[vm.pc]
		dbg(fmt.Sprintf("in[c=%02d, pc=%02d]: 0x%x=%s\n", vm.cursor, vm.pc, op, opNames[op]))

		switch op {
		case opHalt:
			dbg(fmt.Sprintf("vals: %#v\n", vm.values))
			var top Value
			if len(vm.values) > 0 {
				top = vm.values[len(vm.values)-1]
			}
			return top, vm.cursor, nil

		case opAny:
			c, s, err := input.ReadRune()
			if err != nil {
				if err == io.EOF {
					goto fail
				}
				return nil, vm.cursor, err
			}
			vm.updatePos(c, s)
			vm.pc += opAnySizeInBytes

		case opChar:
			c, s, err := input.ReadRune()
			if err != nil {
				if err == io.EOF {
					goto fail
				}
				return nil, vm.cursor, err
			}
			if c != rune(decodeU16(vm.bytecode.code[vm.pc+1:])) {
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
			if c < rune(decodeU16(vm.bytecode.code[vm.pc+1:])) {
				goto fail
			}
			if c > rune(decodeU16(vm.bytecode.code[vm.pc+3:])) {
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

		case opBackCommit:
			vm.backtrackToFrame(vm.stack.pop())
			vm.pc = int(decodeU16(vm.bytecode.code[vm.pc+1:]))

		case opCall:
			vm.stack.push(frame{t: frameType_Call, pc: vm.pc + opCallSizeInBytes})
			vm.pc = int(decodeU16(vm.bytecode.code[vm.pc+1:]))

		case opReturn:
			vm.pc = vm.stack.pop().pc

		case opThrow:
			if vm.predicate {
				vm.pc += opThrowSizeInBytes
				goto fail
			} else {
				// TODO: Lookup recovery table
				lb := int(decodeU16(vm.bytecode.code[vm.pc+1:]))
				id := vm.bytecode.strs[lb]
				return nil, vm.cursor, fmt.Errorf("Labeled Fail: %s", id)
			}

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
	dbg(fmt.Sprintf("fl[c=%02d, pc=%02d]", vm.cursor, vm.pc))

	for vm.stack.len() > 0 {
		f := vm.stack.pop()
		if f.t == frameType_Backtracking {
			vm.pc = f.pc
			vm.predicate = f.predicate
			vm.backtrackToFrame(f)
			input.Seek(int64(vm.cursor), io.SeekStart)
			dbg(fmt.Sprintf(" -> [c=%02d, pc=%02d]\n", vm.cursor, vm.pc))
			goto code
		}
	}
	return nil, vm.cursor, fmt.Errorf("Fail")
}

func (vm *VirtualMachine) updatePos(c rune, s int) {
	vm.cursor += s
	vm.column++
	if c == '\n' {
		vm.column = 0
		vm.line++
	}
}

func (vm *VirtualMachine) backtrackToFrame(f frame) {
	vm.cursor = f.cursor
	vm.line = f.line
	vm.column = f.column
}

func (vm *VirtualMachine) mkBacktrackFrame(pc int) frame {
	return frame{
		t:      frameType_Backtracking,
		pc:     pc,
		cursor: vm.cursor,
		line:   vm.line,
		column: vm.column,
	}
}

func (vm *VirtualMachine) mkBacktrackPredFrame(pc int) frame {
	f := vm.mkBacktrackFrame(pc)
	f.predicate = true
	return f
}

func (vm *VirtualMachine) mkCaptureFrame(id int) frame {
	return frame{
		t:      frameType_Capture,
		capId:  id,
		cursor: vm.cursor,
		line:   vm.line,
		column: vm.column,
	}
}

func (vm *VirtualMachine) newNode(input Input, f frame) {
	var (
		capId = vm.bytecode.strs[f.capId]
		begin = NewLocation(f.line, f.column, f.cursor)
		end   = NewLocation(vm.line, vm.column, vm.cursor)
		span  = NewSpan(begin, end)
		read  = func(size, start int) string {
			val := make([]byte, size)
			if _, err := input.ReadAt(val, int64(start)); err != nil {
				panic(err.Error())
			}
			return string(val)
		}
	)

	if len(f.values) == 0 {
		if vm.cursor-f.cursor > 0 {
			text := read(vm.cursor-f.cursor, f.cursor)
			node := NewNode(capId, NewString(text, span), span)
			vm.capture(node)
		}
		return
	}

	// TODO: this should be moved to the compiler and made optional (within AddCaptures maybe?)
	out := make([]Value, 0, len(f.values))
	prev := NewLocation(f.line, f.column, f.cursor)
	for _, value := range f.values {
		loc := value.Span().Start
		if prev.Cursor < loc.Cursor {
			text := read(loc.Cursor-prev.Cursor, prev.Cursor)
			out = append(out, NewString(text, NewSpan(prev, loc)))
		}
		out = append(out, value)
		prev = value.Span().End
	}
	loc := NewLocation(vm.line, vm.column, vm.cursor)
	if prev.Cursor < loc.Cursor {
		text := read(loc.Cursor-prev.Cursor, prev.Cursor)
		out = append(out, NewString(text, NewSpan(prev, loc)))
	}

	switch len(out) {
	case 0:
	case 1:
		vm.capture(NewNode(capId, out[0], span))
	default:
		vm.capture(NewNode(capId, NewSequence(out, span), span))
	}
}

func (vm *VirtualMachine) capture(values ...Value) {
	if capFrame, ok := vm.stack.findCaptureFrame(); ok {
		capFrame.values = append(capFrame.values, values...)
		return
	}
	if len(vm.values) == 0 {
		vm.values = values
	} else {
		vm.values = append(vm.values, values...)
	}
}
