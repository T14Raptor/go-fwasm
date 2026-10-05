package compile

// loadAdd gives the form of a load that adds a constant to its address
// first.
var loadAdd = [numOps]Op{
	OpLoad8U: OpLoad8UAdd, OpLoad16U: OpLoad16UAdd, OpLoad32: OpLoad32Add, OpLoad64: OpLoad64Add,
}

// fuse merges instruction pairs that lowering one instruction at a time,
// and inlining, leave next to each other:
//
//	r = i32.add_imm x k1;  r = i32.add_imm r k2   →  r = i32.add_imm x k1+k2
//	r = i32.add_imm x k;   r = loadN r+off        →  r = loadN_add x k off
//
// The second instruction overwrites r, so r's first value is dead, and it
// must not be a branch target. A load is not merged across the edge of
// inlined code, so a trap still shows in the right function.
//
// It also drops a copy whose source is overwritten right after it:
//
//	t = op …;  l = t;  t = op2 … t …   →   l = op …;  t = op2 … l …
func (f *Func) fuse() {
	targets := branchTargets(f.Code, f.BrTable)
	edge := make([]bool, len(f.Code)+1)
	for _, s := range f.Inlined {
		edge[s.Start], edge[s.End] = true, true
	}
	code := make([]Instr, 0, len(f.Code))
	newPC := make([]uint32, len(f.Code)+1)
	dropped := make([]bool, len(f.Code))
	var jumps []int
	for pc, in := range f.Code {
		newPC[pc] = uint32(len(code))
		if k := len(code) - 1; k >= 0 && !targets[pc] && in.A == in.B {
			p := &code[k]
			if p.Op == OpI32AddImm && p.A == in.B {
				switch {
				case in.Op == OpI32AddImm:
					p.C += in.C
					dropped[pc] = true
					continue
				case loadAdd[in.Op] != 0 && in.C <= 0xffff && !edge[pc]:
					*p = Instr{Op: loadAdd[in.Op], D: uint16(in.C), A: in.A, B: p.B, C: p.C}
					dropped[pc] = true
					continue
				}
			}
		}
		if in.Op == OpCopy && pc+1 < len(f.Code) && !targets[pc] && !targets[pc+1] && len(code) > 0 {
			p, next := &code[len(code)-1], &f.Code[pc+1]
			t, pinfo, ninfo := in.B, opInfo[p.Op], opInfo[next.Op]
			if p.A == t && pinfo&fDst != 0 && ninfo&fAW != 0 && ninfo&(fA|fRange) == 0 && next.A == t {
				p.A = in.A
				if ninfo&fB != 0 && next.B == t {
					next.B = in.A
				}
				if ninfo&fC != 0 && next.C == t {
					next.C = in.A
				}
				dropped[pc] = true
				continue
			}
		}
		if opInfo[in.Op]&fPC != 0 {
			jumps = append(jumps, len(code))
		}
		code = append(code, in)
	}
	if len(code) == len(f.Code) {
		return
	}
	newPC[len(f.Code)] = uint32(len(code))
	for _, i := range jumps {
		code[i].A = newPC[code[i].A]
	}
	for i := range f.BrTable {
		f.BrTable[i].PC = newPC[f.BrTable[i].PC]
	}
	for i := range f.Inlined {
		s := &f.Inlined[i]
		var pcs []uint32
		for pc := s.Start; pc < s.End; pc++ {
			if !dropped[pc] {
				pcs = append(pcs, s.PCs[pc-s.Start])
			}
		}
		s.Start, s.End, s.PCs = newPC[s.Start], newPC[s.End], pcs
	}
	f.Code = code
}
