//go:build !(amd64 || arm64 || 386 || ppc64le)

package runtime

import "encoding/binary"

// Memory accesses at ea, which the interpreter has bounds checked.

func load16(m []byte, ea uint64) uint16 { return binary.LittleEndian.Uint16(m[ea:]) }
func load32(m []byte, ea uint64) uint32 { return binary.LittleEndian.Uint32(m[ea:]) }
func load64(m []byte, ea uint64) uint64 { return binary.LittleEndian.Uint64(m[ea:]) }

func store16(m []byte, ea uint64, v uint16) { binary.LittleEndian.PutUint16(m[ea:], v) }
func store32(m []byte, ea uint64, v uint32) { binary.LittleEndian.PutUint32(m[ea:], v) }
func store64(m []byte, ea uint64, v uint64) { binary.LittleEndian.PutUint64(m[ea:], v) }
