package instruction

import "github.com/t14raptor/go-fwasm/types"

type Instruction interface {
	Opcode() Opcode
	VisitWith(Visitor)
	VisitChildrenWith(Visitor)
	instruction()
}

// Category interfaces

type ControlInstr interface {
	Instruction
	controlInstr()
}

type ReferenceInstr interface {
	Instruction
	referenceInstr()
}

type ParametricInstr interface {
	Instruction
	parametricInstr()
}

type VariableInstr interface {
	Instruction
	variableInstr()
}

type MemoryInstr interface {
	Instruction
	memoryInstr()
}

type NumericInstr interface {
	Instruction
	numericInstr()
}

// Wrapper types

type Control struct {
	Instr ControlInstr
}

func (c Control) Opcode() Opcode { return c.Instr.Opcode() }
func (Control) instruction()     {}

type Reference struct {
	Instr ReferenceInstr
}

func (r Reference) Opcode() Opcode { return r.Instr.Opcode() }
func (Reference) instruction()     {}

type Parametric struct {
	Instr ParametricInstr
}

func (p Parametric) Opcode() Opcode { return p.Instr.Opcode() }
func (Parametric) instruction()     {}

type Variable struct {
	Instr VariableInstr
}

func (v Variable) Opcode() Opcode { return v.Instr.Opcode() }
func (Variable) instruction()     {}

type Memory struct {
	Instr MemoryInstr
}

func (m Memory) Opcode() Opcode { return m.Instr.Opcode() }
func (Memory) instruction()     {}

type Numeric struct {
	Instr NumericInstr
}

func (n Numeric) Opcode() Opcode { return n.Instr.Opcode() }
func (Numeric) instruction()     {}

// Helper types

type MemArg struct {
	Align  uint32
	Offset uint32
}

type BlockType = types.BlockType
