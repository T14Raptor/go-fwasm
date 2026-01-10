package instruction

type I32Const struct{ Value int32 }

func (I32Const) Opcode() Opcode  { return OpI32Const }
func (I32Const) instruction()    {}
func (I32Const) numericInstr()   {}

type I64Const struct{ Value int64 }

func (I64Const) Opcode() Opcode  { return OpI64Const }
func (I64Const) instruction()    {}
func (I64Const) numericInstr()   {}

type F32Const struct{ Value float32 }

func (F32Const) Opcode() Opcode  { return OpF32Const }
func (F32Const) instruction()    {}
func (F32Const) numericInstr()   {}

type F64Const struct{ Value float64 }

func (F64Const) Opcode() Opcode  { return OpF64Const }
func (F64Const) instruction()    {}
func (F64Const) numericInstr()   {}

// i32 comparison

type I32Eqz struct{}

func (I32Eqz) Opcode() Opcode  { return OpI32Eqz }
func (I32Eqz) instruction()    {}
func (I32Eqz) numericInstr()   {}

type I32Eq struct{}

func (I32Eq) Opcode() Opcode  { return OpI32Eq }
func (I32Eq) instruction()    {}
func (I32Eq) numericInstr()   {}

type I32Ne struct{}

func (I32Ne) Opcode() Opcode  { return OpI32Ne }
func (I32Ne) instruction()    {}
func (I32Ne) numericInstr()   {}

type I32LtS struct{}

func (I32LtS) Opcode() Opcode  { return OpI32LtS }
func (I32LtS) instruction()    {}
func (I32LtS) numericInstr()   {}

type I32LtU struct{}

func (I32LtU) Opcode() Opcode  { return OpI32LtU }
func (I32LtU) instruction()    {}
func (I32LtU) numericInstr()   {}

type I32GtS struct{}

func (I32GtS) Opcode() Opcode  { return OpI32GtS }
func (I32GtS) instruction()    {}
func (I32GtS) numericInstr()   {}

type I32GtU struct{}

func (I32GtU) Opcode() Opcode  { return OpI32GtU }
func (I32GtU) instruction()    {}
func (I32GtU) numericInstr()   {}

type I32LeS struct{}

func (I32LeS) Opcode() Opcode  { return OpI32LeS }
func (I32LeS) instruction()    {}
func (I32LeS) numericInstr()   {}

type I32LeU struct{}

func (I32LeU) Opcode() Opcode  { return OpI32LeU }
func (I32LeU) instruction()    {}
func (I32LeU) numericInstr()   {}

type I32GeS struct{}

func (I32GeS) Opcode() Opcode  { return OpI32GeS }
func (I32GeS) instruction()    {}
func (I32GeS) numericInstr()   {}

type I32GeU struct{}

func (I32GeU) Opcode() Opcode  { return OpI32GeU }
func (I32GeU) instruction()    {}
func (I32GeU) numericInstr()   {}

// i64 comparison

type I64Eqz struct{}

func (I64Eqz) Opcode() Opcode  { return OpI64Eqz }
func (I64Eqz) instruction()    {}
func (I64Eqz) numericInstr()   {}

type I64Eq struct{}

func (I64Eq) Opcode() Opcode  { return OpI64Eq }
func (I64Eq) instruction()    {}
func (I64Eq) numericInstr()   {}

type I64Ne struct{}

func (I64Ne) Opcode() Opcode  { return OpI64Ne }
func (I64Ne) instruction()    {}
func (I64Ne) numericInstr()   {}

type I64LtS struct{}

func (I64LtS) Opcode() Opcode  { return OpI64LtS }
func (I64LtS) instruction()    {}
func (I64LtS) numericInstr()   {}

type I64LtU struct{}

func (I64LtU) Opcode() Opcode  { return OpI64LtU }
func (I64LtU) instruction()    {}
func (I64LtU) numericInstr()   {}

