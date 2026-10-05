package compile

import (
	"errors"
	"fmt"

	"github.com/t14raptor/go-fwasm/instruction"
	"github.com/t14raptor/go-fwasm/module"
	"github.com/t14raptor/go-fwasm/types"
)

// Limits on what a module may declare, as the WebAssembly JS API sets them
// for web embeddings.
const (
	MaxMemoryPages = 65536
	MaxTableSize   = 10_000_000
	MaxLocals      = 50_000
	MaxParams      = 1000
	MaxResults     = 1000
)

// ErrInvalid wraps every validation failure.
var ErrInvalid = errors.New("invalid module")

func invalidf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

// Module is a validated module whose function bodies have been lowered.
type Module struct {
	*module.Module

	// Index spaces, imports first.
	FuncTypes []*types.FuncType
	Tables    []types.TableType
	Memories  []types.MemoryType
	Globals   []types.GlobalType

	NumImportedFuncs   int
	NumImportedTables  int
	NumImportedMems    int
	NumImportedGlobals int

	// Funcs holds the defined functions, Funcs[i] being function index
	// NumImportedFuncs+i.
	Funcs []*Func
}

// Func is a lowered function body.
type Func struct {
	Type       *types.FuncType
	NumLocals  int // parameters plus declared locals
	MaxHeight  int // deepest operand stack above the locals
	Code       []Instr
	BrTable    []BrTarget
	LocalTypes []types.ValueType

	// Locals in [ZeroFrom, ZeroTo) must be zeroed on entry. Every other
	// declared local is written before it is read.
	ZeroFrom, ZeroTo int

	// Inlined lists the calls replaced by the callee's body, in pc order.
	// Plain is then the function without them, for when every call must
	// be observable; otherwise it is nil.
	Inlined []InlineSite
	Plain   *Func
}

// Compile validates m against the WebAssembly 2.0 rules (without SIMD) and
// lowers every function body.
func Compile(m *module.Module) (*Module, error) {
	c := &Module{Module: m}
	if err := c.validateTypes(); err != nil {
		return nil, err
	}
	if err := c.buildIndexSpaces(); err != nil {
		return nil, err
	}
	refs, err := c.validateModuleFields()
	if err != nil {
		return nil, err
	}
	if len(m.Functions) != len(m.Codes) {
		return nil, invalidf("function and code section have inconsistent lengths")
	}
	c.Funcs = make([]*Func, len(m.Codes))
	for i := range m.Codes {
		fn, err := compileFunc(c, refs, c.FuncTypes[c.NumImportedFuncs+i], &m.Codes[i])
		if err != nil {
			return nil, fmt.Errorf("func[%d]: %w", c.NumImportedFuncs+i, err)
		}
		c.Funcs[i] = fn
	}
	c.inline()
	for i, fn := range c.Funcs {
		for f := fn; f != nil; f = f.Plain {
			if err := c.verify(c.NumImportedFuncs+i, f); err != nil {
				return nil, err
			}
		}
	}
	return c, nil
}

func validValueType(t types.ValueType) bool {
	switch t {
	case types.I32, types.I64, types.F32, types.F64, types.FuncRef, types.ExternRef:
		return true
	}
	return false
}

func isRefType(t types.ValueType) bool { return t == types.FuncRef || t == types.ExternRef }

func (c *Module) validateTypes() error {
	for i, ft := range c.Types {
		if len(ft.Params) > MaxParams || len(ft.Results) > MaxResults {
			return invalidf("type[%d]: too many params or results", i)
		}
		for _, t := range ft.Params {
			if !validValueType(t) {
				return invalidf("type[%d]: unsupported value type 0x%02x", i, byte(t))
			}
		}
		for _, t := range ft.Results {
			if !validValueType(t) {
				return invalidf("type[%d]: unsupported value type 0x%02x", i, byte(t))
			}
		}
	}
	return nil
}

func (c *Module) funcType(idx uint32) (*types.FuncType, error) {
	if idx >= uint32(len(c.Types)) {
		return nil, invalidf("unknown type %d", idx)
	}
	return &c.Types[idx], nil
}

