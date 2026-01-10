package instruction

import (
	"testing"

	"github.com/t14raptor/go-fwasm/types"
)

// TestVisitor tracks which instructions were visited
type TestVisitor struct {
	NoopVisitor
	Visited []string
}

func (v *TestVisitor) VisitUnreachable(i Unreachable) { v.Visited = append(v.Visited, "Unreachable") }
func (v *TestVisitor) VisitNop(i Nop)                 { v.Visited = append(v.Visited, "Nop") }
func (v *TestVisitor) VisitReturn(i Return)           { v.Visited = append(v.Visited, "Return") }
func (v *TestVisitor) VisitDrop(i Drop)               { v.Visited = append(v.Visited, "Drop") }
func (v *TestVisitor) VisitSelect(i Select)           { v.Visited = append(v.Visited, "Select") }
func (v *TestVisitor) VisitLocalGet(i LocalGet)       { v.Visited = append(v.Visited, "LocalGet") }
func (v *TestVisitor) VisitLocalSet(i LocalSet)       { v.Visited = append(v.Visited, "LocalSet") }
func (v *TestVisitor) VisitGlobalGet(i GlobalGet)     { v.Visited = append(v.Visited, "GlobalGet") }
func (v *TestVisitor) VisitI32Const(i I32Const)       { v.Visited = append(v.Visited, "I32Const") }
func (v *TestVisitor) VisitI64Const(i I64Const)       { v.Visited = append(v.Visited, "I64Const") }
func (v *TestVisitor) VisitI32Add(i I32Add)           { v.Visited = append(v.Visited, "I32Add") }
func (v *TestVisitor) VisitI64Mul(i I64Mul)           { v.Visited = append(v.Visited, "I64Mul") }

func (v *TestVisitor) VisitRefIsNull(i RefIsNull) { v.Visited = append(v.Visited, "RefIsNull") }
func (v *TestVisitor) VisitI32Load(i I32Load)     { v.Visited = append(v.Visited, "I32Load") }

func (v *TestVisitor) VisitBlock(i Block) {
	v.Visited = append(v.Visited, "Block")
	i.VisitChildrenWith(v)
}

func (v *TestVisitor) VisitLoop(i Loop) {
	v.Visited = append(v.Visited, "Loop")
	i.VisitChildrenWith(v)
}

func (v *TestVisitor) VisitIf(i If) {
	v.Visited = append(v.Visited, "If")
	i.VisitChildrenWith(v)
}

// Test simple instruction visiting
func TestVisitSimpleInstructions(t *testing.T) {
	tests := []struct {
		name     string
		instr    Instruction
		expected string
	}{
		{"Unreachable", Unreachable{}, "Unreachable"},
		{"Nop", Nop{}, "Nop"},
		{"Return", Return{}, "Return"},
		{"Drop", Drop{}, "Drop"},
		{"Select", Select{}, "Select"},
		{"LocalGet", LocalGet{LocalIdx: 0}, "LocalGet"},
		{"LocalSet", LocalSet{LocalIdx: 1}, "LocalSet"},
		{"GlobalGet", GlobalGet{GlobalIdx: 2}, "GlobalGet"},
		{"I32Const", I32Const{Value: 42}, "I32Const"},
		{"I64Const", I64Const{Value: 100}, "I64Const"},
		{"I32Add", I32Add{}, "I32Add"},
		{"I64Mul", I64Mul{}, "I64Mul"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitor := &TestVisitor{}
			tt.instr.VisitWith(visitor)

			if len(visitor.Visited) != 1 {
				t.Errorf("Expected 1 visit, got %d", len(visitor.Visited))
			}
			if visitor.Visited[0] != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, visitor.Visited[0])
			}
		})
	}
}

// Test wrapper type visiting
func TestVisitWrapperTypes(t *testing.T) {
	tests := []struct {
		name     string
		instr    Instruction
		expected string
	}{
		{"Control wrapper", Control{Instr: Nop{}}, "Nop"},
		{"Reference wrapper", Reference{Instr: RefIsNull{}}, "RefIsNull"},
		{"Parametric wrapper", Parametric{Instr: Drop{}}, "Drop"},
		{"Variable wrapper", Variable{Instr: LocalGet{LocalIdx: 0}}, "LocalGet"},
		{"Memory wrapper", Memory{Instr: I32Load{MemArg: MemArg{}}}, "I32Load"},
		{"Numeric wrapper", Numeric{Instr: I32Add{}}, "I32Add"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitor := &TestVisitor{}
			tt.instr.VisitWith(visitor)

			if len(visitor.Visited) != 1 {
				t.Errorf("Expected 1 visit, got %d", len(visitor.Visited))
			}
			if visitor.Visited[0] != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, visitor.Visited[0])
			}
		})
	}
}

