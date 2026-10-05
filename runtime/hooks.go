package runtime

// Hooks observe execution, for tracing and analysis. Nil fields cost
// nothing, and a nil *Hooks costs one predictable branch per call.
//
// Slices passed to hooks alias the value stack and are only valid during
// the hook. Hooks may read memory but must not call into the store.
type Hooks struct {
	// Enter runs when a wasm function starts, with its arguments.
	Enter func(fn *Function, args []uint64)
	// Exit runs when a wasm function returns normally, with its results.
	// It does not run for frames unwound by a trap or host error.
	Exit func(fn *Function, results []uint64)

	// HostCall and HostReturn wrap every call from wasm to a host function.
	HostCall   func(fn *Function, args []uint64)
	HostReturn func(fn *Function, results []uint64, err error)

	// MemWrite runs after a wasm store, memory.fill, memory.copy or
	// memory.init writes bytes overlapping one of Watch. addr and size
	// describe the whole write.
	Watch    []AddrRange
	MemWrite func(fn *Function, addr uint64, size uint64)
}

// AddrRange is the byte range [Start, End).
type AddrRange struct {
	Start, End uint64
}

func (h *Hooks) watching() bool { return h != nil && h.MemWrite != nil && len(h.Watch) > 0 }

func (h *Hooks) memWrite(fn *Function, addr, size uint64) {
	for _, r := range h.Watch {
		if addr < r.End && addr+size > r.Start {
			h.MemWrite(fn, addr, size)
			return
		}
	}
}
