package instruction

type I32WrapI64 struct{}

func (I32WrapI64) Opcode() Opcode  { return OpI32WrapI64 }
func (I32WrapI64) instruction()    {}
func (I32WrapI64) numericInstr()   {}

type I32TruncF32S struct{}

func (I32TruncF32S) Opcode() Opcode  { return OpI32TruncF32S }
func (I32TruncF32S) instruction()    {}
func (I32TruncF32S) numericInstr()   {}

type I32TruncF32U struct{}

func (I32TruncF32U) Opcode() Opcode  { return OpI32TruncF32U }
func (I32TruncF32U) instruction()    {}
func (I32TruncF32U) numericInstr()   {}

type I32TruncF64S struct{}

func (I32TruncF64S) Opcode() Opcode  { return OpI32TruncF64S }
func (I32TruncF64S) instruction()    {}
func (I32TruncF64S) numericInstr()   {}

type I32TruncF64U struct{}

func (I32TruncF64U) Opcode() Opcode  { return OpI32TruncF64U }
func (I32TruncF64U) instruction()    {}
func (I32TruncF64U) numericInstr()   {}

type I64ExtendI32S struct{}

func (I64ExtendI32S) Opcode() Opcode  { return OpI64ExtendI32S }
func (I64ExtendI32S) instruction()    {}
func (I64ExtendI32S) numericInstr()   {}

type I64ExtendI32U struct{}

func (I64ExtendI32U) Opcode() Opcode  { return OpI64ExtendI32U }
func (I64ExtendI32U) instruction()    {}
func (I64ExtendI32U) numericInstr()   {}

type I64TruncF32S struct{}

func (I64TruncF32S) Opcode() Opcode  { return OpI64TruncF32S }
func (I64TruncF32S) instruction()    {}
func (I64TruncF32S) numericInstr()   {}

type I64TruncF32U struct{}

func (I64TruncF32U) Opcode() Opcode  { return OpI64TruncF32U }
func (I64TruncF32U) instruction()    {}
func (I64TruncF32U) numericInstr()   {}

type I64TruncF64S struct{}

func (I64TruncF64S) Opcode() Opcode  { return OpI64TruncF64S }
func (I64TruncF64S) instruction()    {}
func (I64TruncF64S) numericInstr()   {}

type I64TruncF64U struct{}

func (I64TruncF64U) Opcode() Opcode  { return OpI64TruncF64U }
func (I64TruncF64U) instruction()    {}
func (I64TruncF64U) numericInstr()   {}

type F32ConvertI32S struct{}

func (F32ConvertI32S) Opcode() Opcode  { return OpF32ConvertI32S }
func (F32ConvertI32S) instruction()    {}
func (F32ConvertI32S) numericInstr()   {}

type F32ConvertI32U struct{}

func (F32ConvertI32U) Opcode() Opcode  { return OpF32ConvertI32U }
func (F32ConvertI32U) instruction()    {}
func (F32ConvertI32U) numericInstr()   {}

type F32ConvertI64S struct{}

func (F32ConvertI64S) Opcode() Opcode  { return OpF32ConvertI64S }
func (F32ConvertI64S) instruction()    {}
func (F32ConvertI64S) numericInstr()   {}

type F32ConvertI64U struct{}

func (F32ConvertI64U) Opcode() Opcode  { return OpF32ConvertI64U }
func (F32ConvertI64U) instruction()    {}
func (F32ConvertI64U) numericInstr()   {}

type F32DemoteF64 struct{}

func (F32DemoteF64) Opcode() Opcode  { return OpF32DemoteF64 }
func (F32DemoteF64) instruction()    {}
func (F32DemoteF64) numericInstr()   {}

type F64ConvertI32S struct{}

func (F64ConvertI32S) Opcode() Opcode  { return OpF64ConvertI32S }
func (F64ConvertI32S) instruction()    {}
func (F64ConvertI32S) numericInstr()   {}

type F64ConvertI32U struct{}

func (F64ConvertI32U) Opcode() Opcode  { return OpF64ConvertI32U }
func (F64ConvertI32U) instruction()    {}
func (F64ConvertI32U) numericInstr()   {}

type F64ConvertI64S struct{}

func (F64ConvertI64S) Opcode() Opcode  { return OpF64ConvertI64S }
func (F64ConvertI64S) instruction()    {}
func (F64ConvertI64S) numericInstr()   {}

type F64ConvertI64U struct{}

func (F64ConvertI64U) Opcode() Opcode  { return OpF64ConvertI64U }
func (F64ConvertI64U) instruction()    {}
func (F64ConvertI64U) numericInstr()   {}

type F64PromoteF32 struct{}

func (F64PromoteF32) Opcode() Opcode  { return OpF64PromoteF32 }
func (F64PromoteF32) instruction()    {}
func (F64PromoteF32) numericInstr()   {}

type I32ReinterpretF32 struct{}

func (I32ReinterpretF32) Opcode() Opcode  { return OpI32ReinterpretF32 }
func (I32ReinterpretF32) instruction()    {}
func (I32ReinterpretF32) numericInstr()   {}

type I64ReinterpretF64 struct{}

func (I64ReinterpretF64) Opcode() Opcode  { return OpI64ReinterpretF64 }
func (I64ReinterpretF64) instruction()    {}
func (I64ReinterpretF64) numericInstr()   {}

type F32ReinterpretI32 struct{}

func (F32ReinterpretI32) Opcode() Opcode  { return OpF32ReinterpretI32 }
func (F32ReinterpretI32) instruction()    {}
func (F32ReinterpretI32) numericInstr()   {}

type F64ReinterpretI64 struct{}

func (F64ReinterpretI64) Opcode() Opcode  { return OpF64ReinterpretI64 }
func (F64ReinterpretI64) instruction()    {}
func (F64ReinterpretI64) numericInstr()   {}

// sign extension

type I32Extend8S struct{}

func (I32Extend8S) Opcode() Opcode  { return OpI32Extend8S }
func (I32Extend8S) instruction()    {}
func (I32Extend8S) numericInstr()   {}

type I32Extend16S struct{}

func (I32Extend16S) Opcode() Opcode  { return OpI32Extend16S }
func (I32Extend16S) instruction()    {}
func (I32Extend16S) numericInstr()   {}

type I64Extend8S struct{}

func (I64Extend8S) Opcode() Opcode  { return OpI64Extend8S }
func (I64Extend8S) instruction()    {}
func (I64Extend8S) numericInstr()   {}

type I64Extend16S struct{}

func (I64Extend16S) Opcode() Opcode  { return OpI64Extend16S }
func (I64Extend16S) instruction()    {}
func (I64Extend16S) numericInstr()   {}

type I64Extend32S struct{}

func (I64Extend32S) Opcode() Opcode  { return OpI64Extend32S }
func (I64Extend32S) instruction()    {}
func (I64Extend32S) numericInstr()   {}
