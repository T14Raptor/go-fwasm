package instruction

type Opcode byte

// Control instructions
const (
	OpUnreachable Opcode = 0x00
	OpNop         Opcode = 0x01
	OpBlock       Opcode = 0x02
	OpLoop        Opcode = 0x03
	OpIf          Opcode = 0x04
	OpElse        Opcode = 0x05
	OpEnd         Opcode = 0x0B
	OpBr          Opcode = 0x0C
	OpBrIf        Opcode = 0x0D
	OpBrTable     Opcode = 0x0E
	OpReturn      Opcode = 0x0F
	OpCall        Opcode = 0x10
	OpCallIndirect Opcode = 0x11
)

// Reference instructions
const (
	OpRefNull   Opcode = 0xD0
	OpRefIsNull Opcode = 0xD1
	OpRefFunc   Opcode = 0xD2
)

// Parametric instructions
const (
	OpDrop       Opcode = 0x1A
	OpSelect     Opcode = 0x1B
	OpSelectT    Opcode = 0x1C
)

// Variable instructions
const (
	OpLocalGet  Opcode = 0x20
	OpLocalSet  Opcode = 0x21
	OpLocalTee  Opcode = 0x22
	OpGlobalGet Opcode = 0x23
	OpGlobalSet Opcode = 0x24
)

// Table instructions
const (
	OpTableGet  Opcode = 0x25
	OpTableSet  Opcode = 0x26
)

// Memory instructions
const (
	OpI32Load    Opcode = 0x28
	OpI64Load    Opcode = 0x29
	OpF32Load    Opcode = 0x2A
	OpF64Load    Opcode = 0x2B
	OpI32Load8S  Opcode = 0x2C
	OpI32Load8U  Opcode = 0x2D
	OpI32Load16S Opcode = 0x2E
	OpI32Load16U Opcode = 0x2F
	OpI64Load8S  Opcode = 0x30
	OpI64Load8U  Opcode = 0x31
	OpI64Load16S Opcode = 0x32
	OpI64Load16U Opcode = 0x33
	OpI64Load32S Opcode = 0x34
	OpI64Load32U Opcode = 0x35
	OpI32Store   Opcode = 0x36
	OpI64Store   Opcode = 0x37
	OpF32Store   Opcode = 0x38
	OpF64Store   Opcode = 0x39
	OpI32Store8  Opcode = 0x3A
	OpI32Store16 Opcode = 0x3B
	OpI64Store8  Opcode = 0x3C
	OpI64Store16 Opcode = 0x3D
	OpI64Store32 Opcode = 0x3E
	OpMemorySize Opcode = 0x3F
	OpMemoryGrow Opcode = 0x40
)

// Numeric instructions - Constants
const (
	OpI32Const Opcode = 0x41
	OpI64Const Opcode = 0x42
	OpF32Const Opcode = 0x43
	OpF64Const Opcode = 0x44
)

// Numeric instructions - i32 comparisons
const (
	OpI32Eqz Opcode = 0x45
	OpI32Eq  Opcode = 0x46
	OpI32Ne  Opcode = 0x47
	OpI32LtS Opcode = 0x48
	OpI32LtU Opcode = 0x49
	OpI32GtS Opcode = 0x4A
	OpI32GtU Opcode = 0x4B
	OpI32LeS Opcode = 0x4C
	OpI32LeU Opcode = 0x4D
	OpI32GeS Opcode = 0x4E
	OpI32GeU Opcode = 0x4F
)

// Numeric instructions - i64 comparisons
const (
	OpI64Eqz Opcode = 0x50
	OpI64Eq  Opcode = 0x51
	OpI64Ne  Opcode = 0x52
	OpI64LtS Opcode = 0x53
	OpI64LtU Opcode = 0x54
	OpI64GtS Opcode = 0x55
	OpI64GtU Opcode = 0x56
	OpI64LeS Opcode = 0x57
	OpI64LeU Opcode = 0x58
	OpI64GeS Opcode = 0x59
	OpI64GeU Opcode = 0x5A
)

// Numeric instructions - f32 comparisons
const (
	OpF32Eq Opcode = 0x5B
	OpF32Ne Opcode = 0x5C
	OpF32Lt Opcode = 0x5D
	OpF32Gt Opcode = 0x5E
	OpF32Le Opcode = 0x5F
	OpF32Ge Opcode = 0x60
)

// Numeric instructions - f64 comparisons
const (
	OpF64Eq Opcode = 0x61
	OpF64Ne Opcode = 0x62
	OpF64Lt Opcode = 0x63
	OpF64Gt Opcode = 0x64
	OpF64Le Opcode = 0x65
	OpF64Ge Opcode = 0x66
)

