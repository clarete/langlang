package langlang

import (
	"encoding/binary"
	"fmt"
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

// lrMemoKey is the key for the left recursion memoization table.  It
// uniquely identifies a left-recursive call by production address and
// input position.
type lrMemoKey struct {
	address int
	cursor  int
}

// lrMemoEntry stores the memoized result of a left-recursive call.
type lrMemoEntry struct {
	// cursor is the result cursor position after matching, or
	// lrResultLeftRec if in initial left-recursive state
	cursor int
	// bound bookkeeps growth of the left-recursive match
	bound int
	// precedence is the precedence level of the successful match
	precedence int
	// captures holds the NodeIDs captured during the last
	// successful iteration
	captures []NodeID
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

const maxVars = 64

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
	lrmemo         map[lrMemoKey]*lrMemoEntry

	// Tree rewriting state (nil/zero when in parse mode)
	treeInput  *tree    // the tree being matched against
	treeCursor NodeID   // current position in the input tree
	treeNav    []NodeID // navigation stack for enter_child/pop_cursor
	vars       [maxVars]NodeID
	buildStack []NodeID // construction stack for build_* instructions
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
	opCallLR
	opReturnLR
	opCapReturnLR

	// Tree-rewriting opcodes (pattern matching against trees)
	opMatchNode    // assert tree cursor is NodeType_Node with given nameID (u16)
	opMatchString  // assert tree cursor is NodeType_String with text == strs[strID] (u16)
	opMatchAnyNode // wildcard: succeed on any node type
	opMatchSeq     // assert tree cursor is NodeType_Sequence
	opEnterChild   // push tree cursor, descend into NamedNode/ErrNode child
	opEnterIndex   // push tree cursor, descend into SeqNode child at index (u16)
	opPopCursor    // restore tree cursor from navigation stack

	// Tree-rewriting opcodes (variable binding)
	opBind      // save subtree at tree cursor into var slot (u16 varID)
	opCheckBind // assert current subtree equals var slot (u16 varID)

	// Tree-rewriting opcodes (construction)
	opBuildNode   // pop fieldCount children from build stack, create NamedNode (u16 nameID, u8 fieldCount)
	opBuildSeq    // pop count children from build stack, create SeqNode (u16 count)
	opBuildStr    // push a new StrNode from strs[strID] onto build stack (u16 strID)
	opBuildRef    // push var slot value onto build stack (u16 varID)
	opBuildCopy   // push current tree cursor subtree onto build stack

	// Tree-rewriting opcodes (traversal)
	opForEachChild // iterate over children, calling sub-program for each (u16 label)
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
	opCallLR:                "call_lr",
	opReturnLR:              "return_lr",
	opCapReturnLR:           "cap_return_lr",
	opMatchNode:             "match_node",
	opMatchString:           "match_string",
	opMatchAnyNode:          "match_any_node",
	opMatchSeq:              "match_seq",
	opEnterChild:            "enter_child",
	opEnterIndex:            "enter_index",
	opPopCursor:             "pop_cursor",
	opBind:                  "bind",
	opCheckBind:             "check_bind",
	opBuildNode:             "build_node",
	opBuildSeq:              "build_seq",
	opBuildStr:              "build_str",
	opBuildRef:              "build_ref",
	opBuildCopy:             "build_copy",
	opForEachChild:          "for_each_child",
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
	// opCallSizeInBytes: 1 opcode + 2 label + 1 precedence (backward compatible)
	// We keep 4 bytes even for non-LR calls to maintain ABI compatibility
	// with pre-compiled bytecode (e.g., bootstrap parser)
	opCallSizeInBytes   = 4
	opCallLRSizeInBytes = 4
	// opReturnSizeInBytes contains just one byte for the operator
	opReturnSizeInBytes = 1
	// opReturnLRSizeInBytes: same size, different behavior
	opReturnLRSizeInBytes              = 1
	opCapReturnLRSizeInBytes           = 1
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

	// Tree-rewriting instruction sizes
	opMatchNodeSizeInBytes    = 3 // 1 opcode + 2 nameID
	opMatchStringSizeInBytes  = 3 // 1 opcode + 2 strID
	opMatchAnyNodeSizeInBytes = 1
	opMatchSeqSizeInBytes     = 1
	opEnterChildSizeInBytes   = 1
	opEnterIndexSizeInBytes   = 3 // 1 opcode + 2 index
	opPopCursorSizeInBytes    = 1
	opBindSizeInBytes         = 3 // 1 opcode + 2 varID
	opCheckBindSizeInBytes    = 3 // 1 opcode + 2 varID
	opBuildNodeSizeInBytes    = 4 // 1 opcode + 2 nameID + 1 fieldCount
	opBuildSeqSizeInBytes     = 3 // 1 opcode + 2 count
	opBuildStrSizeInBytes     = 3 // 1 opcode + 2 strID
	opBuildRefSizeInBytes     = 3 // 1 opcode + 2 varID
	opBuildCopySizeInBytes    = 1
	opForEachChildSizeInBytes = 3 // 1 opcode + 2 label
)

func NewVirtualMachine(bytecode *Bytecode) *virtualMachine {
	tr := &tree{
		nodes:       make([]node, 0, 256),
		children:    make([]NodeID, 0, 512),
		childRanges: make([]struct{ start, end int32 }, 0, 256),
	}
	stk := &stack{
		frames:    make([]frame, 0, 256),
		nodeArena: make([]NodeID, 0, 256),
		nodes:     make([]NodeID, 0, 256),
		tree:      tr,
	}
	stk.tree.bindStrings(bytecode.strs)
	vm := &virtualMachine{
		stack:    stk,
		bytecode: bytecode,
		ffp:      -1,
	}
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

// Rewrite applies the rewrite bytecode (starting at PC=0) to the
// given tree rooted at rootID.  On success it returns the NodeID of
// the newly constructed tree (from the build stack).  New nodes are
// appended to the same arena as the input tree.
func (vm *virtualMachine) Rewrite(inputTree *tree, rootID NodeID) (NodeID, error) {
	vm.treeInput = inputTree
	vm.treeCursor = rootID
	vm.treeNav = vm.treeNav[:0]
	vm.buildStack = vm.buildStack[:0]

	// Point the output tree at the input tree so build_* opcodes
	// append to the same arena and NodeIDs from bind/copy are valid.
	vm.stack.tree = inputTree

	_, _, err := vm.MatchRule(inputTree.input, 0)
	if err != nil {
		return 0, err
	}
	if len(vm.buildStack) == 0 {
		return 0, fmt.Errorf("rewrite produced no output")
	}
	return vm.buildStack[len(vm.buildStack)-1], nil
}

// internStr ensures s is in t's string table and returns its index.
func (vm *virtualMachine) internStr(t *tree, s string) int32 {
	for i, existing := range t.strs {
		if existing == s {
			return int32(i)
		}
	}
	id := int32(len(t.strs))
	t.strs = append(t.strs, s)
	return id
}

func (vm *virtualMachine) MatchRule(data []byte, ruleAddress int) (Tree, int, error) {
	// dbg := func(m string) {}
	// dbg = func(m string) { fmt.Print(m) }

	// take a local reference of important data
	stack := vm.stack
	code := vm.bytecode.code
	sets := vm.bytecode.sets
	ilen := len(data)
	cursor := 0
	pc := 0

	// reset the vm state
	stack.reset()
	if vm.treeInput == nil {
		// Parse mode: reset the output tree
		stack.tree.reset()
		stack.tree.bindInput(data)
	}
	// In rewrite mode, stack.tree IS the input tree; don't reset it.
	vm.ffp = -1
	vm.ffpPC = 0
	if vm.showFails {
		vm.expected.clear()
	}
	if vm.lrmemo != nil {
		for k := range vm.lrmemo {
			delete(vm.lrmemo, k)
		}
	}

	// if a rule was received, push a call frame for it and set
	// the program appropriately
	if ruleAddress > 0 {
		stack.push(vm.mkCallFrame(opCallSizeInBytes))
		pc = ruleAddress
	}
code:
	for {
		op := code[pc]
		// dbg(fmt.Sprintf("in[c=%02d, pc=%02d]: 0x%x=%s\n", cursor, pc, op, opNames[op]))

		switch op {
		case opHalt:
			// dbg(fmt.Sprintf("nodes: %#v\n", vm.stack.nodes))
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
			if c < utf8.RuneSelf {
				cursor++
			} else {
				_, s := decodeRune(data, cursor)
				cursor += s
			}
			pc += opSetSizeInBytes

		case opSpan:
			sid := decodeU16(code, pc+1)
			set := sets[sid]
			for cursor < ilen {
				c := data[cursor]
				if set.hasByte(c) {
					if c < utf8.RuneSelf {
						cursor++
					} else {
						_, s := decodeRune(data, cursor)
						cursor += s
					}
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

		case opCallLR:
			var failed bool
			if pc, cursor, failed = vm.doCallLR(stack, pc, cursor); failed {
				goto fail
			}

		case opReturn:
			f := stack.pop()
			pc = int(f.pc)

		case opReturnLR:
			pc, cursor = vm.doReturnLR(stack, cursor)

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
			if len(stack.nodes) > 0 {
				idx := len(stack.nodes) - 1
				nid := stack.nodes[idx]
				stack.tree.SetRoot(nid)
			}
			return stack.tree, cursor, vm.mkErr(data, lb, cursor, vm.ffp)

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
			offset := int(decodeU16(code, pc+1))
			if offset > 0 {
				begin := cursor - offset
				nodeID := stack.tree.AddString(begin, cursor)
				stack.capture(nodeID)
			}
			pc += opCapTermSizeInBytes

		case opCapNonTerm:
			id := int(decodeU16(code, pc+1))
			offset := int(decodeU16(code, pc+3))
			if offset > 0 {
				begin := cursor - offset
				named := stack.tree.AddNamedString(int32(id), begin, cursor)
				stack.capture(named)
			}
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
			if offset > 0 {
				begin := cursor - offset
				if vm.capOffsetId < 0 {
					nodeID := stack.tree.AddString(begin, cursor)
					stack.capture(nodeID)
				} else {
					named := stack.tree.AddNamedString(int32(vm.capOffsetId), begin, cursor)
					stack.capture(named)
				}
			}

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

		case opCapReturnLR:
			pc, cursor = vm.doCapReturnLR(stack, cursor)

		// ---- Tree-rewriting: pattern matching ----

		case opMatchNode:
			nameID := int(binary.LittleEndian.Uint16(code[pc+1:]))
			pc += opMatchNodeSizeInBytes
			ti := vm.treeInput
			if ti.Type(vm.treeCursor) != NodeType_Node || ti.Name(vm.treeCursor) != vm.bytecode.strs[nameID] {
				goto fail
			}

		case opMatchString:
			strID := int(binary.LittleEndian.Uint16(code[pc+1:]))
			pc += opMatchStringSizeInBytes
			ti := vm.treeInput
			if ti.Type(vm.treeCursor) != NodeType_String || ti.Text(vm.treeCursor) != vm.bytecode.strs[strID] {
				goto fail
			}

		case opMatchAnyNode:
			pc += opMatchAnyNodeSizeInBytes

		case opMatchSeq:
			pc += opMatchSeqSizeInBytes
			if vm.treeInput.Type(vm.treeCursor) != NodeType_Sequence {
				goto fail
			}

		case opEnterChild:
			pc += opEnterChildSizeInBytes
			ti := vm.treeInput
			child, ok := ti.Child(vm.treeCursor)
			if !ok {
				goto fail
			}
			vm.treeNav = append(vm.treeNav, vm.treeCursor)
			vm.treeCursor = child

		case opEnterIndex:
			idx := int(binary.LittleEndian.Uint16(code[pc+1:]))
			pc += opEnterIndexSizeInBytes
			ti := vm.treeInput
			children := ti.Children(vm.treeCursor)
			if idx >= len(children) {
				goto fail
			}
			vm.treeNav = append(vm.treeNav, vm.treeCursor)
			vm.treeCursor = children[idx]

		case opPopCursor:
			pc += opPopCursorSizeInBytes
			n := len(vm.treeNav)
			vm.treeCursor = vm.treeNav[n-1]
			vm.treeNav = vm.treeNav[:n-1]

		case opBind:
			varID := int(binary.LittleEndian.Uint16(code[pc+1:]))
			pc += opBindSizeInBytes
			vm.vars[varID] = vm.treeCursor

		case opCheckBind:
			varID := int(binary.LittleEndian.Uint16(code[pc+1:]))
			pc += opCheckBindSizeInBytes
			if vm.vars[varID] != vm.treeCursor {
				goto fail
			}

		// ---- Tree-rewriting: construction ----

		case opBuildNode:
			nameID := int(binary.LittleEndian.Uint16(code[pc+1:]))
			fieldCount := int(code[pc+3])
			pc += opBuildNodeSizeInBytes
			outTree := stack.tree
			// Intern the name into the output tree's string table
			outNameID := vm.internStr(outTree, vm.bytecode.strs[nameID])
			n := len(vm.buildStack)
			var childID NodeID
			if fieldCount == 0 {
				childID = outTree.AddString(0, 0)
			} else if fieldCount == 1 {
				childID = vm.buildStack[n-1]
				vm.buildStack = vm.buildStack[:n-1]
			} else {
				children := make([]NodeID, fieldCount)
				copy(children, vm.buildStack[n-fieldCount:])
				vm.buildStack = vm.buildStack[:n-fieldCount]
				childID = outTree.AddSequence(children, 0, 0)
			}
			nodeID := outTree.AddNode(outNameID, childID, 0, 0)
			vm.buildStack = append(vm.buildStack, nodeID)

		case opBuildSeq:
			count := int(binary.LittleEndian.Uint16(code[pc+1:]))
			pc += opBuildSeqSizeInBytes
			outTree := stack.tree
			n := len(vm.buildStack)
			children := make([]NodeID, count)
			copy(children, vm.buildStack[n-count:])
			vm.buildStack = vm.buildStack[:n-count]
			seqID := outTree.AddSequence(children, 0, 0)
			vm.buildStack = append(vm.buildStack, seqID)

		case opBuildStr:
			strID := int(binary.LittleEndian.Uint16(code[pc+1:]))
			pc += opBuildStrSizeInBytes
			outTree := stack.tree
			outStrID := vm.internStr(outTree, vm.bytecode.strs[strID])
			nodeID := outTree.AddNamedString(outStrID, 0, 0)
			vm.buildStack = append(vm.buildStack, nodeID)

		case opBuildRef:
			varID := int(binary.LittleEndian.Uint16(code[pc+1:]))
			pc += opBuildRefSizeInBytes
			// In rewrite mode, input and output share the same arena,
			// so the bound NodeID is valid in the output tree.
			vm.buildStack = append(vm.buildStack, vm.vars[varID])

		case opBuildCopy:
			pc += opBuildCopySizeInBytes
			vm.buildStack = append(vm.buildStack, vm.treeCursor)

		case opForEachChild:
			label := int(binary.LittleEndian.Uint16(code[pc+1:]))
			pc += opForEachChildSizeInBytes
			ti := vm.treeInput
			children := ti.Children(vm.treeCursor)
			savedCursor := vm.treeCursor
			for _, child := range children {
				vm.treeCursor = child
				// Push a call frame so the sub-program can return
				stack.push(frame{
					pc: uint32(pc),
					t:  frameType_Call,
				})
				pc = label
				// The sub-program will execute and return via opReturn,
				// which pops the call frame and restores pc.
				// We need to break out and let the main loop handle it.
				// Store iteration state: the remaining children and
				// the saved cursor are managed by the sub-program.
			}
			vm.treeCursor = savedCursor
			// Note: for_each_child with the current loop structure
			// needs coroutine-like behavior. For now, it pushes the
			// first child call and the main loop handles return.
			// Full iteration is handled by the compiler emitting
			// explicit loops with enter_index.

		default:
			panic("NO ENTIENDO SENOR")
		}
	}

fail:
	if cursor > vm.ffp {
		vm.ffp = cursor
		vm.ffpPC = pc
	}

	// dbg(fmt.Sprintf("fl[c=%02d, pc=%02d]", cursor, pc))

	for stack.len() > 0 {
		f := stack.pop()
		stack.truncateArena(f.nodesStart)

		if f.t == frameType_Backtracking {
			pc = int(f.pc)
			vm.predicate = f.predicate
			cursor = f.cursor
			// dbg(fmt.Sprintf(" -> [c=%02d, pc=%02d]\n", cursor, pc))
			goto code
		}
		if f.t == frameType_LRCall {
			var gotoCode bool
			if pc, cursor, gotoCode = vm.doFailLR(stack, &f); gotoCode {
				goto code
			}
		}
	}

	// dbg(fmt.Sprintf(" -> boom: %d, %d\n", cursor, vm.ffp))

	if len(stack.nodes) > 0 {
		idx := len(stack.nodes) - 1
		nid := stack.nodes[idx]
		stack.tree.SetRoot(nid)
	}
	return stack.tree, cursor, vm.mkErr(data, 0, cursor, vm.ffp)
}

// Left-Recursion Helpers
//
// These are kept as separate //go:noinline methods so their machine
// code lives outside the MatchRule dispatch loop.  For non-LR
// grammars the opcodes are never emitted, so these are never called
// and never pollute the instruction cache.

//go:noinline
func (vm *virtualMachine) doCallLR(stack *stack, pc, cursor int) (int, int, bool) {
	var (
		code = vm.bytecode.code
		addr = int(decodeU16(code, pc+1))
		prec = int(code[pc+3])
		key  = lrMemoKey{address: addr, cursor: cursor}
	)
	if vm.lrmemo == nil {
		vm.lrmemo = make(map[lrMemoKey]*lrMemoEntry)
	}
	entry := vm.lrmemo[key]

	if entry == nil {
		// (lvar.1, lvar.2) First LR call at this position
		vm.lrmemo[key] = &lrMemoEntry{
			cursor:     lrResultLeftRec,
			bound:      0,
			precedence: prec,
			captures:   nil,
		}
		captureStart := uint32(len(stack.nodeArena))
		lrIdx := stack.pushLR(lrFrameData{
			address:      addr,
			precedence:   prec,
			result:       lrResultLeftRec,
			committedEnd: captureStart,
		})
		stack.push(frame{
			t:          frameType_LRCall,
			pc:         uint32(pc + opCallLRSizeInBytes),
			cursor:     cursor,
			lrIdx:      lrIdx,
			nodesStart: captureStart,
			nodesEnd:   captureStart,
		})
		return addr, cursor, false
	}

	if entry.cursor == lrResultLeftRec || prec < entry.precedence {
		// (lvar.3, lvar.5) In LR loop or precedence too low
		return 0, 0, true // fail
	}

	// (lvar.4) Use memoized result and inject captures
	for _, nodeID := range entry.captures {
		stack.capture(nodeID)
	}
	return pc + opCallLRSizeInBytes, entry.cursor, false
}

//go:noinline
func (vm *virtualMachine) doReturnLR(stack *stack, cursor int) (int, int) {
	var (
		f     = stack.top()
		lr    = stack.lr(f)
		key   = lrMemoKey{address: lr.address, cursor: f.cursor}
		entry = vm.lrmemo[key]
	)
	if lr.result == lrResultLeftRec || cursor > lr.result {
		// (inc.1) We grew the match, try again
		entry.cursor = cursor
		entry.bound++
		entry.precedence = lr.precedence
		lr.result = cursor
		return lr.address, f.cursor
	}
	// (inc.3) No more progress, finalize
	stack.pop()
	var (
		newCursor = lr.result
		newPC     = int(f.pc)
	)
	delete(vm.lrmemo, key)
	return newPC, newCursor
}

//go:noinline
func (vm *virtualMachine) doCapReturnLR(stack *stack, cursor int) (int, int) {
	var (
		f     = stack.top()
		lr    = stack.lr(f)
		key   = lrMemoKey{address: lr.address, cursor: f.cursor}
		entry = vm.lrmemo[key]
	)
	if lr.result == lrResultLeftRec || cursor > lr.result {
		// (inc.1) We grew the match, let's try again
		currentCaptures := make([]NodeID, f.nodesEnd-f.nodesStart)
		copy(currentCaptures, stack.nodeArena[f.nodesStart:f.nodesEnd])

		entry.captures = currentCaptures
		entry.cursor = cursor
		entry.bound++
		entry.precedence = lr.precedence

		stack.truncateArena(f.nodesStart)

		lr.result = cursor
		lr.committedEnd = f.nodesStart
		f.nodesEnd = f.nodesStart

		return lr.address, f.cursor
	}

	// (inc.3) No more progress, finalize
	stack.pop()
	stack.truncateArena(f.nodesStart)

	for _, nodeID := range entry.captures {
		stack.capture(nodeID)
	}
	var (
		newCursor = lr.result
		newPC     = int(f.pc)
	)
	delete(vm.lrmemo, key)
	return newPC, newCursor
}

// doFailLR handles a left-recursive frame during backtracking.  The
// caller has already truncated the arena to f.nodesStart.  Returns
// `(pc, cursor, gotoCode)`.  If `gotoCode` is true, the VM should
// resume execution; otherwise it continues popping frames.
//
//go:noinline
func (vm *virtualMachine) doFailLR(stack *stack, f *frame) (int, int, bool) {
	var (
		lr    = stack.lr(f)
		key   = lrMemoKey{address: lr.address, cursor: f.cursor}
		entry = vm.lrmemo[key]
	)
	if lr.result == lrResultLeftRec {
		// (lvar.2) First iteration failed, clean up memo
		delete(vm.lrmemo, key)
		return 0, 0, false
	}
	if lr.result > 0 && entry != nil {
		// (inc.2) Backtrack to previous successful iteration
		// Arena was already truncated by caller; re-inject
		// the captures from the last successful iteration.
		for _, nodeID := range entry.captures {
			stack.capture(nodeID)
		}
		delete(vm.lrmemo, key)
		return int(f.pc), lr.result, true
	}
	// Clean up and continue popping
	delete(vm.lrmemo, key)
	return 0, 0, false
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
	if errCursor >= len(data) {
		isEof = true
	} else {
		c, _ = decodeRune(data, errCursor)
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