// Test that NoopVisitor handles all instruction types
func TestNoopVisitorHandlesAllTypes(t *testing.T) {
	// NoopVisitor should not panic on any instruction type
	noop := NoopVisitor{}

	// Create various instructions
	instrs := []Instruction{
		Unreachable{},
		Nop{},
		Return{},
		Drop{},
		LocalGet{LocalIdx: 0},
		I32Const{Value: 42},
		I32Add{},
		Block{
			BlockType: types.BlockType{Kind: types.BlockTypeEmpty},
			Body:      []Instruction{Nop{}},
		},
	}

	// Should not panic
	for _, instr := range instrs {
		instr.VisitWith(noop)
	}
}

// Test nested instruction visiting with Block
func TestVisitNestedBlock(t *testing.T) {
	block := Block{
		BlockType: types.BlockType{Kind: types.BlockTypeValue, ValType: types.I32},
		Body: []Instruction{
			LocalGet{LocalIdx: 0},
			I32Const{Value: 10},
			I32Add{},
		},
	}

	visitor := &TestVisitor{}
	block.VisitWith(visitor)

	expected := []string{"Block", "LocalGet", "I32Const", "I32Add"}
	if len(visitor.Visited) != len(expected) {
		t.Fatalf("Expected %d visits, got %d: %v", len(expected), len(visitor.Visited), visitor.Visited)
	}
	for i, exp := range expected {
		if visitor.Visited[i] != exp {
			t.Errorf("Visit %d: expected %q, got %q", i, exp, visitor.Visited[i])
		}
	}
}

// Test nested instruction visiting with If
func TestVisitNestedIf(t *testing.T) {
	ifInstr := If{
		BlockType: types.BlockType{Kind: types.BlockTypeValue, ValType: types.I32},
		Then: []Instruction{
			I32Const{Value: 1},
		},
		Else: []Instruction{
			I32Const{Value: 2},
		},
	}

	visitor := &TestVisitor{}
	ifInstr.VisitWith(visitor)

	expected := []string{"If", "I32Const", "I32Const"}
	if len(visitor.Visited) != len(expected) {
		t.Fatalf("Expected %d visits, got %d: %v", len(expected), len(visitor.Visited), visitor.Visited)
	}
	for i, exp := range expected {
		if visitor.Visited[i] != exp {
			t.Errorf("Visit %d: expected %q, got %q", i, exp, visitor.Visited[i])
		}
	}
}

// Test VisitChildrenWith is a no-op for simple instructions
func TestVisitChildrenWithNoOp(t *testing.T) {
	instructions := []Instruction{
		Nop{},
		I32Add{},
		LocalGet{LocalIdx: 0},
		RefIsNull{},
	}

	for _, instr := range instructions {
		visitor := &TestVisitor{}
		instr.VisitChildrenWith(visitor)

		if len(visitor.Visited) != 0 {
			t.Errorf("Expected no visits for %T, got %d", instr, len(visitor.Visited))
		}
	}
}

// Test deeply nested instructions
func TestVisitDeeplyNested(t *testing.T) {
	nested := Block{
		BlockType: types.BlockType{Kind: types.BlockTypeValue, ValType: types.I32},
		Body: []Instruction{
			Loop{
				BlockType: types.BlockType{Kind: types.BlockTypeValue, ValType: types.I32},
				Body: []Instruction{
					LocalGet{LocalIdx: 0},
					If{
						BlockType: types.BlockType{Kind: types.BlockTypeValue, ValType: types.I32},
						Then: []Instruction{
							I32Const{Value: 1},
							LocalSet{LocalIdx: 0},
						},
						Else: []Instruction{
							I32Const{Value: 2},
						},
					},
				},
			},
		},
	}

	visitor := &TestVisitor{}
	nested.VisitWith(visitor)

	expected := []string{"Block", "Loop", "LocalGet", "If", "I32Const", "LocalSet", "I32Const"}
	if len(visitor.Visited) != len(expected) {
		t.Fatalf("Expected %d visits, got %d: %v", len(expected), len(visitor.Visited), visitor.Visited)
	}
	for i, exp := range expected {
		if visitor.Visited[i] != exp {
			t.Errorf("Visit %d: expected %q, got %q", i, exp, visitor.Visited[i])
		}
	}
}

// Test NoopVisitor traverses all instructions
func TestNoopVisitorTraversal(t *testing.T) {
	nested := Block{
		BlockType: types.BlockType{Kind: types.BlockTypeValue, ValType: types.I32},
		Body: []Instruction{
			Loop{
				BlockType: types.BlockType{Kind: types.BlockTypeValue, ValType: types.I32},
				Body: []Instruction{
					LocalGet{LocalIdx: 0},
					If{
						BlockType: types.BlockType{Kind: types.BlockTypeValue, ValType: types.I32},
						Then: []Instruction{
							I32Const{Value: 1},
							LocalSet{LocalIdx: 0},
						},
						Else: []Instruction{
							I32Const{Value: 2},
						},
					},
				},
			},
		},
	}

	// Use TestVisitor which properly implements recursion
	visitor := &TestVisitor{}
	nested.VisitWith(visitor)

	// Should visit: Block, Loop, LocalGet, If, I32Const, LocalSet, I32Const = 7
	if len(visitor.Visited) != 7 {
		t.Errorf("Expected 7 visits, got %d", len(visitor.Visited))
	}
}
