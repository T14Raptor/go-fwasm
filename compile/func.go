package compile

import (
	"fmt"
	"math"

	"github.com/t14raptor/go-fwasm/instruction"
	"github.com/t14raptor/go-fwasm/module"
	"github.com/t14raptor/go-fwasm/types"
)

// unknown is the bottom type of the polymorphic stack in unreachable code.
const unknown types.ValueType = 0

type ctrlKind uint8

const (
	ctrlFunc ctrlKind = iota
	ctrlBlock
	ctrlLoop
	ctrlIf
	ctrlElse
)

type ctrlFrame struct {
	kind    ctrlKind
	params  []types.ValueType
	results []types.ValueType
	// height is the operand stack height below the frame's parameters.
	// The label's values go to the slots from there up.
	height      int
	unreachable bool
	// dead means the whole frame was entered from unreachable code, so
	// nothing inside it is emitted.
	dead bool

	startPC     int   // loop: branch target
	fixups      []int // instructions whose A becomes the end pc
	tableFixups []int // BrTable entries whose PC becomes the end pc
	elseFixup   int   // if: the jump to patch, or -1
}

func (f *ctrlFrame) labelTypes() []types.ValueType {
	if f.kind == ctrlLoop {
		return f.params
	}
	return f.results
}

// operandKind says where a value on the operand stack is.
type operandKind uint8

const (
	inSlot  operandKind = iota // in the slot of its stack position
	inLocal                    // nowhere yet: it is the current value of a local
	isConst                    // nowhere yet: it is a constant
)

type operand struct {
	kind operandKind
	slot uint32 // inSlot, inLocal
	bits uint64 // isConst
}

type stackVal struct {
	t  types.ValueType
	op operand
}

type funcCompiler struct {
	mod       *Module
	refs      map[uint32]bool
	locals    []types.ValueType
	numLocals uint32
	vals      []stackVal
	ctrls     []ctrlFrame
	code      []Instr
	brTable   []BrTarget
	maxVals   int

	// lazy is a lower bound on the stack positions that hold inLocal or
	// isConst values.
	lazy int
	// last is the index of the last instruction when all it does is write
	// slot A, so that a local.set right after it can redirect it, and
	// nothing branches to the pc after it. Otherwise it is -1.
	last int
}

// work is one instruction sequence being walked. Bodies nest thousands deep
// in real modules, so the walk keeps its own stack instead of recursing.
type work struct {
	body []instruction.Instruction
	i    int
	// ifElse is the else body still to walk once this then-body ends.
	ifElse   []instruction.Instruction
	isThen   bool
	hasFrame bool // ending this body ends a control frame
}

func compileFunc(mod *Module, refs map[uint32]bool, ft *types.FuncType, code *module.Code) (*Func, error) {
	c := &funcCompiler{mod: mod, refs: refs, last: -1}
	c.locals = append(c.locals, ft.Params...)
	total := uint64(len(ft.Params))
	for _, l := range code.Locals {
		total += uint64(l.Count)
		if total > MaxLocals {
			return nil, invalidf("too many locals")
		}
		if !validValueType(l.Type) {
			return nil, invalidf("unsupported local type 0x%02x", byte(l.Type))
		}
		for j := uint32(0); j < l.Count; j++ {
			c.locals = append(c.locals, l.Type)
		}
	}
	c.numLocals = uint32(len(c.locals))

	c.ctrls = append(c.ctrls, ctrlFrame{kind: ctrlFunc, results: ft.Results, elseFixup: -1})
	stack := []work{{body: code.Body, hasFrame: true}}
	for len(stack) > 0 {
		w := &stack[len(stack)-1]
		if w.i == len(w.body) {
			if w.isThen && w.ifElse != nil {
				if err := c.elseOp(); err != nil {
					return nil, err
				}
				*w = work{body: w.ifElse, hasFrame: true}
				continue
			}
			if w.hasFrame {
				if err := c.endOp(); err != nil {
					return nil, err
				}
			}
			stack = stack[:len(stack)-1]
			continue
		}
		in := w.body[w.i]
		w.i++

		ctl, ok := in.(instruction.Control)
		if ok {
			switch x := ctl.Instr.(type) {
			case instruction.Block:
				if err := c.blockOp(ctrlBlock, x.BlockType); err != nil {
					return nil, err
				}
				stack = append(stack, work{body: x.Body, hasFrame: true})
				continue
			case instruction.Loop:
				if err := c.blockOp(ctrlLoop, x.BlockType); err != nil {
					return nil, err
				}
				stack = append(stack, work{body: x.Body, hasFrame: true})
				continue
			case instruction.If:
				if err := c.blockOp(ctrlIf, x.BlockType); err != nil {
					return nil, err
				}
				stack = append(stack, work{body: x.Then, isThen: true, ifElse: x.Else, hasFrame: true})
				continue
			}
		}
		if err := c.instr(in); err != nil {
			return nil, fmt.Errorf("%w (at %s)", err, describe(in))
		}
	}

	return &Func{
		Type:       ft,
		ZeroFrom:   len(ft.Params),
		ZeroTo:     len(c.locals),
		NumLocals:  len(c.locals),
		MaxHeight:  c.maxVals,
		Code:       c.code,
		BrTable:    c.brTable,
		LocalTypes: c.locals,
	}, nil
}

