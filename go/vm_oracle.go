package langlang

func (vm *virtualMachine) advancePrefix(data []byte, pc, cursor int) (int, int, bool) {

	var (
		stack = vm.stack
		code  = vm.bytecode.code
		sets  = vm.bytecode.sets
		ilen  = len(data)
	)

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
			return pc, cursor, true

		case opAny:
			if cursor >= ilen {
				return pc, cursor, true

			}
			_, s := decodeRune(data, cursor)
			cursor += s
			pc += opAnySizeInBytes

		case opChar:
			e := rune(decodeU16(code, pc+1))
			if cursor >= ilen {
				return pc, cursor, true

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
				return pc, cursor, true

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
				return pc, cursor, true

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
				return pc, cursor, true

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
				return pc, cursor, true

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
			matchedAny := false
			for cursor < ilen {
				c := data[cursor]
				if set.hasByte(c) {
					cursor++
					matchedAny = true
					continue
				}
				break
			}
			if cursor >= ilen && matchedAny {
				return pc, cursor, true
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
			return 0, 0, false

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
	return 0, 0, false

}