// Numeric instructions - i32 operations
const (
	OpI32Clz    Opcode = 0x67
	OpI32Ctz    Opcode = 0x68
	OpI32Popcnt Opcode = 0x69
	OpI32Add    Opcode = 0x6A
	OpI32Sub    Opcode = 0x6B
	OpI32Mul    Opcode = 0x6C
	OpI32DivS   Opcode = 0x6D
	OpI32DivU   Opcode = 0x6E
	OpI32RemS   Opcode = 0x6F
	OpI32RemU   Opcode = 0x70
	OpI32And    Opcode = 0x71
	OpI32Or     Opcode = 0x72
	OpI32Xor    Opcode = 0x73
	OpI32Shl    Opcode = 0x74
	OpI32ShrS   Opcode = 0x75
	OpI32ShrU   Opcode = 0x76
	OpI32Rotl   Opcode = 0x77
	OpI32Rotr   Opcode = 0x78
)

// Numeric instructions - i64 operations
const (
	OpI64Clz    Opcode = 0x79
	OpI64Ctz    Opcode = 0x7A
	OpI64Popcnt Opcode = 0x7B
	OpI64Add    Opcode = 0x7C
	OpI64Sub    Opcode = 0x7D
	OpI64Mul    Opcode = 0x7E
	OpI64DivS   Opcode = 0x7F
	OpI64DivU   Opcode = 0x80
	OpI64RemS   Opcode = 0x81
	OpI64RemU   Opcode = 0x82
	OpI64And    Opcode = 0x83
	OpI64Or     Opcode = 0x84
	OpI64Xor    Opcode = 0x85
	OpI64Shl    Opcode = 0x86
	OpI64ShrS   Opcode = 0x87
	OpI64ShrU   Opcode = 0x88
	OpI64Rotl   Opcode = 0x89
	OpI64Rotr   Opcode = 0x8A
)

// Numeric instructions - f32 operations
const (
	OpF32Abs      Opcode = 0x8B
	OpF32Neg      Opcode = 0x8C
	OpF32Ceil     Opcode = 0x8D
	OpF32Floor    Opcode = 0x8E
	OpF32Trunc    Opcode = 0x8F
	OpF32Nearest  Opcode = 0x90
	OpF32Sqrt     Opcode = 0x91
	OpF32Add      Opcode = 0x92
	OpF32Sub      Opcode = 0x93
	OpF32Mul      Opcode = 0x94
	OpF32Div      Opcode = 0x95
	OpF32Min      Opcode = 0x96
	OpF32Max      Opcode = 0x97
	OpF32Copysign Opcode = 0x98
)

// Numeric instructions - f64 operations
const (
	OpF64Abs      Opcode = 0x99
	OpF64Neg      Opcode = 0x9A
	OpF64Ceil     Opcode = 0x9B
	OpF64Floor    Opcode = 0x9C
	OpF64Trunc    Opcode = 0x9D
	OpF64Nearest  Opcode = 0x9E
	OpF64Sqrt     Opcode = 0x9F
	OpF64Add      Opcode = 0xA0
	OpF64Sub      Opcode = 0xA1
	OpF64Mul      Opcode = 0xA2
	OpF64Div      Opcode = 0xA3
	OpF64Min      Opcode = 0xA4
	OpF64Max      Opcode = 0xA5
	OpF64Copysign Opcode = 0xA6
)

// Numeric instructions - conversions
const (
	OpI32WrapI64      Opcode = 0xA7
	OpI32TruncF32S    Opcode = 0xA8
	OpI32TruncF32U    Opcode = 0xA9
	OpI32TruncF64S    Opcode = 0xAA
	OpI32TruncF64U    Opcode = 0xAB
	OpI64ExtendI32S   Opcode = 0xAC
	OpI64ExtendI32U   Opcode = 0xAD
	OpI64TruncF32S    Opcode = 0xAE
	OpI64TruncF32U    Opcode = 0xAF
	OpI64TruncF64S    Opcode = 0xB0
	OpI64TruncF64U    Opcode = 0xB1
	OpF32ConvertI32S  Opcode = 0xB2
	OpF32ConvertI32U  Opcode = 0xB3
	OpF32ConvertI64S  Opcode = 0xB4
	OpF32ConvertI64U  Opcode = 0xB5
	OpF32DemoteF64    Opcode = 0xB6
	OpF64ConvertI32S  Opcode = 0xB7
	OpF64ConvertI32U  Opcode = 0xB8
	OpF64ConvertI64S  Opcode = 0xB9
	OpF64ConvertI64U  Opcode = 0xBA
	OpF64PromoteF32   Opcode = 0xBB
	OpI32ReinterpretF32 Opcode = 0xBC
	OpI64ReinterpretF64 Opcode = 0xBD
	OpF32ReinterpretI32 Opcode = 0xBE
	OpF64ReinterpretI64 Opcode = 0xBF
)

