package instruction

import "github.com/t14raptor/go-fwasm/types"

type Unreachable struct{}

func (Unreachable) Opcode() Opcode { return OpUnreachable }
func (Unreachable) instruction()   {}
func (Unreachable) controlInstr()  {}

type Nop struct{}

func (Nop) Opcode() Opcode { return OpNop }
func (Nop) instruction()   {}
func (Nop) controlInstr()  {}

type Block struct {
	BlockType types.BlockType
	Body      []Instruction
}

func (Block) Opcode() Opcode { return OpBlock }
func (Block) instruction()   {}
func (Block) controlInstr()  {}

type Loop struct {
	BlockType types.BlockType
	Body      []Instruction
}

func (Loop) Opcode() Opcode { return OpLoop }
func (Loop) instruction()   {}
func (Loop) controlInstr()  {}

type If struct {
	BlockType types.BlockType
	Then      []Instruction
	Else      []Instruction
}

func (If) Opcode() Opcode { return OpIf }
func (If) instruction()   {}
func (If) controlInstr()  {}

type Br struct {
	LabelIdx uint32
}

func (Br) Opcode() Opcode { return OpBr }
func (Br) instruction()   {}
func (Br) controlInstr()  {}

type BrIf struct {
	LabelIdx uint32
}

func (BrIf) Opcode() Opcode { return OpBrIf }
func (BrIf) instruction()   {}
func (BrIf) controlInstr()  {}

type BrTable struct {
	Labels       []uint32
	DefaultLabel uint32
}

func (BrTable) Opcode() Opcode { return OpBrTable }
func (BrTable) instruction()   {}
func (BrTable) controlInstr()  {}

type Return struct{}

func (Return) Opcode() Opcode { return OpReturn }
func (Return) instruction()   {}
func (Return) controlInstr()  {}

type Call struct {
	FuncIdx uint32
}

func (Call) Opcode() Opcode { return OpCall }
func (Call) instruction()   {}
func (Call) controlInstr()  {}

type CallIndirect struct {
	TypeIdx  uint32
	TableIdx uint32
}

func (CallIndirect) Opcode() Opcode { return OpCallIndirect }
func (CallIndirect) instruction()   {}
func (CallIndirect) controlInstr()  {}
