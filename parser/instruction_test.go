package parser

import (
	"math"
	"reflect"
	"testing"

	"github.com/t14raptor/go-fwasm/instruction"
)

func TestParseI32ConstExtremes(t *testing.T) {
	// Test i32.const with INT32_MIN and INT32_MAX
	// INT32_MAX (2147483647) in signed LEB128: ff ff ff ff 07
	// INT32_MIN (-2147483648) in signed LEB128: 80 80 80 80 78
	tests := []struct {
		name     string
		lebBytes []byte
		expected int32
	}{
		{"INT32_MAX", []byte{0xff, 0xff, 0xff, 0xff, 0x07}, 2147483647},
		{"INT32_MIN", []byte{0x80, 0x80, 0x80, 0x80, 0x78}, -2147483648},
		{"zero", []byte{0x00}, 0},
		{"minus_one", []byte{0x7f}, -1},
		{"one", []byte{0x01}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build module with i32.const instruction
			// codeSize = locals(1) + i32.const(1) + value(len) + end(1)
			codeSize := byte(1 + 1 + len(tt.lebBytes) + 1)
			sectionSize := byte(1 + 1 + int(codeSize)) // num entries + entry size byte + entry content
			data := []byte{
				0x00, 0x61, 0x73, 0x6d, // magic
				0x01, 0x00, 0x00, 0x00, // version
				// Type section: () -> i32
				0x01, 0x05, 0x01, 0x60, 0x00, 0x01, 0x7f,
				// Function section
				0x03, 0x02, 0x01, 0x00,
				// Code section
				0x0a, sectionSize, 0x01, codeSize, 0x00, // code header
				0x41, // i32.const
			}
			data = append(data, tt.lebBytes...)
			data = append(data, 0x0b) // end

			mod, err := Parse(data)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			if len(mod.Codes) != 1 || len(mod.Codes[0].Body) != 1 {
				t.Fatalf("expected 1 instruction, got %d", len(mod.Codes[0].Body))
			}

			numWrapper, ok := mod.Codes[0].Body[0].(instruction.Numeric)
			if !ok {
				t.Fatalf("expected Numeric wrapper, got %T", mod.Codes[0].Body[0])
			}
			i32const, ok := numWrapper.Instr.(instruction.I32Const)
			if !ok {
				t.Fatalf("expected I32Const, got %T", numWrapper.Instr)
			}
			if i32const.Value != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, i32const.Value)
			}
		})
	}
}

func TestParseI64ConstExtremes(t *testing.T) {
	// Test i64.const with INT64_MIN and INT64_MAX
	// INT64_MAX in signed LEB128: ff ff ff ff ff ff ff ff ff 00
	// INT64_MIN in signed LEB128: 80 80 80 80 80 80 80 80 80 7f
	tests := []struct {
		name     string
		lebBytes []byte
		expected int64
	}{
		{"INT64_MAX", []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00}, 9223372036854775807},
		{"INT64_MIN", []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x7f}, -9223372036854775808},
		{"zero", []byte{0x00}, 0},
		{"minus_one", []byte{0x7f}, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// codeSize = locals(1) + i64.const(1) + value(len) + end(1)
			codeSize := byte(1 + 1 + len(tt.lebBytes) + 1)
			sectionSize := byte(1 + 1 + int(codeSize))
			data := []byte{
				0x00, 0x61, 0x73, 0x6d,
				0x01, 0x00, 0x00, 0x00,
				// Type section: () -> i64
				0x01, 0x05, 0x01, 0x60, 0x00, 0x01, 0x7e,
				// Function section
				0x03, 0x02, 0x01, 0x00,
				// Code section
				0x0a, sectionSize, 0x01, codeSize, 0x00,
				0x42, // i64.const
			}
			data = append(data, tt.lebBytes...)
			data = append(data, 0x0b)

			mod, err := Parse(data)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			if len(mod.Codes) != 1 || len(mod.Codes[0].Body) != 1 {
				t.Fatalf("expected 1 instruction")
			}

			numWrapper := mod.Codes[0].Body[0].(instruction.Numeric)
			i64const := numWrapper.Instr.(instruction.I64Const)
			if i64const.Value != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, i64const.Value)
			}
		})
	}
}

