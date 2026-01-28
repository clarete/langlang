package langlang

// importOracleState loads an OracleState into the VM, preparing it for
// advancePrefix. This converts StackFrames to the VM's internal frame type.
func (vm *virtualMachine) importOracleState(s OracleState) {
	vm.stack.reset()
	vm.predicate = false

	for _, sf := range s.Stack {
		var t frameType
		switch sf.Type {
		case StackFrameBacktrack:
			t = frameType_Backtracking
		case StackFrameCall:
			t = frameType_Call
		case StackFrameCapture:
			t = frameType_Capture
		default:
			t = frameType_Backtracking
		}
		f := frame{
			pc:        uint32(sf.PC),
			cursor:    sf.Cursor,
			t:         t,
			predicate: sf.Predicate,
		}
		vm.stack.push(f)

		if sf.Predicate {
			vm.predicate = true
		}
	}
}

// exportOracleState extracts the VM state after advancePrefix into an OracleState.
// The baseCursor is added to the cursor to compute the total prefix length.
func (vm *virtualMachine) exportOracleState(pc, cursor, baseCursor int) OracleState {
	stack := make([]StackFrame, 0, vm.stack.len())
	for i := 0; i < vm.stack.len(); i++ {
		f := vm.stack.frames[i]
		var t StackFrameType
		switch f.t {
		case frameType_Backtracking:
			t = StackFrameBacktrack
		case frameType_Call:
			t = StackFrameCall
		case frameType_Capture:
			t = StackFrameCapture
		}
		stack = append(stack, StackFrame{
			PC:        int(f.pc),
			Cursor:    f.cursor,
			Type:      t,
			Predicate: f.predicate,
		})
	}
	return OracleState{
		PC:     pc,
		Cursor: baseCursor + cursor,
		Stack:  stack,
	}
}