func (c *Module) buildIndexSpaces() error {
	for _, imp := range c.Imports {
		switch imp.Desc.Kind {
		case types.ImportFunc:
			ft, err := c.funcType(imp.Desc.TypeIdx)
			if err != nil {
				return err
			}
			c.FuncTypes = append(c.FuncTypes, ft)
			c.NumImportedFuncs++
		case types.ImportTable:
			if err := validateTableType(imp.Desc.TableType); err != nil {
				return err
			}
			c.Tables = append(c.Tables, imp.Desc.TableType)
			c.NumImportedTables++
		case types.ImportMem:
			if err := validateMemoryType(imp.Desc.MemType); err != nil {
				return err
			}
			c.Memories = append(c.Memories, imp.Desc.MemType)
			c.NumImportedMems++
		case types.ImportGlobal:
			if !validValueType(imp.Desc.GlobalType.ValType) {
				return invalidf("import %s.%s: unsupported global type", imp.Module, imp.Name)
			}
			c.Globals = append(c.Globals, imp.Desc.GlobalType)
			c.NumImportedGlobals++
		default:
			return invalidf("import %s.%s: unknown kind %d", imp.Module, imp.Name, imp.Desc.Kind)
		}
	}
	for _, typeIdx := range c.Functions {
		ft, err := c.funcType(typeIdx)
		if err != nil {
			return err
		}
		c.FuncTypes = append(c.FuncTypes, ft)
	}
	for _, t := range c.Module.Tables {
		if err := validateTableType(t); err != nil {
			return err
		}
		c.Tables = append(c.Tables, t)
	}
	for _, mt := range c.Module.Memories {
		if err := validateMemoryType(mt); err != nil {
			return err
		}
		c.Memories = append(c.Memories, mt)
	}
	if len(c.Memories) > 1 {
		return invalidf("multiple memories")
	}
	for _, g := range c.Module.Globals {
		if !validValueType(g.Type.ValType) {
			return invalidf("unsupported global type 0x%02x", byte(g.Type.ValType))
		}
		c.Globals = append(c.Globals, g.Type)
	}
	return nil
}

func validateTableType(t types.TableType) error {
	if !isRefType(t.ElemType) {
		return invalidf("table element type must be a reference type")
	}
	if t.Limits.HasMax && t.Limits.Max < t.Limits.Min {
		return invalidf("size minimum must not be greater than maximum")
	}
	return nil
}

func validateMemoryType(t types.MemoryType) error {
	if t.Limits.Min > MaxMemoryPages || (t.Limits.HasMax && t.Limits.Max > MaxMemoryPages) {
		return invalidf("memory size must be at most 65536 pages (4GiB)")
	}
	if t.Limits.HasMax && t.Limits.Max < t.Limits.Min {
		return invalidf("size minimum must not be greater than maximum")
	}
	return nil
}