func describe(in instruction.Instruction) string {
	if in == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%T", in)
}

// Operand stack.

// slot is the slot of stack position pos.
func (c *funcCompiler) slot(pos int) uint32 { return c.numLocals + uint32(pos) }

// push pushes a value computed into its stack slot.
func (c *funcCompiler) push(t types.ValueType) {
	c.pushOp(t, operand{kind: inSlot, slot: c.slot(len(c.vals))})
}

func (c *funcCompiler) pushOp(t types.ValueType, op operand) {
	if op.kind != inSlot && len(c.vals) < c.lazy {
		c.lazy = len(c.vals)
	}
	c.vals = append(c.vals, stackVal{t, op})
	if len(c.vals) > c.maxVals {
		c.maxVals = len(c.vals)
	}
}

func (c *funcCompiler) pushN(ts []types.ValueType) {
	for _, t := range ts {
		c.push(t)
	}
}

func (c *funcCompiler) pop() (stackVal, error) {
	f := &c.ctrls[len(c.ctrls)-1]
	if len(c.vals) == f.height {
		if f.unreachable {
			return stackVal{t: unknown, op: operand{slot: c.slot(len(c.vals))}}, nil
		}
		return stackVal{}, invalidf("type mismatch: operand stack underflow")
	}
	v := c.vals[len(c.vals)-1]
	c.vals = c.vals[:len(c.vals)-1]
	return v, nil
}

func (c *funcCompiler) popExpect(want types.ValueType) (stackVal, error) {
	v, err := c.pop()
	if err != nil {
		return stackVal{}, err
	}
	if v.t != want && v.t != unknown && want != unknown {
		return stackVal{}, invalidf("type mismatch: expected %s, got %s", want, v.t)
	}
	if v.t == unknown {
		v.t = want
	}
	return v, nil
}

func (c *funcCompiler) popN(ts []types.ValueType) error {
	for i := len(ts) - 1; i >= 0; i-- {
		if _, err := c.popExpect(ts[i]); err != nil {
			return err
		}
	}
	return nil
}

// Operand placement.

// load emits dst ← op.
func (c *funcCompiler) load(dst uint32, op operand) {
	switch {
	case op.kind == isConst:
		c.emitDst(OpConst, dst, uint32(op.bits), uint32(op.bits>>32))
	case op.slot != dst:
		c.emitDst(OpCopy, dst, op.slot, 0)
	}
}

// materialize moves the value at stack position i into its slot.
func (c *funcCompiler) materialize(i int) {
	v := &c.vals[i]
	if v.op.kind == inSlot {
		return
	}
	s := c.slot(i)
	c.load(s, v.op)
	v.op = operand{kind: inSlot, slot: s}
}

// flush materializes the whole stack, as a label expects it.
func (c *funcCompiler) flush() {
	if !c.reachable() {
		return
	}
	for i := c.lazy; i < len(c.vals); i++ {
		c.materialize(i)
	}
	c.lazy = len(c.vals)
}

// flushTop materializes the top n values of the stack.
func (c *funcCompiler) flushTop(n int) {
	if !c.reachable() {
		return
	}
	for i := max(len(c.vals)-n, c.cur().height); i < len(c.vals); i++ {
		c.materialize(i)
	}
}

// preserve materializes the stack values that are still local, before the
// local changes.
func (c *funcCompiler) preserve(local uint32) {
	for i := c.lazy; i < len(c.vals); i++ {
		if op := c.vals[i].op; op.kind == inLocal && op.slot == local {
			c.materialize(i)
		}
	}
}

// use returns the slot holding v, which was popped from stack position pos.
// A constant is first put in that position's slot.
func (c *funcCompiler) use(v stackVal, pos int) uint32 {
	if v.op.kind == isConst {
		s := c.slot(pos)
		c.load(s, v.op)
		return s
	}
	return v.op.slot
}

// has reports whether the current frame holds at least n values. Code that
// is reachable but fails it is invalid, and validation will say so.
func (c *funcCompiler) has(n int) bool {
	return len(c.vals)-n >= c.cur().height
}

// Control frames.

func (c *funcCompiler) cur() *ctrlFrame { return &c.ctrls[len(c.ctrls)-1] }

func (c *funcCompiler) reachable() bool {
	f := c.cur()
	return !f.unreachable && !f.dead
}

func (c *funcCompiler) setUnreachable() {
	f := c.cur()
	c.vals = c.vals[:f.height]
	f.unreachable = true
	c.last = -1
}

func (c *funcCompiler) emit(op Op, a, b, cc uint32) int {
	if !c.reachable() {
		return -1
	}
	c.code = append(c.code, Instr{Op: op, A: a, B: b, C: cc})
	c.last = -1
	return len(c.code) - 1
}

