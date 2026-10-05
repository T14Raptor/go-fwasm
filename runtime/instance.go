package runtime

import (
	"errors"
	"fmt"
	"math"

	"github.com/t14raptor/go-fwasm/compile"
	"github.com/t14raptor/go-fwasm/instruction"
	"github.com/t14raptor/go-fwasm/module"
	"github.com/t14raptor/go-fwasm/parser"
	"github.com/t14raptor/go-fwasm/types"
)

// Module is a parsed, validated and compiled module. It can be instantiated
// any number of times, in any store.
type Module struct {
	c *compile.Module
}

// Compile parses, validates and compiles a binary module.
func Compile(wasm []byte) (*Module, error) {
	m, err := parser.Parse(wasm)
	if err != nil {
		return nil, err
	}
	return CompileModule(m)
}

// CompileModule validates and compiles an already parsed module.
func CompileModule(m *module.Module) (*Module, error) {
	c, err := compile.Compile(m)
	if err != nil {
		return nil, err
	}
	return &Module{c: c}, nil
}

// Compiled exposes the lowered module, for tools.
func (m *Module) Compiled() *compile.Module { return m.c }

// ImportType describes one import. Exactly one of the type fields applies,
// per Kind.
type ImportType struct {
	Module, Name string
	Kind         types.ImportDescKind
	Func         *types.FuncType
	Table        types.TableType
	Memory       types.MemoryType
	Global       types.GlobalType
}

// Imports lists the module's imports in order.
func (m *Module) Imports() []ImportType {
	out := make([]ImportType, len(m.c.Imports))
	for i, imp := range m.c.Imports {
		it := ImportType{Module: imp.Module, Name: imp.Name, Kind: imp.Desc.Kind,
			Table: imp.Desc.TableType, Memory: imp.Desc.MemType, Global: imp.Desc.GlobalType}
		if imp.Desc.Kind == types.ImportFunc {
			it.Func = &m.c.Types[imp.Desc.TypeIdx]
		}
		out[i] = it
	}
	return out
}

// Exports lists the module's exports in order.
func (m *Module) Exports() []types.Export { return m.c.Exports }

// Imports maps module name, then field name, to the extern to link.
type Imports map[string]map[string]Extern

// LinkError reports an import that is missing or has the wrong type.
type LinkError struct {
	Module, Name string
	Reason       string
}

func (e *LinkError) Error() string {
	return fmt.Sprintf("link error: import %s.%s: %s", e.Module, e.Name, e.Reason)
}

// Instance is an instantiated module.
type Instance struct {
	store   *Store
	module  *Module
	funcs   []*Function
	tables  []*Table
	mems    []*Memory
	mem     *Memory // memory 0, or nil
	globals []*Global
	typeIDs []uint32
	elems   [][]uint64 // nil once dropped
	datas   [][]byte   // nil once dropped
	exports map[string]Extern
	caller  Caller
}

