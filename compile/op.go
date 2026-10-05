// Package compile validates a parsed module and lowers every function body to
// flat bytecode for the interpreter in package runtime.
//
// The bytecode is a register machine over 64-bit slots. A function frame is
// NumLocals+MaxHeight slots: its locals (parameters first) in slots
// 0 … NumLocals-1, then one slot per wasm operand stack position, so the
// value at stack height h lives in slot NumLocals+h. Operands name slots
// relative to the frame, and since every stack height is known statically,
// the interpreter keeps no stack pointer. Every value takes one slot: i32
// and f32 are stored zero-extended, which makes the reinterpret and
// i64.extend_i32_u instructions no-ops, and references are 0 for null.
//
// local.get and constants usually cost nothing: the compiler tracks which
// stack values are still just a local or a constant, and the instruction
// that consumes one reads the local's slot or takes the constant as an
// immediate. Such values are copied into their own slots only where control
// flow needs the stack in place (block boundaries, branches and call
// arguments) or before the local they mirror is overwritten. A local.set
// usually redirects the instruction that computed its value instead of
// copying.
//
// Control flow is resolved at compile time. Blocks disappear, if/else becomes
// conditional jumps, and every branch carries its target pc plus the slots
// that carry the label's values.
package compile

import "fmt"

// Op is a bytecode opcode. It is not the wasm opcode: loads, stores and
// constants that are identical at the bit level share one Op.
type Op uint16

// Instr is one bytecode instruction. The meaning of A, B, C and D depends on
// Op and is listed next to each constant. Unless noted otherwise, A is the
// destination slot and B and C are operand slots.
type Instr struct {
	Op Op
	D  uint16
	A  uint32
	B  uint32
	C  uint32
}

// BrTarget is one br_table entry, or the target of an OpBrIf that moves
// values. N values move from slot Src to slot Dst before the jump.
type BrTarget struct {
	PC       uint32
	N        uint32
	Src, Dst uint32
}

