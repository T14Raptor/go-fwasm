package instruction

import "github.com/t14raptor/go-fwasm/types"

type Drop struct{}

func (Drop) Opcode() Opcode   { return OpDrop }
func (Drop) instruction()     {}
func (Drop) parametricInstr() {}

type Select struct{}

func (Select) Opcode() Opcode   { return OpSelect }
func (Select) instruction()     {}
func (Select) parametricInstr() {}

type SelectT struct {
	Types []types.ValueType
}

func (SelectT) Opcode() Opcode   { return OpSelectT }
func (SelectT) instruction()     {}
func (SelectT) parametricInstr() {}
