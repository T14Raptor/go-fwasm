// Package runtime instantiates and runs compiled WebAssembly modules.
//
// A Store owns functions, memories, tables and globals, and the execution
// stack they run on. It is not safe for concurrent use: run one Store per
// goroutine. Calls are re-entrant, so a host function may call back into any
// function of the same Store, and an error it returns unwinds the wasm frames
// above it and comes back out of the Call that entered them.
package runtime

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/t14raptor/go-fwasm/types"
)

// Store defaults.
const (
	DefaultMaxCallDepth  = 10_000
	DefaultMaxStackSlots = 8 << 20 // 64 MiB of 8-byte slots
)

// Store holds runtime objects and the execution state shared by every
// instance created in it.
type Store struct {
	// MaxCallDepth bounds nested wasm calls. Exceeding it traps with
	// "call stack exhausted".
	MaxCallDepth int
	// MaxStackSlots bounds the value stack (locals plus operands).
	MaxStackSlots int
	// MaxMemoryPages caps every memory created afterwards, below whatever
	// maximum the module declares. Zero means the 65536-page spec limit.
	MaxMemoryPages uint32

	funcs   []*Function // indexed by funcref value - 1
	typeIDs map[string]uint32

	stack  []uint64
	sp     int
	frames []frame

	hooks     *Hooks
	interrupt atomic.Bool
}

// frame is a caller waiting for a call to return. It holds no pointers, so
// pushing one needs no write barrier.
type frame struct {
	fn uint32 // index in Store.funcs
	pc uint32 // return address
	fp int
}

// NewStore returns an empty store with default limits.
func NewStore() *Store {
	return &Store{
		MaxCallDepth:  DefaultMaxCallDepth,
		MaxStackSlots: DefaultMaxStackSlots,
		typeIDs:       map[string]uint32{},
		stack:         make([]uint64, 1<<12),
	}
}

// SetHooks installs tracing hooks, or removes them when h is nil. Hooks are
// read when a call enters wasm, so changing them from inside a host function
// takes effect at the next Call.
func (s *Store) SetHooks(h *Hooks) { s.hooks = h }

// Interrupt makes running wasm code trap with TrapInterrupted at its next
// loop iteration. It may be called from any goroutine. The flag stays set
// until ClearInterrupt.
func (s *Store) Interrupt() { s.interrupt.Store(true) }

// ClearInterrupt resets the flag set by Interrupt.
func (s *Store) ClearInterrupt() { s.interrupt.Store(false) }

// typeID interns a function signature, so call_indirect can compare
// signatures across instances with one integer comparison.
func (s *Store) typeID(ft *types.FuncType) uint32 {
	var b strings.Builder
	for _, t := range ft.Params {
		b.WriteByte(byte(t))
	}
	b.WriteByte(0)
	for _, t := range ft.Results {
		b.WriteByte(byte(t))
	}
	key := b.String()
	id, ok := s.typeIDs[key]
	if !ok {
		id = uint32(len(s.typeIDs))
		s.typeIDs[key] = id
	}
	return id
}

func (s *Store) addFunc(f *Function) {
	s.funcs = append(s.funcs, f)
	f.addr = uint64(len(s.funcs))
}

// growStack makes room for at least n slots.
func (s *Store) growStack(n int) bool {
	if n <= len(s.stack) {
		return true
	}
	if n > s.MaxStackSlots {
		return false
	}
	size := max(2*len(s.stack), n)
	size = min(size, s.MaxStackSlots)
	stack := make([]uint64, size)
	copy(stack, s.stack)
	s.stack = stack
	return true
}

func (s *Store) memoryPageCap() uint32 {
	if s.MaxMemoryPages == 0 || s.MaxMemoryPages > 65536 {
		return 65536
	}
	return s.MaxMemoryPages
}

// NewHostFunc creates a function implemented in Go. name only shows up in
// traps and hooks.
func (s *Store) NewHostFunc(name string, ft types.FuncType, fn HostFunc) *Function {
	f := &Function{store: s, typ: &ft, host: fn, name: name,
		numParams: len(ft.Params), numResults: len(ft.Results)}
	f.typeID = s.typeID(f.typ)
	s.addFunc(f)
	return f
}

// NewMemory creates a memory of t.Limits.Min pages.
func (s *Store) NewMemory(t types.MemoryType) (*Memory, error) {
	if t.Limits.Min > 65536 || (t.Limits.HasMax && (t.Limits.Max > 65536 || t.Limits.Max < t.Limits.Min)) {
		return nil, fmt.Errorf("invalid memory limits %+v", t.Limits)
	}
	return newMemory(t, s.memoryPageCap())
}

// NewTable creates a table of t.Limits.Min null references.
func (s *Store) NewTable(t types.TableType) (*Table, error) {
	if t.Limits.HasMax && t.Limits.Max < t.Limits.Min {
		return nil, fmt.Errorf("invalid table limits %+v", t.Limits)
	}
	return newTable(t)
}

// NewGlobal creates a global holding v.
func (s *Store) NewGlobal(t types.GlobalType, v uint64) *Global {
	return &Global{typ: t, val: normalize(t.ValType, v)}
}