// emitDst emits an instruction whose only effect is to write slot a.
func (c *funcCompiler) emitDst(op Op, a, b, cc uint32) {
	if i := c.emit(op, a, b, cc); i >= 0 {
		c.last = i
	}
}

// bindLabel marks the next pc as a branch target.
func (c *funcCompiler) bindLabel() { c.last = -1 }

func (c *funcCompiler) blockSig(bt types.BlockType) ([]types.ValueType, []types.ValueType, error) {
	switch bt.Kind {
	case types.BlockTypeEmpty:
		return nil, nil, nil
	case types.BlockTypeValue:
		if !validValueType(bt.ValType) {
			return nil, nil, invalidf("unsupported block type 0x%02x", byte(bt.ValType))
		}
		return nil, []types.ValueType{bt.ValType}, nil
	case types.BlockTypeIndex:
		ft, err := c.mod.funcType(bt.TypeIdx)
		if err != nil {
			return nil, nil, err
		}
		return ft.Params, ft.Results, nil
	}
	return nil, nil, invalidf("invalid block type")
}

func (c *funcCompiler) blockOp(kind ctrlKind, bt types.BlockType) error {
	params, results, err := c.blockSig(bt)
	if err != nil {
		return err
	}
	var cond stackVal
	var cmp *Instr
	condPos := 0
	if kind == ctrlIf {
		if cond, err = c.popExpect(types.I32); err != nil {
			return err
		}
		condPos = len(c.vals)
		cmp = c.takeCompare(cond)
	}
	// Code inside the frame can branch past anything that would later
	// materialize the values below it, so they go in place now.
	c.flush()
	if err := c.popN(params); err != nil {
		return err
	}
	dead := !c.reachable()
	f := ctrlFrame{kind: kind, params: params, results: results, height: len(c.vals), dead: dead, elseFixup: -1}
	if !dead {
		switch kind {
		case ctrlLoop:
			f.startPC = len(c.code)
			c.code = append(c.code, Instr{Op: OpLoopHeader})
			c.bindLabel()
		case ctrlIf:
			f.elseFixup = c.condJump(cond, condPos, cmp, false)
		}
	}
	c.ctrls = append(c.ctrls, f)
	c.pushN(params)
	return nil
}

// popFrame checks that the frame's results are exactly what is left on the
// operand stack, then removes it.
func (c *funcCompiler) popFrame() (ctrlFrame, error) {
	f := c.cur()
	if err := c.popN(f.results); err != nil {
		return ctrlFrame{}, err
	}
	if len(c.vals) != f.height {
		return ctrlFrame{}, invalidf("type mismatch: values remaining on stack at end of block")
	}
	frame := *f
	c.ctrls = c.ctrls[:len(c.ctrls)-1]
	return frame, nil
}

func (c *funcCompiler) patch(f *ctrlFrame, pc int) {
	for _, i := range f.fixups {
		c.code[i].A = uint32(pc)
	}
	for _, i := range f.tableFixups {
		c.brTable[i].PC = uint32(pc)
	}
}

func (c *funcCompiler) elseOp() error {
	if c.cur().kind != ctrlIf {
		return invalidf("else without if")
	}
	// Leaving the then-branch: jump over the else-branch.
	c.flush()
	jump := c.emit(OpJump, 0, 0, 0)
	f, err := c.popFrame()
	if err != nil {
		return err
	}
	if jump >= 0 {
		f.fixups = append(f.fixups, jump)
	}
	if f.elseFixup >= 0 {
		c.code[f.elseFixup].A = uint32(len(c.code))
	}
	c.bindLabel()
	c.ctrls = append(c.ctrls, ctrlFrame{
		kind: ctrlElse, params: f.params, results: f.results, height: len(c.vals),
		dead: f.dead, fixups: f.fixups, tableFixups: f.tableFixups, elseFixup: -1,
	})
	c.pushN(f.params)
	return nil
}

func (c *funcCompiler) endOp() error {
	cur := c.cur()
	returned := false
	if cur.kind == ctrlFunc && len(cur.fixups) == 0 && len(cur.tableFixups) == 0 && c.reachable() {
		// Nothing branches to the end, so return straight from wherever
		// the results are.
		returned = c.emitReturn()
	} else {
		c.flush()
	}
	f, err := c.popFrame()
	if err != nil {
		return err
	}
	if f.kind == ctrlIf {
		// No else: the implicit empty else-branch must map params to results.
		if !sameTypes(f.params, f.results) {
			return invalidf("type mismatch: if without else must not change the stack")
		}
		if f.elseFixup >= 0 {
			c.code[f.elseFixup].A = uint32(len(c.code))
		}
	}
	c.bindLabel()
	if f.kind == ctrlFunc {
		if !returned {
			// Branches to the function's label land on the final return,
			// with the results at the bottom of the operand stack.
			c.patch(&f, len(c.code))
			c.code = append(c.code, Instr{Op: OpReturn, A: c.slot(0)})
			c.maxVals = max(c.maxVals, len(f.results))
		}
		return nil
	}
	if f.kind != ctrlLoop {
		c.patch(&f, len(c.code))
	}
	c.pushN(f.results)
	return nil
}