func TestParseF32ConstSpecialValues(t *testing.T) {
	// Test f32.const with special float values
	// F32 is 4 bytes little-endian
	tests := []struct {
		name     string
		bytes    []byte
		checkFn  func(float32) bool
		describe string
	}{
		{"positive_infinity", []byte{0x00, 0x00, 0x80, 0x7f}, func(v float32) bool { return math.IsInf(float64(v), 1) }, "+Inf"},
		{"negative_infinity", []byte{0x00, 0x00, 0x80, 0xff}, func(v float32) bool { return math.IsInf(float64(v), -1) }, "-Inf"},
		{"nan", []byte{0x00, 0x00, 0xc0, 0x7f}, func(v float32) bool { return math.IsNaN(float64(v)) }, "NaN"},
		{"negative_zero", []byte{0x00, 0x00, 0x00, 0x80}, func(v float32) bool { return v == 0 && math.Signbit(float64(v)) }, "-0"},
		{"positive_zero", []byte{0x00, 0x00, 0x00, 0x00}, func(v float32) bool { return v == 0 && !math.Signbit(float64(v)) }, "+0"},
		{"one", []byte{0x00, 0x00, 0x80, 0x3f}, func(v float32) bool { return v == 1.0 }, "1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// f32 is 4 bytes, code entry = locals(1) + f32.const(1) + 4 bytes + end(1) = 7
			data := []byte{
				0x00, 0x61, 0x73, 0x6d,
				0x01, 0x00, 0x00, 0x00,
				// Type section: () -> f32
				0x01, 0x05, 0x01, 0x60, 0x00, 0x01, 0x7d,
				// Function section
				0x03, 0x02, 0x01, 0x00,
				// Code section: section_size=9, 1 entry, entry_size=7
				0x0a, 0x09, 0x01, 0x07, 0x00,
				0x43, // f32.const
			}
			data = append(data, tt.bytes...)
			data = append(data, 0x0b)

			mod, err := Parse(data)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			if len(mod.Codes) != 1 || len(mod.Codes[0].Body) != 1 {
				t.Fatalf("expected 1 instruction")
			}

			numWrapper := mod.Codes[0].Body[0].(instruction.Numeric)
			f32const := numWrapper.Instr.(instruction.F32Const)
			if !tt.checkFn(f32const.Value) {
				t.Errorf("expected %s, got %v", tt.describe, f32const.Value)
			}
		})
	}
}

func TestParseF64ConstSpecialValues(t *testing.T) {
	// Test f64.const with special double values
	// F64 is 8 bytes little-endian
	tests := []struct {
		name     string
		bytes    []byte
		checkFn  func(float64) bool
		describe string
	}{
		{"positive_infinity", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0x7f}, func(v float64) bool { return math.IsInf(v, 1) }, "+Inf"},
		{"negative_infinity", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0xff}, func(v float64) bool { return math.IsInf(v, -1) }, "-Inf"},
		{"nan", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf8, 0x7f}, func(v float64) bool { return math.IsNaN(v) }, "NaN"},
		{"negative_zero", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80}, func(v float64) bool { return v == 0 && math.Signbit(v) }, "-0"},
		{"positive_zero", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, func(v float64) bool { return v == 0 && !math.Signbit(v) }, "+0"},
		{"one", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0x3f}, func(v float64) bool { return v == 1.0 }, "1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// f64 is 8 bytes, code entry = locals(1) + f64.const(1) + 8 bytes + end(1) = 11
			data := []byte{
				0x00, 0x61, 0x73, 0x6d,
				0x01, 0x00, 0x00, 0x00,
				// Type section: () -> f64
				0x01, 0x05, 0x01, 0x60, 0x00, 0x01, 0x7c,
				// Function section
				0x03, 0x02, 0x01, 0x00,
				// Code section: section_size=13, 1 entry, entry_size=11
				0x0a, 0x0d, 0x01, 0x0b, 0x00,
				0x44, // f64.const
			}
			data = append(data, tt.bytes...)
			data = append(data, 0x0b)

			mod, err := Parse(data)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			if len(mod.Codes) != 1 || len(mod.Codes[0].Body) != 1 {
				t.Fatalf("expected 1 instruction")
			}

			numWrapper := mod.Codes[0].Body[0].(instruction.Numeric)
			f64const := numWrapper.Instr.(instruction.F64Const)
			if !tt.checkFn(f64const.Value) {
				t.Errorf("expected %s, got %v", tt.describe, f64const.Value)
			}
		})
	}
}

