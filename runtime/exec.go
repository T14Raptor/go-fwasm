package runtime

import (
	"fmt"
	"math"
	"math/bits"
	"unsafe"

	"github.com/t14raptor/go-fwasm/compile"
)

// invoke calls f with args on top of the current stack. It is the boundary
// between Go and wasm: whatever happens inside, the stack and frames are
// restored on the way out, so an outer activation can carry on after a
// host function swallows the error.
func (s *Store) invoke(f *Function, args []uint64) ([]uint64, error) {
	ft := f.typ
	if len(args) != len(ft.Params) {
		return nil, fmt.Errorf("%s: got %d arguments, want %d", f, len(args), len(ft.Params))
	}
	if f.host != nil {
		in := make([]uint64, len(args))
		for i, a := range args {
			in[i] = normalize(ft.Params[i], a)
		}
		res, err := f.host(&Caller{}, in)
		if err != nil {
			return nil, err
		}
		if len(res) != len(ft.Results) {
			return nil, fmt.Errorf("%s returned %d values, want %d", f, len(res), len(ft.Results))
		}
		return res, nil
	}

	if len(s.frames) >= s.MaxCallDepth {
		return nil, &Trap{Code: TrapCallStackExhausted}
	}
	base, nframes := s.sp, len(s.frames)
	defer func() {
		s.sp = base
		s.frames = s.frames[:nframes]
	}()
	if !s.growStack(base + f.frameSize) {
		return nil, &Trap{Code: TrapCallStackExhausted}
	}
	for i, a := range args {
		s.stack[base+i] = normalize(ft.Params[i], a)
	}
	if err := s.run(f, base); err != nil {
		return nil, err
	}
	res := make([]uint64, len(ft.Results))
	copy(res, s.stack[base:])
	return res, nil
}

// callHost calls a host function from wasm. The arguments are in the
// stack from at.
func (s *Store) callHost(f *Function, caller *Instance, at int) ([]uint64, error) {
	top := at + f.numParams
	args := s.stack[at:top]
	s.sp = top // nested calls start above the arguments
	h := s.hooks
	if h != nil && h.HostCall != nil {
		h.HostCall(f, args)
	}
	res, err := f.host(&caller.caller, args)
	if h != nil && h.HostReturn != nil {
		h.HostReturn(f, res, err)
	}
	if err != nil {
		return nil, err
	}
	if len(res) != len(f.typ.Results) {
		return nil, fmt.Errorf("%s returned %d values, want %d", f, len(res), len(f.typ.Results))
	}
	return res, nil
}

