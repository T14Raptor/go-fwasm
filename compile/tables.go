package compile

import (
	"github.com/t14raptor/go-fwasm/instruction"
	"github.com/t14raptor/go-fwasm/types"
)

const (
	i32 = types.I32
	i64 = types.I64
	f32 = types.F32
	f64 = types.F64
)

// numericSig is the type of a numeric instruction: one or two operands of
// type in, and one result.
type numericSig struct {
	name   string
	in     types.ValueType
	binary bool
	out    types.ValueType
}

// numericSigs covers wasm opcodes 0x45…0xC4, indexed by opcode-0x45.
var numericSigs = [...]numericSig{
	{"i32.eqz", i32, false, i32},
	{"i32.eq", i32, true, i32}, {"i32.ne", i32, true, i32},
	{"i32.lt_s", i32, true, i32}, {"i32.lt_u", i32, true, i32},
	{"i32.gt_s", i32, true, i32}, {"i32.gt_u", i32, true, i32},
	{"i32.le_s", i32, true, i32}, {"i32.le_u", i32, true, i32},
	{"i32.ge_s", i32, true, i32}, {"i32.ge_u", i32, true, i32},
	{"i64.eqz", i64, false, i32},
	{"i64.eq", i64, true, i32}, {"i64.ne", i64, true, i32},
	{"i64.lt_s", i64, true, i32}, {"i64.lt_u", i64, true, i32},
	{"i64.gt_s", i64, true, i32}, {"i64.gt_u", i64, true, i32},
	{"i64.le_s", i64, true, i32}, {"i64.le_u", i64, true, i32},
	{"i64.ge_s", i64, true, i32}, {"i64.ge_u", i64, true, i32},
	{"f32.eq", f32, true, i32}, {"f32.ne", f32, true, i32},
	{"f32.lt", f32, true, i32}, {"f32.gt", f32, true, i32},
	{"f32.le", f32, true, i32}, {"f32.ge", f32, true, i32},
	{"f64.eq", f64, true, i32}, {"f64.ne", f64, true, i32},
	{"f64.lt", f64, true, i32}, {"f64.gt", f64, true, i32},
	{"f64.le", f64, true, i32}, {"f64.ge", f64, true, i32},
	{"i32.clz", i32, false, i32}, {"i32.ctz", i32, false, i32}, {"i32.popcnt", i32, false, i32},
	{"i32.add", i32, true, i32}, {"i32.sub", i32, true, i32}, {"i32.mul", i32, true, i32},
	{"i32.div_s", i32, true, i32}, {"i32.div_u", i32, true, i32},
	{"i32.rem_s", i32, true, i32}, {"i32.rem_u", i32, true, i32},
	{"i32.and", i32, true, i32}, {"i32.or", i32, true, i32}, {"i32.xor", i32, true, i32},
	{"i32.shl", i32, true, i32}, {"i32.shr_s", i32, true, i32}, {"i32.shr_u", i32, true, i32},
	{"i32.rotl", i32, true, i32}, {"i32.rotr", i32, true, i32},
	{"i64.clz", i64, false, i64}, {"i64.ctz", i64, false, i64}, {"i64.popcnt", i64, false, i64},
	{"i64.add", i64, true, i64}, {"i64.sub", i64, true, i64}, {"i64.mul", i64, true, i64},
	{"i64.div_s", i64, true, i64}, {"i64.div_u", i64, true, i64},
	{"i64.rem_s", i64, true, i64}, {"i64.rem_u", i64, true, i64},
	{"i64.and", i64, true, i64}, {"i64.or", i64, true, i64}, {"i64.xor", i64, true, i64},
	{"i64.shl", i64, true, i64}, {"i64.shr_s", i64, true, i64}, {"i64.shr_u", i64, true, i64},
	{"i64.rotl", i64, true, i64}, {"i64.rotr", i64, true, i64},
	{"f32.abs", f32, false, f32}, {"f32.neg", f32, false, f32},
	{"f32.ceil", f32, false, f32}, {"f32.floor", f32, false, f32},
	{"f32.trunc", f32, false, f32}, {"f32.nearest", f32, false, f32},
	{"f32.sqrt", f32, false, f32},
	{"f32.add", f32, true, f32}, {"f32.sub", f32, true, f32},
	{"f32.mul", f32, true, f32}, {"f32.div", f32, true, f32},
	{"f32.min", f32, true, f32}, {"f32.max", f32, true, f32},
	{"f32.copysign", f32, true, f32},
	{"f64.abs", f64, false, f64}, {"f64.neg", f64, false, f64},
	{"f64.ceil", f64, false, f64}, {"f64.floor", f64, false, f64},
	{"f64.trunc", f64, false, f64}, {"f64.nearest", f64, false, f64},
	{"f64.sqrt", f64, false, f64},
	{"f64.add", f64, true, f64}, {"f64.sub", f64, true, f64},
	{"f64.mul", f64, true, f64}, {"f64.div", f64, true, f64},
	{"f64.min", f64, true, f64}, {"f64.max", f64, true, f64},
	{"f64.copysign", f64, true, f64},
	{"i32.wrap_i64", i64, false, i32},
	{"i32.trunc_f32_s", f32, false, i32}, {"i32.trunc_f32_u", f32, false, i32},
	{"i32.trunc_f64_s", f64, false, i32}, {"i32.trunc_f64_u", f64, false, i32},
	{"i64.extend_i32_s", i32, false, i64}, {"i64.extend_i32_u", i32, false, i64},
	{"i64.trunc_f32_s", f32, false, i64}, {"i64.trunc_f32_u", f32, false, i64},
	{"i64.trunc_f64_s", f64, false, i64}, {"i64.trunc_f64_u", f64, false, i64},
	{"f32.convert_i32_s", i32, false, f32}, {"f32.convert_i32_u", i32, false, f32},
	{"f32.convert_i64_s", i64, false, f32}, {"f32.convert_i64_u", i64, false, f32},
	{"f32.demote_f64", f64, false, f32},
	{"f64.convert_i32_s", i32, false, f64}, {"f64.convert_i32_u", i32, false, f64},
	{"f64.convert_i64_s", i64, false, f64}, {"f64.convert_i64_u", i64, false, f64},
	{"f64.promote_f32", f32, false, f64},
	{"i32.reinterpret_f32", f32, false, i32}, {"i64.reinterpret_f64", f64, false, i64},
	{"f32.reinterpret_i32", i32, false, f32}, {"f64.reinterpret_i64", i64, false, f64},
	{"i32.extend8_s", i32, false, i32}, {"i32.extend16_s", i32, false, i32},
	{"i64.extend8_s", i64, false, i64}, {"i64.extend16_s", i64, false, i64},
	{"i64.extend32_s", i64, false, i64},
}