// Numeric instructions - sign extension
const (
	OpI32Extend8S  Opcode = 0xC0
	OpI32Extend16S Opcode = 0xC1
	OpI64Extend8S  Opcode = 0xC2
	OpI64Extend16S Opcode = 0xC3
	OpI64Extend32S Opcode = 0xC4
)

// Prefix for multi-byte opcodes
const (
	OpPrefix      Opcode = 0xFC
	OpPrefixSIMD  Opcode = 0xFD
)

// Extended opcodes (0xFC prefix)
type ExtOpcode uint32

const (
	ExtOpI32TruncSatF32S ExtOpcode = 0
	ExtOpI32TruncSatF32U ExtOpcode = 1
	ExtOpI32TruncSatF64S ExtOpcode = 2
	ExtOpI32TruncSatF64U ExtOpcode = 3
	ExtOpI64TruncSatF32S ExtOpcode = 4
	ExtOpI64TruncSatF32U ExtOpcode = 5
	ExtOpI64TruncSatF64S ExtOpcode = 6
	ExtOpI64TruncSatF64U ExtOpcode = 7
	ExtOpMemoryInit      ExtOpcode = 8
	ExtOpDataDrop        ExtOpcode = 9
	ExtOpMemoryCopy      ExtOpcode = 10
	ExtOpMemoryFill      ExtOpcode = 11
	ExtOpTableInit       ExtOpcode = 12
	ExtOpElemDrop        ExtOpcode = 13
	ExtOpTableCopy       ExtOpcode = 14
	ExtOpTableGrow       ExtOpcode = 15
	ExtOpTableSize       ExtOpcode = 16
	ExtOpTableFill       ExtOpcode = 17
)

