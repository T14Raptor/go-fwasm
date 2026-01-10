package instruction

// 0xFC prefix instructions

type I32TruncSatF32S struct{}

func (I32TruncSatF32S) Opcode() Opcode  { return OpPrefix }
func (I32TruncSatF32S) instruction()    {}
func (I32TruncSatF32S) numericInstr()   {}

type I32TruncSatF32U struct{}

func (I32TruncSatF32U) Opcode() Opcode  { return OpPrefix }
func (I32TruncSatF32U) instruction()    {}
func (I32TruncSatF32U) numericInstr()   {}

type I32TruncSatF64S struct{}

func (I32TruncSatF64S) Opcode() Opcode  { return OpPrefix }
func (I32TruncSatF64S) instruction()    {}
func (I32TruncSatF64S) numericInstr()   {}

type I32TruncSatF64U struct{}

func (I32TruncSatF64U) Opcode() Opcode  { return OpPrefix }
func (I32TruncSatF64U) instruction()    {}
func (I32TruncSatF64U) numericInstr()   {}

type I64TruncSatF32S struct{}

func (I64TruncSatF32S) Opcode() Opcode  { return OpPrefix }
func (I64TruncSatF32S) instruction()    {}
func (I64TruncSatF32S) numericInstr()   {}

type I64TruncSatF32U struct{}

func (I64TruncSatF32U) Opcode() Opcode  { return OpPrefix }
func (I64TruncSatF32U) instruction()    {}
func (I64TruncSatF32U) numericInstr()   {}

type I64TruncSatF64S struct{}

func (I64TruncSatF64S) Opcode() Opcode  { return OpPrefix }
func (I64TruncSatF64S) instruction()    {}
func (I64TruncSatF64S) numericInstr()   {}

type I64TruncSatF64U struct{}

func (I64TruncSatF64U) Opcode() Opcode  { return OpPrefix }
func (I64TruncSatF64U) instruction()    {}
func (I64TruncSatF64U) numericInstr()   {}

type MemoryInit struct {
	DataIdx uint32
	MemIdx  uint32
}

func (MemoryInit) Opcode() Opcode  { return OpPrefix }
func (MemoryInit) instruction()    {}
func (MemoryInit) memoryInstr()    {}

type DataDrop struct {
	DataIdx uint32
}

func (DataDrop) Opcode() Opcode  { return OpPrefix }
func (DataDrop) instruction()    {}
func (DataDrop) memoryInstr()    {}

type MemoryCopy struct {
	DstMemIdx uint32
	SrcMemIdx uint32
}

func (MemoryCopy) Opcode() Opcode  { return OpPrefix }
func (MemoryCopy) instruction()    {}
func (MemoryCopy) memoryInstr()    {}

type MemoryFill struct {
	MemIdx uint32
}

func (MemoryFill) Opcode() Opcode  { return OpPrefix }
func (MemoryFill) instruction()    {}
func (MemoryFill) memoryInstr()    {}

type TableInit struct {
	ElemIdx  uint32
	TableIdx uint32
}

func (TableInit) Opcode() Opcode  { return OpPrefix }
func (TableInit) instruction()    {}
func (TableInit) variableInstr()  {}

type ElemDrop struct {
	ElemIdx uint32
}

func (ElemDrop) Opcode() Opcode  { return OpPrefix }
func (ElemDrop) instruction()    {}
func (ElemDrop) variableInstr()  {}

type TableCopy struct {
	DstTableIdx uint32
	SrcTableIdx uint32
}

func (TableCopy) Opcode() Opcode  { return OpPrefix }
func (TableCopy) instruction()    {}
func (TableCopy) variableInstr()  {}

type TableGrow struct {
	TableIdx uint32
}

func (TableGrow) Opcode() Opcode  { return OpPrefix }
func (TableGrow) instruction()    {}
func (TableGrow) variableInstr()  {}

type TableSize struct {
	TableIdx uint32
}

func (TableSize) Opcode() Opcode  { return OpPrefix }
func (TableSize) instruction()    {}
func (TableSize) variableInstr()  {}

type TableFill struct {
	TableIdx uint32
}

func (TableFill) Opcode() Opcode  { return OpPrefix }
func (TableFill) instruction()    {}
func (TableFill) variableInstr()  {}