// Instantiate links m against imports, initializes its tables and memory and
// runs its start function. A *LinkError means an import did not match. If a
// segment or the start function traps, the error is a *Trap, and like in the
// spec any writes that already happened to imported tables or memories stay.
func (s *Store) Instantiate(m *Module, imports Imports) (*Instance, error) {
	c := m.c
	inst := &Instance{store: s, module: m, exports: map[string]Extern{}}
	inst.caller = Caller{inst: inst}

	for _, imp := range c.Imports {
		ext, ok := imports[imp.Module][imp.Name]
		if !ok || ext == nil {
			return nil, &LinkError{imp.Module, imp.Name, "unknown import"}
		}
		if err := inst.link(imp, ext); err != nil {
			return nil, err
		}
	}

	inst.typeIDs = make([]uint32, len(c.Types))
	for i := range c.Types {
		inst.typeIDs[i] = s.typeID(&c.Types[i])
	}

	for i, fn := range c.Funcs {
		idx := uint32(c.NumImportedFuncs + i)
		f := &Function{store: s, typ: fn.Type, typeID: s.typeID(fn.Type), code: fn, inst: inst, index: idx,
			numParams: len(fn.Type.Params), numResults: len(fn.Type.Results),
			numLocals: fn.NumLocals, frameSize: fn.NumLocals + fn.MaxHeight}
		s.addFunc(f)
		inst.funcs = append(inst.funcs, f)
	}
	for _, t := range c.Module.Tables {
		tab, err := newTable(t)
		if err != nil {
			return nil, err
		}
		inst.tables = append(inst.tables, tab)
	}
	for _, mt := range c.Module.Memories {
		mem, err := newMemory(mt, s.memoryPageCap())
		if err != nil {
			return nil, err
		}
		inst.mems = append(inst.mems, mem)
	}
	if len(inst.mems) > 0 {
		inst.mem = inst.mems[0]
	}
	for _, g := range c.Module.Globals {
		v, err := inst.eval(g.Init)
		if err != nil {
			return nil, err
		}
		inst.globals = append(inst.globals, &Global{typ: g.Type, val: v})
	}

	for _, exp := range c.Exports {
		var ext Extern
		switch exp.Desc.Kind {
		case types.ExportFunc:
			f := inst.funcs[exp.Desc.Idx]
			if f.name == "" && f.inst == inst {
				f.name = exp.Name
			}
			ext = f
		case types.ExportTable:
			ext = inst.tables[exp.Desc.Idx]
		case types.ExportMem:
			ext = inst.mems[exp.Desc.Idx]
		case types.ExportGlobal:
			ext = inst.globals[exp.Desc.Idx]
		}
		inst.exports[exp.Name] = ext
	}

	inst.elems = make([][]uint64, len(c.Elements))
	for i, e := range c.Elements {
		refs := make([]uint64, len(e.Init))
		for j, init := range e.Init {
			v, err := inst.eval(init)
			if err != nil {
				return nil, err
			}
			refs[j] = v
		}
		inst.elems[i] = refs
	}
	inst.datas = make([][]byte, len(c.Datas))
	for i, d := range c.Datas {
		inst.datas[i] = d.Init
	}

	for i, e := range c.Elements {
		switch e.Mode {
		case module.ElementModeActive:
			off, err := inst.eval(e.Offset)
			if err != nil {
				return nil, err
			}
			if code := inst.tableInit(inst.tables[e.TableIdx], uint32(i), uint32(off), 0, uint32(len(inst.elems[i]))); code != 0 {
				return inst, &Trap{Code: code}
			}
			inst.elems[i] = nil
		case module.ElementModeDeclarative:
			inst.elems[i] = nil
		}
	}
	for i, d := range c.Datas {
		if d.Mode != module.DataModeActive {
			continue
		}
		off, err := inst.eval(d.Offset)
		if err != nil {
			return nil, err
		}
		if code := inst.memoryInit(uint32(i), uint32(off), 0, uint32(len(d.Init))); code != 0 {
			return inst, &Trap{Code: code}
		}
		inst.datas[i] = nil
	}

	if c.Start != nil {
		if _, err := inst.funcs[*c.Start].Call(); err != nil {
			return inst, err
		}
	}
	return inst, nil
}

func (inst *Instance) link(imp types.Import, ext Extern) error {
	fail := func(reason string) error { return &LinkError{imp.Module, imp.Name, reason} }
	switch imp.Desc.Kind {
	case types.ImportFunc:
		f, ok := ext.(*Function)
		if !ok {
			return fail("incompatible import type: expected function")
		}
		if f.store != inst.store {
			return fail("function belongs to another store")
		}
		if !sameFuncType(f.typ, &inst.module.c.Types[imp.Desc.TypeIdx]) {
			return fail("incompatible import type: function signature mismatch")
		}
		inst.funcs = append(inst.funcs, f)
	case types.ImportTable:
		t, ok := ext.(*Table)
		if !ok {
			return fail("incompatible import type: expected table")
		}
		want := imp.Desc.TableType
		if t.typ.ElemType != want.ElemType || !limitsMatch(t.Type().Limits, want.Limits) {
			return fail("incompatible import type: table type mismatch")
		}
		inst.tables = append(inst.tables, t)
	case types.ImportMem:
		m, ok := ext.(*Memory)
		if !ok {
			return fail("incompatible import type: expected memory")
		}
		if !limitsMatch(m.Type().Limits, imp.Desc.MemType.Limits) {
			return fail("incompatible import type: memory type mismatch")
		}
		inst.mems = append(inst.mems, m)
		if inst.mem == nil {
			inst.mem = m
		}
	case types.ImportGlobal:
		g, ok := ext.(*Global)
		if !ok {
			return fail("incompatible import type: expected global")
		}
		if g.typ != imp.Desc.GlobalType {
			return fail("incompatible import type: global type mismatch")
		}
		inst.globals = append(inst.globals, g)
	}
	return nil
}

