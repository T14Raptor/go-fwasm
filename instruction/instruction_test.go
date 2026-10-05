package instruction

import "testing"

func TestCategoryInterfaces(t *testing.T) {
	// Test that instructions implement their category interfaces
	var _ ControlInstr = Block{}
	var _ ControlInstr = If{}
	var _ ControlInstr = Call{}

	var _ VariableInstr = LocalGet{}
	var _ VariableInstr = GlobalSet{}

	var _ NumericInstr = I32Add{}
	var _ NumericInstr = I32Const{}
	var _ NumericInstr = F64Mul{}

	var _ MemoryInstr = I32Load{}
	var _ MemoryInstr = I64Store{}

	var _ ReferenceInstr = RefNull{}
	var _ ReferenceInstr = RefFunc{}

	var _ ParametricInstr = Drop{}
	var _ ParametricInstr = Select{}
}

func TestWrapperTypes(t *testing.T) {
	// Test wrapper types work correctly
	block := Block{}
	ctrl := Control{Instr: block}

	if ctrl.Opcode() != OpBlock {
		t.Errorf("expected OpBlock, got %v", ctrl.Opcode())
	}

	// Verify Control implements Instruction
	var _ Instruction = ctrl

	add := I32Add{}
	num := Numeric{Instr: add}

	if num.Opcode() != OpI32Add {
		t.Errorf("expected OpI32Add, got %v", num.Opcode())
	}
}