// validateModuleFields checks everything outside function bodies and returns
// the set of functions that ref.func may name (the spec's C.refs).
func (c *Module) validateModuleFields() (map[uint32]bool, error) {
	refs := map[uint32]bool{}

	for i, g := range c.Module.Globals {
		t, err := c.constExpr(g.Init, refs)
		if err != nil {
			return nil, fmt.Errorf("global[%d]: %w", c.NumImportedGlobals+i, err)
		}
		if t != g.Type.ValType {
			return nil, invalidf("global[%d]: type mismatch", c.NumImportedGlobals+i)
		}
	}

	for i, e := range c.Elements {
		if !isRefType(e.Type) {
			return nil, invalidf("elem[%d]: element type must be a reference type", i)
		}
		for _, init := range e.Init {
			t, err := c.constExpr(init, refs)
			if err != nil {
				return nil, fmt.Errorf("elem[%d]: %w", i, err)
			}
			if t != e.Type {
				return nil, invalidf("elem[%d]: type mismatch", i)
			}
		}
		if e.Mode == module.ElementModeActive {
			if e.TableIdx >= uint32(len(c.Tables)) {
				return nil, invalidf("elem[%d]: unknown table %d", i, e.TableIdx)
			}
			if c.Tables[e.TableIdx].ElemType != e.Type {
				return nil, invalidf("elem[%d]: type mismatch", i)
			}
			t, err := c.constExpr(e.Offset, refs)
			if err != nil {
				return nil, fmt.Errorf("elem[%d]: %w", i, err)
			}
			if t != types.I32 {
				return nil, invalidf("elem[%d]: type mismatch", i)
			}
		}
	}

	for i, d := range c.Datas {
		if d.Mode != module.DataModeActive {
			continue
		}
		if d.MemIdx >= uint32(len(c.Memories)) {
			return nil, invalidf("data[%d]: unknown memory %d", i, d.MemIdx)
		}
		t, err := c.constExpr(d.Offset, refs)
		if err != nil {
			return nil, fmt.Errorf("data[%d]: %w", i, err)
		}
		if t != types.I32 {
			return nil, invalidf("data[%d]: type mismatch", i)
		}
	}

	names := map[string]bool{}
	for _, exp := range c.Exports {
		if names[exp.Name] {
			return nil, invalidf("duplicate export name %q", exp.Name)
		}
		names[exp.Name] = true
		var n int
		switch exp.Desc.Kind {
		case types.ExportFunc:
			n = len(c.FuncTypes)
			refs[exp.Desc.Idx] = true
		case types.ExportTable:
			n = len(c.Tables)
		case types.ExportMem:
			n = len(c.Memories)
		case types.ExportGlobal:
			n = len(c.Globals)
		default:
			return nil, invalidf("export %q: unknown kind %d", exp.Name, exp.Desc.Kind)
		}
		if exp.Desc.Idx >= uint32(n) {
			return nil, invalidf("export %q: unknown index %d", exp.Name, exp.Desc.Idx)
		}
	}

	if c.Start != nil {
		if *c.Start >= uint32(len(c.FuncTypes)) {
			return nil, invalidf("unknown function %d", *c.Start)
		}
		ft := c.FuncTypes[*c.Start]
		if len(ft.Params) != 0 || len(ft.Results) != 0 {
			return nil, invalidf("start function must have type [] -> []")
		}
	}
	return refs, nil
}

// constExpr validates a constant expression and returns its type. Any
// function it names is added to refs.
func (c *Module) constExpr(expr []instruction.Instruction, refs map[uint32]bool) (types.ValueType, error) {
	if len(expr) != 1 {
		if len(expr) == 0 {
			return 0, invalidf("type mismatch: empty constant expression")
		}
		return 0, invalidf("constant expression required")
	}
	switch x := expr[0].(type) {
	case instruction.Numeric:
		switch x.Instr.(type) {
		case instruction.I32Const:
			return types.I32, nil
		case instruction.I64Const:
			return types.I64, nil
		case instruction.F32Const:
			return types.F32, nil
		case instruction.F64Const:
			return types.F64, nil
		}
	case instruction.Reference:
		switch r := x.Instr.(type) {
		case instruction.RefNull:
			if !isRefType(r.Type) {
				return 0, invalidf("malformed reference type")
			}
			return r.Type, nil
		case instruction.RefFunc:
			if r.FuncIdx >= uint32(len(c.FuncTypes)) {
				return 0, invalidf("unknown function %d", r.FuncIdx)
			}
			refs[r.FuncIdx] = true
			return types.FuncRef, nil
		}
	case instruction.Variable:
		if g, ok := x.Instr.(instruction.GlobalGet); ok {
			// Only imported, immutable globals are visible to constant
			// expressions.
			if g.GlobalIdx >= uint32(c.NumImportedGlobals) {
				return 0, invalidf("unknown global %d", g.GlobalIdx)
			}
			if c.Globals[g.GlobalIdx].Mutable {
				return 0, invalidf("constant expression required")
			}
			return c.Globals[g.GlobalIdx].ValType, nil
		}
	}
	return 0, invalidf("constant expression required")
}
