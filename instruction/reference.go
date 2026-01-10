package instruction

import "github.com/t14raptor/go-fwasm/types"

type RefNull struct {
	Type types.ValueType
}

func (RefNull) Opcode() Opcode  { return OpRefNull }
func (RefNull) instruction()    {}
func (RefNull) referenceInstr() {}

type RefIsNull struct{}

func (RefIsNull) Opcode() Opcode  { return OpRefIsNull }
func (RefIsNull) instruction()    {}
func (RefIsNull) referenceInstr() {}

type RefFunc struct {
	FuncIdx uint32
}

func (RefFunc) Opcode() Opcode  { return OpRefFunc }
func (RefFunc) instruction()    {}
func (RefFunc) referenceInstr() {}