const (
	OpUnreachable  Op = iota
	OpLoopHeader      // first instruction of every loop; checks for interrupts
	OpJump            // A: target pc
	OpBr              // A: target pc. Moves D values from slot B to slot C first
	OpBrIf            // A: BrTable entry (target and moves), B: condition. Branches when it is nonzero
	OpBrIfNez         // A: target pc, B: condition. Jumps when it is nonzero
	OpBrIfEqz         // A: target pc, B: condition. Jumps when it is zero
	OpBrTable         // A: first entry in Func.BrTable, B: index, C: entry count (the last is the default)
	OpReturn          // A: first result slot
	OpCall            // A: function index, B: first argument slot, where the callee's frame starts
	OpCallIndirect    // A: type index, B: table index, C: element index, with the arguments right below it

	OpSelect // A: first operand and result, B: second operand, C: condition
	OpCopy   // A ← B
	OpConst  // A ← B | C<<32

	OpGlobalGet // A ← global B
	OpGlobalSet // global A ← B

	// Rare instructions take their operands from consecutive slots starting
	// at B, and leave any result in B.
	OpTableGet  // A: table index. B: index → ref
	OpTableSet  // A: table index. B: index, ref
	OpTableSize // A: table index. B ← size
	OpTableGrow // A: table index. B: ref, delta → old size
	OpTableFill // A: table index. B: dest, ref, count
	OpTableCopy // A: destination table, C: source table. B: dest, src, count
	OpTableInit // A: element segment, C: table index. B: dest, src, count
	OpElemDrop  // A: element segment

	OpRefIsNull // A ← B == null
	OpRefFunc   // A ← reference to function B

	// Loads: A ← memory[B + C]. Stores: memory[A + C] ← B. C is the static
	// offset. All touch memory 0.
	OpLoad8U  // i32.load8_u, i64.load8_u
	OpLoad16U // i32.load16_u, i64.load16_u
	OpLoad32  // i32.load, f32.load, i64.load32_u
	OpLoad64  // i64.load, f64.load
	OpI32Load8S
	OpI32Load16S
	OpI64Load8S
	OpI64Load16S
	OpI64Load32S
	OpStore8  // i32.store8, i64.store8
	OpStore16 // i32.store16, i64.store16
	OpStore32 // i32.store, f32.store, i64.store32
	OpStore64 // i64.store, f64.store

	OpMemorySize // A ← size
	OpMemoryGrow // A ← grow by B
	OpMemoryInit // A: data segment. B: dest, src, count
	OpDataDrop   // A: data segment
	OpMemoryCopy // B: dest, src, count
	OpMemoryFill // B: dest, value, count

	// Numeric instructions, in wasm opcode order 0x45…0xC4 so that
	// numericOp maps them arithmetically. Unary ones are A ← op B, binary
	// ones A ← B op C. The reinterprets and i64.extend_i32_u are never
	// emitted (see the package doc).
	OpI32Eqz
	OpI32Eq
	OpI32Ne
	OpI32LtS
	OpI32LtU
	OpI32GtS
	OpI32GtU
	OpI32LeS
	OpI32LeU
	OpI32GeS
	OpI32GeU
	OpI64Eqz
	OpI64Eq
	OpI64Ne
	OpI64LtS
	OpI64LtU
	OpI64GtS
	OpI64GtU
	OpI64LeS
	OpI64LeU
	OpI64GeS
	OpI64GeU
	OpF32Eq
	OpF32Ne
	OpF32Lt
	OpF32Gt
	OpF32Le
	OpF32Ge
	OpF64Eq
	OpF64Ne
	OpF64Lt
	OpF64Gt
	OpF64Le
	OpF64Ge
	OpI32Clz
	OpI32Ctz
	OpI32Popcnt
	OpI32Add
	OpI32Sub
	OpI32Mul
	OpI32DivS
	OpI32DivU
	OpI32RemS
	OpI32RemU
	OpI32And
	OpI32Or
	OpI32Xor
	OpI32Shl
	OpI32ShrS
	OpI32ShrU
	OpI32Rotl
	OpI32Rotr
	OpI64Clz
	OpI64Ctz
	OpI64Popcnt
	OpI64Add
	OpI64Sub
	OpI64Mul
	OpI64DivS
	OpI64DivU
	OpI64RemS
	OpI64RemU
	OpI64And
	OpI64Or
	OpI64Xor
	OpI64Shl
	OpI64ShrS
	OpI64ShrU
	OpI64Rotl
	OpI64Rotr
	OpF32Abs
	OpF32Neg
	OpF32Ceil
	OpF32Floor
	OpF32Trunc
	OpF32Nearest
	OpF32Sqrt
	OpF32Add
	OpF32Sub
	OpF32Mul
	OpF32Div
	OpF32Min
	OpF32Max
	OpF32Copysign
	OpF64Abs
	OpF64Neg
	OpF64Ceil
	OpF64Floor
	OpF64Trunc
	OpF64Nearest
	OpF64Sqrt
	OpF64Add
	OpF64Sub
	OpF64Mul
	OpF64Div
	OpF64Min
	OpF64Max
	OpF64Copysign
	OpI32WrapI64
	OpI32TruncF32S
	OpI32TruncF32U
	OpI32TruncF64S
	OpI32TruncF64U
	OpI64ExtendI32S
	OpI64ExtendI32U // never emitted
	OpI64TruncF32S
	OpI64TruncF32U
	OpI64TruncF64S
	OpI64TruncF64U
	OpF32ConvertI32S
	OpF32ConvertI32U
	OpF32ConvertI64S
	OpF32ConvertI64U
	OpF32DemoteF64
	OpF64ConvertI32S
	OpF64ConvertI32U
	OpF64ConvertI64S
	OpF64ConvertI64U
	OpF64PromoteF32
	OpI32ReinterpretF32 // never emitted
	OpI64ReinterpretF64 // never emitted
	OpF32ReinterpretI32 // never emitted
	OpF64ReinterpretI64 // never emitted
	OpI32Extend8S
	OpI32Extend16S
	OpI64Extend8S
	OpI64Extend16S
	OpI64Extend32S

	OpI32TruncSatF32S
	OpI32TruncSatF32U
	OpI32TruncSatF64S
	OpI32TruncSatF64U
	OpI64TruncSatF32S
	OpI64TruncSatF32U
	OpI64TruncSatF64S
	OpI64TruncSatF64U

	// i32 binary instructions with a constant right operand: A ← B op C.
	// i32.sub uses OpI32AddImm, shift and rotate counts are already masked,
	// and the divisor of OpI32DivUImm and OpI32RemUImm is never zero.
	OpI32AddImm
	OpI32MulImm
	OpI32DivUImm
	OpI32RemUImm
	OpI32AndImm
	OpI32OrImm
	OpI32XorImm
	OpI32ShlImm
	OpI32ShrSImm
	OpI32ShrUImm
	OpI32RotlImm
	OpI32EqImm
	OpI32NeImm
	OpI32LtSImm
	OpI32LtUImm
	OpI32GtSImm
	OpI32GtUImm
	OpI32LeSImm
	OpI32LeUImm
	OpI32GeSImm
	OpI32GeUImm

	// Fused i32 compare and branch: A is the target pc, and the jump is
	// taken when B compares true against slot C (or against the constant C
	// for the Imm forms).
	OpBrIfI32Eq
	OpBrIfI32Ne
	OpBrIfI32LtS
	OpBrIfI32LtU
	OpBrIfI32GtS
	OpBrIfI32GtU
	OpBrIfI32LeS
	OpBrIfI32LeU
	OpBrIfI32GeS
	OpBrIfI32GeU
	OpBrIfI32EqImm
	OpBrIfI32NeImm
	OpBrIfI32LtSImm
	OpBrIfI32LtUImm
	OpBrIfI32GtSImm
	OpBrIfI32GtUImm
	OpBrIfI32LeSImm
	OpBrIfI32LeUImm
	OpBrIfI32GeSImm
	OpBrIfI32GeUImm

	// Loads from B plus the constant C, wrapped to 32 bits: A ← memory[(B + C) + D].
	// D is the static offset.
	OpLoad8UAdd
	OpLoad16UAdd
	OpLoad32Add
	OpLoad64Add

	numOps
)

