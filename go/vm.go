package langlang

import (
	"fmt"
	"io"
)

type Input interface {
	io.Seeker
	io.RuneReader
}

type Bytecode struct {
	code []byte
}

const (
	opHalt byte = iota
	opAny
	opLiteral
	opChoice
	opCommit
	opCall
	opReturn
)

var opNames = map[byte]string{
	opHalt:    "halt",
	opAny:     "any",
	opLiteral: "literal",
	opChoice:  "choice",
	opCommit:  "commit",
	opCall:    "call",
	opReturn:  "return",
}

type frame struct {
	pc     int
	cursor int
}

type stack []frame

func (bytecode *Bytecode) Match(input Input) (Value, int, error) {
	var (
		pc     = 0
		cursor = 0
		s      = &stack{}
		dbg    = func(m string) {}
	)

	dbg = func(m string) { fmt.Print(m) }

loop:
	for {
		dbg(fmt.Sprintf("pc: %d\n", pc))
		op := bytecode.code[pc]
		dbg(fmt.Sprintf("in: 0x%x=%s\n", op, opNames[op]))

		switch op {
		case opHalt:
			break loop

		case opAny:
			_, s, err := input.ReadRune()
			if err != nil {
				return nil, cursor, err
			}
			cursor += s
			pc += opAnySizeInBytes

		case opLiteral:
			pc++

		case opCall:
			pc++
			s.push(frame{pc: pc + 3})
			pc = int(decodeU16(bytecode.code[pc:]))

		case opReturn:
			f := s.pop()
			pc = f.pc
		}
	}
	return nil, cursor, nil
}

func (s *stack) push(f frame) {
	*s = append(*s, f)
}

func (s *stack) pop() frame {
	f := (*s)[len(*s)-1]
	*s = (*s)[len(*s)-1:]
	return f
}