func (op Opcode) String() string {
	switch op {
	case OpUnreachable:
		return "unreachable"
	case OpNop:
		return "nop"
	case OpBlock:
		return "block"
	case OpLoop:
		return "loop"
	case OpIf:
		return "if"
	case OpElse:
		return "else"
	case OpEnd:
		return "end"
	case OpBr:
		return "br"
	case OpBrIf:
		return "br_if"
	case OpBrTable:
		return "br_table"
	case OpReturn:
		return "return"
	case OpCall:
		return "call"
	case OpCallIndirect:
		return "call_indirect"
	case OpRefNull:
		return "ref.null"
	case OpRefIsNull:
		return "ref.is_null"
	case OpRefFunc:
		return "ref.func"
	case OpDrop:
		return "drop"
	case OpSelect:
		return "select"
	case OpSelectT:
		return "select"
	case OpLocalGet:
		return "local.get"
	case OpLocalSet:
		return "local.set"
	case OpLocalTee:
		return "local.tee"
	case OpGlobalGet:
		return "global.get"
	case OpGlobalSet:
		return "global.set"
	case OpTableGet:
		return "table.get"
	case OpTableSet:
		return "table.set"
	case OpI32Load:
		return "i32.load"
	case OpI64Load:
		return "i64.load"
	case OpF32Load:
		return "f32.load"
	case OpF64Load:
		return "f64.load"
	case OpI32Load8S:
		return "i32.load8_s"
	case OpI32Load8U:
		return "i32.load8_u"
	case OpI32Load16S:
		return "i32.load16_s"
	case OpI32Load16U:
		return "i32.load16_u"
	case OpI64Load8S:
		return "i64.load8_s"
	case OpI64Load8U:
		return "i64.load8_u"
	case OpI64Load16S:
		return "i64.load16_s"
	case OpI64Load16U:
		return "i64.load16_u"
	case OpI64Load32S:
		return "i64.load32_s"
	case OpI64Load32U:
		return "i64.load32_u"
	case OpI32Store:
		return "i32.store"
	case OpI64Store:
		return "i64.store"
	case OpF32Store:
		return "f32.store"
	case OpF64Store:
		return "f64.store"
	case OpI32Store8:
		return "i32.store8"
	case OpI32Store16:
		return "i32.store16"
	case OpI64Store8:
		return "i64.store8"
	case OpI64Store16:
		return "i64.store16"
	case OpI64Store32:
		return "i64.store32"
	case OpMemorySize:
		return "memory.size"
	case OpMemoryGrow:
		return "memory.grow"
	case OpI32Const:
		return "i32.const"
	case OpI64Const:
		return "i64.const"
	case OpF32Const:
		return "f32.const"
	case OpF64Const:
		return "f64.const"
	case OpI32Eqz:
		return "i32.eqz"
	case OpI32Eq:
		return "i32.eq"
	case OpI32Ne:
		return "i32.ne"
	case OpI32LtS:
		return "i32.lt_s"
	case OpI32LtU:
		return "i32.lt_u"
	case OpI32GtS:
		return "i32.gt_s"
	case OpI32GtU:
		return "i32.gt_u"
	case OpI32LeS:
		return "i32.le_s"
	case OpI32LeU:
		return "i32.le_u"
	case OpI32GeS:
		return "i32.ge_s"
	case OpI32GeU:
		return "i32.ge_u"
	case OpI64Eqz:
		return "i64.eqz"
	case OpI64Eq:
		return "i64.eq"
	case OpI64Ne:
		return "i64.ne"
	case OpI64LtS:
		return "i64.lt_s"
	case OpI64LtU:
		return "i64.lt_u"
	case OpI64GtS:
		return "i64.gt_s"
	case OpI64GtU:
		return "i64.gt_u"
	case OpI64LeS:
		return "i64.le_s"
	case OpI64LeU:
		return "i64.le_u"
	case OpI64GeS:
		return "i64.ge_s"
	case OpI64GeU:
		return "i64.ge_u"
	case OpF32Eq:
		return "f32.eq"
	case OpF32Ne:
		return "f32.ne"
	case OpF32Lt:
		return "f32.lt"
	case OpF32Gt:
		return "f32.gt"
	case OpF32Le:
		return "f32.le"
	case OpF32Ge:
		return "f32.ge"
	case OpF64Eq:
		return "f64.eq"
	case OpF64Ne:
		return "f64.ne"
	case OpF64Lt:
		return "f64.lt"
	case OpF64Gt:
		return "f64.gt"
	case OpF64Le:
		return "f64.le"
	case OpF64Ge:
		return "f64.ge"
	case OpI32Clz:
		return "i32.clz"
	case OpI32Ctz:
		return "i32.ctz"
	case OpI32Popcnt:
		return "i32.popcnt"
	case OpI32Add:
		return "i32.add"
	case OpI32Sub:
		return "i32.sub"
	case OpI32Mul:
		return "i32.mul"
	case OpI32DivS:
		return "i32.div_s"
	case OpI32DivU:
		return "i32.div_u"
	case OpI32RemS:
		return "i32.rem_s"
	case OpI32RemU:
		return "i32.rem_u"
	case OpI32And:
		return "i32.and"
	case OpI32Or:
		return "i32.or"
	case OpI32Xor:
		return "i32.xor"
	case OpI32Shl:
		return "i32.shl"
	case OpI32ShrS:
		return "i32.shr_s"
	case OpI32ShrU:
		return "i32.shr_u"
	case OpI32Rotl:
		return "i32.rotl"
	case OpI32Rotr:
		return "i32.rotr"
	case OpI64Clz:
		return "i64.clz"
	case OpI64Ctz:
		return "i64.ctz"
	case OpI64Popcnt:
		return "i64.popcnt"
	case OpI64Add:
		return "i64.add"
	case OpI64Sub:
		return "i64.sub"
	case OpI64Mul:
		return "i64.mul"
	case OpI64DivS:
		return "i64.div_s"
	case OpI64DivU:
		return "i64.div_u"
	case OpI64RemS:
		return "i64.rem_s"
	case OpI64RemU:
		return "i64.rem_u"
	case OpI64And:
		return "i64.and"
	case OpI64Or:
		return "i64.or"
	case OpI64Xor:
		return "i64.xor"
	case OpI64Shl:
		return "i64.shl"
	case OpI64ShrS:
		return "i64.shr_s"
	case OpI64ShrU:
		return "i64.shr_u"
	case OpI64Rotl:
		return "i64.rotl"
	case OpI64Rotr:
		return "i64.rotr"
	case OpF32Abs:
		return "f32.abs"
	case OpF32Neg:
		return "f32.neg"
	case OpF32Ceil:
		return "f32.ceil"
	case OpF32Floor:
		return "f32.floor"
	case OpF32Trunc:
		return "f32.trunc"
	case OpF32Nearest:
		return "f32.nearest"
	case OpF32Sqrt:
		return "f32.sqrt"
	case OpF32Add:
		return "f32.add"
	case OpF32Sub:
		return "f32.sub"
	case OpF32Mul:
		return "f32.mul"
	case OpF32Div:
		return "f32.div"
	case OpF32Min:
		return "f32.min"
	case OpF32Max:
		return "f32.max"
	case OpF32Copysign:
		return "f32.copysign"
	case OpF64Abs:
		return "f64.abs"
	case OpF64Neg:
		return "f64.neg"
	case OpF64Ceil:
		return "f64.ceil"
	case OpF64Floor:
		return "f64.floor"
	case OpF64Trunc:
		return "f64.trunc"
	case OpF64Nearest:
		return "f64.nearest"
	case OpF64Sqrt:
		return "f64.sqrt"
	case OpF64Add:
		return "f64.add"
	case OpF64Sub:
		return "f64.sub"
	case OpF64Mul:
		return "f64.mul"
	case OpF64Div:
		return "f64.div"
	case OpF64Min:
		return "f64.min"
	case OpF64Max:
		return "f64.max"
	case OpF64Copysign:
		return "f64.copysign"
	case OpI32WrapI64:
		return "i32.wrap_i64"
	case OpI32TruncF32S:
		return "i32.trunc_f32_s"
	case OpI32TruncF32U:
		return "i32.trunc_f32_u"
	case OpI32TruncF64S:
		return "i32.trunc_f64_s"
	case OpI32TruncF64U:
		return "i32.trunc_f64_u"
	case OpI64ExtendI32S:
		return "i64.extend_i32_s"
	case OpI64ExtendI32U:
		return "i64.extend_i32_u"
	case OpI64TruncF32S:
		return "i64.trunc_f32_s"
	case OpI64TruncF32U:
		return "i64.trunc_f32_u"
	case OpI64TruncF64S:
		return "i64.trunc_f64_s"
	case OpI64TruncF64U:
		return "i64.trunc_f64_u"
	case OpF32ConvertI32S:
		return "f32.convert_i32_s"
	case OpF32ConvertI32U:
		return "f32.convert_i32_u"
	case OpF32ConvertI64S:
		return "f32.convert_i64_s"
	case OpF32ConvertI64U:
		return "f32.convert_i64_u"
	case OpF32DemoteF64:
		return "f32.demote_f64"
	case OpF64ConvertI32S:
		return "f64.convert_i32_s"
	case OpF64ConvertI32U:
		return "f64.convert_i32_u"
	case OpF64ConvertI64S:
		return "f64.convert_i64_s"
	case OpF64ConvertI64U:
		return "f64.convert_i64_u"
	case OpF64PromoteF32:
		return "f64.promote_f32"
	case OpI32ReinterpretF32:
		return "i32.reinterpret_f32"
	case OpI64ReinterpretF64:
		return "i64.reinterpret_f64"
	case OpF32ReinterpretI32:
		return "f32.reinterpret_i32"
	case OpF64ReinterpretI64:
		return "f64.reinterpret_i64"
	case OpI32Extend8S:
		return "i32.extend8_s"
	case OpI32Extend16S:
		return "i32.extend16_s"
	case OpI64Extend8S:
		return "i64.extend8_s"
	case OpI64Extend16S:
		return "i64.extend16_s"
	case OpI64Extend32S:
		return "i64.extend32_s"
	default:
		return "unknown"
	}
}

