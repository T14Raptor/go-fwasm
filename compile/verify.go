package compile

import "fmt"

// verify checks the invariants the interpreter relies on instead of bounds
// checks: every slot an instruction names is inside the frame, every
// branch lands inside the code, and the code cannot run off its end. A
// failure is a bug in this package.
func (m *Module) verify(idx int, f *Func) error {
	frame := uint64(f.NumLocals + f.MaxHeight)
	n := uint64(len(f.Code))
	fail := func(pc int, format string, args ...any) error {
		return fmt.Errorf("func[%d]: internal error: bad bytecode at pc %d (%s): %s",
			idx, pc, f.Code[pc], fmt.Sprintf(format, args...))
	}
	inFrame := func(slot uint32, count uint64) bool { return uint64(slot)+count <= frame }
	for _, e := range f.BrTable {
		if uint64(e.PC) >= n || !inFrame(e.Src, uint64(e.N)) || !inFrame(e.Dst, uint64(e.N)) {
			return fmt.Errorf("func[%d]: internal error: bad branch target %+v", idx, e)
		}
	}
	if n == 0 {
		return fmt.Errorf("func[%d]: internal error: no code", idx)
	}
	switch f.Code[n-1].Op {
	case OpReturn, OpJump, OpBr, OpBrTable, OpUnreachable:
	default:
		return fail(int(n-1), "code can run off its end")
	}
	for pc, in := range f.Code {
		switch in.Op {
		case OpReturn:
			if !inFrame(in.A, uint64(len(f.Type.Results))) {
				return fail(pc, "results outside the frame")
			}
			continue
		case OpCall:
			if int(in.A) >= len(m.FuncTypes) || !inFrame(in.B, uint64(len(m.FuncTypes[in.A].Params))) {
				return fail(pc, "arguments outside the frame")
			}
			continue
		case OpCallIndirect:
			if int(in.A) >= len(m.Types) || uint64(in.C) < uint64(len(m.Types[in.A].Params)) || !inFrame(in.C, 1) {
				return fail(pc, "arguments outside the frame")
			}
			continue
		case OpBrIf:
			if uint64(in.A) >= uint64(len(f.BrTable)) || !inFrame(in.B, 1) {
				return fail(pc, "bad operands")
			}
			continue
		case OpBrTable:
			if in.C == 0 || uint64(in.A)+uint64(in.C) > uint64(len(f.BrTable)) || !inFrame(in.B, 1) {
				return fail(pc, "bad operands")
			}
			continue
		}
		info := opInfo[in.Op]
		if info&fKnown == 0 {
			return fail(pc, "unknown opcode")
		}
		if info&fRange != 0 {
			// The rare instructions that take a run of slots access
			// the frame with bounds checks; only OpBr is unchecked.
			if in.Op == OpBr && (!inFrame(in.B, uint64(in.D)) || !inFrame(in.C, uint64(in.D))) {
				return fail(pc, "moves outside the frame")
			}
			info &^= fB | fC
		}
		if info&(fA|fAW) != 0 && !inFrame(in.A, 1) || info&fB != 0 && !inFrame(in.B, 1) ||
			info&fC != 0 && !inFrame(in.C, 1) {
			return fail(pc, "slot outside the frame")
		}
		if info&fPC != 0 && uint64(in.A) >= n {
			return fail(pc, "branch outside the code")
		}
	}
	return nil
}
