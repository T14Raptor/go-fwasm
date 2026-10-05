package runtime

import (
	"fmt"

	"github.com/t14raptor/go-fwasm/types"
)

// PageSize is the size of a wasm memory page.
const PageSize = 65536

// Memory is a linear memory.
type Memory struct {
	typ    types.MemoryType
	buf    []byte
	max    uint32 // pages
	onGrow []func(oldPages, newPages uint32)
}

func newMemory(t types.MemoryType, pageCap uint32) (*Memory, error) {
	max := pageCap
	if t.Limits.HasMax {
		max = min(max, t.Limits.Max)
	}
	if t.Limits.Min > max {
		return nil, fmt.Errorf("memory minimum %d pages exceeds limit %d", t.Limits.Min, max)
	}
	return &Memory{typ: t, buf: make([]byte, int(t.Limits.Min)*PageSize), max: max}, nil
}

// Type returns the memory's type with its current size as the minimum.
func (m *Memory) Type() types.MemoryType {
	mt := m.typ
	mt.Limits.Min = m.Pages()
	return mt
}

// Bytes returns the memory's contents. The slice aliases the memory until it
// grows; after that it is stale and Bytes must be called again (see OnGrow).
func (m *Memory) Bytes() []byte { return m.buf }

// Pages returns the current size in pages.
func (m *Memory) Pages() uint32 { return uint32(len(m.buf) / PageSize) }

// OnGrow registers fn to run after every successful grow, including
// memory.grow from wasm and grows by zero pages.
func (m *Memory) OnGrow(fn func(oldPages, newPages uint32)) {
	m.onGrow = append(m.onGrow, fn)
}

// Grow adds delta pages and returns the previous size in pages. It fails,
// leaving the memory unchanged, past the memory's maximum.
func (m *Memory) Grow(delta uint32) (uint32, bool) {
	old := m.Pages()
	if uint64(old)+uint64(delta) > uint64(m.max) {
		return 0, false
	}
	if delta > 0 {
		n := int(old+delta) * PageSize
		if n <= cap(m.buf) {
			// Bytes past len were never handed out, so they are still zero.
			m.buf = m.buf[:n]
		} else {
			buf := make([]byte, n, min(max(n, 2*cap(m.buf)), int(m.max)*PageSize))
			copy(buf, m.buf)
			m.buf = buf
		}
	}
	for _, fn := range m.onGrow {
		fn(old, old+delta)
	}
	return old, true
}