func (op ExtOpcode) String() string {
	switch op {
	case ExtOpI32TruncSatF32S:
		return "i32.trunc_sat_f32_s"
	case ExtOpI32TruncSatF32U:
		return "i32.trunc_sat_f32_u"
	case ExtOpI32TruncSatF64S:
		return "i32.trunc_sat_f64_s"
	case ExtOpI32TruncSatF64U:
		return "i32.trunc_sat_f64_u"
	case ExtOpI64TruncSatF32S:
		return "i64.trunc_sat_f32_s"
	case ExtOpI64TruncSatF32U:
		return "i64.trunc_sat_f32_u"
	case ExtOpI64TruncSatF64S:
		return "i64.trunc_sat_f64_s"
	case ExtOpI64TruncSatF64U:
		return "i64.trunc_sat_f64_u"
	case ExtOpMemoryInit:
		return "memory.init"
	case ExtOpDataDrop:
		return "data.drop"
	case ExtOpMemoryCopy:
		return "memory.copy"
	case ExtOpMemoryFill:
		return "memory.fill"
	case ExtOpTableInit:
		return "table.init"
	case ExtOpElemDrop:
		return "elem.drop"
	case ExtOpTableCopy:
		return "table.copy"
	case ExtOpTableGrow:
		return "table.grow"
	case ExtOpTableSize:
		return "table.size"
	case ExtOpTableFill:
		return "table.fill"
	default:
		return "unknown"
	}
}

