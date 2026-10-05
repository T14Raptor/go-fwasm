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

func TestParseManyTypes(t *testing.T) {
	// Test module with 10 different function types
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section with 10 types
		0x01, // section id
	}

	// Build type section content
	typeSection := []byte{0x0a} // 10 types
	for i := 0; i < 10; i++ {
		// Each type: (i32 * i) -> i32
		typeSection = append(typeSection, 0x60)    // func
		typeSection = append(typeSection, byte(i)) // num params
		for j := 0; j < i; j++ {
			typeSection = append(typeSection, 0x7f) // i32
		}
		typeSection = append(typeSection, 0x01, 0x7f) // 1 result: i32
	}

	data = append(data, byte(len(typeSection)))
	data = append(data, typeSection...)

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(mod.Types) != 10 {
		t.Errorf("expected 10 types, got %d", len(mod.Types))
	}

	// Verify param counts
	for i := 0; i < 10; i++ {
		if len(mod.Types[i].Params) != i {
			t.Errorf("type %d: expected %d params, got %d", i, i, len(mod.Types[i].Params))
		}
	}
}

func TestParseMultipleImports(t *testing.T) {
	// Test module with imports of each kind (func, table, memory, global)
	// Build imports dynamically to ensure correct sizes
	imports := []byte{
		0x04, // 4 imports
		// Import 0: func - "m"."f" -> type 0
		0x01, 'm', // module name
		0x01, 'f', // name
		0x00, 0x00, // kind=func, type index 0
		// Import 1: table - "m"."t" -> funcref table min=0
		0x01, 'm',
		0x01, 't',
		0x01, 0x70, 0x00, 0x01, // kind=table, funcref, flags=0, min=1
		// Import 2: memory - "m"."m" -> min=1
		0x01, 'm',
		0x01, 'm',
		0x02, 0x00, 0x01, // kind=memory, flags=0, min=1
		// Import 3: global - "m"."g" -> i32 immutable
		0x01, 'm',
		0x01, 'g',
		0x03, 0x7f, 0x00, // kind=global, i32, immutable
	}

	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section: () -> ()
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Import section
		0x02, byte(len(imports)),
	}
	data = append(data, imports...)

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(mod.Imports) != 4 {
		t.Fatalf("expected 4 imports, got %d", len(mod.Imports))
	}

	// Verify import kinds
	if mod.Imports[0].Desc.Kind != 0x00 { // func
		t.Errorf("import 0: expected func, got %d", mod.Imports[0].Desc.Kind)
	}
	if mod.Imports[1].Desc.Kind != 0x01 { // table
		t.Errorf("import 1: expected table, got %d", mod.Imports[1].Desc.Kind)
	}
	if mod.Imports[2].Desc.Kind != 0x02 { // memory
		t.Errorf("import 2: expected memory, got %d", mod.Imports[2].Desc.Kind)
	}
	if mod.Imports[3].Desc.Kind != 0x03 { // global
		t.Errorf("import 3: expected global, got %d", mod.Imports[3].Desc.Kind)
	}
}

func TestParseMultipleExports(t *testing.T) {
	// Test module with exports of each kind
	exports := []byte{
		0x04,                  // 4 exports
		0x01, 'f', 0x00, 0x00, // func export: name="f", kind=0, idx=0
		0x01, 't', 0x01, 0x00, // table export: name="t", kind=1, idx=0
		0x01, 'm', 0x02, 0x00, // memory export: name="m", kind=2, idx=0
		0x01, 'g', 0x03, 0x00, // global export: name="g", kind=3, idx=0
	}

	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Table section
		0x04, 0x04, 0x01, 0x70, 0x00, 0x01,
		// Memory section
		0x05, 0x03, 0x01, 0x00, 0x01,
		// Global section
		0x06, 0x06, 0x01, 0x7f, 0x00, 0x41, 0x00, 0x0b, // i32 const 0
		// Export section
		0x07, byte(len(exports)),
	}
	data = append(data, exports...)
	// Code section
	data = append(data, 0x0a, 0x04, 0x01, 0x02, 0x00, 0x0b)

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(mod.Exports) != 4 {
		t.Fatalf("expected 4 exports, got %d", len(mod.Exports))
	}
}

func TestParseStartFunction(t *testing.T) {
	// Test module with start function
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Type section
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Start section
		0x08, 0x01, 0x00, // start function index 0
		// Code section
		0x0a, 0x04, 0x01, 0x02, 0x00, 0x0b,
	}

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if mod.Start == nil {
		t.Fatal("expected start function")
	}
	if *mod.Start != 0 {
		t.Errorf("expected start function index 0, got %d", *mod.Start)
	}
}

