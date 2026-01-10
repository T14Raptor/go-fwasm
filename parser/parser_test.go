package parser

import (
	"testing"

	"github.com/t14raptor/go-fwasm/instruction"
)

func TestParseMinimalModule(t *testing.T) {
	// Minimal valid WASM module: magic + version only
	data := []byte{
		0x00, 0x61, 0x73, 0x6d, // magic: \0asm
		0x01, 0x00, 0x00, 0x00, // version: 1
	}

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse minimal module: %v", err)
	}
	if mod == nil {
		t.Fatal("expected non-nil module")
	}
}

func TestParseWithTypeSection(t *testing.T) {
	// WASM module with type section defining (i32, i32) -> i32
	data := []byte{
		0x00, 0x61, 0x73, 0x6d, // magic
		0x01, 0x00, 0x00, 0x00, // version
		// Type section
		0x01,       // section id: type
		0x07,       // section size: 7 bytes
		0x01,       // num types: 1
		0x60,       // func type marker
		0x02,       // num params: 2
		0x7f, 0x7f, // params: i32, i32
		0x01, // num results: 1
		0x7f, // results: i32
	}

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(mod.Types) != 1 {
		t.Fatalf("expected 1 type, got %d", len(mod.Types))
	}
	if len(mod.Types[0].Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(mod.Types[0].Params))
	}
	if len(mod.Types[0].Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(mod.Types[0].Results))
	}
}

func TestParseWithFunction(t *testing.T) {
	// WASM module with a simple add function
	data := []byte{
		0x00, 0x61, 0x73, 0x6d, // magic
		0x01, 0x00, 0x00, 0x00, // version
		// Type section
		0x01,       // section id: type
		0x07,       // section size
		0x01,       // num types
		0x60,       // func type
		0x02,       // 2 params
		0x7f, 0x7f, // i32, i32
		0x01, // 1 result
		0x7f, // i32
		// Function section
		0x03, // section id: function
		0x02, // section size
		0x01, // num functions
		0x00, // type index 0
		// Code section
		0x0a, // section id: code
		0x09, // section size
		0x01, // num code entries
		0x07, // code size
		0x00, // num locals
		// Function body: local.get 0, local.get 1, i32.add, end
		0x20, 0x00, // local.get 0
		0x20, 0x01, // local.get 1
		0x6a, // i32.add
		0x0b, // end
	}

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(mod.Codes) != 1 {
		t.Fatalf("expected 1 code, got %d", len(mod.Codes))
	}
	if len(mod.Codes[0].Body) != 3 {
		t.Fatalf("expected 3 instructions, got %d", len(mod.Codes[0].Body))
	}

	// Verify instruction types - now wrapped in category wrappers
	if v, ok := mod.Codes[0].Body[0].(instruction.Variable); !ok {
		t.Errorf("expected Variable wrapper, got %T", mod.Codes[0].Body[0])
	} else if _, ok := v.Instr.(instruction.LocalGet); !ok {
		t.Errorf("expected LocalGet inside Variable, got %T", v.Instr)
	}
	if v, ok := mod.Codes[0].Body[1].(instruction.Variable); !ok {
		t.Errorf("expected Variable wrapper, got %T", mod.Codes[0].Body[1])
	} else if _, ok := v.Instr.(instruction.LocalGet); !ok {
		t.Errorf("expected LocalGet inside Variable, got %T", v.Instr)
	}
	if n, ok := mod.Codes[0].Body[2].(instruction.Numeric); !ok {
		t.Errorf("expected Numeric wrapper, got %T", mod.Codes[0].Body[2])
	} else if _, ok := n.Instr.(instruction.I32Add); !ok {
		t.Errorf("expected I32Add inside Numeric, got %T", n.Instr)
	}
}

func TestCategoryInterfaces(t *testing.T) {
	// Test that instructions implement their category interfaces
	var _ instruction.ControlInstr = instruction.Block{}
	var _ instruction.ControlInstr = instruction.If{}
	var _ instruction.ControlInstr = instruction.Call{}

	var _ instruction.VariableInstr = instruction.LocalGet{}
	var _ instruction.VariableInstr = instruction.GlobalSet{}

	var _ instruction.NumericInstr = instruction.I32Add{}
	var _ instruction.NumericInstr = instruction.I32Const{}
	var _ instruction.NumericInstr = instruction.F64Mul{}

	var _ instruction.MemoryInstr = instruction.I32Load{}
	var _ instruction.MemoryInstr = instruction.I64Store{}

	var _ instruction.ReferenceInstr = instruction.RefNull{}
	var _ instruction.ReferenceInstr = instruction.RefFunc{}

	var _ instruction.ParametricInstr = instruction.Drop{}
	var _ instruction.ParametricInstr = instruction.Select{}
}

func TestWrapperTypes(t *testing.T) {
	// Test wrapper types work correctly
	block := instruction.Block{}
	ctrl := instruction.Control{Instr: block}

	if ctrl.Opcode() != instruction.OpBlock {
		t.Errorf("expected OpBlock, got %v", ctrl.Opcode())
	}

	// Verify Control implements Instruction
	var _ instruction.Instruction = ctrl

	add := instruction.I32Add{}
	num := instruction.Numeric{Instr: add}

	if num.Opcode() != instruction.OpI32Add {
		t.Errorf("expected OpI32Add, got %v", num.Opcode())
	}
}