// numericOp maps a wasm numeric opcode (0x45…0xC4) to its Op.
func numericOp(wasmOp byte) Op { return OpI32Eqz + Op(wasmOp-0x45) }

// immOp gives the constant-operand form of an i32 binary instruction.
var immOp = [numOps]Op{
	OpI32Add: OpI32AddImm, OpI32Mul: OpI32MulImm, OpI32DivU: OpI32DivUImm, OpI32RemU: OpI32RemUImm,
	OpI32And: OpI32AndImm, OpI32Or: OpI32OrImm, OpI32Xor: OpI32XorImm,
	OpI32Shl: OpI32ShlImm, OpI32ShrS: OpI32ShrSImm, OpI32ShrU: OpI32ShrUImm, OpI32Rotl: OpI32RotlImm,
	OpI32Eq: OpI32EqImm, OpI32Ne: OpI32NeImm,
	OpI32LtS: OpI32LtSImm, OpI32LtU: OpI32LtUImm, OpI32GtS: OpI32GtSImm, OpI32GtU: OpI32GtUImm,
	OpI32LeS: OpI32LeSImm, OpI32LeU: OpI32LeUImm, OpI32GeS: OpI32GeSImm, OpI32GeU: OpI32GeUImm,
}

// swapOp gives the instruction that computes the same result with the
// operands swapped, for the i32 instructions that have one.
var swapOp = [numOps]Op{
	OpI32Add: OpI32Add, OpI32Mul: OpI32Mul, OpI32And: OpI32And, OpI32Or: OpI32Or, OpI32Xor: OpI32Xor,
	OpI32Eq: OpI32Eq, OpI32Ne: OpI32Ne,
	OpI32LtS: OpI32GtS, OpI32LtU: OpI32GtU, OpI32GtS: OpI32LtS, OpI32GtU: OpI32LtU,
	OpI32LeS: OpI32GeS, OpI32LeU: OpI32GeU, OpI32GeS: OpI32LeS, OpI32GeU: OpI32LeU,
}

// brIfOp gives the fused branch for an i32 comparison, and brUnlessOp the
// one that branches when the comparison is false.
var brIfOp, brUnlessOp [numOps]Op