func TestParseDeeplyNestedBlocks(t *testing.T) {
	// Test 10 levels of nested blocks
	// block -> block -> block -> ... -> i32.const 42 -> end -> end -> ...
	const depth = 10

	// Build the code body
	var codeBody []byte
	for i := 0; i < depth; i++ {
		codeBody = append(codeBody, 0x02, 0x7f) // block (result i32)
	}
	codeBody = append(codeBody, 0x41, 0x2a) // i32.const 42
	for i := 0; i < depth; i++ {
		codeBody = append(codeBody, 0x0b) // end
	}
	codeBody = append(codeBody, 0x0b) // final end

	codeSize := len(codeBody) + 1 // +1 for locals count
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section: () -> i32
		0x01, 0x05, 0x01, 0x60, 0x00, 0x01, 0x7f,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Code section
		0x0a,
	}
	// Add code section size and code entry header
	codeSectionSize := 1 + 1 + 1 + len(codeBody) // num funcs + code size (1 byte) + locals + body
	data = append(data, byte(codeSectionSize))
	data = append(data, 0x01)           // 1 function
	data = append(data, byte(codeSize)) // code size
	data = append(data, 0x00)           // 0 locals
	data = append(data, codeBody...)

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse deeply nested blocks: %v", err)
	}

	// Verify we have nested blocks
	if len(mod.Codes) != 1 {
		t.Fatalf("expected 1 code entry")
	}

	// Count nesting depth by walking the AST
	actualDepth := countBlockDepth(mod.Codes[0].Body)
	if actualDepth != depth {
		t.Errorf("expected depth %d, got %d", depth, actualDepth)
	}
}

func countBlockDepth(instrs []instruction.Instruction) int {
	maxDepth := 0
	for _, instr := range instrs {
		switch i := instr.(type) {
		case instruction.Control:
			switch blk := i.Instr.(type) {
			case instruction.Block:
				d := 1 + countBlockDepth(blk.Body)
				if d > maxDepth {
					maxDepth = d
				}
			case instruction.Loop:
				d := 1 + countBlockDepth(blk.Body)
				if d > maxDepth {
					maxDepth = d
				}
			case instruction.If:
				d := 1 + countBlockDepth(blk.Then)
				if ed := 1 + countBlockDepth(blk.Else); ed > d {
					d = ed
				}
				if d > maxDepth {
					maxDepth = d
				}
			}
		}
	}
	return maxDepth
}

func TestParseLargeBrTable(t *testing.T) {
	// Test br_table with 50 labels
	const numLabels = 50

	var codeBody []byte
	codeBody = append(codeBody, 0x02, 0x40)      // block (empty)
	codeBody = append(codeBody, 0x41, 0x00)      // i32.const 0 (selector)
	codeBody = append(codeBody, 0x0e)            // br_table
	codeBody = append(codeBody, byte(numLabels)) // label count
	for i := 0; i < numLabels; i++ {
		codeBody = append(codeBody, 0x00) // all labels point to 0
	}
	codeBody = append(codeBody, 0x00) // default label
	codeBody = append(codeBody, 0x0b) // end block
	codeBody = append(codeBody, 0x0b) // end func

	codeSize := len(codeBody) + 1
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section: () -> ()
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Code section
		0x0a,
	}
	codeSectionSize := 1 + 1 + 1 + len(codeBody)
	data = append(data, byte(codeSectionSize))
	data = append(data, 0x01)
	data = append(data, byte(codeSize))
	data = append(data, 0x00)
	data = append(data, codeBody...)

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse large br_table: %v", err)
	}

	// Find the br_table instruction
	if len(mod.Codes) != 1 {
		t.Fatalf("expected 1 code entry")
	}

	found := false
	for _, instr := range mod.Codes[0].Body {
		if ctrl, ok := instr.(instruction.Control); ok {
			if blk, ok := ctrl.Instr.(instruction.Block); ok {
				for _, bi := range blk.Body {
					if ctrl2, ok := bi.(instruction.Control); ok {
						if brt, ok := ctrl2.Instr.(instruction.BrTable); ok {
							found = true
							if len(brt.Labels) != numLabels {
								t.Errorf("expected %d labels, got %d", numLabels, len(brt.Labels))
							}
						}
					}
				}
			}
		}
	}
	if !found {
		t.Error("br_table instruction not found")
	}
}