func sameTypes(a, b []types.ValueType) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Branches.

func (c *funcCompiler) label(depth uint32) (*ctrlFrame, error) {
	if depth >= uint32(len(c.ctrls)) {
		return nil, invalidf("unknown label %d", depth)
	}
	return &c.ctrls[len(c.ctrls)-1-int(depth)], nil
}

// target records that branch instruction idx goes to target's label.
func (c *funcCompiler) target(target *ctrlFrame, idx int) {
	if idx < 0 {
		return
	}
	if target.kind == ctrlLoop {
		c.code[idx].A = uint32(target.startPC)
	} else {
		target.fixups = append(target.fixups, idx)
	}
}

// tableTarget adds a BrTarget for a branch to target taken with the label's
// values at the top of the stack.
func (c *funcCompiler) tableTarget(target *ctrlFrame, arity int) {
	src, dst := len(c.vals)-arity, target.height
	e := BrTarget{}
	if arity > 0 && src != dst {
		e = BrTarget{N: uint32(arity), Src: c.slot(src), Dst: c.slot(dst)}
	}
	if target.kind == ctrlLoop {
		e.PC = uint32(target.startPC)
	} else {
		target.tableFixups = append(target.tableFixups, len(c.brTable))
	}
	c.brTable = append(c.brTable, e)
}

// emitReturn returns the top results, and reports whether it emitted
// anything.
func (c *funcCompiler) emitReturn() bool {
	nr := len(c.ctrls[0].results)
	if !c.reachable() || !c.has(nr) {
		return false
	}
	switch nr {
	case 0:
		c.emit(OpReturn, 0, 0, 0)
	case 1:
		pos := len(c.vals) - 1
		c.emit(OpReturn, c.use(c.vals[pos], pos), 0, 0)
	default:
		c.flushTop(nr)
		c.emit(OpReturn, c.slot(len(c.vals)-nr), 0, 0)
	}
	return true
}

// br branches to target unconditionally, with the label's values on top
// of the stack.
func (c *funcCompiler) br(target *ctrlFrame) {
	if target.kind == ctrlFunc {
		c.emitReturn()
		return
	}
	k := len(target.labelTypes())
	if !c.reachable() || !c.has(k) {
		return
	}
	c.flushTop(k)
	src, dst := len(c.vals)-k, target.height
	if k == 0 || src == dst {
		c.target(target, c.emit(OpJump, 0, 0, 0))
		return
	}
	idx := c.emit(OpBr, 0, c.slot(src), c.slot(dst))
	c.code[idx].D = uint16(k)
	c.target(target, idx)
}

// takeCompare removes the instruction that just computed cond if it is an
// i32 comparison a branch can fuse. Its operands are slots no flush writes,
// so it can move past one.
func (c *funcCompiler) takeCompare(cond stackVal) *Instr {
	if !c.reachable() || cond.op.kind != inSlot || c.last < 0 {
		return nil
	}
	in := c.code[c.last]
	if in.A != cond.op.slot || brIfOp[in.Op] == 0 {
		return nil
	}
	c.code = c.code[:c.last]
	c.last = -1
	return &in
}

// condJump emits a jump taken when cond, popped from stack position pos, is
// nonzero (or zero, if !ifTrue). cmp is the comparison that computed cond,
// if takeCompare took it. It returns the jump to patch with the target.
func (c *funcCompiler) condJump(cond stackVal, pos int, cmp *Instr, ifTrue bool) int {
	if cmp != nil {
		op := brIfOp[cmp.Op]
		if !ifTrue {
			op = brUnlessOp[cmp.Op]
		}
		return c.emit(op, 0, cmp.B, cmp.C)
	}
	op := OpBrIfNez
	if !ifTrue {
		op = OpBrIfEqz
	}
	return c.emit(op, 0, c.use(cond, pos), 0)
}

// brIf branches to target when cond, popped from stack position pos, is
// nonzero.
func (c *funcCompiler) brIf(target *ctrlFrame, cond stackVal, pos int) {
	k := len(target.labelTypes())
	if !c.reachable() || !c.has(k) {
		return
	}
	src, dst := len(c.vals)-k, target.height
	if k == 0 || src == dst {
		cmp := c.takeCompare(cond)
		c.flushTop(k)
		c.target(target, c.condJump(cond, pos, cmp, true))
		return
	}
	c.flushTop(k)
	e := len(c.brTable)
	c.tableTarget(target, k)
	c.emit(OpBrIf, uint32(e), c.use(cond, pos), 0)
}