type I64GtS struct{}

func (I64GtS) Opcode() Opcode  { return OpI64GtS }
func (I64GtS) instruction()    {}
func (I64GtS) numericInstr()   {}

type I64GtU struct{}

func (I64GtU) Opcode() Opcode  { return OpI64GtU }
func (I64GtU) instruction()    {}
func (I64GtU) numericInstr()   {}

type I64LeS struct{}

func (I64LeS) Opcode() Opcode  { return OpI64LeS }
func (I64LeS) instruction()    {}
func (I64LeS) numericInstr()   {}

type I64LeU struct{}

func (I64LeU) Opcode() Opcode  { return OpI64LeU }
func (I64LeU) instruction()    {}
func (I64LeU) numericInstr()   {}

type I64GeS struct{}

func (I64GeS) Opcode() Opcode  { return OpI64GeS }
func (I64GeS) instruction()    {}
func (I64GeS) numericInstr()   {}

type I64GeU struct{}

func (I64GeU) Opcode() Opcode  { return OpI64GeU }
func (I64GeU) instruction()    {}
func (I64GeU) numericInstr()   {}

// f32 comparison

type F32Eq struct{}

func (F32Eq) Opcode() Opcode  { return OpF32Eq }
func (F32Eq) instruction()    {}
func (F32Eq) numericInstr()   {}

type F32Ne struct{}

func (F32Ne) Opcode() Opcode  { return OpF32Ne }
func (F32Ne) instruction()    {}
func (F32Ne) numericInstr()   {}

type F32Lt struct{}

func (F32Lt) Opcode() Opcode  { return OpF32Lt }
func (F32Lt) instruction()    {}
func (F32Lt) numericInstr()   {}

type F32Gt struct{}

func (F32Gt) Opcode() Opcode  { return OpF32Gt }
func (F32Gt) instruction()    {}
func (F32Gt) numericInstr()   {}

type F32Le struct{}

func (F32Le) Opcode() Opcode  { return OpF32Le }
func (F32Le) instruction()    {}
func (F32Le) numericInstr()   {}

type F32Ge struct{}

func (F32Ge) Opcode() Opcode  { return OpF32Ge }
func (F32Ge) instruction()    {}
func (F32Ge) numericInstr()   {}

// f64 comparison

type F64Eq struct{}

func (F64Eq) Opcode() Opcode  { return OpF64Eq }
func (F64Eq) instruction()    {}
func (F64Eq) numericInstr()   {}

type F64Ne struct{}

func (F64Ne) Opcode() Opcode  { return OpF64Ne }
func (F64Ne) instruction()    {}
func (F64Ne) numericInstr()   {}

type F64Lt struct{}

func (F64Lt) Opcode() Opcode  { return OpF64Lt }
func (F64Lt) instruction()    {}
func (F64Lt) numericInstr()   {}

type F64Gt struct{}

func (F64Gt) Opcode() Opcode  { return OpF64Gt }
func (F64Gt) instruction()    {}
func (F64Gt) numericInstr()   {}

type F64Le struct{}

func (F64Le) Opcode() Opcode  { return OpF64Le }
func (F64Le) instruction()    {}
func (F64Le) numericInstr()   {}

type F64Ge struct{}

func (F64Ge) Opcode() Opcode  { return OpF64Ge }
func (F64Ge) instruction()    {}
func (F64Ge) numericInstr()   {}

// i32 arithmetic

type I32Clz struct{}

func (I32Clz) Opcode() Opcode  { return OpI32Clz }
func (I32Clz) instruction()    {}
func (I32Clz) numericInstr()   {}

type I32Ctz struct{}

func (I32Ctz) Opcode() Opcode  { return OpI32Ctz }
func (I32Ctz) instruction()    {}
func (I32Ctz) numericInstr()   {}

type I32Popcnt struct{}