func TestParseNestedIfElse(t *testing.T) {
	// Test nested if/else with both branches populated
	// if (result i32)
	//   if (result i32)
	//     i32.const 1
	//   else
	//     i32.const 2
	//   end
	// else
	//   if (result i32)
	//     i32.const 3
	//   else
	//     i32.const 4
	//   end
	// end
	codeBody := []byte{
		0x41, 0x01, // i32.const 1 (condition)
		0x04, 0x7f, // if (result i32)
		0x41, 0x01, // i32.const 1 (inner condition)
		0x04, 0x7f, // if (result i32)
		0x41, 0x01, // i32.const 1
		0x05,       // else
		0x41, 0x02, // i32.const 2
		0x0b,       // end inner if
		0x05,       // else
		0x41, 0x01, // i32.const 1 (inner condition)
		0x04, 0x7f, // if (result i32)
		0x41, 0x03, // i32.const 3
		0x05,       // else
		0x41, 0x04, // i32.const 4
		0x0b, // end inner if
		0x0b, // end outer if
		0x0b, // end func
	}

	codeSize := len(codeBody) + 1
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section: () -> i32
		0x01, 0x05, 0x01, 0x60, 0x00, 0x01, 0x7f,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Code section
		0x0a,
	}
	codeSectionSize := 1 + 1 + 1 + len(codeBody)
	data = append(data, byte(codeSectionSize))
	data = append(data, 0x01)
	data = append(data, byte(codeSize))
	data = append(data, 0x00)
	data = append(data, codeBody...)

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse nested if/else: %v", err)
	}

	if len(mod.Codes) != 1 {
		t.Fatalf("expected 1 code entry")
	}

	// Count total If instructions (1 outer + 2 inner = 3)
	ifCount := countIfInstructions(mod.Codes[0].Body)
	if ifCount != 3 {
		t.Errorf("expected 3 If instructions, got %d", ifCount)
	}
}

func countIfInstructions(instrs []instruction.Instruction) int {
	count := 0
	for _, instr := range instrs {
		switch i := instr.(type) {
		case instruction.Control:
			switch blk := i.Instr.(type) {
			case instruction.If:
				count++
				count += countIfInstructions(blk.Then)
				count += countIfInstructions(blk.Else)
			case instruction.Block:
				count += countIfInstructions(blk.Body)
			case instruction.Loop:
				count += countIfInstructions(blk.Body)
			}
		}
	}
	return count
}

func TestParseBlockWithTypeIndex(t *testing.T) {
	// Test block with type index (multi-value)
	// type 0: (i32, i32) -> (i32, i32)
	// block (type 0) ... end
	codeBody := []byte{
		0x41, 0x01, // i32.const 1
		0x41, 0x02, // i32.const 2
		0x02, 0x00, // block (type 0)
		0x0b, // end block
		0x0b, // end func
	}

	codeSize := len(codeBody) + 1
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section: (i32, i32) -> (i32, i32)
		0x01, 0x08, 0x01, 0x60,
		0x02, 0x7f, 0x7f, // 2 params: i32, i32
		0x02, 0x7f, 0x7f, // 2 results: i32, i32
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Code section
		0x0a,
	}
	codeSectionSize := 1 + 1 + 1 + len(codeBody)
	data = append(data, byte(codeSectionSize))
	data = append(data, 0x01)
	data = append(data, byte(codeSize))
	data = append(data, 0x00)
	data = append(data, codeBody...)

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse block with type index: %v", err)
	}

	// Check that the block has a type index
	found := false
	for _, instr := range mod.Codes[0].Body {
		if ctrl, ok := instr.(instruction.Control); ok {
			if blk, ok := ctrl.Instr.(instruction.Block); ok {
				if blk.BlockType.Kind == 2 { // BlockTypeIndex
					found = true
					if blk.BlockType.TypeIdx != 0 {
						t.Errorf("expected type index 0, got %d", blk.BlockType.TypeIdx)
					}
				}
			}
		}
	}
	if !found {
		t.Error("block with type index not found")
	}
}

func TestParseExtendedTruncSatInstructions(t *testing.T) {
	// Test all saturating truncation instructions (0xFC 0x00 through 0xFC 0x07)
	tests := []struct {
		name   string
		extOp  byte
		expect string
	}{
		{"i32.trunc_sat_f32_s", 0x00, "I32TruncSatF32S"},
		{"i32.trunc_sat_f32_u", 0x01, "I32TruncSatF32U"},
		{"i32.trunc_sat_f64_s", 0x02, "I32TruncSatF64S"},
		{"i32.trunc_sat_f64_u", 0x03, "I32TruncSatF64U"},
		{"i64.trunc_sat_f32_s", 0x04, "I64TruncSatF32S"},
		{"i64.trunc_sat_f32_u", 0x05, "I64TruncSatF32U"},
		{"i64.trunc_sat_f64_s", 0x06, "I64TruncSatF64S"},
		{"i64.trunc_sat_f64_u", 0x07, "I64TruncSatF64U"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// f32.const 1.0, trunc_sat, drop
			codeBody := []byte{
				0x43, 0x00, 0x00, 0x80, 0x3f, // f32.const 1.0
				0xfc, tt.extOp, // extended opcode
				0x1a, // drop
				0x0b, // end
			}

			codeSize := len(codeBody) + 1
			data := []byte{
				0x00, 0x61, 0x73, 0x6d,
				0x01, 0x00, 0x00, 0x00,
				// Type section: () -> ()
				0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
				// Function section
				0x03, 0x02, 0x01, 0x00,
				// Code section
				0x0a,
			}
			codeSectionSize := 1 + 1 + 1 + len(codeBody)
			data = append(data, byte(codeSectionSize))
			data = append(data, 0x01)
			data = append(data, byte(codeSize))
			data = append(data, 0x00)
			data = append(data, codeBody...)

			mod, err := Parse(data)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			if len(mod.Codes) != 1 || len(mod.Codes[0].Body) != 3 {
				t.Fatalf("expected 3 instructions, got %d", len(mod.Codes[0].Body))
			}

			// Check the middle instruction is the trunc_sat
			numWrapper, ok := mod.Codes[0].Body[1].(instruction.Numeric)
			if !ok {
				t.Fatalf("expected Numeric wrapper, got %T", mod.Codes[0].Body[1])
			}

			typeName := reflect.TypeOf(numWrapper.Instr).Name()
			if typeName != tt.expect {
				t.Errorf("expected %s, got %s", tt.expect, typeName)
			}
		})
	}
}

