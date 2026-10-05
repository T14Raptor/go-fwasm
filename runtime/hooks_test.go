package runtime

import "testing"

func TestHooks(t *testing.T) {
	f := newFixture(t)
	f.fact = func(n uint64) ([]uint64, error) { return f.inst.Func("fact").Call(n) }
	var enters, exits, hostCalls, hostReturns int
	var writes []uint64
	f.store.SetHooks(&Hooks{
		Enter:      func(*Function, []uint64) { enters++ },
		Exit:       func(*Function, []uint64) { exits++ },
		HostCall:   func(*Function, []uint64) { hostCalls++ },
		HostReturn: func(*Function, []uint64, error) { hostReturns++ },
		Watch:      []AddrRange{{Start: 200, End: 204}},
		MemWrite:   func(_ *Function, addr, _ uint64) { writes = append(writes, addr) },
	})
	f.call(t, "fact", 3)
	if enters != 4 || exits != 4 || hostCalls != 3 || hostReturns != 3 {
		t.Fatalf("enters=%d exits=%d hostCalls=%d hostReturns=%d", enters, exits, hostCalls, hostReturns)
	}
	f.call(t, "store", 196, 1) // ends at 200: outside
	f.call(t, "store", 198, 1) // overlaps
	f.call(t, "store", 204, 1) // outside
	if len(writes) != 1 || writes[0] != 198 {
		t.Fatalf("watched writes %v", writes)
	}
}