func (I32Popcnt) Opcode() Opcode  { return OpI32Popcnt }
func (I32Popcnt) instruction()    {}
func (I32Popcnt) numericInstr()   {}

type I32Add struct{}

func (I32Add) Opcode() Opcode  { return OpI32Add }
func (I32Add) instruction()    {}
func (I32Add) numericInstr()   {}

type I32Sub struct{}

func (I32Sub) Opcode() Opcode  { return OpI32Sub }
func (I32Sub) instruction()    {}
func (I32Sub) numericInstr()   {}

type I32Mul struct{}

func (I32Mul) Opcode() Opcode  { return OpI32Mul }
func (I32Mul) instruction()    {}
func (I32Mul) numericInstr()   {}

type I32DivS struct{}

func (I32DivS) Opcode() Opcode  { return OpI32DivS }
func (I32DivS) instruction()    {}
func (I32DivS) numericInstr()   {}

type I32DivU struct{}

func (I32DivU) Opcode() Opcode  { return OpI32DivU }
func (I32DivU) instruction()    {}
func (I32DivU) numericInstr()   {}

type I32RemS struct{}

func (I32RemS) Opcode() Opcode  { return OpI32RemS }
func (I32RemS) instruction()    {}
func (I32RemS) numericInstr()   {}

type I32RemU struct{}

func (I32RemU) Opcode() Opcode  { return OpI32RemU }
func (I32RemU) instruction()    {}
func (I32RemU) numericInstr()   {}

type I32And struct{}

func (I32And) Opcode() Opcode  { return OpI32And }
func (I32And) instruction()    {}
func (I32And) numericInstr()   {}

type I32Or struct{}

func (I32Or) Opcode() Opcode  { return OpI32Or }
func (I32Or) instruction()    {}
func (I32Or) numericInstr()   {}

type I32Xor struct{}

func (I32Xor) Opcode() Opcode  { return OpI32Xor }
func (I32Xor) instruction()    {}
func (I32Xor) numericInstr()   {}

type I32Shl struct{}

func (I32Shl) Opcode() Opcode  { return OpI32Shl }
func (I32Shl) instruction()    {}
func (I32Shl) numericInstr()   {}

type I32ShrS struct{}

func (I32ShrS) Opcode() Opcode  { return OpI32ShrS }
func (I32ShrS) instruction()    {}
func (I32ShrS) numericInstr()   {}

type I32ShrU struct{}

func (I32ShrU) Opcode() Opcode  { return OpI32ShrU }
func (I32ShrU) instruction()    {}
func (I32ShrU) numericInstr()   {}

type I32Rotl struct{}

func (I32Rotl) Opcode() Opcode  { return OpI32Rotl }
func (I32Rotl) instruction()    {}
func (I32Rotl) numericInstr()   {}

type I32Rotr struct{}

func (I32Rotr) Opcode() Opcode  { return OpI32Rotr }
func (I32Rotr) instruction()    {}
func (I32Rotr) numericInstr()   {}

// i64 arithmetic

type I64Clz struct{}

func (I64Clz) Opcode() Opcode  { return OpI64Clz }
func (I64Clz) instruction()    {}
func (I64Clz) numericInstr()   {}

type I64Ctz struct{}

func (I64Ctz) Opcode() Opcode  { return OpI64Ctz }
func (I64Ctz) instruction()    {}
func (I64Ctz) numericInstr()   {}

type I64Popcnt struct{}

func (I64Popcnt) Opcode() Opcode  { return OpI64Popcnt }
func (I64Popcnt) instruction()    {}
func (I64Popcnt) numericInstr()   {}

type I64Add struct{}

func (I64Add) Opcode() Opcode  { return OpI64Add }
func (I64Add) instruction()    {}
func (I64Add) numericInstr()   {}

type I64Sub struct{}

func (I64Sub) Opcode() Opcode  { return OpI64Sub }
func (I64Sub) instruction()    {}
func (I64Sub) numericInstr()   {}