func TestParseBulkMemoryInstructions(t *testing.T) {
	// Test memory.init, data.drop, memory.copy, memory.fill
	tests := []struct {
		name      string
		codeBytes []byte
		expect    string
	}{
		{
			"memory.init",
			[]byte{
				0x41, 0x00, // i32.const 0 (dest)
				0x41, 0x00, // i32.const 0 (src offset)
				0x41, 0x00, // i32.const 0 (len)
				0xfc, 0x08, 0x00, 0x00, // memory.init data_idx=0 mem_idx=0
				0x0b,
			},
			"MemoryInit",
		},
		{
			"data.drop",
			[]byte{
				0xfc, 0x09, 0x00, // data.drop data_idx=0
				0x0b,
			},
			"DataDrop",
		},
		{
			"memory.copy",
			[]byte{
				0x41, 0x00, // i32.const 0 (dest)
				0x41, 0x00, // i32.const 0 (src)
				0x41, 0x00, // i32.const 0 (len)
				0xfc, 0x0a, 0x00, 0x00, // memory.copy dst_mem=0 src_mem=0
				0x0b,
			},
			"MemoryCopy",
		},
		{
			"memory.fill",
			[]byte{
				0x41, 0x00, // i32.const 0 (dest)
				0x41, 0x00, // i32.const 0 (value)
				0x41, 0x00, // i32.const 0 (len)
				0xfc, 0x0b, 0x00, // memory.fill mem_idx=0
				0x0b,
			},
			"MemoryFill",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codeSize := len(tt.codeBytes) + 1
			data := []byte{
				0x00, 0x61, 0x73, 0x6d,
				0x01, 0x00, 0x00, 0x00,
				// Type section: () -> ()
				0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
				// Function section
				0x03, 0x02, 0x01, 0x00,
				// Memory section
				0x05, 0x03, 0x01, 0x00, 0x01, // 1 memory, min=0, max=1
				// Data count section (required for bulk memory)
				0x0c, 0x01, 0x01, // 1 data segment
				// Code section
				0x0a,
			}
			codeSectionSize := 1 + 1 + 1 + len(tt.codeBytes)
			data = append(data, byte(codeSectionSize))
			data = append(data, 0x01)
			data = append(data, byte(codeSize))
			data = append(data, 0x00)
			data = append(data, tt.codeBytes...)
			// Data section (passive)
			data = append(data, 0x0b, 0x03, 0x01, 0x01, 0x00) // 1 passive data segment

			mod, err := Parse(data)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", tt.name, err)
			}

			if len(mod.Codes) != 1 {
				t.Fatalf("expected 1 code entry")
			}

			// Find the expected instruction
			found := false
			for _, instr := range mod.Codes[0].Body {
				if mem, ok := instr.(instruction.Memory); ok {
					typeName := reflect.TypeOf(mem.Instr).Name()
					if typeName == tt.expect {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("%s instruction not found", tt.expect)
			}
		})
	}
}

