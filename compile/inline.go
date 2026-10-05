package compile

// maxInline is the largest callee that gets inlined, counting its
// instructions plus the locals it must zero.
const maxInline = 32

// InlineSite is a call that was replaced by the callee's body.
type InlineSite struct {
	Start, End uint32 // the body's pcs in the caller
	Func       uint32 // the callee's function index
	PCs        []uint32
}

// CalleePC maps a pc in [Start, End) to the callee's own pc, in the
// callee's Code. That may be inside an InlineSite of the callee.
func (s *InlineSite) CalleePC(pc int) int { return int(s.PCs[pc-int(s.Start)]) }

// Operand flags.
const (
	fKnown = 1 << iota // the inliner knows the instruction's operands
	fA                 // A is a slot it reads
	fAW                // A is a slot it writes
	fB                 // B is a slot
	fC                 // C is a slot
	fPC                // A is a pc
	fDst               // all it does is write slot A
	fRange             // B is the first of several consecutive slots
)

var opInfo [numOps]uint8

func init() {
	set := func(f uint8, ops ...Op) {
		for _, op := range ops {
			opInfo[op] = fKnown | f
		}
	}
	set(0, OpUnreachable, OpLoopHeader, OpElemDrop, OpDataDrop)
	set(fPC, OpJump)
	set(fPC|fB|fC|fRange, OpBr)
	set(fPC|fB, OpBrIfNez, OpBrIfEqz)
	for op := OpBrIfI32Eq; op <= OpBrIfI32GeU; op++ {
		set(fPC|fB|fC, op)
	}
	for op := OpBrIfI32EqImm; op <= OpBrIfI32GeUImm; op++ {
		set(fPC|fB, op)
	}
	set(fA|fAW|fB|fC, OpSelect)
	set(fAW|fB|fDst, OpCopy, OpRefIsNull, OpMemoryGrow)
	set(fAW|fDst, OpConst, OpGlobalGet, OpRefFunc, OpMemorySize)
	set(fB, OpGlobalSet)
	set(fB|fRange, OpTableGet, OpTableSet, OpTableSize, OpTableGrow, OpTableFill, OpTableCopy, OpTableInit,
		OpMemoryInit, OpMemoryCopy, OpMemoryFill)
	for op := OpLoad8U; op <= OpI64Load32S; op++ {
		set(fAW|fB|fDst, op)
	}
	for op := OpLoad8UAdd; op <= OpLoad64Add; op++ {
		set(fAW|fB|fDst, op)
	}
	for op := OpStore8; op <= OpStore64; op++ {
		set(fA|fB, op)
	}
	for op := OpI32Eqz; op <= OpI64TruncSatF64U; op++ {
		switch {
		case isBinary(op):
			set(fAW|fB|fC|fDst, op)
		case isUnary(op):
			set(fAW|fB|fDst, op)
		}
	}
	for op := OpI32AddImm; op <= OpI32GeUImm; op++ {
		set(fAW|fB|fDst, op)
	}
}

// inlinable reports whether calls to g can be replaced by its body: it is
// small, calls nothing, and every operand is one the inliner can remap.
func inlinable(g *Func) bool {
	if len(g.Type.Results) > 1 || len(g.Code)+g.ZeroTo-g.ZeroFrom > maxInline {
		return false
	}
	for _, in := range g.Code {
		info := opInfo[in.Op]
		if in.Op == OpReturn {
			continue
		}
		if info&fKnown == 0 {
			return false
		}
		// Slot ranges are always operand stack slots; make sure, since
		// they are remapped as a whole.
		if info&fRange != 0 && (in.B < uint32(g.NumLocals) || in.Op == OpBr && in.C < uint32(g.NumLocals)) {
			return false
		}
	}
	return true
}

// writesSlot reports whether in may write slot s.
func writesSlot(in Instr, s uint32) bool {
	info := opInfo[in.Op]
	switch {
	case info&fAW != 0 && in.A == s:
		return true
	case in.Op == OpBr:
		return s >= in.C && s < in.C+uint32(in.D)
	case info&fRange != 0:
		return s >= in.B && s < in.B+3
	}
	return false
}

// branchTargets marks every pc that a branch in f can jump to.
func branchTargets(code []Instr, brTable []BrTarget) []bool {
	t := make([]bool, len(code)+1)
	for _, in := range code {
		if opInfo[in.Op]&fPC != 0 {
			t[in.A] = true
		}
	}
	for _, e := range brTable {
		t[e.PC] = true
	}
	return t
}

// inline replaces calls to small leaf functions with their bodies. It
// goes bottom-up through the call graph, so a function that only called
// such functions can be inlined in turn.
func (m *Module) inline() {
	done := make([]bool, len(m.Funcs))
	var visit func(i int)
	visit = func(i int) {
		done[i] = true
		for _, in := range m.Funcs[i].Code {
			if j := int(in.A) - m.NumImportedFuncs; in.Op == OpCall && j >= 0 && !done[j] {
				visit(j)
			}
		}
		if g := m.inlineCalls(m.Funcs[i]); g != nil {
			m.Funcs[i] = g
			g.Plain.setZero()
		}
		m.Funcs[i].fuse()
		m.Funcs[i].setZero()
	}
	for i := range m.Funcs {
		if !done[i] {
			visit(i)
		}
	}
}