func (c *funcCompiler) brTableOp(x instruction.BrTable) error {
	idx, err := c.popExpect(types.I32)
	if err != nil {
		return err
	}
	idxPos := len(c.vals)
	def, err := c.label(x.DefaultLabel)
	if err != nil {
		return err
	}
	arity := len(def.labelTypes())

	if c.reachable() && c.has(arity) {
		c.flushTop(arity)
		start := len(c.brTable)
		labels := append(append([]uint32(nil), x.Labels...), x.DefaultLabel)
		for _, depth := range labels {
			target, err := c.label(depth)
			if err != nil {
				return err
			}
			c.tableTarget(target, arity)
		}
		c.emit(OpBrTable, uint32(start), c.use(idx, idxPos), uint32(len(labels)))
	}

	for _, depth := range x.Labels {
		target, err := c.label(depth)
		if err != nil {
			return err
		}
		lt := target.labelTypes()
		if len(lt) != arity {
			return invalidf("type mismatch: br_table targets have different arities")
		}
		// Check the operands against this label without consuming them.
		saved := append([]stackVal(nil), c.vals...)
		if err := c.popN(lt); err != nil {
			return err
		}
		c.vals = saved
	}
	if err := c.popN(def.labelTypes()); err != nil {
		return err
	}
	c.setUnreachable()
	return nil
}

// instr validates and lowers one non-block instruction.
func (c *funcCompiler) instr(in instruction.Instruction) error {
	switch x := in.(type) {
	case instruction.Control:
		return c.control(x.Instr)
	case instruction.Numeric:
		return c.numeric(x.Instr)
	case instruction.Memory:
		return c.memory(x.Instr)
	case instruction.Variable:
		return c.variable(x.Instr)
	case instruction.Parametric:
		return c.parametric(x.Instr)
	case instruction.Reference:
		return c.reference(x.Instr)
	}
	return invalidf("unexpected instruction %T", in)
}

func (c *funcCompiler) control(in instruction.ControlInstr) error {
	switch x := in.(type) {
	case instruction.Unreachable:
		c.emit(OpUnreachable, 0, 0, 0)
		c.setUnreachable()
	case instruction.Nop:
	case instruction.Br:
		target, err := c.label(x.LabelIdx)
		if err != nil {
			return err
		}
		c.br(target)
		if err := c.popN(target.labelTypes()); err != nil {
			return err
		}
		c.setUnreachable()
	case instruction.BrIf:
		cond, err := c.popExpect(types.I32)
		if err != nil {
			return err
		}
		pos := len(c.vals)
		target, err := c.label(x.LabelIdx)
		if err != nil {
			return err
		}
		// Type-check the label's values, which stay on the stack where
		// they are.
		lt := target.labelTypes()
		var kept []operand
		if c.reachable() && c.has(len(lt)) {
			for _, v := range c.vals[len(c.vals)-len(lt):] {
				kept = append(kept, v.op)
			}
		}
		if err := c.popN(lt); err != nil {
			return err
		}
		c.pushN(lt)
		for i, op := range kept {
			p := len(c.vals) - len(kept) + i
			c.vals[p].op = op
			if op.kind != inSlot && p < c.lazy {
				c.lazy = p
			}
		}
		c.brIf(target, cond, pos)
	case instruction.BrTable:
		return c.brTableOp(x)
	case instruction.Return:
		c.emitReturn()
		if err := c.popN(c.ctrls[0].results); err != nil {
			return err
		}
		c.setUnreachable()
	case instruction.Call:
		if x.FuncIdx >= uint32(len(c.mod.FuncTypes)) {
			return invalidf("unknown function %d", x.FuncIdx)
		}
		ft := c.mod.FuncTypes[x.FuncIdx]
		c.flushTop(len(ft.Params))
		if err := c.popN(ft.Params); err != nil {
			return err
		}
		base := c.slot(len(c.vals))
		c.pushN(ft.Results)
		c.emit(OpCall, x.FuncIdx, base, 0)
	case instruction.CallIndirect:
		if x.TableIdx >= uint32(len(c.mod.Tables)) {
			return invalidf("unknown table %d", x.TableIdx)
		}
		if c.mod.Tables[x.TableIdx].ElemType != types.FuncRef {
			return invalidf("type mismatch: call_indirect on a non-funcref table")
		}
		ft, err := c.mod.funcType(x.TypeIdx)
		if err != nil {
			return err
		}
		c.flushTop(len(ft.Params) + 1)
		if _, err := c.popExpect(types.I32); err != nil {
			return err
		}
		if err := c.popN(ft.Params); err != nil {
			return err
		}
		base := c.slot(len(c.vals))
		c.pushN(ft.Results)
		c.emit(OpCallIndirect, x.TypeIdx, x.TableIdx, base+uint32(len(ft.Params)))
	default:
		return invalidf("unexpected control instruction %T", in)
	}
	return nil
}

func (c *funcCompiler) pushConst(t types.ValueType, bits uint64) {
	c.pushOp(t, operand{kind: isConst, bits: bits})
}

func (c *funcCompiler) numeric(in instruction.NumericInstr) error {
	switch x := in.(type) {
	case instruction.I32Const:
		c.pushConst(types.I32, uint64(uint32(x.Value)))
		return nil
	case instruction.I64Const:
		c.pushConst(types.I64, uint64(x.Value))
		return nil
	case instruction.F32Const:
		c.pushConst(types.F32, uint64(math.Float32bits(x.Value)))
		return nil
	case instruction.F64Const:
		c.pushConst(types.F64, math.Float64bits(x.Value))
		return nil
	}
	if op, from, to, ok := satTrunc(in); ok {
		return c.unary(op, from, to)
	}
	wasmOp := byte(in.Opcode())
	if wasmOp < 0x45 || int(wasmOp-0x45) >= len(numericSigs) {
		return invalidf("unexpected numeric instruction %T", in)
	}
	sig := numericSigs[wasmOp-0x45]
	if sig.binary {
		return c.binary(numericOp(wasmOp), sig.in, sig.out)
	}
	return c.unary(numericOp(wasmOp), sig.in, sig.out)
}

