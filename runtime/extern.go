package runtime

import (
	"fmt"

	"github.com/t14raptor/go-fwasm/compile"
	"github.com/t14raptor/go-fwasm/types"
)

// Extern is an importable or exportable object: *Function, *Memory, *Table
// or *Global.
type Extern interface{ externKind() types.ExportDescKind }

func (*Function) externKind() types.ExportDescKind { return types.ExportFunc }
func (*Table) externKind() types.ExportDescKind    { return types.ExportTable }
func (*Memory) externKind() types.ExportDescKind   { return types.ExportMem }
func (*Global) externKind() types.ExportDescKind   { return types.ExportGlobal }

// HostFunc implements a function in Go. args holds one slot per parameter
// (see value.go for the encoding) and is only valid until the function
// returns. It must return exactly one slot per result. A non-nil error
// aborts the wasm frames between this call and the innermost Call, and
// that Call returns the same error value.
type HostFunc func(c *Caller, args []uint64) ([]uint64, error)

// Caller is what a host function sees of the wasm code calling it.
type Caller struct {
	inst *Instance
}

// Instance returns the calling instance, or nil when the host function was
// called directly from Go.
func (c *Caller) Instance() *Instance { return c.inst }

// Memory returns memory 0 of the calling instance, or nil.
func (c *Caller) Memory() *Memory {
	if c.inst == nil {
		return nil
	}
	return c.inst.mem
}

// Function is a wasm or host function.
type Function struct {
	store  *Store
	typ    *types.FuncType
	typeID uint32
	addr   uint64 // funcref value

	// Wasm functions.
	code *compile.Func
	inst *Instance
	// Copied out of typ and code so a call touches one object.
	numParams, numResults int
	numLocals, frameSize  int

	// Host functions.
	host HostFunc

	index uint32 // in the defining module's function index space
	name  string
}

// Type returns the function's signature.
func (f *Function) Type() *types.FuncType { return f.typ }

// IsHost reports whether f is implemented in Go.
func (f *Function) IsHost() bool { return f.host != nil }

// Index returns f's index in its module's function index space. It is 0 for
// host functions.
func (f *Function) Index() uint32 { return f.index }

// Name returns the export or import name f was created with, if any.
func (f *Function) Name() string { return f.name }

// Instance returns the instance that defines f, or nil for host functions.
func (f *Function) Instance() *Instance { return f.inst }

func (f *Function) String() string {
	if f.host != nil {
		return fmt.Sprintf("host %s", f.name)
	}
	if f.name != "" {
		return fmt.Sprintf("func[%d] <%s>", f.index, f.name)
	}
	return fmt.Sprintf("func[%d]", f.index)
}

// Call runs f with args encoded as in value.go and returns its results.
func (f *Function) Call(args ...uint64) ([]uint64, error) {
	return f.store.invoke(f, args)
}

// Global is a global variable.
type Global struct {
	typ types.GlobalType
	val uint64
}

// Type returns the global's type.
func (g *Global) Type() types.GlobalType { return g.typ }

// Get returns the global's value.
func (g *Global) Get() uint64 { return g.val }

// Set changes a mutable global.
func (g *Global) Set(v uint64) error {
	if !g.typ.Mutable {
		return fmt.Errorf("global is immutable")
	}
	g.val = normalize(g.typ.ValType, v)
	return nil
}

// Table is a table of references: funcref values from Function.Ref, or host
// chosen externref values. Zero is null.
type Table struct {
	typ   types.TableType
	elems []uint64
	max   uint32
}

func newTable(t types.TableType) (*Table, error) {
	max := uint32(compile.MaxTableSize)
	if t.Limits.HasMax {
		max = min(max, t.Limits.Max)
	}
	if t.Limits.Min > max {
		return nil, fmt.Errorf("table minimum %d exceeds limit %d", t.Limits.Min, max)
	}
	return &Table{typ: t, elems: make([]uint64, t.Limits.Min), max: max}, nil
}

// Type returns the table's type with its current size as the minimum.
func (t *Table) Type() types.TableType {
	tt := t.typ
	tt.Limits.Min = uint32(len(t.elems))
	return tt
}

// Size returns the number of elements.
func (t *Table) Size() uint32 { return uint32(len(t.elems)) }

// Get returns element i.
func (t *Table) Get(i uint32) (uint64, bool) {
	if i >= uint32(len(t.elems)) {
		return 0, false
	}
	return t.elems[i], true
}

// Set changes element i.
func (t *Table) Set(i uint32, ref uint64) bool {
	if i >= uint32(len(t.elems)) {
		return false
	}
	t.elems[i] = ref
	return true
}

// Grow adds delta elements set to init and returns the previous size.
func (t *Table) Grow(delta uint32, init uint64) (uint32, bool) {
	old := uint32(len(t.elems))
	if uint64(old)+uint64(delta) > uint64(t.max) {
		return 0, false
	}
	t.elems = append(t.elems, make([]uint64, delta)...)
	if init != 0 {
		for i := old; i < old+delta; i++ {
			t.elems[i] = init
		}
	}
	return old, true
}

// Ref returns the funcref value for f.
func (f *Function) Ref() uint64 { return f.addr }

// FuncByRef resolves a funcref value from this store, or returns nil for
// null and foreign values.
func (s *Store) FuncByRef(ref uint64) *Function {
	if ref == 0 || ref > uint64(len(s.funcs)) {
		return nil
	}
	return s.funcs[ref-1]
}
