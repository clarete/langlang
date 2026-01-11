package langlang

import "encoding/binary"

func fitsU16Rune(r rune) bool { return r >= 0 && r <= 0xFFFF }

func Encode(p *Program, cfg *Config) *Bytecode {
	var (
		code    []byte
		cursor  int
		labels  = map[ILabel]int{}
		rxps    = map[int]int{}
		rxbs    = bitset512{}
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
			if fitsU16Rune(ii.Char) {
				code = append(code, opChar)
				code = encodeU16(code, uint16(ii.Char))
			} else {
				code = append(code, opChar32)
				code = encodeU32(code, uint32(ii.Char))
			}
		case IRange:
			if fitsU16Rune(ii.Lo) && fitsU16Rune(ii.Hi) {
				code = append(code, opRange)
				code = encodeU16(code, uint16(ii.Lo))
				code = encodeU16(code, uint16(ii.Hi))
			} else {
				code = append(code, opRange32)
				code = encodeU32(code, uint32(ii.Lo))
				code = encodeU32(code, uint32(ii.Hi))
			}
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
		case ICapCommit:
			code = encodeJmp(code, opCapCommit, labels[ii.Label])
		case IPartialCommit:
			code = encodeJmp(code, opPartialCommit, labels[ii.Label])
		case ICapPartialCommit:
			code = encodeJmp(code, opCapPartialCommit, labels[ii.Label])
		case IBackCommit:
			code = encodeJmp(code, opBackCommit, labels[ii.Label])
		case ICapBackCommit:
			code = encodeJmp(code, opCapBackCommit, labels[ii.Label])
		case ICall:
			code = encodeJmp(code, opCall, labels[ii.Label])
			code = append(code, byte(ii.Precedence))
		case IReturn:
			code = append(code, opReturn)
		case ICapReturn:
			code = append(code, opCapReturn)
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
		case ICapTermBeginOffset:
			code = append(code, opCapTermBeginOffset)
		case ICapNonTermBeginOffset:
			code = append(code, opCapNonTermBeginOffset)
			code = encodeU16(code, uint16(ii.ID))
		case ICapEndOffset:
			code = append(code, opCapEndOffset)
		}
	}
	for id, entry := range p.recovery {
		rxps[id] = labels[entry.label]
		rxbs.Set(id)
	}

	return &Bytecode{
		code: code,
		strs: p.strings,
		smap: p.stringsMap,
		rxps: rxps,
		rxbs: rxbs,
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
var encodeU32 = binary.LittleEndian.AppendUint32