func (c *funcCompiler) unary(op Op, in, out types.ValueType) error {
	a, err := c.popExpect(in)
	if err != nil {
		return err
	}
	pos := len(c.vals)
	if isNoop(op) {
		c.pushOp(out, a.op)
		return nil
	}
	c.push(out)
	c.emitDst(op, c.slot(pos), c.use(a, pos), 0)
	return nil
}

func (c *funcCompiler) binary(op Op, in, out types.ValueType) error {
	b, err := c.popExpect(in)
	if err != nil {
		return err
	}
	a, err := c.popExpect(in)
	if err != nil {
		return err
	}
	pos := len(c.vals)
	c.push(out)
	if !c.reachable() {
		return nil
	}
	if a.op.kind == isConst && b.op.kind != isConst && swapOp[op] != 0 {
		a, b, op = b, a, swapOp[op]
	}
	if b.op.kind == isConst {
		if imm, k, ok := immForm(op, uint32(b.op.bits)); ok {
			c.emitDst(imm, c.slot(pos), c.use(a, pos), k)
			return nil
		}
	}
	c.emitDst(op, c.slot(pos), c.use(a, pos), c.use(b, pos+1))
	return nil
}

// immForm returns the instruction computing op with the constant right
// operand k, and the immediate it takes.
func immForm(op Op, k uint32) (Op, uint32, bool) {
	switch op {
	case OpI32Sub:
		return OpI32AddImm, -k, true
	case OpI32Shl, OpI32ShrS, OpI32ShrU, OpI32Rotl:
		k &= 31
	case OpI32Rotr:
		return OpI32RotlImm, -k & 31, true
	case OpI32DivU, OpI32RemU:
		if k == 0 {
			return 0, 0, false // traps at run time
		}
	}
	imm := immOp[op]
	return imm, k, imm != 0
}

func (c *funcCompiler) needMemory(idx uint32) error {
	if idx >= uint32(len(c.mod.Memories)) {
		return invalidf("unknown memory %d", idx)
	}
	return nil
}

func (c *funcCompiler) needData(idx uint32) error {
	if c.mod.DataCount == nil {
		return invalidf("data count section required")
	}
	if idx >= *c.mod.DataCount {
		return invalidf("unknown data segment %d", idx)
	}
	return nil
}

var i32x3 = []types.ValueType{types.I32, types.I32, types.I32}

// stackOpBase is the first operand slot of a rare instruction that takes
// its operands in consecutive slots, once they are flushed and popped.
func (c *funcCompiler) stackOpBase() uint32 { return c.slot(len(c.vals)) }

func (c *funcCompiler) memory(in instruction.MemoryInstr) error {
	if acc, arg, ok := memAccessOf(in); ok {
		if err := c.needMemory(0); err != nil {
			return err
		}
		if arg.Align >= 32 || uint64(1)<<arg.Align > uint64(acc.size) {
			return invalidf("alignment must not be larger than natural")
		}
		if acc.store {
			val, err := c.popExpect(acc.typ)
			if err != nil {
				return err
			}
			addr, err := c.popExpect(types.I32)
			if err != nil {
				return err
			}
			pos := len(c.vals)
			c.emit(acc.op, c.use(addr, pos), c.use(val, pos+1), arg.Offset)
		} else {
			addr, err := c.popExpect(types.I32)
			if err != nil {
				return err
			}
			pos := len(c.vals)
			c.push(acc.typ)
			c.emitDst(acc.op, c.slot(pos), c.use(addr, pos), arg.Offset)
		}
		return nil
	}

	switch x := in.(type) {
	case instruction.MemorySize:
		if err := c.needMemory(x.MemIdx); err != nil {
			return err
		}
		pos := len(c.vals)
		c.push(types.I32)
		c.emitDst(OpMemorySize, c.slot(pos), 0, 0)
	case instruction.MemoryGrow:
		if err := c.needMemory(x.MemIdx); err != nil {
			return err
		}
		delta, err := c.popExpect(types.I32)
		if err != nil {
			return err
		}
		pos := len(c.vals)
		c.push(types.I32)
		c.emit(OpMemoryGrow, c.slot(pos), c.use(delta, pos), 0)
	case instruction.MemoryInit:
		if err := c.needMemory(x.MemIdx); err != nil {
			return err
		}
		if err := c.needData(x.DataIdx); err != nil {
			return err
		}
		c.flushTop(3)
		if err := c.popN(i32x3); err != nil {
			return err
		}
		c.emit(OpMemoryInit, x.DataIdx, c.stackOpBase(), 0)
	case instruction.DataDrop:
		if err := c.needData(x.DataIdx); err != nil {
			return err
		}
		c.emit(OpDataDrop, x.DataIdx, 0, 0)
	case instruction.MemoryCopy:
		if err := c.needMemory(x.DstMemIdx); err != nil {
			return err
		}
		if err := c.needMemory(x.SrcMemIdx); err != nil {
			return err
		}
		c.flushTop(3)
		if err := c.popN(i32x3); err != nil {
			return err
		}
		c.emit(OpMemoryCopy, 0, c.stackOpBase(), 0)
	case instruction.MemoryFill:
		if err := c.needMemory(x.MemIdx); err != nil {
			return err
		}
		c.flushTop(3)
		if err := c.popN(i32x3); err != nil {
			return err
		}
		c.emit(OpMemoryFill, 0, c.stackOpBase(), 0)
	default:
		return invalidf("unexpected memory instruction %T", in)
	}
	return nil
}

