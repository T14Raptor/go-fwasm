package compile

// setZero works out which declared locals of f must be zeroed on entry: those
// that some path reads before writing. It sets f.ZeroFrom and f.ZeroTo to
// the smallest range of local slots covering them.
func (f *Func) setZero() {
	np, nl := len(f.Type.Params), f.NumLocals
	f.ZeroFrom, f.ZeroTo = np, nl
	if np == nl {
		return
	}
	// Track the first 64 declared locals; any after them are always
	// zeroed.
	tracked := min(nl-np, 64)
	all := uint64(1)<<tracked - 1 // all ones when tracked is 64

	// unset[pc] is the set of tracked locals that may not have been
	// written yet when pc runs.
	unset := make([]uint64, len(f.Code))
	seen := make([]bool, len(f.Code))
	unset[0], seen[0] = all, true
	work := []int{0}
	var reads uint64
	flow := func(pc int, set uint64) {
		if pc >= len(f.Code) {
			return
		}
		if !seen[pc] || unset[pc]|set != unset[pc] {
			seen[pc] = true
			unset[pc] |= set
			work = append(work, pc)
		}
	}
	for len(work) > 0 {
		pc := work[len(work)-1]
		work = work[:len(work)-1]
		in := f.Code[pc]
		set := unset[pc]
		r, w, next, targets := f.access(in)
		reads |= r & set
		set &^= w
		if next {
			flow(pc+1, set)
		}
		for _, t := range targets {
			flow(int(t), set)
		}
	}
	if nl-np > tracked {
		// Untracked locals are zeroed; so are tracked ones read early.
		lo := np + tracked
		for i := tracked - 1; i >= 0; i-- {
			if reads&(1<<i) != 0 {
				lo = np + i
			}
		}
		f.ZeroFrom = lo
		return
	}
	if reads == 0 {
		f.ZeroFrom, f.ZeroTo = np, np
		return
	}
	lo, hi := -1, -1
	for i := 0; i < tracked; i++ {
		if reads&(1<<i) != 0 {
			if lo < 0 {
				lo = i
			}
			hi = i
		}
	}
	f.ZeroFrom, f.ZeroTo = np+lo, np+hi+1
}

// access returns the tracked local bits in reads and writes for in, whether
// it can fall through to the next instruction, and where else it can go.
// Instructions that only touch operand stack slots report nothing, since
// those are never locals.
func (f *Func) access(in Instr) (reads, writes uint64, next bool, targets []uint32) {
	np := len(f.Type.Params)
	tracked := min(f.NumLocals-np, 64)
	bit := func(s uint32) uint64 {
		if i := int(s) - np; i >= 0 && i < tracked {
			return 1 << i
		}
		return 0
	}
	span := func(s, n uint32) (b uint64) {
		for i := range n {
			b |= bit(s + i)
		}
		return b
	}
	next = true
	switch in.Op {
	case OpReturn:
		return span(in.A, uint32(len(f.Type.Results))), 0, false, nil
	case OpUnreachable:
		return 0, 0, false, nil
	case OpCall, OpCallIndirect:
		// Arguments and results are operand stack slots.
		return 0, 0, true, nil
	case OpBrIf:
		e := f.BrTable[in.A]
		return bit(in.B) | span(e.Src, e.N), span(e.Dst, e.N), true, []uint32{e.PC}
	case OpBrTable:
		reads = bit(in.B)
		for _, e := range f.BrTable[in.A : in.A+in.C] {
			reads |= span(e.Src, e.N)
			writes |= span(e.Dst, e.N)
			targets = append(targets, e.PC)
		}
		return reads, writes, false, targets
	case OpBr:
		return span(in.B, uint32(in.D)), span(in.C, uint32(in.D)), false, []uint32{in.A}
	case OpJump:
		return 0, 0, false, []uint32{in.A}
	}
	info := opInfo[in.Op]
	if info&fRange != 0 {
		// table and bulk memory operands: a few consecutive slots
		return span(in.B, 3), span(in.B, 1), true, nil
	}
	if info&fA != 0 {
		reads |= bit(in.A)
	}
	if info&fB != 0 {
		reads |= bit(in.B)
	}
	if info&fC != 0 {
		reads |= bit(in.C)
	}
	if info&fAW != 0 {
		writes = bit(in.A)
		if in.Op == OpSelect {
			writes = 0 // only sometimes
		}
	}
	if info&fPC != 0 {
		targets = []uint32{in.A}
	}
	return reads, writes, true, targets
}