func TestParseTableInstructions(t *testing.T) {
	// Test table.grow, table.size, table.fill (no element section needed)
	tests := []struct {
		name      string
		codeBytes []byte
		expect    string
	}{
		{
			"table.size",
			[]byte{
				0xfc, 0x10, 0x00, // table.size table_idx=0
				0x1a, // drop
				0x0b,
			},
			"TableSize",
		},
		{
			"table.grow",
			[]byte{
				0xd0, 0x70, // ref.null funcref
				0x41, 0x01, // i32.const 1 (delta)
				0xfc, 0x0f, 0x00, // table.grow table_idx=0
				0x1a, // drop
				0x0b,
			},
			"TableGrow",
		},
		{
			"table.fill",
			[]byte{
				0x41, 0x00, // i32.const 0 (start)
				0xd0, 0x70, // ref.null funcref
				0x41, 0x01, // i32.const 1 (len)
				0xfc, 0x11, 0x00, // table.fill table_idx=0
				0x0b,
			},
			"TableFill",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codeSize := len(tt.codeBytes) + 1
			data := []byte{
				0x00, 0x61, 0x73, 0x6d,
				0x01, 0x00, 0x00, 0x00,
				// Type section: () -> ()
				0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
				// Function section
				0x03, 0x02, 0x01, 0x00,
				// Table section: 1 funcref table, min=0, no max
				0x04, 0x04, 0x01, 0x70, 0x00, 0x10,
				// Code section
				0x0a,
			}
			codeSectionSize := 1 + 1 + 1 + len(tt.codeBytes)
			data = append(data, byte(codeSectionSize))
			data = append(data, 0x01)
			data = append(data, byte(codeSize))
			data = append(data, 0x00)
			data = append(data, tt.codeBytes...)

			mod, err := Parse(data)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", tt.name, err)
			}

			if len(mod.Codes) != 1 {
				t.Fatalf("expected 1 code entry")
			}

			// Find the expected instruction
			found := false
			for _, instr := range mod.Codes[0].Body {
				if v, ok := instr.(instruction.Variable); ok {
					typeName := reflect.TypeOf(v.Instr).Name()
					if typeName == tt.expect {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("%s instruction not found", tt.expect)
			}
		})
	}
}

func TestParseElemDrop(t *testing.T) {
	// Test elem.drop with a proper declarative element section
	// Element section format for declarative (flags=0x03):
	// flags(1) + elemkind(1) + vec count(1) = at minimum 3 bytes for empty
	codeBody := []byte{
		0xfc, 0x0d, 0x00, // elem.drop elem_idx=0
		0x0b,
	}

	codeSize := len(codeBody) + 1
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section: () -> ()
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Table section
		0x04, 0x04, 0x01, 0x70, 0x00, 0x10,
		// Element section: 1 declarative element with 0 items
		// flags=0x03 (declarative), elemkind=0x00 (funcref), count=0
		0x09, 0x04, 0x01, 0x03, 0x00, 0x00,
		// Code section
		0x0a,
	}
	codeSectionSize := 1 + 1 + 1 + len(codeBody)
	data = append(data, byte(codeSectionSize))
	data = append(data, 0x01)
	data = append(data, byte(codeSize))
	data = append(data, 0x00)
	data = append(data, codeBody...)

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse elem.drop: %v", err)
	}

	if len(mod.Codes) != 1 {
		t.Fatalf("expected 1 code entry")
	}

	// Find the elem.drop instruction
	found := false
	for _, instr := range mod.Codes[0].Body {
		if v, ok := instr.(instruction.Variable); ok {
			if _, ok := v.Instr.(instruction.ElemDrop); ok {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("ElemDrop instruction not found")
	}
}

func TestParseAllMemoryLoadVariants(t *testing.T) {
	// Test all load instruction variants with various offsets and alignments
	tests := []struct {
		name   string
		opcode byte
		align  byte   // log2 of alignment
		offset uint32 // LEB128 encoded
	}{
		{"i32.load", 0x28, 2, 0},
		{"i64.load", 0x29, 3, 0},
		{"f32.load", 0x2a, 2, 0},
		{"f64.load", 0x2b, 3, 0},
		{"i32.load8_s", 0x2c, 0, 0},
		{"i32.load8_u", 0x2d, 0, 0},
		{"i32.load16_s", 0x2e, 1, 0},
		{"i32.load16_u", 0x2f, 1, 0},
		{"i64.load8_s", 0x30, 0, 0},
		{"i64.load8_u", 0x31, 0, 0},
		{"i64.load16_s", 0x32, 1, 0},
		{"i64.load16_u", 0x33, 1, 0},
		{"i64.load32_s", 0x34, 2, 0},
		{"i64.load32_u", 0x35, 2, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codeBody := []byte{
				0x41, 0x00, // i32.const 0 (address)
				tt.opcode, tt.align, 0x00, // load with align and offset=0
				0x1a, // drop
				0x0b, // end
			}

			codeSize := len(codeBody) + 1
			data := []byte{
				0x00, 0x61, 0x73, 0x6d,
				0x01, 0x00, 0x00, 0x00,
				// Type section: () -> ()
				0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
				// Function section
				0x03, 0x02, 0x01, 0x00,
				// Memory section
				0x05, 0x03, 0x01, 0x00, 0x01,
				// Code section
				0x0a,
			}
			codeSectionSize := 1 + 1 + 1 + len(codeBody)
			data = append(data, byte(codeSectionSize))
			data = append(data, 0x01)
			data = append(data, byte(codeSize))
			data = append(data, 0x00)
			data = append(data, codeBody...)

			mod, err := Parse(data)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", tt.name, err)
			}

			if len(mod.Codes) != 1 || len(mod.Codes[0].Body) < 2 {
				t.Fatalf("expected at least 2 instructions")
			}

			// Verify it's a Memory instruction
			if _, ok := mod.Codes[0].Body[1].(instruction.Memory); !ok {
				t.Errorf("expected Memory wrapper for %s, got %T", tt.name, mod.Codes[0].Body[1])
			}
		})
	}
}