func init() {
	cmps := []Op{OpI32Eq, OpI32Ne, OpI32LtS, OpI32LtU, OpI32GtS, OpI32GtU, OpI32LeS, OpI32LeU, OpI32GeS, OpI32GeU}
	negs := []Op{OpI32Ne, OpI32Eq, OpI32GeS, OpI32GeU, OpI32LeS, OpI32LeU, OpI32GtS, OpI32GtU, OpI32LtS, OpI32LtU}
	for i, op := range cmps {
		neg := negs[i]
		brIfOp[op] = OpBrIfI32Eq + (op - OpI32Eq)
		brUnlessOp[op] = OpBrIfI32Eq + (neg - OpI32Eq)
		brIfOp[immOp[op]] = OpBrIfI32EqImm + (op - OpI32Eq)
		brUnlessOp[immOp[op]] = OpBrIfI32EqImm + (neg - OpI32Eq)
	}
	brIfOp[OpI32Eqz] = OpBrIfEqz
	brUnlessOp[OpI32Eqz] = OpBrIfNez
}

var opNames = [numOps]string{
	OpUnreachable: "unreachable", OpLoopHeader: "loop_header", OpJump: "jump",
	OpBr: "br", OpBrIf: "br_if", OpBrIfNez: "br_if_nez", OpBrIfEqz: "br_if_eqz",
	OpBrTable: "br_table", OpReturn: "return", OpCall: "call", OpCallIndirect: "call_indirect",
	OpSelect: "select", OpCopy: "copy", OpConst: "const",
	OpGlobalGet: "global.get", OpGlobalSet: "global.set",
	OpTableGet: "table.get", OpTableSet: "table.set", OpTableSize: "table.size",
	OpTableGrow: "table.grow", OpTableFill: "table.fill", OpTableCopy: "table.copy",
	OpTableInit: "table.init", OpElemDrop: "elem.drop",
	OpRefIsNull: "ref.is_null", OpRefFunc: "ref.func",
	OpLoad8U: "load8_u", OpLoad16U: "load16_u", OpLoad32: "load32", OpLoad64: "load64",
	OpLoad8UAdd: "load8_u_add", OpLoad16UAdd: "load16_u_add", OpLoad32Add: "load32_add", OpLoad64Add: "load64_add",
	OpI32Load8S: "i32.load8_s", OpI32Load16S: "i32.load16_s", OpI64Load8S: "i64.load8_s",
	OpI64Load16S: "i64.load16_s", OpI64Load32S: "i64.load32_s",
	OpStore8: "store8", OpStore16: "store16", OpStore32: "store32", OpStore64: "store64",
	OpMemorySize: "memory.size", OpMemoryGrow: "memory.grow", OpMemoryInit: "memory.init",
	OpDataDrop: "data.drop", OpMemoryCopy: "memory.copy", OpMemoryFill: "memory.fill",
	OpI32TruncSatF32S: "i32.trunc_sat_f32_s", OpI32TruncSatF32U: "i32.trunc_sat_f32_u",
	OpI32TruncSatF64S: "i32.trunc_sat_f64_s", OpI32TruncSatF64U: "i32.trunc_sat_f64_u",
	OpI64TruncSatF32S: "i64.trunc_sat_f32_s", OpI64TruncSatF32U: "i64.trunc_sat_f32_u",
	OpI64TruncSatF64S: "i64.trunc_sat_f64_s", OpI64TruncSatF64U: "i64.trunc_sat_f64_u",
}

func init() {
	for i := 0; i < len(numericSigs); i++ {
		opNames[numericOp(byte(0x45+i))] = numericSigs[i].name
	}
	for op, imm := range immOp {
		if imm == 0 {
			continue
		}
		opNames[imm] = opNames[op] + "_imm"
		if brIfOp[op] != 0 {
			opNames[brIfOp[op]] = "br_if_" + opNames[op]
			opNames[brIfOp[imm]] = "br_if_" + opNames[op] + "_imm"
		}
	}
}

func (op Op) String() string {
	if op < numOps && opNames[op] != "" {
		return opNames[op]
	}
	return fmt.Sprintf("op(%d)", uint16(op))
}

// isBinary reports whether op is a numeric instruction A ← B op C.
func isBinary(op Op) bool {
	if op < OpI32Eqz || op > OpI64Extend32S {
		return false
	}
	return numericSigs[op-OpI32Eqz].binary
}