func b2u(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

func f32(v uint64) float32 { return math.Float32frombits(uint32(v)) }
func f64(v uint64) float64 { return math.Float64frombits(v) }
func u32f(v float32) uint64 {
	return uint64(math.Float32bits(v))
}
func u64f(v float64) uint64 { return math.Float64bits(v) }

func memBytes(inst *Instance) []byte {
	if inst.mem == nil {
		return nil
	}
	return inst.mem.buf
}

// activation is the state of one run that the dispatch loop does not keep
// in locals. Go's register allocator spills a loop variable that some case
// has to restore after a call at the top of the loop, which would cost
// every instruction a dozen stores. So the loop keeps only pc and views of
// code, regs and mem in locals. Before anything that calls out it stores
// pc here, and after it reloads everything. Cases that may call out go to
// slow, including some that only call out on amd64: math.Ceil and
// bits.OnesCount check for CPU features and fall back to a call there.
type activation struct {
	s     *Store
	hooks *Hooks
	watch bool
	plain bool // run functions without their inlined calls
	base  int  // len(s.frames) when the run started

	fn   *Function
	cf   *compile.Func // the code fn runs
	inst *Instance
	fp   int

	pc   int
	code []compile.Instr
	regs []uint64 // fn's frame: stack[fp:]
	mem  []byte
}

// load returns the views the dispatch loop works on.
func (a *activation) load() (instrs, slots, []byte) {
	return instrs{unsafe.Pointer(unsafe.SliceData(a.code))}, slots{unsafe.Pointer(unsafe.SliceData(a.regs))}, a.mem
}

// slots is a frame without bounds checks. Compile verifies that every slot
// an instruction names is inside its function's frame, and a call makes
// room for the whole frame before it starts.
type slots struct{ p unsafe.Pointer }

func (r slots) get(i uint32) uint64    { return *(*uint64)(unsafe.Add(r.p, uintptr(i)*8)) }
func (r slots) set(i uint32, v uint64) { *(*uint64)(unsafe.Add(r.p, uintptr(i)*8)) = v }

// instrs is code without bounds checks. Compile verifies that branches
// land inside the code and that it cannot run off its end.
type instrs struct{ p unsafe.Pointer }

func (c instrs) at(pc int) *compile.Instr {
	return (*compile.Instr)(unsafe.Add(c.p, uintptr(pc)*unsafe.Sizeof(compile.Instr{})))
}

// trap builds a Trap for the instruction before pc.
func (a *activation) trap(code TrapCode, pc int) error {
	return a.s.trap(code, a.cf, a.fn, pc)
}

func (a *activation) memWrite(addr, size uint64) { a.hooks.memWrite(a.fn, addr, size) }

// enter starts fn, whose frame is at stack[fp] with the arguments in place.
func (a *activation) enter(fn *Function, fp int) {
	a.fn, a.fp = fn, fp
	a.cf = fn.code
	if a.plain && a.cf.Plain != nil {
		a.cf = a.cf.Plain
	}
	a.code = a.cf.Code
	a.regs = a.s.stack[fp:]
	if cf := a.cf; cf.ZeroFrom < cf.ZeroTo {
		clear(a.regs[cf.ZeroFrom:cf.ZeroTo])
	}
	a.pc = 0
	if fn.inst != a.inst {
		a.inst = fn.inst
		a.mem = memBytes(a.inst)
	}
	if h := a.hooks; h != nil && h.Enter != nil {
		a.s.sp = fp + fn.frameSize
		h.Enter(fn, a.regs[:fn.numParams])
	}
}

// call calls callee with its frame at slot at of the current one. a.pc is
// the return address.
func (a *activation) call(callee *Function, at int) error {
	s := a.s
	if callee.host != nil {
		// The caller stays on s.frames during the host call, so wasm
		// the host re-enters counts toward MaxCallDepth and traps in
		// it show this frame.
		s.frames = append(s.frames, frame{uint32(a.fn.addr - 1), uint32(a.pc), a.fp})
		res, err := s.callHost(callee, a.inst, a.fp+at)
		if err != nil {
			return err
		}
		s.frames = s.frames[:len(s.frames)-1]
		a.regs = s.stack[a.fp:]
		for i, t := range callee.typ.Results {
			a.regs[at+i] = normalize(t, res[i])
		}
		a.mem = memBytes(a.inst)
		return nil
	}
	// The depth limit counts frames of every activation, so host
	// re-entrancy cannot get around it.
	if len(s.frames) >= s.MaxCallDepth {
		return a.trap(TrapCallStackExhausted, a.pc)
	}
	if at+callee.frameSize > len(a.regs) && !s.growStack(a.fp+at+callee.frameSize) {
		return a.trap(TrapCallStackExhausted, a.pc)
	}
	s.frames = append(s.frames, frame{uint32(a.fn.addr - 1), uint32(a.pc), a.fp})
	a.enter(callee, a.fp+at)
	return nil
}

func (a *activation) callIndirect(in *compile.Instr) error {
	i := uint32(a.regs[in.C])
	tab := a.inst.tables[in.B]
	if i >= uint32(len(tab.elems)) {
		return a.trap(TrapUndefinedElement, a.pc)
	}
	// FuncByRef is nil for null, and for junk a host put in the table
	// with Table.Set.
	callee := a.s.FuncByRef(tab.elems[i])
	if callee == nil {
		return a.trap(TrapUninitializedElement, a.pc)
	}
	if callee.typeID != a.inst.typeIDs[in.A] {
		return a.trap(TrapIndirectCallTypeMismatch, a.pc)
	}
	return a.call(callee, int(in.C)-callee.numParams)
}

// ret returns from the running function, whose results start at slot
// from. It reports whether that ends the run.
func (a *activation) ret(from uint32) bool {
	fn, regs, s := a.fn, a.regs, a.s
	switch nr := fn.numResults; nr {
	case 0:
	case 1:
		regs[0] = regs[from]
	default:
		copy(regs[:nr], regs[from:int(from)+nr])
	}
	if h := a.hooks; h != nil && h.Exit != nil {
		s.sp = a.fp + fn.numResults
		h.Exit(fn, regs[:fn.numResults])
	}
	if len(s.frames) == a.base {
		s.sp = a.fp + fn.numResults
		return true
	}
	f := &s.frames[len(s.frames)-1]
	a.fn, a.pc, a.fp = s.funcs[f.fn], int(f.pc), f.fp
	s.frames = s.frames[:len(s.frames)-1]
	a.cf = a.fn.code
	if a.plain && a.cf.Plain != nil {
		a.cf = a.cf.Plain
	}
	a.code = a.cf.Code
	a.regs = s.stack[a.fp:]
	// mem tracks grows of the running instance's memory, so it only
	// needs reloading when the instance changes.
	if a.fn.inst != a.inst {
		a.inst = a.fn.inst
		a.mem = memBytes(a.inst)
	}
	return false
}

// slow runs the instructions too rare or too big for the dispatch loop.
func (a *activation) slow(in *compile.Instr) error {
	regs, mem, inst, pc := a.regs, a.mem, a.inst, a.pc
	switch in.Op {
	case compile.OpMemoryGrow:
		a.s.sp = a.fp + a.fn.frameSize // grow callbacks may call back in
		old, ok := inst.mem.Grow(uint32(regs[in.B]))
		if !ok {
			old = math.MaxUint32
		}
		a.regs = a.s.stack[a.fp:]
		a.regs[in.A] = uint64(old)
		a.mem = inst.mem.buf
	case compile.OpTableGet:
		t, b := inst.tables[in.A], int(in.B)
		i := uint32(regs[b])
		if i >= uint32(len(t.elems)) {
			return a.trap(TrapTableOutOfBounds, pc)
		}
		regs[b] = t.elems[i]
	case compile.OpTableSet:
		t, b := inst.tables[in.A], int(in.B)
		i := uint32(regs[b])
		if i >= uint32(len(t.elems)) {
			return a.trap(TrapTableOutOfBounds, pc)
		}
		t.elems[i] = regs[b+1]
	case compile.OpTableSize:
		regs[in.B] = uint64(len(inst.tables[in.A].elems))
	case compile.OpTableGrow:
		b := int(in.B)
		old, ok := inst.tables[in.A].Grow(uint32(regs[b+1]), regs[b])
		if !ok {
			old = math.MaxUint32
		}
		regs[b] = uint64(old)
	case compile.OpTableFill:
		t, b := inst.tables[in.A], int(in.B)
		d, ref, n := uint64(uint32(regs[b])), regs[b+1], uint64(uint32(regs[b+2]))
		if d+n > uint64(len(t.elems)) {
			return a.trap(TrapTableOutOfBounds, pc)
		}
		for i := range t.elems[d : d+n] {
			t.elems[d+uint64(i)] = ref
		}
	case compile.OpTableCopy:
		dt, st, b := inst.tables[in.A], inst.tables[in.C], int(in.B)
		d, src, n := uint64(uint32(regs[b])), uint64(uint32(regs[b+1])), uint64(uint32(regs[b+2]))
		if d+n > uint64(len(dt.elems)) || src+n > uint64(len(st.elems)) {
			return a.trap(TrapTableOutOfBounds, pc)
		}
		copy(dt.elems[d:d+n], st.elems[src:src+n])
	case compile.OpTableInit:
		b := int(in.B)
		if code := inst.tableInit(inst.tables[in.C], in.A, uint32(regs[b]), uint32(regs[b+1]), uint32(regs[b+2])); code != 0 {
			return a.trap(code, pc)
		}
	case compile.OpElemDrop:
		inst.elems[in.A] = nil

	case compile.OpMemoryInit:
		b := int(in.B)
		d, src, n := uint32(regs[b]), uint32(regs[b+1]), uint32(regs[b+2])
		if code := inst.memoryInit(in.A, d, src, n); code != 0 {
			return a.trap(code, pc)
		}
		if a.watch && n > 0 {
			a.memWrite(uint64(d), uint64(n))
		}
	case compile.OpDataDrop:
		inst.datas[in.A] = nil
	case compile.OpMemoryCopy:
		b := int(in.B)
		d, src, n := uint64(uint32(regs[b])), uint64(uint32(regs[b+1])), uint64(uint32(regs[b+2]))
		if d+n > uint64(len(mem)) || src+n > uint64(len(mem)) {
			return a.trap(TrapMemoryOutOfBounds, pc)
		}
		copy(mem[d:d+n], mem[src:src+n])
		if a.watch && n > 0 {
			a.memWrite(d, n)
		}
	case compile.OpMemoryFill:
		b := int(in.B)
		d, v, n := uint64(uint32(regs[b])), byte(regs[b+1]), uint64(uint32(regs[b+2]))
		if d+n > uint64(len(mem)) {
			return a.trap(TrapMemoryOutOfBounds, pc)
		}
		fill := mem[d : d+n]
		for i := range fill {
			fill[i] = v
		}
		if a.watch && n > 0 {
			a.memWrite(d, n)
		}

	case compile.OpI32Popcnt:
		regs[in.A] = uint64(bits.OnesCount32(uint32(regs[in.B])))
	case compile.OpI64Popcnt:
		regs[in.A] = uint64(bits.OnesCount64(regs[in.B]))
	case compile.OpF32Ceil:
		regs[in.A] = u32f(float32(math.Ceil(float64(f32(regs[in.B])))))
	case compile.OpF32Floor:
		regs[in.A] = u32f(float32(math.Floor(float64(f32(regs[in.B])))))
	case compile.OpF32Trunc:
		regs[in.A] = u32f(float32(math.Trunc(float64(f32(regs[in.B])))))
	case compile.OpF32Nearest:
		regs[in.A] = u32f(float32(math.RoundToEven(float64(f32(regs[in.B])))))
	case compile.OpF32Min:
		regs[in.A] = u32f(fmin32(f32(regs[in.B]), f32(regs[in.C])))
	case compile.OpF32Max:
		regs[in.A] = u32f(fmax32(f32(regs[in.B]), f32(regs[in.C])))
	case compile.OpF64Ceil:
		regs[in.A] = u64f(math.Ceil(f64(regs[in.B])))
	case compile.OpF64Floor:
		regs[in.A] = u64f(math.Floor(f64(regs[in.B])))
	case compile.OpF64Trunc:
		regs[in.A] = u64f(math.Trunc(f64(regs[in.B])))
	case compile.OpF64Nearest:
		regs[in.A] = u64f(math.RoundToEven(f64(regs[in.B])))
	case compile.OpF64Min:
		regs[in.A] = u64f(fmin64(f64(regs[in.B]), f64(regs[in.C])))
	case compile.OpF64Max:
		regs[in.A] = u64f(fmax64(f64(regs[in.B]), f64(regs[in.C])))
	case compile.OpI32TruncF32S:
		v, code := truncS32(float64(f32(regs[in.B])))
		if code != 0 {
			return a.trap(code, pc)
		}
		regs[in.A] = v
	case compile.OpI32TruncF32U:
		v, code := truncU32(float64(f32(regs[in.B])))
		if code != 0 {
			return a.trap(code, pc)
		}
		regs[in.A] = v
	case compile.OpI32TruncF64S:
		v, code := truncS32(f64(regs[in.B]))
		if code != 0 {
			return a.trap(code, pc)
		}
		regs[in.A] = v
	case compile.OpI32TruncF64U:
		v, code := truncU32(f64(regs[in.B]))
		if code != 0 {
			return a.trap(code, pc)
		}
		regs[in.A] = v
	case compile.OpI64TruncF32S:
		v, code := truncS64(float64(f32(regs[in.B])))
		if code != 0 {
			return a.trap(code, pc)
		}
		regs[in.A] = v
	case compile.OpI64TruncF32U:
		v, code := truncU64(float64(f32(regs[in.B])))
		if code != 0 {
			return a.trap(code, pc)
		}
		regs[in.A] = v
	case compile.OpI64TruncF64S:
		v, code := truncS64(f64(regs[in.B]))
		if code != 0 {
			return a.trap(code, pc)
		}
		regs[in.A] = v
	case compile.OpI64TruncF64U:
		v, code := truncU64(f64(regs[in.B]))
		if code != 0 {
			return a.trap(code, pc)
		}
		regs[in.A] = v
	case compile.OpI32TruncSatF32S:
		regs[in.A] = satS32(float64(f32(regs[in.B])))
	case compile.OpI32TruncSatF32U:
		regs[in.A] = satU32(float64(f32(regs[in.B])))
	case compile.OpI32TruncSatF64S:
		regs[in.A] = satS32(f64(regs[in.B]))
	case compile.OpI32TruncSatF64U:
		regs[in.A] = satU32(f64(regs[in.B]))
	case compile.OpI64TruncSatF32S:
		regs[in.A] = satS64(float64(f32(regs[in.B])))
	case compile.OpI64TruncSatF32U:
		regs[in.A] = satU64(float64(f32(regs[in.B])))
	case compile.OpI64TruncSatF64S:
		regs[in.A] = satS64(f64(regs[in.B]))
	case compile.OpI64TruncSatF64U:
		regs[in.A] = satU64(f64(regs[in.B]))

	default:
		return fmt.Errorf("%s: unknown opcode %s at pc %d", a.fn, in.Op, pc-1)
	}
	return nil
}

// run executes fn, whose frame starts at stack[fp] with the arguments in
// place, until it returns. The results are left at stack[fp:].
func (s *Store) run(fn *Function, fp int) error {
	a := activation{s: s, hooks: s.hooks, base: len(s.frames)}
	a.watch = a.hooks.watching()
	// With hooks on, functions run without inlined calls, so that every
	// call shows.
	a.plain = a.hooks != nil
	a.enter(fn, fp)
	pc := 0
	code, r, mem := a.load()
	for {
		in := code.at(pc)
		pc++
		switch in.Op {
		case compile.OpUnreachable:
			return a.trap(TrapUnreachable, pc)

		case compile.OpLoopHeader:
			if s.interrupt.Load() {
				return a.trap(TrapInterrupted, pc)
			}

		case compile.OpJump:
			pc = int(in.A)

		// Branch moves go down the stack, so copying upwards is safe.
		case compile.OpBr:
			for i := range uint32(in.D) {
				r.set(in.C+i, r.get(in.B+i))
			}
			pc = int(in.A)

		case compile.OpBrIf:
			if uint32(r.get(in.B)) != 0 {
				t := &a.cf.BrTable[in.A]
				for i := range t.N {
					r.set(t.Dst+i, r.get(t.Src+i))
				}
				pc = int(t.PC)
			}

		case compile.OpBrIfNez:
			if uint32(r.get(in.B)) != 0 {
				pc = int(in.A)
			}

		case compile.OpBrIfEqz:
			if uint32(r.get(in.B)) == 0 {
				pc = int(in.A)
			}

		case compile.OpBrTable:
			i := uint32(r.get(in.B))
			if last := in.C - 1; i > last {
				i = last
			}
			t := &a.cf.BrTable[in.A+i]
			for j := range t.N {
				r.set(t.Dst+j, r.get(t.Src+j))
			}
			pc = int(t.PC)

		// Calls and returns within an instance take a fast path when
		// hooks are off; ret and call do the rest. Both paths reload
		// all the loop's locals, so that none has to survive the
		// register pressure here.
		case compile.OpReturn:
			a.pc = pc
			if n := len(s.frames); n > a.base && !a.plain && a.fn.numResults <= 1 {
				f := s.frames[n-1]
				caller := s.funcs[f.fn]
				if caller.inst == a.inst {
					if a.fn.numResults == 1 {
						r.set(0, r.get(in.A))
					}
					s.frames = s.frames[:n-1]
					a.fn, a.cf, a.fp = caller, caller.code, f.fp
					a.code, a.regs = a.cf.Code, s.stack[f.fp:]
					pc = int(f.pc)
					code, r, mem = a.load()
					break
				}
			}
			if a.ret(in.A) {
				return nil
			}
			pc = a.pc
			code, r, mem = a.load()

		case compile.OpCall:
			callee, at := a.inst.funcs[in.A], in.B
			if n := len(s.frames); callee.inst == a.inst && !a.plain && n < s.MaxCallDepth && n < cap(s.frames) &&
				int(at)+callee.frameSize <= len(a.regs) {
				s.frames = s.frames[:n+1]
				s.frames[n] = frame{uint32(a.fn.addr - 1), uint32(pc), a.fp}
				a.fn, a.cf, a.fp = callee, callee.code, a.fp+int(at)
				a.code, a.regs = a.cf.Code, a.regs[at:]
				code, r, mem = a.load()
				for i := a.cf.ZeroFrom; i < a.cf.ZeroTo; i++ {
					r.set(uint32(i), 0)
				}
				pc = 0
				break
			}
			a.pc = pc
			if err := a.call(callee, int(at)); err != nil {
				return err
			}
			pc = a.pc
			code, r, mem = a.load()

		case compile.OpCallIndirect:
			a.pc = pc
			if err := a.callIndirect(in); err != nil {
				return err
			}
			pc = a.pc
			code, r, mem = a.load()

		// Loads from a base plus a constant, wrapped to 32 bits, plus the
		// static offset.
		case compile.OpLoad8UAdd:
			ea := uint64(uint32(r.get(in.B))+in.C) + uint64(in.D)
			if ea >= uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			r.set(in.A, uint64(mem[ea]))
		case compile.OpLoad16UAdd:
			ea := uint64(uint32(r.get(in.B))+in.C) + uint64(in.D)
			if ea+2 > uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			r.set(in.A, uint64(load16(mem, ea)))
		case compile.OpLoad32Add:
			ea := uint64(uint32(r.get(in.B))+in.C) + uint64(in.D)
			if ea+4 > uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			r.set(in.A, uint64(load32(mem, ea)))
		case compile.OpLoad64Add:
			ea := uint64(uint32(r.get(in.B))+in.C) + uint64(in.D)
			if ea+8 > uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			r.set(in.A, load64(mem, ea))

		case compile.OpSelect:
			if uint32(r.get(in.C)) == 0 {
				r.set(in.A, r.get(in.B))
			}
		case compile.OpCopy:
			r.set(in.A, r.get(in.B))
		case compile.OpConst:
			r.set(in.A, uint64(in.B)|uint64(in.C)<<32)

		case compile.OpGlobalGet:
			r.set(in.A, a.inst.globals[in.B].val)
		case compile.OpGlobalSet:
			a.inst.globals[in.A].val = r.get(in.B)

		case compile.OpRefIsNull:
			r.set(in.A, b2u(r.get(in.B) == 0))
		case compile.OpRefFunc:
			r.set(in.A, a.inst.funcs[in.B].addr)

		// Loads. The effective address is computed in 64 bits, so it
		// cannot wrap.
		case compile.OpLoad8U:
			ea := uint64(uint32(r.get(in.B))) + uint64(in.C)
			if ea >= uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			r.set(in.A, uint64(mem[ea]))
		case compile.OpLoad16U:
			ea := uint64(uint32(r.get(in.B))) + uint64(in.C)
			if ea+2 > uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			r.set(in.A, uint64(load16(mem, ea)))
		case compile.OpLoad32:
			ea := uint64(uint32(r.get(in.B))) + uint64(in.C)
			if ea+4 > uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			r.set(in.A, uint64(load32(mem, ea)))
		case compile.OpLoad64:
			ea := uint64(uint32(r.get(in.B))) + uint64(in.C)
			if ea+8 > uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			r.set(in.A, load64(mem, ea))
		case compile.OpI32Load8S:
			ea := uint64(uint32(r.get(in.B))) + uint64(in.C)
			if ea >= uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			r.set(in.A, uint64(uint32(int32(int8(mem[ea])))))
		case compile.OpI32Load16S:
			ea := uint64(uint32(r.get(in.B))) + uint64(in.C)
			if ea+2 > uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			r.set(in.A, uint64(uint32(int32(int16(load16(mem, ea))))))
		case compile.OpI64Load8S:
			ea := uint64(uint32(r.get(in.B))) + uint64(in.C)
			if ea >= uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			r.set(in.A, uint64(int64(int8(mem[ea]))))
		case compile.OpI64Load16S:
			ea := uint64(uint32(r.get(in.B))) + uint64(in.C)
			if ea+2 > uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			r.set(in.A, uint64(int64(int16(load16(mem, ea)))))
		case compile.OpI64Load32S:
			ea := uint64(uint32(r.get(in.B))) + uint64(in.C)
			if ea+4 > uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			r.set(in.A, uint64(int64(int32(load32(mem, ea)))))

		// Stores.
		case compile.OpStore8:
			ea := uint64(uint32(r.get(in.A))) + uint64(in.C)
			if ea >= uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			mem[ea] = byte(r.get(in.B))
			if a.watch {
				a.pc = pc
				a.memWrite(ea, 1)
				pc = a.pc
				code, r, mem = a.load()
			}
		case compile.OpStore16:
			ea := uint64(uint32(r.get(in.A))) + uint64(in.C)
			if ea+2 > uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			store16(mem, ea, uint16(r.get(in.B)))
			if a.watch {
				a.pc = pc
				a.memWrite(ea, 2)
				pc = a.pc
				code, r, mem = a.load()
			}
		case compile.OpStore32:
			ea := uint64(uint32(r.get(in.A))) + uint64(in.C)
			if ea+4 > uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			store32(mem, ea, uint32(r.get(in.B)))
			if a.watch {
				a.pc = pc
				a.memWrite(ea, 4)
				pc = a.pc
				code, r, mem = a.load()
			}
		case compile.OpStore64:
			ea := uint64(uint32(r.get(in.A))) + uint64(in.C)
			if ea+8 > uint64(len(mem)) {
				return a.trap(TrapMemoryOutOfBounds, pc)
			}
			store64(mem, ea, r.get(in.B))
			if a.watch {
				a.pc = pc
				a.memWrite(ea, 8)
				pc = a.pc
				code, r, mem = a.load()
			}

		case compile.OpMemorySize:
			r.set(in.A, uint64(len(mem)/PageSize))
		// i32 comparisons.
		case compile.OpI32Eqz:
			r.set(in.A, b2u(uint32(r.get(in.B)) == 0))
		case compile.OpI32Eq:
			r.set(in.A, b2u(uint32(r.get(in.B)) == uint32(r.get(in.C))))
		case compile.OpI32Ne:
			r.set(in.A, b2u(uint32(r.get(in.B)) != uint32(r.get(in.C))))
		case compile.OpI32LtS:
			r.set(in.A, b2u(int32(r.get(in.B)) < int32(r.get(in.C))))
		case compile.OpI32LtU:
			r.set(in.A, b2u(uint32(r.get(in.B)) < uint32(r.get(in.C))))
		case compile.OpI32GtS:
			r.set(in.A, b2u(int32(r.get(in.B)) > int32(r.get(in.C))))
		case compile.OpI32GtU:
			r.set(in.A, b2u(uint32(r.get(in.B)) > uint32(r.get(in.C))))
		case compile.OpI32LeS:
			r.set(in.A, b2u(int32(r.get(in.B)) <= int32(r.get(in.C))))
		case compile.OpI32LeU:
			r.set(in.A, b2u(uint32(r.get(in.B)) <= uint32(r.get(in.C))))
		case compile.OpI32GeS:
			r.set(in.A, b2u(int32(r.get(in.B)) >= int32(r.get(in.C))))
		case compile.OpI32GeU:
			r.set(in.A, b2u(uint32(r.get(in.B)) >= uint32(r.get(in.C))))

		// i64 comparisons.
		case compile.OpI64Eqz:
			r.set(in.A, b2u(r.get(in.B) == 0))
		case compile.OpI64Eq:
			r.set(in.A, b2u(r.get(in.B) == r.get(in.C)))
		case compile.OpI64Ne:
			r.set(in.A, b2u(r.get(in.B) != r.get(in.C)))
		case compile.OpI64LtS:
			r.set(in.A, b2u(int64(r.get(in.B)) < int64(r.get(in.C))))
		case compile.OpI64LtU:
			r.set(in.A, b2u(r.get(in.B) < r.get(in.C)))
		case compile.OpI64GtS:
			r.set(in.A, b2u(int64(r.get(in.B)) > int64(r.get(in.C))))
		case compile.OpI64GtU:
			r.set(in.A, b2u(r.get(in.B) > r.get(in.C)))
		case compile.OpI64LeS:
			r.set(in.A, b2u(int64(r.get(in.B)) <= int64(r.get(in.C))))
		case compile.OpI64LeU:
			r.set(in.A, b2u(r.get(in.B) <= r.get(in.C)))
		case compile.OpI64GeS:
			r.set(in.A, b2u(int64(r.get(in.B)) >= int64(r.get(in.C))))
		case compile.OpI64GeU:
			r.set(in.A, b2u(r.get(in.B) >= r.get(in.C)))

		// Float comparisons. Go's operators already give false for NaN
		// operands (true for !=), as wasm requires.
		case compile.OpF32Eq:
			r.set(in.A, b2u(f32(r.get(in.B)) == f32(r.get(in.C))))
		case compile.OpF32Ne:
			r.set(in.A, b2u(f32(r.get(in.B)) != f32(r.get(in.C))))
		case compile.OpF32Lt:
			r.set(in.A, b2u(f32(r.get(in.B)) < f32(r.get(in.C))))
		case compile.OpF32Gt:
			r.set(in.A, b2u(f32(r.get(in.B)) > f32(r.get(in.C))))
		case compile.OpF32Le:
			r.set(in.A, b2u(f32(r.get(in.B)) <= f32(r.get(in.C))))
		case compile.OpF32Ge:
			r.set(in.A, b2u(f32(r.get(in.B)) >= f32(r.get(in.C))))
		case compile.OpF64Eq:
			r.set(in.A, b2u(f64(r.get(in.B)) == f64(r.get(in.C))))
		case compile.OpF64Ne:
			r.set(in.A, b2u(f64(r.get(in.B)) != f64(r.get(in.C))))
		case compile.OpF64Lt:
			r.set(in.A, b2u(f64(r.get(in.B)) < f64(r.get(in.C))))
		case compile.OpF64Gt:
			r.set(in.A, b2u(f64(r.get(in.B)) > f64(r.get(in.C))))
		case compile.OpF64Le:
			r.set(in.A, b2u(f64(r.get(in.B)) <= f64(r.get(in.C))))
		case compile.OpF64Ge:
			r.set(in.A, b2u(f64(r.get(in.B)) >= f64(r.get(in.C))))

		// i32 arithmetic.
		case compile.OpI32Clz:
			r.set(in.A, uint64(bits.LeadingZeros32(uint32(r.get(in.B)))))
		case compile.OpI32Ctz:
			r.set(in.A, uint64(bits.TrailingZeros32(uint32(r.get(in.B)))))
		case compile.OpI32Add:
			r.set(in.A, uint64(uint32(r.get(in.B))+uint32(r.get(in.C))))
		case compile.OpI32Sub:
			r.set(in.A, uint64(uint32(r.get(in.B))-uint32(r.get(in.C))))
		case compile.OpI32Mul:
			r.set(in.A, uint64(uint32(r.get(in.B))*uint32(r.get(in.C))))
		case compile.OpI32DivS:
			x, y := int32(r.get(in.B)), int32(r.get(in.C))
			if y == 0 {
				return a.trap(TrapIntegerDivideByZero, pc)
			}
			if x == math.MinInt32 && y == -1 {
				return a.trap(TrapIntegerOverflow, pc)
			}
			r.set(in.A, uint64(uint32(x/y)))
		case compile.OpI32DivU:
			y := uint32(r.get(in.C))
			if y == 0 {
				return a.trap(TrapIntegerDivideByZero, pc)
			}
			r.set(in.A, uint64(uint32(r.get(in.B))/y))
		case compile.OpI32RemS:
			x, y := int32(r.get(in.B)), int32(r.get(in.C))
			if y == 0 {
				return a.trap(TrapIntegerDivideByZero, pc)
			}
			if y == -1 {
				r.set(in.A, 0)
			} else {
				r.set(in.A, uint64(uint32(x%y)))
			}
		case compile.OpI32RemU:
			y := uint32(r.get(in.C))
			if y == 0 {
				return a.trap(TrapIntegerDivideByZero, pc)
			}
			r.set(in.A, uint64(uint32(r.get(in.B))%y))
		case compile.OpI32And:
			r.set(in.A, r.get(in.B)&(r.get(in.C)))
		case compile.OpI32Or:
			r.set(in.A, r.get(in.B)|(r.get(in.C)))
		case compile.OpI32Xor:
			r.set(in.A, r.get(in.B)^(r.get(in.C)))
		case compile.OpI32Shl:
			r.set(in.A, uint64(uint32(r.get(in.B))<<(r.get(in.C)&31)))
		case compile.OpI32ShrS:
			r.set(in.A, uint64(uint32(int32(r.get(in.B))>>(r.get(in.C)&31))))
		case compile.OpI32ShrU:
			r.set(in.A, uint64(uint32(r.get(in.B))>>(r.get(in.C)&31)))
		case compile.OpI32Rotl:
			r.set(in.A, uint64(bits.RotateLeft32(uint32(r.get(in.B)), int(r.get(in.C)&31))))
		case compile.OpI32Rotr:
			r.set(in.A, uint64(bits.RotateLeft32(uint32(r.get(in.B)), -int(r.get(in.C)&31))))

		// i64 arithmetic.
		case compile.OpI64Clz:
			r.set(in.A, uint64(bits.LeadingZeros64(r.get(in.B))))
		case compile.OpI64Ctz:
			r.set(in.A, uint64(bits.TrailingZeros64(r.get(in.B))))
		case compile.OpI64Add:
			r.set(in.A, r.get(in.B)+(r.get(in.C)))
		case compile.OpI64Sub:
			r.set(in.A, r.get(in.B)-(r.get(in.C)))
		case compile.OpI64Mul:
			r.set(in.A, r.get(in.B)*(r.get(in.C)))
		case compile.OpI64DivS:
			x, y := int64(r.get(in.B)), int64(r.get(in.C))
			if y == 0 {
				return a.trap(TrapIntegerDivideByZero, pc)
			}
			if x == math.MinInt64 && y == -1 {
				return a.trap(TrapIntegerOverflow, pc)
			}
			r.set(in.A, uint64(x/y))
		case compile.OpI64DivU:
			y := r.get(in.C)
			if y == 0 {
				return a.trap(TrapIntegerDivideByZero, pc)
			}
			r.set(in.A, r.get(in.B)/(y))
		case compile.OpI64RemS:
			x, y := int64(r.get(in.B)), int64(r.get(in.C))
			if y == 0 {
				return a.trap(TrapIntegerDivideByZero, pc)
			}
			if y == -1 {
				r.set(in.A, 0)
			} else {
				r.set(in.A, uint64(x%y))
			}
		case compile.OpI64RemU:
			y := r.get(in.C)
			if y == 0 {
				return a.trap(TrapIntegerDivideByZero, pc)
			}
			r.set(in.A, r.get(in.B)%(y))
		case compile.OpI64And:
			r.set(in.A, r.get(in.B)&(r.get(in.C)))
		case compile.OpI64Or:
			r.set(in.A, r.get(in.B)|(r.get(in.C)))
		case compile.OpI64Xor:
			r.set(in.A, r.get(in.B)^(r.get(in.C)))
		case compile.OpI64Shl:
			r.set(in.A, r.get(in.B)<<(r.get(in.C)&63))
		case compile.OpI64ShrS:
			r.set(in.A, uint64(int64(r.get(in.B))>>(r.get(in.C)&63)))
		case compile.OpI64ShrU:
			r.set(in.A, r.get(in.B)>>(r.get(in.C)&63))
		case compile.OpI64Rotl:
			r.set(in.A, bits.RotateLeft64(r.get(in.B), int(r.get(in.C)&63)))
		case compile.OpI64Rotr:
			r.set(in.A, bits.RotateLeft64(r.get(in.B), -int(r.get(in.C)&63)))

		// f32 arithmetic. Sign operations work on the bits so they never
		// touch a NaN's payload.
		case compile.OpF32Abs:
			r.set(in.A, r.get(in.B)&^(1<<31))
		case compile.OpF32Neg:
			r.set(in.A, r.get(in.B)^(1<<31))
		case compile.OpF32Sqrt:
			// Rounding the double result is exact for sqrt.
			r.set(in.A, u32f(float32(math.Sqrt(float64(f32(r.get(in.B)))))))
		case compile.OpF32Add:
			r.set(in.A, u32f(f32(r.get(in.B))+f32(r.get(in.C))))
		case compile.OpF32Sub:
			r.set(in.A, u32f(f32(r.get(in.B))-f32(r.get(in.C))))
		case compile.OpF32Mul:
			r.set(in.A, u32f(f32(r.get(in.B))*f32(r.get(in.C))))
		case compile.OpF32Div:
			r.set(in.A, u32f(f32(r.get(in.B))/f32(r.get(in.C))))
		case compile.OpF32Copysign:
			r.set(in.A, r.get(in.B)&^(1<<31)|r.get(in.C)&(1<<31))

		// f64 arithmetic.
		case compile.OpF64Abs:
			r.set(in.A, r.get(in.B)&^(1<<63))
		case compile.OpF64Neg:
			r.set(in.A, r.get(in.B)^(1<<63))
		case compile.OpF64Sqrt:
			r.set(in.A, u64f(math.Sqrt(f64(r.get(in.B)))))
		case compile.OpF64Add:
			r.set(in.A, u64f(f64(r.get(in.B))+f64(r.get(in.C))))
		case compile.OpF64Sub:
			r.set(in.A, u64f(f64(r.get(in.B))-f64(r.get(in.C))))
		case compile.OpF64Mul:
			r.set(in.A, u64f(f64(r.get(in.B))*f64(r.get(in.C))))
		case compile.OpF64Div:
			r.set(in.A, u64f(f64(r.get(in.B))/f64(r.get(in.C))))
		case compile.OpF64Copysign:
			r.set(in.A, r.get(in.B)&^(1<<63)|r.get(in.C)&(1<<63))

		// Conversions.
		case compile.OpI32WrapI64:
			r.set(in.A, uint64(uint32(r.get(in.B))))
		case compile.OpI64ExtendI32S:
			r.set(in.A, uint64(int64(int32(r.get(in.B)))))
		case compile.OpF32ConvertI32S:
			r.set(in.A, u32f(float32(int32(r.get(in.B)))))
		case compile.OpF32ConvertI32U:
			r.set(in.A, u32f(float32(uint32(r.get(in.B)))))
		case compile.OpF32ConvertI64S:
			r.set(in.A, u32f(float32(int64(r.get(in.B)))))
		case compile.OpF32ConvertI64U:
			r.set(in.A, u32f(float32(r.get(in.B))))
		case compile.OpF32DemoteF64:
			r.set(in.A, u32f(float32(f64(r.get(in.B)))))
		case compile.OpF64ConvertI32S:
			r.set(in.A, u64f(float64(int32(r.get(in.B)))))
		case compile.OpF64ConvertI32U:
			r.set(in.A, u64f(float64(uint32(r.get(in.B)))))
		case compile.OpF64ConvertI64S:
			r.set(in.A, u64f(float64(int64(r.get(in.B)))))
		case compile.OpF64ConvertI64U:
			r.set(in.A, u64f(float64(r.get(in.B))))
		case compile.OpF64PromoteF32:
			r.set(in.A, u64f(float64(f32(r.get(in.B)))))
		case compile.OpI32Extend8S:
			r.set(in.A, uint64(uint32(int32(int8(r.get(in.B))))))
		case compile.OpI32Extend16S:
			r.set(in.A, uint64(uint32(int32(int16(r.get(in.B))))))
		case compile.OpI64Extend8S:
			r.set(in.A, uint64(int64(int8(r.get(in.B)))))
		case compile.OpI64Extend16S:
			r.set(in.A, uint64(int64(int16(r.get(in.B)))))
		case compile.OpI64Extend32S:
			r.set(in.A, uint64(int64(int32(r.get(in.B)))))

		// i32 instructions with a constant right operand.
		case compile.OpI32AddImm:
			r.set(in.A, uint64(uint32(r.get(in.B))+in.C))
		case compile.OpI32MulImm:
			r.set(in.A, uint64(uint32(r.get(in.B))*in.C))
		case compile.OpI32DivUImm:
			r.set(in.A, uint64(uint32(r.get(in.B))/in.C))
		case compile.OpI32RemUImm:
			r.set(in.A, uint64(uint32(r.get(in.B))%in.C))
		case compile.OpI32AndImm:
			r.set(in.A, r.get(in.B)&uint64(in.C))
		case compile.OpI32OrImm:
			r.set(in.A, r.get(in.B)|uint64(in.C))
		case compile.OpI32XorImm:
			r.set(in.A, r.get(in.B)^uint64(in.C))
		case compile.OpI32ShlImm:
			r.set(in.A, uint64(uint32(r.get(in.B))<<in.C))
		case compile.OpI32ShrSImm:
			r.set(in.A, uint64(uint32(int32(r.get(in.B))>>in.C)))
		case compile.OpI32ShrUImm:
			r.set(in.A, uint64(uint32(r.get(in.B))>>in.C))
		case compile.OpI32RotlImm:
			r.set(in.A, uint64(bits.RotateLeft32(uint32(r.get(in.B)), int(in.C))))
		case compile.OpI32EqImm:
			r.set(in.A, b2u(uint32(r.get(in.B)) == in.C))
		case compile.OpI32NeImm:
			r.set(in.A, b2u(uint32(r.get(in.B)) != in.C))
		case compile.OpI32LtSImm:
			r.set(in.A, b2u(int32(r.get(in.B)) < int32(in.C)))
		case compile.OpI32LtUImm:
			r.set(in.A, b2u(uint32(r.get(in.B)) < in.C))
		case compile.OpI32GtSImm:
			r.set(in.A, b2u(int32(r.get(in.B)) > int32(in.C)))
		case compile.OpI32GtUImm:
			r.set(in.A, b2u(uint32(r.get(in.B)) > in.C))
		case compile.OpI32LeSImm:
			r.set(in.A, b2u(int32(r.get(in.B)) <= int32(in.C)))
		case compile.OpI32LeUImm:
			r.set(in.A, b2u(uint32(r.get(in.B)) <= in.C))
		case compile.OpI32GeSImm:
			r.set(in.A, b2u(int32(r.get(in.B)) >= int32(in.C)))
		case compile.OpI32GeUImm:
			r.set(in.A, b2u(uint32(r.get(in.B)) >= in.C))

		// Fused i32 compare and branch.
		case compile.OpBrIfI32Eq:
			if uint32(r.get(in.B)) == uint32(r.get(in.C)) {
				pc = int(in.A)
			}
		case compile.OpBrIfI32Ne:
			if uint32(r.get(in.B)) != uint32(r.get(in.C)) {
				pc = int(in.A)
			}
		case compile.OpBrIfI32LtS:
			if int32(r.get(in.B)) < int32(r.get(in.C)) {
				pc = int(in.A)
			}
		case compile.OpBrIfI32LtU:
			if uint32(r.get(in.B)) < uint32(r.get(in.C)) {
				pc = int(in.A)
			}
		case compile.OpBrIfI32GtS:
			if int32(r.get(in.B)) > int32(r.get(in.C)) {
				pc = int(in.A)
			}
		case compile.OpBrIfI32GtU:
			if uint32(r.get(in.B)) > uint32(r.get(in.C)) {
				pc = int(in.A)
			}
		case compile.OpBrIfI32LeS:
			if int32(r.get(in.B)) <= int32(r.get(in.C)) {
				pc = int(in.A)
			}
		case compile.OpBrIfI32LeU:
			if uint32(r.get(in.B)) <= uint32(r.get(in.C)) {
				pc = int(in.A)
			}
		case compile.OpBrIfI32GeS:
			if int32(r.get(in.B)) >= int32(r.get(in.C)) {
				pc = int(in.A)
			}
		case compile.OpBrIfI32GeU:
			if uint32(r.get(in.B)) >= uint32(r.get(in.C)) {
				pc = int(in.A)
			}
		case compile.OpBrIfI32EqImm:
			if uint32(r.get(in.B)) == in.C {
				pc = int(in.A)
			}
		case compile.OpBrIfI32NeImm:
			if uint32(r.get(in.B)) != in.C {
				pc = int(in.A)
			}
		case compile.OpBrIfI32LtSImm:
			if int32(r.get(in.B)) < int32(in.C) {
				pc = int(in.A)
			}
		case compile.OpBrIfI32LtUImm:
			if uint32(r.get(in.B)) < in.C {
				pc = int(in.A)
			}
		case compile.OpBrIfI32GtSImm:
			if int32(r.get(in.B)) > int32(in.C) {
				pc = int(in.A)
			}
		case compile.OpBrIfI32GtUImm:
			if uint32(r.get(in.B)) > in.C {
				pc = int(in.A)
			}
		case compile.OpBrIfI32LeSImm:
			if int32(r.get(in.B)) <= int32(in.C) {
				pc = int(in.A)
			}
		case compile.OpBrIfI32LeUImm:
			if uint32(r.get(in.B)) <= in.C {
				pc = int(in.A)
			}
		case compile.OpBrIfI32GeSImm:
			if int32(r.get(in.B)) >= int32(in.C) {
				pc = int(in.A)
			}
		case compile.OpBrIfI32GeUImm:
			if uint32(r.get(in.B)) >= in.C {
				pc = int(in.A)
			}

		default:
			a.pc = pc
			if err := a.slow(in); err != nil {
				return err
			}
			pc = a.pc
			code, r, mem = a.load()
		}
	}
}