type I64Mul struct{}

func (I64Mul) Opcode() Opcode  { return OpI64Mul }
func (I64Mul) instruction()    {}
func (I64Mul) numericInstr()   {}

type I64DivS struct{}

func (I64DivS) Opcode() Opcode  { return OpI64DivS }
func (I64DivS) instruction()    {}
func (I64DivS) numericInstr()   {}

type I64DivU struct{}

func (I64DivU) Opcode() Opcode  { return OpI64DivU }
func (I64DivU) instruction()    {}
func (I64DivU) numericInstr()   {}

type I64RemS struct{}

func (I64RemS) Opcode() Opcode  { return OpI64RemS }
func (I64RemS) instruction()    {}
func (I64RemS) numericInstr()   {}

type I64RemU struct{}

func (I64RemU) Opcode() Opcode  { return OpI64RemU }
func (I64RemU) instruction()    {}
func (I64RemU) numericInstr()   {}

type I64And struct{}

func (I64And) Opcode() Opcode  { return OpI64And }
func (I64And) instruction()    {}
func (I64And) numericInstr()   {}

type I64Or struct{}

func (I64Or) Opcode() Opcode  { return OpI64Or }
func (I64Or) instruction()    {}
func (I64Or) numericInstr()   {}

type I64Xor struct{}

func (I64Xor) Opcode() Opcode  { return OpI64Xor }
func (I64Xor) instruction()    {}
func (I64Xor) numericInstr()   {}

type I64Shl struct{}

func (I64Shl) Opcode() Opcode  { return OpI64Shl }
func (I64Shl) instruction()    {}
func (I64Shl) numericInstr()   {}

type I64ShrS struct{}

func (I64ShrS) Opcode() Opcode  { return OpI64ShrS }
func (I64ShrS) instruction()    {}
func (I64ShrS) numericInstr()   {}

type I64ShrU struct{}

func (I64ShrU) Opcode() Opcode  { return OpI64ShrU }
func (I64ShrU) instruction()    {}
func (I64ShrU) numericInstr()   {}

type I64Rotl struct{}

func (I64Rotl) Opcode() Opcode  { return OpI64Rotl }
func (I64Rotl) instruction()    {}
func (I64Rotl) numericInstr()   {}

type I64Rotr struct{}

func (I64Rotr) Opcode() Opcode  { return OpI64Rotr }
func (I64Rotr) instruction()    {}
func (I64Rotr) numericInstr()   {}

// f32 arithmetic

type F32Abs struct{}

func (F32Abs) Opcode() Opcode  { return OpF32Abs }
func (F32Abs) instruction()    {}
func (F32Abs) numericInstr()   {}

type F32Neg struct{}

func (F32Neg) Opcode() Opcode  { return OpF32Neg }
func (F32Neg) instruction()    {}
func (F32Neg) numericInstr()   {}

type F32Ceil struct{}

func (F32Ceil) Opcode() Opcode  { return OpF32Ceil }
func (F32Ceil) instruction()    {}
func (F32Ceil) numericInstr()   {}

type F32Floor struct{}

func (F32Floor) Opcode() Opcode  { return OpF32Floor }
func (F32Floor) instruction()    {}
func (F32Floor) numericInstr()   {}

type F32Trunc struct{}

func (F32Trunc) Opcode() Opcode  { return OpF32Trunc }
func (F32Trunc) instruction()    {}
func (F32Trunc) numericInstr()   {}

type F32Nearest struct{}

func (F32Nearest) Opcode() Opcode  { return OpF32Nearest }
func (F32Nearest) instruction()    {}
func (F32Nearest) numericInstr()   {}

type F32Sqrt struct{}

func (F32Sqrt) Opcode() Opcode  { return OpF32Sqrt }
func (F32Sqrt) instruction()    {}
func (F32Sqrt) numericInstr()   {}

type F32Add struct{}

