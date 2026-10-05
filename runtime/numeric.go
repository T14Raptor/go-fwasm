package runtime

import "math"

// Float min/max: NaN wins, then -0 < +0. A NaN result is the first NaN
// operand made quiet, so canonical NaNs stay canonical.

func fmin32(a, b float32) float32 {
	switch {
	case a != a || b != b:
		return nan32(a, b)
	case a == 0 && b == 0:
		if math.Float32bits(a)>>31 != 0 {
			return a
		}
		return b
	case a < b:
		return a
	}
	return b
}

func fmax32(a, b float32) float32 {
	switch {
	case a != a || b != b:
		return nan32(a, b)
	case a == 0 && b == 0:
		if math.Float32bits(a)>>31 == 0 {
			return a
		}
		return b
	case a > b:
		return a
	}
	return b
}

func nan32(a, b float32) float32 {
	x := a
	if a == a {
		x = b
	}
	return math.Float32frombits(math.Float32bits(x) | 1<<22)
}

func fmin64(a, b float64) float64 {
	switch {
	case a != a || b != b:
		return nan64(a, b)
	case a == 0 && b == 0:
		if math.Signbit(a) {
			return a
		}
		return b
	case a < b:
		return a
	}
	return b
}

func fmax64(a, b float64) float64 {
	switch {
	case a != a || b != b:
		return nan64(a, b)
	case a == 0 && b == 0:
		if !math.Signbit(a) {
			return a
		}
		return b
	case a > b:
		return a
	}
	return b
}

func nan64(a, b float64) float64 {
	x := a
	if a == a {
		x = b
	}
	return math.Float64frombits(math.Float64bits(x) | 1<<51)
}

// Trapping float-to-int truncations. f32 operands are widened first, which
// is exact. Each range check is on the untruncated value, against the first
// value whose truncation is out of range.

func truncS32(x float64) (uint64, TrapCode) {
	if x != x {
		return 0, TrapInvalidConversion
	}
	if x <= -2147483649.0 || x >= 2147483648.0 {
		return 0, TrapIntegerOverflow
	}
	return uint64(uint32(int32(x))), 0
}

func truncU32(x float64) (uint64, TrapCode) {
	if x != x {
		return 0, TrapInvalidConversion
	}
	if x <= -1.0 || x >= 4294967296.0 {
		return 0, TrapIntegerOverflow
	}
	return uint64(uint32(int64(x))), 0
}

func truncS64(x float64) (uint64, TrapCode) {
	if x != x {
		return 0, TrapInvalidConversion
	}
	if x < -9223372036854775808.0 || x >= 9223372036854775808.0 {
		return 0, TrapIntegerOverflow
	}
	return uint64(int64(x)), 0
}

func truncU64(x float64) (uint64, TrapCode) {
	if x != x {
		return 0, TrapInvalidConversion
	}
	if x <= -1.0 || x >= 18446744073709551616.0 {
		return 0, TrapIntegerOverflow
	}
	if x < 1 {
		return 0, 0
	}
	return uint64(x), 0
}

// Saturating truncations: NaN is 0, out-of-range clamps.

func satS32(x float64) uint64 {
	switch {
	case x != x:
		return 0
	case x <= math.MinInt32:
		return 1 << 31
	case x >= math.MaxInt32:
		return math.MaxInt32
	}
	return uint64(uint32(int32(x)))
}

func satU32(x float64) uint64 {
	switch {
	case x != x || x <= 0:
		return 0
	case x >= math.MaxUint32:
		return math.MaxUint32
	}
	return uint64(uint32(int64(x)))
}

func satS64(x float64) uint64 {
	switch {
	case x != x:
		return 0
	case x <= math.MinInt64:
		return 1 << 63
	case x >= 9223372036854775808.0:
		return math.MaxInt64
	}
	return uint64(int64(x))
}

func satU64(x float64) uint64 {
	switch {
	case x != x || x < 1:
		return 0
	case x >= 18446744073709551616.0:
		return math.MaxUint64
	}
	return uint64(x)
}
