package runtime

import (
	"math"

	"github.com/t14raptor/go-fwasm/types"
)

// Values cross the API as raw 64-bit slots:
//
//	i32      zero-extended two's complement
//	i64      two's complement
//	f32      IEEE bits, zero-extended
//	f64      IEEE bits
//	funcref  Function.Ref, or 0 for null
//	externref  any host-chosen value, 0 for null
//
// Values passed into the runtime are normalized, so a sign-extended i32 is
// accepted.

func I32(v int32) uint64   { return uint64(uint32(v)) }
func I64(v int64) uint64   { return uint64(v) }
func F32(v float32) uint64 { return uint64(math.Float32bits(v)) }
func F64(v float64) uint64 { return math.Float64bits(v) }

func AsI32(v uint64) int32   { return int32(v) }
func AsI64(v uint64) int64   { return int64(v) }
func AsF32(v uint64) float32 { return math.Float32frombits(uint32(v)) }
func AsF64(v uint64) float64 { return math.Float64frombits(v) }

// normalize clears the upper half of 32-bit values.
func normalize(t types.ValueType, v uint64) uint64 {
	if t == types.I32 || t == types.F32 {
		return uint64(uint32(v))
	}
	return v
}
