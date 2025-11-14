package langlang

import "encoding/binary"

func Encode(p *Program) *Bytecode {
	var (
		code    []byte
		cursor  int
		labels  = map[ILabel]int{}
		rxps    = map[int]int{}
		setsMap = map[string]int{}
		sets    []charset
		sexp    [][]expected
		addSet  = func(cs *charset) uint16 {
			s := cs.encoded()
			if pos, ok := setsMap[s]; ok {
				return uint16(pos)
			}
			idx := len(sets)
			setsMap[s] = idx
			sets = append(sets, *cs)
			sexp = append(sexp, cs.precomputeExpectedSet())
			return uint16(idx)
		}
	)
	for _, instruction := range p.code {
		switch ii := instruction.(type) {
		case ILabel:
			labels[ii] = cursor
		default:
			cursor += instruction.SizeInBytes()
		}
	}
	for _, instruction := range p.code {
		switch ii := instruction.(type) {
		case ILabel:
			// doesn't translate to anything
		case IHalt:
			code = append(code, opHalt)
		case IAny:
			code = append(code, opAny)
		case IChar:
			code = append(code, opChar)
			code = encodeU16(code, uint16(ii.Char))
		case IRange:
			code = append(code, opRange)
			code = encodeU16(code, uint16(ii.Lo))
			code = encodeU16(code, uint16(ii.Hi))
		case ISet:
			code = append(code, opSet)
			code = encodeU16(code, addSet(ii.cs))
		case ISpan:
			code = append(code, opSpan)
			code = encodeU16(code, addSet(ii.cs))
		case IChoice:
			code = encodeJmp(code, opChoice, labels[ii.Label])
		case IChoicePred:
			code = encodeJmp(code, opChoicePred, labels[ii.Label])
		case ICommit:
			code = encodeJmp(code, opCommit, labels[ii.Label])
		case IPartialCommit:
			code = encodeJmp(code, opPartialCommit, labels[ii.Label])
		case IBackCommit:
			code = encodeJmp(code, opBackCommit, labels[ii.Label])
		case ICall:
			code = encodeJmp(code, opCall, labels[ii.Label])
			code = append(code, byte(ii.Precedence))
		case IReturn:
			code = append(code, opReturn)
		case IJump:
			code = encodeJmp(code, opJump, labels[ii.Label])
		case IFail:
			code = append(code, opFail)
		case IFailTwice:
			code = append(code, opFailTwice)
		case IThrow:
			code = append(code, opThrow)
			code = encodeU16(code, uint16(ii.ErrorLabel))
		case ICapBegin:
			code = append(code, opCapBegin)
			code = encodeU16(code, uint16(ii.ID))
		case ICapEnd:
			code = append(code, opCapEnd)
		case ICapTerm:
			code = append(code, opCapTerm)
			code = encodeU16(code, uint16(ii.Offset))
		case ICapNonTerm:
			code = append(code, opCapNonTerm)
			code = encodeU16(code, uint16(ii.ID))
			code = encodeU16(code, uint16(ii.Offset))
		}
	}
	for id, entry := range p.recovery {
		rxps[id] = labels[entry.label]
	}
	return &Bytecode{
		code: code,
		strs: p.strings,
		smap: p.stringsMap,
		rxps: rxps,
		sets: sets,
		sexp: sexp,
	}
}

func encodeJmp(code []byte, op byte, label int) []byte {
	code = append(code, op)
	code = encodeU16(code, uint16(label))
	return code
}

var encodeU16 = binary.LittleEndian.AppendUint16
