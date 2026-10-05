package runtime

import "testing"

func TestMemoryGrow(t *testing.T) {
	f := newFixture(t)
	mem := f.inst.Memory("mem")
	before := mem.Bytes()
	var grows [][2]uint32
	mem.OnGrow(func(old, new uint32) { grows = append(grows, [2]uint32{old, new}) })

	f.call(t, "store", 100, 0xdeadbeef)
	if got := f.call(t, "grow", 2)[0]; got != 1 {
		t.Fatalf("grow returned %d, want 1", got)
	}
	if got := f.call(t, "grow", 0)[0]; got != 3 {
		t.Fatalf("grow(0) returned %d, want 3", got)
	}
	if got := AsI32(f.call(t, "grow", 2)[0]); got != -1 {
		t.Fatalf("grow past max returned %d, want -1", got)
	}
	if len(grows) != 2 || grows[0] != [2]uint32{1, 3} || grows[1] != [2]uint32{3, 3} {
		t.Fatalf("grow callbacks %v", grows)
	}
	if len(mem.Bytes()) != 3*PageSize || len(before) != PageSize {
		t.Fatalf("sizes: now %d, before %d", len(mem.Bytes()), len(before))
	}
	if got := f.call(t, "load", 100)[0]; got != 0xdeadbeef {
		t.Fatalf("contents lost across grow: %#x", got)
	}
}