func (F32Add) Opcode() Opcode  { return OpF32Add }
func (F32Add) instruction()    {}
func (F32Add) numericInstr()   {}

type F32Sub struct{}

func (F32Sub) Opcode() Opcode  { return OpF32Sub }
func (F32Sub) instruction()    {}
func (F32Sub) numericInstr()   {}

type F32Mul struct{}

func (F32Mul) Opcode() Opcode  { return OpF32Mul }
func (F32Mul) instruction()    {}
func (F32Mul) numericInstr()   {}

type F32Div struct{}

func (F32Div) Opcode() Opcode  { return OpF32Div }
func (F32Div) instruction()    {}
func (F32Div) numericInstr()   {}

type F32Min struct{}

func (F32Min) Opcode() Opcode  { return OpF32Min }
func (F32Min) instruction()    {}
func (F32Min) numericInstr()   {}

type F32Max struct{}

func (F32Max) Opcode() Opcode  { return OpF32Max }
func (F32Max) instruction()    {}
func (F32Max) numericInstr()   {}

type F32Copysign struct{}

func (F32Copysign) Opcode() Opcode  { return OpF32Copysign }
func (F32Copysign) instruction()    {}
func (F32Copysign) numericInstr()   {}

// f64 arithmetic

type F64Abs struct{}

func (F64Abs) Opcode() Opcode  { return OpF64Abs }
func (F64Abs) instruction()    {}
func (F64Abs) numericInstr()   {}

type F64Neg struct{}

func (F64Neg) Opcode() Opcode  { return OpF64Neg }
func (F64Neg) instruction()    {}
func (F64Neg) numericInstr()   {}

type F64Ceil struct{}

func (F64Ceil) Opcode() Opcode  { return OpF64Ceil }
func (F64Ceil) instruction()    {}
func (F64Ceil) numericInstr()   {}

type F64Floor struct{}

func (F64Floor) Opcode() Opcode  { return OpF64Floor }
func (F64Floor) instruction()    {}
func (F64Floor) numericInstr()   {}

type F64Trunc struct{}

func (F64Trunc) Opcode() Opcode  { return OpF64Trunc }
func (F64Trunc) instruction()    {}
func (F64Trunc) numericInstr()   {}

type F64Nearest struct{}

func (F64Nearest) Opcode() Opcode  { return OpF64Nearest }
func (F64Nearest) instruction()    {}
func (F64Nearest) numericInstr()   {}

type F64Sqrt struct{}

func (F64Sqrt) Opcode() Opcode  { return OpF64Sqrt }
func (F64Sqrt) instruction()    {}
func (F64Sqrt) numericInstr()   {}

type F64Add struct{}

func (F64Add) Opcode() Opcode  { return OpF64Add }
func (F64Add) instruction()    {}
func (F64Add) numericInstr()   {}

type F64Sub struct{}

func (F64Sub) Opcode() Opcode  { return OpF64Sub }
func (F64Sub) instruction()    {}
func (F64Sub) numericInstr()   {}

type F64Mul struct{}

func (F64Mul) Opcode() Opcode  { return OpF64Mul }
func (F64Mul) instruction()    {}
func (F64Mul) numericInstr()   {}

type F64Div struct{}

func (F64Div) Opcode() Opcode  { return OpF64Div }
func (F64Div) instruction()    {}
func (F64Div) numericInstr()   {}

type F64Min struct{}

func (F64Min) Opcode() Opcode  { return OpF64Min }
func (F64Min) instruction()    {}
func (F64Min) numericInstr()   {}

type F64Max struct{}

func (F64Max) Opcode() Opcode  { return OpF64Max }
func (F64Max) instruction()    {}
func (F64Max) numericInstr()   {}

type F64Copysign struct{}

func (F64Copysign) Opcode() Opcode  { return OpF64Copysign }
func (F64Copysign) instruction()    {}
func (F64Copysign) numericInstr()   {}