func TestParseActiveDataSection(t *testing.T) {
	// Test module with active data segment
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Memory section
		0x05, 0x03, 0x01, 0x00, 0x01,
		// Data section
		0x0b, 0x0b, // section id and size
		0x01,             // 1 data segment
		0x00,             // active, memory 0
		0x41, 0x00, 0x0b, // offset: i32.const 0, end
		0x05, 'h', 'e', 'l', 'l', 'o', // data
	}

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(mod.Datas) != 1 {
		t.Fatalf("expected 1 data segment, got %d", len(mod.Datas))
	}

	if mod.Datas[0].Mode != 1 { // active
		t.Errorf("expected active data mode (1), got %d", mod.Datas[0].Mode)
	}
	if string(mod.Datas[0].Init) != "hello" {
		t.Errorf("expected 'hello', got '%s'", string(mod.Datas[0].Init))
	}
}

func TestParsePassiveDataSection(t *testing.T) {
	// Test module with passive data segment (requires data count section)
	// Passive data format: 0x01 (flag) + vec(byte)
	dataSection := []byte{
		0x01,                          // 1 data segment
		0x01,                          // passive flag
		0x05, 'w', 'o', 'r', 'l', 'd', // vec length + data bytes
	}

	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Memory section
		0x05, 0x03, 0x01, 0x00, 0x01,
		// Data count section
		0x0c, 0x01, 0x01, // 1 data segment
		// Data section
		0x0b, byte(len(dataSection)),
	}
	data = append(data, dataSection...)

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(mod.Datas) != 1 {
		t.Fatalf("expected 1 data segment, got %d", len(mod.Datas))
	}

	if mod.Datas[0].Mode != 0 { // passive
		t.Errorf("expected passive data mode (0), got %d", mod.Datas[0].Mode)
	}
}

func TestParseGlobalsWithInit(t *testing.T) {
	// Test module with multiple globals with initializers
	globalSection := []byte{
		0x03, // 3 globals
		// Global 0: i32 mutable = 42
		0x7f, 0x01, 0x41, 0x2a, 0x0b, // i32, mut, i32.const 42, end
		// Global 1: i64 immutable = 100
		0x7e, 0x00, 0x42, 0xe4, 0x00, 0x0b, // i64, immut, i64.const 100, end
		// Global 2: f32 mutable = 3.14
		0x7d, 0x01, 0x43, 0xc3, 0xf5, 0x48, 0x40, 0x0b, // f32, mut, f32.const, 4 bytes, end
	}

	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		// Global section
		0x06, byte(len(globalSection)),
	}
	data = append(data, globalSection...)

	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(mod.Globals) != 3 {
		t.Fatalf("expected 3 globals, got %d", len(mod.Globals))
	}

	// Check global 0: i32 mutable
	if !mod.Globals[0].Type.Mutable {
		t.Error("global 0 should be mutable")
	}

	// Check global 1: i64 immutable
	if mod.Globals[1].Type.Mutable {
		t.Error("global 1 should be immutable")
	}
}

func TestParseEmptyInput(t *testing.T) {
	// Empty input
	data := []byte{}

	_, err := Parse(data)
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseInvalidMagic(t *testing.T) {
	// Invalid magic number
	data := []byte{
		0x00, 0x00, 0x00, 0x00, // wrong magic
		0x01, 0x00, 0x00, 0x00, // version
	}

	_, err := Parse(data)
	if err == nil {
		t.Error("expected error for invalid magic")
	}
}

func TestParseInvalidVersion(t *testing.T) {
	// Invalid version number
	data := []byte{
		0x00, 0x61, 0x73, 0x6d, // correct magic
		0x02, 0x00, 0x00, 0x00, // wrong version (2 instead of 1)
	}

	_, err := Parse(data)
	if err == nil {
		t.Error("expected error for invalid version")
	}
}

func TestParseTruncatedHeader(t *testing.T) {
	// Truncated header (only magic, no version)
	data := []byte{
		0x00, 0x61, 0x73, 0x6d, // magic only
	}

	_, err := Parse(data)
	if err == nil {
		t.Error("expected error for truncated header")
	}
}

func TestParseTruncatedSection(t *testing.T) {
	// Section with incorrect size (claims 100 bytes but has fewer)
	data := []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		0x01, 0x64, // type section, size=100 (but data ends)
		0x01, // only 1 byte of content
	}

	_, err := Parse(data)
	if err == nil {
		t.Error("expected error for truncated section")
	}
}
