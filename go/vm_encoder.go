package langlang

import (
	"encoding/binary"
	"fmt"
)

func Encode(p *Program) *Bytecode {
	var (
		code   []byte
		cursor uint16
		labels = map[ILabel]uint16{}
	)
	for _, instruction := range p.code {
		switch ii := instruction.(type) {
		case ILabel:
			labels[ii] = cursor
		default:
			cursor += uint16(instruction.SizeInBytes())
		}
	}
	for _, instruction := range p.code {
		switch ii := instruction.(type) {
		case ILabel:
			// doesn't translate to anything
		case IHalt:
			code = append(code, opHalt)
			fmt.Printf("code: %#v\n", code)
		case IAny:
			code = append(code, opAny)
			fmt.Printf("code: %#v\n", code)
		case IChoice:
			code = encodeJmp(code, opChoice, labels[ii.Label])
			fmt.Printf("code: %#v\n", code)
		case ICommit:
			code = encodeJmp(code, opCommit, labels[ii.Label])
			fmt.Printf("code: %#v\n", code)
		case ICall:
			code = encodeJmp(code, opCall, labels[ii.Label])
			code = append(code, byte(ii.Precedence))
			fmt.Printf("code: %#v\n", code)
		case IReturn:
			code = append(code, opReturn)
			fmt.Printf("code: %#v\n", code)
		}
	}
	return &Bytecode{
		code: code,
	}
}

func encodeJmp(code []byte, op byte, label uint16) []byte {
	code = append(code, op)
	code = encodeU16(code, label)
	return code
}

var encodeU16 = binary.LittleEndian.AppendUint16
var decodeU16 = binary.LittleEndian.Uint16
