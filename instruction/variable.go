package instruction

type LocalGet struct {
	LocalIdx uint32
}

func (LocalGet) Opcode() Opcode   { return OpLocalGet }
func (LocalGet) instruction()     {}
func (LocalGet) variableInstr()   {}

type LocalSet struct {
	LocalIdx uint32
}

func (LocalSet) Opcode() Opcode   { return OpLocalSet }
func (LocalSet) instruction()     {}
func (LocalSet) variableInstr()   {}

type LocalTee struct {
	LocalIdx uint32
}

func (LocalTee) Opcode() Opcode   { return OpLocalTee }
func (LocalTee) instruction()     {}
func (LocalTee) variableInstr()   {}

type GlobalGet struct {
	GlobalIdx uint32
}

func (GlobalGet) Opcode() Opcode   { return OpGlobalGet }
func (GlobalGet) instruction()     {}
func (GlobalGet) variableInstr()   {}

type GlobalSet struct {
	GlobalIdx uint32
}

func (GlobalSet) Opcode() Opcode   { return OpGlobalSet }
func (GlobalSet) instruction()     {}
func (GlobalSet) variableInstr()   {}

type TableGet struct {
	TableIdx uint32
}

func (TableGet) Opcode() Opcode   { return OpTableGet }
func (TableGet) instruction()     {}
func (TableGet) variableInstr()   {}

type TableSet struct {
	TableIdx uint32
}

func (TableSet) Opcode() Opcode   { return OpTableSet }
func (TableSet) instruction()     {}
func (TableSet) variableInstr()   {}
