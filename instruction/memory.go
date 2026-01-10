package instruction

type I32Load struct{ MemArg MemArg }

func (I32Load) Opcode() Opcode  { return OpI32Load }
func (I32Load) instruction()    {}
func (I32Load) memoryInstr()    {}

type I64Load struct{ MemArg MemArg }

func (I64Load) Opcode() Opcode  { return OpI64Load }
func (I64Load) instruction()    {}
func (I64Load) memoryInstr()    {}

type F32Load struct{ MemArg MemArg }

func (F32Load) Opcode() Opcode  { return OpF32Load }
func (F32Load) instruction()    {}
func (F32Load) memoryInstr()    {}

type F64Load struct{ MemArg MemArg }

func (F64Load) Opcode() Opcode  { return OpF64Load }
func (F64Load) instruction()    {}
func (F64Load) memoryInstr()    {}

type I32Load8S struct{ MemArg MemArg }

func (I32Load8S) Opcode() Opcode  { return OpI32Load8S }
func (I32Load8S) instruction()    {}
func (I32Load8S) memoryInstr()    {}

type I32Load8U struct{ MemArg MemArg }

func (I32Load8U) Opcode() Opcode  { return OpI32Load8U }
func (I32Load8U) instruction()    {}
func (I32Load8U) memoryInstr()    {}

type I32Load16S struct{ MemArg MemArg }

func (I32Load16S) Opcode() Opcode  { return OpI32Load16S }
func (I32Load16S) instruction()    {}
func (I32Load16S) memoryInstr()    {}

type I32Load16U struct{ MemArg MemArg }

func (I32Load16U) Opcode() Opcode  { return OpI32Load16U }
func (I32Load16U) instruction()    {}
func (I32Load16U) memoryInstr()    {}

type I64Load8S struct{ MemArg MemArg }

func (I64Load8S) Opcode() Opcode  { return OpI64Load8S }
func (I64Load8S) instruction()    {}
func (I64Load8S) memoryInstr()    {}

type I64Load8U struct{ MemArg MemArg }

func (I64Load8U) Opcode() Opcode  { return OpI64Load8U }
func (I64Load8U) instruction()    {}
func (I64Load8U) memoryInstr()    {}

type I64Load16S struct{ MemArg MemArg }

func (I64Load16S) Opcode() Opcode  { return OpI64Load16S }
func (I64Load16S) instruction()    {}
func (I64Load16S) memoryInstr()    {}

type I64Load16U struct{ MemArg MemArg }

func (I64Load16U) Opcode() Opcode  { return OpI64Load16U }
func (I64Load16U) instruction()    {}
func (I64Load16U) memoryInstr()    {}

type I64Load32S struct{ MemArg MemArg }

func (I64Load32S) Opcode() Opcode  { return OpI64Load32S }
func (I64Load32S) instruction()    {}
func (I64Load32S) memoryInstr()    {}

type I64Load32U struct{ MemArg MemArg }

func (I64Load32U) Opcode() Opcode  { return OpI64Load32U }
func (I64Load32U) instruction()    {}
func (I64Load32U) memoryInstr()    {}

type I32Store struct{ MemArg MemArg }

func (I32Store) Opcode() Opcode  { return OpI32Store }
func (I32Store) instruction()    {}
func (I32Store) memoryInstr()    {}

type I64Store struct{ MemArg MemArg }

func (I64Store) Opcode() Opcode  { return OpI64Store }
func (I64Store) instruction()    {}
func (I64Store) memoryInstr()    {}

type F32Store struct{ MemArg MemArg }

func (F32Store) Opcode() Opcode  { return OpF32Store }
func (F32Store) instruction()    {}
func (F32Store) memoryInstr()    {}

type F64Store struct{ MemArg MemArg }

func (F64Store) Opcode() Opcode  { return OpF64Store }
func (F64Store) instruction()    {}
func (F64Store) memoryInstr()    {}

type I32Store8 struct{ MemArg MemArg }

func (I32Store8) Opcode() Opcode  { return OpI32Store8 }
func (I32Store8) instruction()    {}
func (I32Store8) memoryInstr()    {}

type I32Store16 struct{ MemArg MemArg }

func (I32Store16) Opcode() Opcode  { return OpI32Store16 }
func (I32Store16) instruction()    {}
func (I32Store16) memoryInstr()    {}

type I64Store8 struct{ MemArg MemArg }

func (I64Store8) Opcode() Opcode  { return OpI64Store8 }
func (I64Store8) instruction()    {}
func (I64Store8) memoryInstr()    {}

type I64Store16 struct{ MemArg MemArg }

func (I64Store16) Opcode() Opcode  { return OpI64Store16 }
func (I64Store16) instruction()    {}
func (I64Store16) memoryInstr()    {}

type I64Store32 struct{ MemArg MemArg }

func (I64Store32) Opcode() Opcode  { return OpI64Store32 }
func (I64Store32) instruction()    {}
func (I64Store32) memoryInstr()    {}

type MemorySize struct{ MemIdx uint32 }

func (MemorySize) Opcode() Opcode  { return OpMemorySize }
func (MemorySize) instruction()    {}
func (MemorySize) memoryInstr()    {}

type MemoryGrow struct{ MemIdx uint32 }

func (MemoryGrow) Opcode() Opcode  { return OpMemoryGrow }
func (MemoryGrow) instruction()    {}
func (MemoryGrow) memoryInstr()    {}