func TestParseAllMemoryStoreVariants(t *testing.T) {
	// Test all store instruction variants
	tests := []struct {
		name   string
		opcode byte
		align  byte
	}{
		{"i32.store", 0x36, 2},
		{"i64.store", 0x37, 3},
		{"f32.store", 0x38, 2},
		{"f64.store", 0x39, 3},
		{"i32.store8", 0x3a, 0},
		{"i32.store16", 0x3b, 1},
		{"i64.store8", 0x3c, 0},
		{"i64.store16", 0x3d, 1},
		{"i64.store32", 0x3e, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codeBody := []byte{
				0x41, 0x00, // i32.const 0 (address)
				0x41, 0x00, // i32.const 0 (value) - works for all types at bytecode level
				tt.opcode, tt.align, 0x00, // store with align and offset=0
				0x0b, // end
			}

			codeSize := len(codeBody) + 1
			data := []byte{
				0x00, 0x61, 0x73, 0x6d,
				0x01, 0x00, 0x00, 0x00,
				// Type section: () -> ()
				0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
				// Function section
				0x03, 0x02, 0x01, 0x00,
				// Memory section
				0x05, 0x03, 0x01, 0x00, 0x01,
				// Code section
				0x0a,
			}
			codeSectionSize := 1 + 1 + 1 + len(codeBody)
			data = append(data, byte(codeSectionSize))
			data = append(data, 0x01)
			data = append(data, byte(codeSize))
			data = append(data, 0x00)
			data = append(data, codeBody...)

			mod, err := Parse(data)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", tt.name, err)
			}

			if len(mod.Codes) != 1 || len(mod.Codes[0].Body) < 3 {
				t.Fatalf("expected at least 3 instructions")
			}

			// Verify it's a Memory instruction
			if _, ok := mod.Codes[0].Body[2].(instruction.Memory); !ok {
				t.Errorf("expected Memory wrapper for %s, got %T", tt.name, mod.Codes[0].Body[2])
			}
		})
	}
}

func TestParseMemoryWithLargeOffset(t *testing.T) {
	// Test i32.load with a large offset (0x7FFFFFFF - max positive i32)
	// LEB128 encoding of 0x7FFFFFFF: ff ff ff ff 07
	codeBody := []byte{
		0x41, 0x00, // i32.const 0
		0x28,                         // i32.load
		0x02,                         // align = 4
		0xff, 0xff, 0xff, 0xff, 0x07, // offset = 0x7FFFFFFF
		0x1a, // drop
		0x0b, // end
	}

	codeSize := len(codeBody) + 1
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section: () -> ()
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Memory section
		0x05, 0x03, 0x01, 0x00, 0x01,
		// Code section
		0x0a,
	}
	codeSectionSize := 1 + 1 + 1 + len(codeBody)
	data = append(data, byte(codeSectionSize))
	data = append(data, 0x01)
	data = append(data, byte(codeSize))
	data = append(data, 0x00)
	data = append(data, codeBody...)

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(mod.Codes) != 1 || len(mod.Codes[0].Body) < 2 {
		t.Fatalf("expected at least 2 instructions")
	}

	mem, ok := mod.Codes[0].Body[1].(instruction.Memory)
	if !ok {
		t.Fatalf("expected Memory wrapper, got %T", mod.Codes[0].Body[1])
	}

	load, ok := mem.Instr.(instruction.I32Load)
	if !ok {
		t.Fatalf("expected I32Load, got %T", mem.Instr)
	}

	if load.MemArg.Offset != 0x7FFFFFFF {
		t.Errorf("expected offset 0x7FFFFFFF, got 0x%X", load.MemArg.Offset)
	}
}

