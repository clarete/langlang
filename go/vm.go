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
	stack     *stack
	bytecode  *Bytecode
	predicate bool
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
		dbg(fmt.Sprintf("in[pc=%02d]: 0x%x=%s\n", vm.pc, op, opNames[op]))

		switch op {
		case opHalt:
			dbg(fmt.Sprintf("vals: %#v\n", vm.stack.values))
			var top Value
			if len(vm.stack.values) > 0 {
				top = vm.stack.values[len(vm.stack.values)-1]
			}
			return top, vm.cursor, nil

		case opAny:
			_, s, err := input.ReadRune()
			if err != nil {
				if err == io.EOF {
					goto fail
				}
				return nil, vm.cursor, err
			}
			vm.cursor += s
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
			vm.cursor += s
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
			vm.cursor += s
			vm.pc += opSpanSizeInBytes

		case opFail:
			goto fail

		case opFailTwice:
			vm.stack.pop()
			goto fail

		case opChoice:
			lb := int(decodeU16(vm.bytecode.code[vm.pc+1:]))
			vm.stack.pushBacktrack(lb, vm.cursor)
			vm.pc += opChoiceSizeInBytes

		case opChoicePred:
			lb := int(decodeU16(vm.bytecode.code[vm.pc+1:]))
			vm.stack.pushBacktrackPred(lb, vm.cursor)
			vm.pc += opChoiceSizeInBytes
			vm.predicate = true

		case opCommit:
			vm.stack.pop()
			vm.pc = int(decodeU16(vm.bytecode.code[vm.pc+1:]))

		case opPartialCommit:
			vm.stack.top().cursor = vm.cursor
			vm.pc = int(decodeU16(vm.bytecode.code[vm.pc+1:]))

		case opBackCommit:
			vm.cursor = vm.stack.pop().cursor
			vm.pc = int(decodeU16(vm.bytecode.code[vm.pc+1:]))

		case opCall:
			vm.stack.pushCall(vm.pc + opCallSizeInBytes)
			vm.pc = int(decodeU16(vm.bytecode.code[vm.pc+1:]))

		case opReturn:
			vm.pc = vm.stack.pop().pc

		case opCapBegin:
			id := int(decodeU16(vm.bytecode.code[vm.pc+1:]))
			vm.stack.push(frame{t: frameType_Capture, capId: id, cursor: vm.cursor})
			vm.pc += opCapBeginSizeInBytes

		case opCapEnd:
			frame := vm.stack.pop()
			capId := vm.bytecode.strs[frame.capId]
			value := newNode(input, capId, frame.cursor, vm.cursor)
			vm.stack.capture(value)
			vm.pc += opCapEndSizeInBytes

		default:
			panic("NO ENTIENDO SENOR")
		}
	}

fail:
	dbg(fmt.Sprintf("fl[pc=%d, cursor=%d]", vm.pc, vm.cursor))

	for vm.stack.len() > 0 {
		f := vm.stack.pop()
		if f.t == frameType_Backtracking {
			vm.pc = f.pc
			vm.cursor = f.cursor
			vm.predicate = f.predicate
			input.Seek(int64(vm.cursor), io.SeekStart)
			dbg(fmt.Sprintf(" -> [pc=%d, cursor=%d]\n", vm.pc, vm.cursor))
			goto code
		}
	}
	return nil, vm.cursor, nil
}

func newNode(input Input, name string, begin, end int) Value {
	if begin == end {
		return nil
	}

	span := NewSpan(NewLocation(0, 0, begin), NewLocation(0, 1, end))
	val, err := readSubstring(input, int64(begin), int64(end))
	if err != nil {
		panic(err.Error())
	}

	return NewNode(name, NewString(val, span), span)
}

func readSubstring(r io.ReaderAt, offset, length int64) (string, error) {
	sectionReader := io.NewSectionReader(r, offset, length)
	buffer := make([]byte, length)
	n, err := sectionReader.Read(buffer)
	if err != nil {
		return "", err
	}
	return string(buffer[:n]), nil
}