// isNoop reports whether a numeric instruction leaves the slot unchanged.
func isNoop(op Op) bool {
	switch op {
	case OpI64ExtendI32U, OpI32ReinterpretF32, OpI64ReinterpretF64, OpF32ReinterpretI32, OpF64ReinterpretI64:
		return true
	}
	return false
}

// memAccess describes a load or store.
type memAccess struct {
	op    Op
	typ   types.ValueType
	size  uint32 // bytes accessed; also the natural alignment
	store bool
}

func memAccessOf(in instruction.MemoryInstr) (memAccess, instruction.MemArg, bool) {
	switch x := in.(type) {
	case instruction.I32Load:
		return memAccess{OpLoad32, i32, 4, false}, x.MemArg, true
	case instruction.I64Load:
		return memAccess{OpLoad64, i64, 8, false}, x.MemArg, true
	case instruction.F32Load:
		return memAccess{OpLoad32, f32, 4, false}, x.MemArg, true
	case instruction.F64Load:
		return memAccess{OpLoad64, f64, 8, false}, x.MemArg, true
	case instruction.I32Load8S:
		return memAccess{OpI32Load8S, i32, 1, false}, x.MemArg, true
	case instruction.I32Load8U:
		return memAccess{OpLoad8U, i32, 1, false}, x.MemArg, true
	case instruction.I32Load16S:
		return memAccess{OpI32Load16S, i32, 2, false}, x.MemArg, true
	case instruction.I32Load16U:
		return memAccess{OpLoad16U, i32, 2, false}, x.MemArg, true
	case instruction.I64Load8S:
		return memAccess{OpI64Load8S, i64, 1, false}, x.MemArg, true
	case instruction.I64Load8U:
		return memAccess{OpLoad8U, i64, 1, false}, x.MemArg, true
	case instruction.I64Load16S:
		return memAccess{OpI64Load16S, i64, 2, false}, x.MemArg, true
	case instruction.I64Load16U:
		return memAccess{OpLoad16U, i64, 2, false}, x.MemArg, true
	case instruction.I64Load32S:
		return memAccess{OpI64Load32S, i64, 4, false}, x.MemArg, true
	case instruction.I64Load32U:
		return memAccess{OpLoad32, i64, 4, false}, x.MemArg, true
	case instruction.I32Store:
		return memAccess{OpStore32, i32, 4, true}, x.MemArg, true
	case instruction.I64Store:
		return memAccess{OpStore64, i64, 8, true}, x.MemArg, true
	case instruction.F32Store:
		return memAccess{OpStore32, f32, 4, true}, x.MemArg, true
	case instruction.F64Store:
		return memAccess{OpStore64, f64, 8, true}, x.MemArg, true
	case instruction.I32Store8:
		return memAccess{OpStore8, i32, 1, true}, x.MemArg, true
	case instruction.I32Store16:
		return memAccess{OpStore16, i32, 2, true}, x.MemArg, true
	case instruction.I64Store8:
		return memAccess{OpStore8, i64, 1, true}, x.MemArg, true
	case instruction.I64Store16:
		return memAccess{OpStore16, i64, 2, true}, x.MemArg, true
	case instruction.I64Store32:
		return memAccess{OpStore32, i64, 4, true}, x.MemArg, true
	}
	return memAccess{}, instruction.MemArg{}, false
}

// satTrunc maps the 0xFC saturating truncations to their Op and types.
func satTrunc(in instruction.NumericInstr) (Op, types.ValueType, types.ValueType, bool) {
	switch in.(type) {
	case instruction.I32TruncSatF32S:
		return OpI32TruncSatF32S, f32, i32, true
	case instruction.I32TruncSatF32U:
		return OpI32TruncSatF32U, f32, i32, true
	case instruction.I32TruncSatF64S:
		return OpI32TruncSatF64S, f64, i32, true
	case instruction.I32TruncSatF64U:
		return OpI32TruncSatF64U, f64, i32, true
	case instruction.I64TruncSatF32S:
		return OpI64TruncSatF32S, f32, i64, true
	case instruction.I64TruncSatF32U:
		return OpI64TruncSatF32U, f32, i64, true
	case instruction.I64TruncSatF64S:
		return OpI64TruncSatF64S, f64, i64, true
	case instruction.I64TruncSatF64U:
		return OpI64TruncSatF64U, f64, i64, true
	}
	return 0, 0, 0, false
}