func sameFuncType(a, b *types.FuncType) bool {
	if len(a.Params) != len(b.Params) || len(a.Results) != len(b.Results) {
		return false
	}
	for i := range a.Params {
		if a.Params[i] != b.Params[i] {
			return false
		}
	}
	for i := range a.Results {
		if a.Results[i] != b.Results[i] {
			return false
		}
	}
	return true
}

// limitsMatch is import subtyping: the provided limits must fit the declared.
func limitsMatch(have, want types.Limits) bool {
	if have.Min < want.Min {
		return false
	}
	if want.HasMax {
		return have.HasMax && have.Max <= want.Max
	}
	return true
}

// eval runs a validated constant expression.
func (inst *Instance) eval(expr []instruction.Instruction) (uint64, error) {
	switch x := expr[0].(type) {
	case instruction.Numeric:
		switch v := x.Instr.(type) {
		case instruction.I32Const:
			return uint64(uint32(v.Value)), nil
		case instruction.I64Const:
			return uint64(v.Value), nil
		case instruction.F32Const:
			return uint64(math.Float32bits(v.Value)), nil
		case instruction.F64Const:
			return math.Float64bits(v.Value), nil
		}
	case instruction.Reference:
		switch v := x.Instr.(type) {
		case instruction.RefNull:
			return 0, nil
		case instruction.RefFunc:
			return inst.funcs[v.FuncIdx].addr, nil
		}
	case instruction.Variable:
		if g, ok := x.Instr.(instruction.GlobalGet); ok {
			return inst.globals[g.GlobalIdx].val, nil
		}
	}
	return 0, errors.New("unsupported constant expression")
}

// tableInit copies n refs from element segment seg at s to table t at d.
func (inst *Instance) tableInit(t *Table, seg, d, s, n uint32) TrapCode {
	refs := inst.elems[seg]
	if uint64(s)+uint64(n) > uint64(len(refs)) || uint64(d)+uint64(n) > uint64(len(t.elems)) {
		return TrapTableOutOfBounds
	}
	copy(t.elems[d:d+n], refs[s:s+n])
	return 0
}

// memoryInit copies n bytes from data segment seg at s to memory 0 at d.
func (inst *Instance) memoryInit(seg, d, s, n uint32) TrapCode {
	data := inst.datas[seg]
	mem := inst.mem.buf
	if uint64(s)+uint64(n) > uint64(len(data)) || uint64(d)+uint64(n) > uint64(len(mem)) {
		return TrapMemoryOutOfBounds
	}
	copy(mem[d:d+n], data[s:s+n])
	return 0
}

// Store returns the store the instance lives in.
func (inst *Instance) Store() *Store { return inst.store }

// Module returns the module the instance was created from.
func (inst *Instance) Module() *Module { return inst.module }

// Export returns the named export, or nil.
func (inst *Instance) Export(name string) Extern { return inst.exports[name] }

// Exports returns every export by name.
func (inst *Instance) Exports() map[string]Extern { return inst.exports }

// Func returns the named exported function, or nil.
func (inst *Instance) Func(name string) *Function {
	f, _ := inst.exports[name].(*Function)
	return f
}

// Memory returns the named exported memory, or nil.
func (inst *Instance) Memory(name string) *Memory {
	m, _ := inst.exports[name].(*Memory)
	return m
}

// Global returns the named exported global, or nil.
func (inst *Instance) Global(name string) *Global {
	g, _ := inst.exports[name].(*Global)
	return g
}

// Table returns the named exported table, or nil.
func (inst *Instance) Table(name string) *Table {
	t, _ := inst.exports[name].(*Table)
	return t
}

// Function returns a function by index in the instance's function index
// space, imports included, or nil.
func (inst *Instance) Function(idx uint32) *Function {
	if idx >= uint32(len(inst.funcs)) {
		return nil
	}
	return inst.funcs[idx]
}

// DefaultMemory returns memory 0, or nil.
func (inst *Instance) DefaultMemory() *Memory { return inst.mem }