func (c *funcCompiler) local(idx uint32) (types.ValueType, error) {
	if idx >= uint32(len(c.locals)) {
		return 0, invalidf("unknown local %d", idx)
	}
	return c.locals[idx], nil
}

// setLocal stores v in local x. It reports whether it redirected the
// instruction that computed v, so that v is now only in x.
func (c *funcCompiler) setLocal(x uint32, v stackVal) bool {
	if !c.reachable() {
		return false
	}
	c.preserve(x)
	if v.op.kind == inSlot && c.last >= 0 && c.code[c.last].A == v.op.slot {
		c.code[c.last].A = x
		c.last = -1
		return true
	}
	c.load(x, v.op)
	return false
}

func (c *funcCompiler) table(idx uint32) (types.TableType, error) {
	if idx >= uint32(len(c.mod.Tables)) {
		return types.TableType{}, invalidf("unknown table %d", idx)
	}
	return c.mod.Tables[idx], nil
}

func (c *funcCompiler) elem(idx uint32) (types.ValueType, error) {
	if idx >= uint32(len(c.mod.Elements)) {
		return 0, invalidf("unknown elem segment %d", idx)
	}
	return c.mod.Elements[idx].Type, nil
}

func (c *funcCompiler) variable(in instruction.VariableInstr) error {
	switch x := in.(type) {
	case instruction.LocalGet:
		t, err := c.local(x.LocalIdx)
		if err != nil {
			return err
		}
		c.pushOp(t, operand{kind: inLocal, slot: x.LocalIdx})
	case instruction.LocalSet:
		t, err := c.local(x.LocalIdx)
		if err != nil {
			return err
		}
		v, err := c.popExpect(t)
		if err != nil {
			return err
		}
		c.setLocal(x.LocalIdx, v)
	case instruction.LocalTee:
		t, err := c.local(x.LocalIdx)
		if err != nil {
			return err
		}
		v, err := c.popExpect(t)
		if err != nil {
			return err
		}
		if c.setLocal(x.LocalIdx, v) {
			c.pushOp(t, operand{kind: inLocal, slot: x.LocalIdx})
		} else {
			c.pushOp(t, v.op)
		}
	case instruction.GlobalGet:
		if x.GlobalIdx >= uint32(len(c.mod.Globals)) {
			return invalidf("unknown global %d", x.GlobalIdx)
		}
		pos := len(c.vals)
		c.push(c.mod.Globals[x.GlobalIdx].ValType)
		c.emitDst(OpGlobalGet, c.slot(pos), x.GlobalIdx, 0)
	case instruction.GlobalSet:
		if x.GlobalIdx >= uint32(len(c.mod.Globals)) {
			return invalidf("unknown global %d", x.GlobalIdx)
		}
		g := c.mod.Globals[x.GlobalIdx]
		if !g.Mutable {
			return invalidf("global is immutable")
		}
		v, err := c.popExpect(g.ValType)
		if err != nil {
			return err
		}
		c.emit(OpGlobalSet, x.GlobalIdx, c.use(v, len(c.vals)), 0)
	case instruction.TableGet:
		t, err := c.table(x.TableIdx)
		if err != nil {
			return err
		}
		c.flushTop(1)
		if _, err := c.popExpect(types.I32); err != nil {
			return err
		}
		base := c.stackOpBase()
		c.push(t.ElemType)
		c.emit(OpTableGet, x.TableIdx, base, 0)
	case instruction.TableSet:
		t, err := c.table(x.TableIdx)
		if err != nil {
			return err
		}
		c.flushTop(2)
		if err := c.popN([]types.ValueType{types.I32, t.ElemType}); err != nil {
			return err
		}
		c.emit(OpTableSet, x.TableIdx, c.stackOpBase(), 0)
	case instruction.TableSize:
		if _, err := c.table(x.TableIdx); err != nil {
			return err
		}
		base := c.stackOpBase()
		c.push(types.I32)
		c.emit(OpTableSize, x.TableIdx, base, 0)
	case instruction.TableGrow:
		t, err := c.table(x.TableIdx)
		if err != nil {
			return err
		}
		c.flushTop(2)
		if err := c.popN([]types.ValueType{t.ElemType, types.I32}); err != nil {
			return err
		}
		base := c.stackOpBase()
		c.push(types.I32)
		c.emit(OpTableGrow, x.TableIdx, base, 0)
	case instruction.TableFill:
		t, err := c.table(x.TableIdx)
		if err != nil {
			return err
		}
		c.flushTop(3)
		if err := c.popN([]types.ValueType{types.I32, t.ElemType, types.I32}); err != nil {
			return err
		}
		c.emit(OpTableFill, x.TableIdx, c.stackOpBase(), 0)
	case instruction.TableCopy:
		dst, err := c.table(x.DstTableIdx)
		if err != nil {
			return err
		}
		src, err := c.table(x.SrcTableIdx)
		if err != nil {
			return err
		}
		if dst.ElemType != src.ElemType {
			return invalidf("type mismatch: table.copy between different element types")
		}
		c.flushTop(3)
		if err := c.popN(i32x3); err != nil {
			return err
		}
		c.emit(OpTableCopy, x.DstTableIdx, c.stackOpBase(), x.SrcTableIdx)
	case instruction.TableInit:
		t, err := c.table(x.TableIdx)
		if err != nil {
			return err
		}
		et, err := c.elem(x.ElemIdx)
		if err != nil {
			return err
		}
		if et != t.ElemType {
			return invalidf("type mismatch: table.init element type")
		}
		c.flushTop(3)
		if err := c.popN(i32x3); err != nil {
			return err
		}
		c.emit(OpTableInit, x.ElemIdx, c.stackOpBase(), x.TableIdx)
	case instruction.ElemDrop:
		if _, err := c.elem(x.ElemIdx); err != nil {
			return err
		}
		c.emit(OpElemDrop, x.ElemIdx, 0, 0)
	default:
		return invalidf("unexpected variable instruction %T", in)
	}
	return nil
}