func isUnary(op Op) bool {
	return op >= OpI32Eqz && op <= OpI64Extend32S && !isBinary(op) ||
		op >= OpI32TruncSatF32S && op <= OpI64TruncSatF64U
}

func (in Instr) String() string {
	r := func(slot uint32) string { return fmt.Sprintf("r%d", slot) }
	switch op := in.Op; {
	case op == OpJump:
		return fmt.Sprintf("jump %d", in.A)
	case op == OpBr:
		return fmt.Sprintf("br %d move %d %s→%s", in.A, in.D, r(in.B), r(in.C))
	case op == OpBrIf:
		return fmt.Sprintf("br_if %s target[%d]", r(in.B), in.A)
	case op == OpBrIfNez, op == OpBrIfEqz:
		return fmt.Sprintf("%s %s %d", op, r(in.B), in.A)
	case op >= OpBrIfI32Eq && op <= OpBrIfI32GeU:
		return fmt.Sprintf("%s %s %s %d", op, r(in.B), r(in.C), in.A)
	case op >= OpBrIfI32EqImm && op <= OpBrIfI32GeUImm:
		return fmt.Sprintf("%s %s 0x%x %d", op, r(in.B), in.C, in.A)
	case op == OpBrTable:
		return fmt.Sprintf("br_table %s targets[%d:+%d]", r(in.B), in.A, in.C)
	case op == OpReturn:
		return fmt.Sprintf("return %s", r(in.A))
	case op == OpCall:
		return fmt.Sprintf("call %d frame=%s", in.A, r(in.B))
	case op == OpCallIndirect:
		return fmt.Sprintf("call_indirect type=%d table=%d %s", in.A, in.B, r(in.C))
	case op == OpSelect:
		return fmt.Sprintf("%s = select %s %s %s", r(in.A), r(in.A), r(in.B), r(in.C))
	case op == OpCopy:
		return fmt.Sprintf("%s = %s", r(in.A), r(in.B))
	case op == OpConst:
		return fmt.Sprintf("%s = 0x%x", r(in.A), uint64(in.B)|uint64(in.C)<<32)
	case op == OpGlobalGet:
		return fmt.Sprintf("%s = global %d", r(in.A), in.B)
	case op == OpGlobalSet:
		return fmt.Sprintf("global %d = %s", in.A, r(in.B))
	case op == OpRefIsNull:
		return fmt.Sprintf("%s = ref.is_null %s", r(in.A), r(in.B))
	case op == OpRefFunc:
		return fmt.Sprintf("%s = ref.func %d", r(in.A), in.B)
	case op == OpMemorySize:
		return fmt.Sprintf("%s = memory.size", r(in.A))
	case op == OpMemoryGrow:
		return fmt.Sprintf("%s = memory.grow %s", r(in.A), r(in.B))
	case op >= OpLoad8U && op <= OpI64Load32S:
		return fmt.Sprintf("%s = %s %s+%d", r(in.A), op, r(in.B), in.C)
	case op >= OpLoad8UAdd && op <= OpLoad64Add:
		return fmt.Sprintf("%s = %s (%s+0x%x)+%d", r(in.A), op, r(in.B), in.C, in.D)
	case op >= OpStore8 && op <= OpStore64:
		return fmt.Sprintf("%s %s+%d %s", op, r(in.A), in.C, r(in.B))
	case op >= OpTableGet && op <= OpTableInit, op == OpMemoryInit, op == OpMemoryCopy, op == OpMemoryFill:
		return fmt.Sprintf("%s %d %d args=%s", op, in.A, in.C, r(in.B))
	case op == OpElemDrop, op == OpDataDrop:
		return fmt.Sprintf("%s %d", op, in.A)
	case op >= OpI32AddImm && op <= OpI32GeUImm:
		return fmt.Sprintf("%s = %s %s 0x%x", r(in.A), op, r(in.B), in.C)
	case isBinary(op):
		return fmt.Sprintf("%s = %s %s %s", r(in.A), op, r(in.B), r(in.C))
	case isUnary(op):
		return fmt.Sprintf("%s = %s %s", r(in.A), op, r(in.B))
	}
	return in.Op.String()
}