// inlineCalls returns f with its calls to inlinable functions inlined, or
// nil if it has none.
func (m *Module) inlineCalls(f *Func) *Func {
	callee := func(in Instr) *Func {
		if in.Op != OpCall || int(in.A) < m.NumImportedFuncs {
			return nil
		}
		if g := m.Funcs[int(in.A)-m.NumImportedFuncs]; g != f && inlinable(g) {
			return g
		}
		return nil
	}
	found := false
	for _, in := range f.Code {
		if callee(in) != nil {
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	targets := branchTargets(f.Code, f.BrTable)
	out := &Func{Type: f.Type, NumLocals: f.NumLocals, MaxHeight: f.MaxHeight, LocalTypes: f.LocalTypes, Plain: f}
	code := make([]Instr, 0, len(f.Code)+len(f.Code)/2)
	newPC := make([]uint32, len(f.Code)+1)
	removed := make([]bool, len(f.Code))
	var jumps []int // instructions in code whose A is a pc in f

	// subs[pc] maps the parameters of the call at pc to caller slots that
	// already hold the arguments; the copies that put them in place are
	// removed.
	subs := map[int][]int{}
	for pc, in := range f.Code {
		g := callee(in)
		if g == nil {
			continue
		}
		np := uint32(len(g.Type.Params))
		sub := make([]int, np)
		for i := range sub {
			sub[i] = -1
		}
		base := in.B
		for k := pc - 1; k >= 0 && !targets[k+1]; k-- {
			arg := f.Code[k]
			if arg.Op != OpCopy && arg.Op != OpConst || arg.A < base || arg.A >= base+np {
				break
			}
			if arg.Op == OpCopy && arg.B >= base {
				break // reads a slot the call's frame covers
			}
			p := arg.A - base
			if arg.Op != OpCopy || sub[p] >= 0 || written(g, p) {
				continue
			}
			sub[p] = int(arg.B)
			removed[k] = true
		}
		subs[pc] = sub
	}

	for pc, in := range f.Code {
		newPC[pc] = uint32(len(code))
		if removed[pc] {
			continue
		}
		if g := callee(in); g != nil {
			code = m.inlineBody(out, code, g, in, subs[pc])
			continue
		}
		if opInfo[in.Op]&fPC != 0 {
			jumps = append(jumps, len(code))
		}
		code = append(code, in)
	}
	newPC[len(f.Code)] = uint32(len(code))
	for _, i := range jumps {
		code[i].A = newPC[code[i].A]
	}
	out.Code = code
	out.BrTable = make([]BrTarget, len(f.BrTable))
	for i, e := range f.BrTable {
		e.PC = newPC[e.PC]
		out.BrTable[i] = e
	}
	return out
}

// written reports whether g ever writes its parameter p.
func written(g *Func, p uint32) bool {
	for _, in := range g.Code {
		if writesSlot(in, p) {
			return true
		}
	}
	return false
}

// inlineBody appends g's body in place of call, whose arguments start at
// slot call.B. sub maps parameters to the caller slots holding them.
func (m *Module) inlineBody(f *Func, code []Instr, g *Func, call Instr, sub []int) []Instr {
	base := call.B
	np := uint32(len(g.Type.Params))
	slot := func(s uint32) uint32 {
		if s < np && sub[s] >= 0 {
			return uint32(sub[s])
		}
		return base + s
	}
	site := InlineSite{Start: uint32(len(code)), Func: call.A}
	for s := uint32(g.ZeroFrom); s < uint32(g.ZeroTo); s++ {
		code = append(code, Instr{Op: OpConst, A: base + s})
		site.PCs = append(site.PCs, 0)
	}
	targets := branchTargets(g.Code, nil)
	pcs := make([]uint32, len(g.Code)+1)
	var inner, ends []int
	for j, in := range g.Code {
		pcs[j] = uint32(len(code))
		if in.Op == OpReturn {
			last := j == len(g.Code)-1
			if len(g.Type.Results) == 1 {
				src := slot(in.A)
				prev := len(code) - 1
				switch {
				case src == base:
				case j > 0 && !targets[j] && opInfo[g.Code[j-1].Op]&fDst != 0 && code[prev].A == src:
					// Have the instruction that computed the result
					// write it where the caller expects it.
					code[prev].A = base
				default:
					code = append(code, Instr{Op: OpCopy, A: base, B: src})
					site.PCs = append(site.PCs, uint32(j))
				}
			}
			if !last {
				ends = append(ends, len(code))
				code = append(code, Instr{Op: OpJump})
				site.PCs = append(site.PCs, uint32(j))
			}
			continue
		}
		info := opInfo[in.Op]
		if info&(fA|fAW) != 0 {
			in.A = slot(in.A)
		}
		if info&fB != 0 {
			in.B = slot(in.B)
		}
		if info&fC != 0 {
			in.C = slot(in.C)
		}
		if info&fPC != 0 {
			inner = append(inner, len(code))
		}
		code = append(code, in)
		site.PCs = append(site.PCs, uint32(j))
	}
	pcs[len(g.Code)] = uint32(len(code))
	for _, i := range inner {
		code[i].A = pcs[code[i].A]
	}
	for _, i := range ends {
		code[i].A = uint32(len(code))
	}
	site.End = uint32(len(code))
	f.Inlined = append(f.Inlined, site)
	f.MaxHeight = max(f.MaxHeight, int(base)+g.NumLocals+g.MaxHeight-f.NumLocals)
	return code
}
