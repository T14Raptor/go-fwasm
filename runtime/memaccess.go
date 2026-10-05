//go:build amd64 || arm64 || 386 || ppc64le

package runtime

import "unsafe"

// Memory accesses at ea, which the interpreter has bounds checked. These
// architectures are little-endian and allow unaligned access, so values
// are read and written in place. The helpers are small enough that the
// compiler inlines them even into the interpreter loop.

func at(m []byte, ea uint64) unsafe.Pointer {
	return unsafe.Add(unsafe.Pointer(unsafe.SliceData(m)), ea)
}

func load16(m []byte, ea uint64) uint16 { return *(*uint16)(at(m, ea)) }
func load32(m []byte, ea uint64) uint32 { return *(*uint32)(at(m, ea)) }
func load64(m []byte, ea uint64) uint64 { return *(*uint64)(at(m, ea)) }

func store16(m []byte, ea uint64, v uint16) { *(*uint16)(at(m, ea)) = v }
func store32(m []byte, ea uint64, v uint32) { *(*uint32)(at(m, ea)) = v }
func store64(m []byte, ea uint64, v uint64) { *(*uint64)(at(m, ea)) = v }