func isNumType(t types.ValueType) bool {
	switch t {
	case types.I32, types.I64, types.F32, types.F64, unknown:
		return true
	}
	return false
}

// selectOp emits select once its operands are popped. v1 was at position
// pos.
func (c *funcCompiler) selectOp(v1, v2, cond stackVal, pos int) {
	if !c.reachable() {
		return
	}
	dst := c.slot(pos)
	c.load(dst, v1.op)
	c.emit(OpSelect, dst, c.use(v2, pos+1), c.use(cond, pos+2))
}

func (c *funcCompiler) parametric(in instruction.ParametricInstr) error {
	switch x := in.(type) {
	case instruction.Drop:
		if _, err := c.pop(); err != nil {
			return err
		}
	case instruction.Select:
		cond, err := c.popExpect(types.I32)
		if err != nil {
			return err
		}
		v2, err := c.pop()
		if err != nil {
			return err
		}
		v1, err := c.pop()
		if err != nil {
			return err
		}
		t1, t2 := v2.t, v1.t
		if !isNumType(t1) || !isNumType(t2) {
			return invalidf("type mismatch: untyped select needs numeric operands")
		}
		if t1 != t2 && t1 != unknown && t2 != unknown {
			return invalidf("type mismatch: select operands differ")
		}
		if t1 == unknown {
			t1 = t2
		}
		pos := len(c.vals)
		c.push(t1)
		c.selectOp(v1, v2, cond, pos)
	case instruction.SelectT:
		if len(x.Types) != 1 {
			return invalidf("invalid result arity for select")
		}
		t := x.Types[0]
		if !validValueType(t) {
			return invalidf("unsupported select type")
		}
		cond, err := c.popExpect(types.I32)
		if err != nil {
			return err
		}
		v2, err := c.popExpect(t)
		if err != nil {
			return err
		}
		v1, err := c.popExpect(t)
		if err != nil {
			return err
		}
		pos := len(c.vals)
		c.push(t)
		c.selectOp(v1, v2, cond, pos)
	default:
		return invalidf("unexpected parametric instruction %T", in)
	}
	return nil
}

func (c *funcCompiler) reference(in instruction.ReferenceInstr) error {
	switch x := in.(type) {
	case instruction.RefNull:
		if !isRefType(x.Type) {
			return invalidf("malformed reference type")
		}
		c.pushConst(x.Type, 0)
	case instruction.RefIsNull:
		v, err := c.pop()
		if err != nil {
			return err
		}
		if v.t != unknown && !isRefType(v.t) {
			return invalidf("type mismatch: ref.is_null needs a reference")
		}
		pos := len(c.vals)
		c.push(types.I32)
		c.emitDst(OpRefIsNull, c.slot(pos), c.use(v, pos), 0)
	case instruction.RefFunc:
		if x.FuncIdx >= uint32(len(c.mod.FuncTypes)) {
			return invalidf("unknown function %d", x.FuncIdx)
		}
		if !c.refs[x.FuncIdx] {
			return invalidf("undeclared function reference")
		}
		pos := len(c.vals)
		c.push(types.FuncRef)
		c.emitDst(OpRefFunc, c.slot(pos), x.FuncIdx, 0)
	default:
		return invalidf("unexpected reference instruction %T", in)
	}
	return nil
}