func TestParseMemoryWithNonNaturalAlignment(t *testing.T) {
	// Test i32.load with non-natural alignment (align=1 instead of 4)
	codeBody := []byte{
		0x41, 0x00, // i32.const 0
		0x28, // i32.load
		0x00, // align = 1 (non-natural for i32)
		0x00, // offset = 0
		0x1a, // drop
		0x0b, // end
	}

	codeSize := len(codeBody) + 1
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section: () -> ()
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Memory section
		0x05, 0x03, 0x01, 0x00, 0x01,
		// Code section
		0x0a,
	}
	codeSectionSize := 1 + 1 + 1 + len(codeBody)
	data = append(data, byte(codeSectionSize))
	data = append(data, 0x01)
	data = append(data, byte(codeSize))
	data = append(data, 0x00)
	data = append(data, codeBody...)

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	mem := mod.Codes[0].Body[1].(instruction.Memory)
	load := mem.Instr.(instruction.I32Load)

	if load.MemArg.Align != 0 {
		t.Errorf("expected align 0 (log2 of 1), got %d", load.MemArg.Align)
	}
}

func TestParseMemorySizeAndGrow(t *testing.T) {
	// Test memory.size and memory.grow instructions
	codeBody := []byte{
		0x3f, 0x00, // memory.size 0
		0x40, 0x00, // memory.grow 0
		0x1a, // drop
		0x0b, // end
	}

	codeSize := len(codeBody) + 1
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section: () -> ()
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Memory section
		0x05, 0x03, 0x01, 0x00, 0x01,
		// Code section
		0x0a,
	}
	codeSectionSize := 1 + 1 + 1 + len(codeBody)
	data = append(data, byte(codeSectionSize))
	data = append(data, 0x01)
	data = append(data, byte(codeSize))
	data = append(data, 0x00)
	data = append(data, codeBody...)

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(mod.Codes) != 1 || len(mod.Codes[0].Body) != 3 {
		t.Fatalf("expected 3 instructions, got %d", len(mod.Codes[0].Body))
	}

	// Check memory.size
	mem1, ok := mod.Codes[0].Body[0].(instruction.Memory)
	if !ok {
		t.Fatalf("expected Memory wrapper for memory.size")
	}
	if _, ok := mem1.Instr.(instruction.MemorySize); !ok {
		t.Errorf("expected MemorySize, got %T", mem1.Instr)
	}

	// Check memory.grow
	mem2, ok := mod.Codes[0].Body[1].(instruction.Memory)
	if !ok {
		t.Fatalf("expected Memory wrapper for memory.grow")
	}
	if _, ok := mem2.Instr.(instruction.MemoryGrow); !ok {
		t.Errorf("expected MemoryGrow, got %T", mem2.Instr)
	}
}

func TestParseUnknownOpcode(t *testing.T) {
	// Code section with unknown opcode
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Code section with unknown opcode
		0x0a, 0x05, 0x01, 0x03, 0x00,
		0xff, // unknown opcode
		0x0b, // end
	}

	_, err := Parse(data)
	if err == nil {
		t.Error("expected error for unknown opcode")
	}
}

func TestParseTruncatedInstruction(t *testing.T) {
	// i32.const without the value
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Code section with truncated i32.const
		0x0a, 0x04, 0x01, 0x02, 0x00,
		0x41, // i32.const but no value follows
	}

	_, err := Parse(data)
	if err == nil {
		t.Error("expected error for truncated instruction")
	}
}

func TestParseTruncatedMemArg(t *testing.T) {
	// i32.load without full memarg
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Memory section
		0x05, 0x03, 0x01, 0x00, 0x01,
		// Code section with truncated memarg
		0x0a, 0x05, 0x01, 0x03, 0x00,
		0x28, 0x02, // i32.load with align but no offset
	}

	_, err := Parse(data)
	if err == nil {
		t.Error("expected error for truncated memarg")
	}
}

func TestParseInvalidBlockType(t *testing.T) {
	// Block with invalid block type byte
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Code section with negative type index (invalid)
		0x0a, 0x06, 0x01, 0x04, 0x00,
		0x02,        // block
		0x7f ^ 0x80, // This creates an invalid negative s33 when decoded
		0x0b, 0x0b,  // end block, end func
	}

	// This may or may not error depending on how the parser handles it
	// The important thing is it doesn't panic
	_, _ = Parse(data)
}

func TestParseUnterminatedBlock(t *testing.T) {
	// Block without end instruction
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Code section - block without end
		0x0a, 0x05, 0x01, 0x03, 0x00,
		0x02, 0x40, // block (empty result)
		// no end instruction - truncated
	}

	_, err := Parse(data)
	if err == nil {
		t.Error("expected error for unterminated block")
	}
}
