package runtime

import (
	"errors"
	"testing"
	"time"

	"github.com/t14raptor/go-fwasm/types"
)

// reentryWasm is compiled from:
//
//	;; Host round trips: fact recurses through the host, outer/catch exercise
//	;; host errors, grow/store/spin/runaway exercise memory, hooks and limits.
//	(module
//	  (import "host" "fact" (func $host_fact (param i32) (result i32)))
//	  (import "host" "fail" (func $fail))
//	  (import "host" "catch" (func $catch (result i32)))
//	  (memory (export "mem") 1 4)
//
//	  ;; fact(n) = n * host.fact(n-1); the host calls back into fact.
//	  (func (export "fact") (param $n i32) (result i32)
//	    (if (result i32) (i32.eqz (local.get $n))
//	      (then (i32.const 1))
//	      (else (i32.mul (local.get $n) (call $host_fact (i32.sub (local.get $n) (i32.const 1)))))))
//
//	  ;; Fails through the host after some wasm frames.
//	  (func $deep (param i32)
//	    (if (local.get 0)
//	      (then (call $deep (i32.sub (local.get 0) (i32.const 1))))
//	      (else (call $fail))))
//	  (func (export "outer") (call $deep (i32.const 5)))
//
//	  ;; host.catch calls outer, swallows its error and returns 7.
//	  (func (export "catcher") (result i32) (i32.add (call $catch) (i32.const 1)))
//
//	  (func (export "grow") (param i32) (result i32) (memory.grow (local.get 0)))
//	  (func (export "store") (param i32 i32) (i32.store (local.get 0) (local.get 1)))
//	  (func (export "load") (param i32) (result i32) (i32.load (local.get 0)))
//	  (func (export "spin") (loop $l (br $l)))
//	  (func $runaway (export "runaway") (call $runaway))
//	  (func (export "div") (param i32 i32) (result i32) (i32.div_s (local.get 0) (local.get 1)))
//	)
var reentryWasm = []byte{
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x1c, 0x06, 0x60, 0x01, 0x7f, 0x01, 0x7f,
	0x60, 0x00, 0x00, 0x60, 0x00, 0x01, 0x7f, 0x60, 0x01, 0x7f, 0x00, 0x60, 0x02, 0x7f, 0x7f, 0x00,
	0x60, 0x02, 0x7f, 0x7f, 0x01, 0x7f, 0x02, 0x26, 0x03, 0x04, 0x68, 0x6f, 0x73, 0x74, 0x04, 0x66,
	0x61, 0x63, 0x74, 0x00, 0x00, 0x04, 0x68, 0x6f, 0x73, 0x74, 0x04, 0x66, 0x61, 0x69, 0x6c, 0x00,
	0x01, 0x04, 0x68, 0x6f, 0x73, 0x74, 0x05, 0x63, 0x61, 0x74, 0x63, 0x68, 0x00, 0x02, 0x03, 0x0b,
	0x0a, 0x00, 0x03, 0x01, 0x02, 0x00, 0x04, 0x00, 0x01, 0x01, 0x05, 0x05, 0x04, 0x01, 0x01, 0x01,
	0x04, 0x07, 0x4d, 0x0a, 0x03, 0x6d, 0x65, 0x6d, 0x02, 0x00, 0x04, 0x66, 0x61, 0x63, 0x74, 0x00,
	0x03, 0x05, 0x6f, 0x75, 0x74, 0x65, 0x72, 0x00, 0x05, 0x07, 0x63, 0x61, 0x74, 0x63, 0x68, 0x65,
	0x72, 0x00, 0x06, 0x04, 0x67, 0x72, 0x6f, 0x77, 0x00, 0x07, 0x05, 0x73, 0x74, 0x6f, 0x72, 0x65,
	0x00, 0x08, 0x04, 0x6c, 0x6f, 0x61, 0x64, 0x00, 0x09, 0x04, 0x73, 0x70, 0x69, 0x6e, 0x00, 0x0a,
	0x07, 0x72, 0x75, 0x6e, 0x61, 0x77, 0x61, 0x79, 0x00, 0x0b, 0x03, 0x64, 0x69, 0x76, 0x00, 0x0c,
	0x0a, 0x66, 0x0a, 0x15, 0x00, 0x20, 0x00, 0x45, 0x04, 0x7f, 0x41, 0x01, 0x05, 0x20, 0x00, 0x20,
	0x00, 0x41, 0x01, 0x6b, 0x10, 0x00, 0x6c, 0x0b, 0x0b, 0x11, 0x00, 0x20, 0x00, 0x04, 0x40, 0x20,
	0x00, 0x41, 0x01, 0x6b, 0x10, 0x04, 0x05, 0x10, 0x01, 0x0b, 0x0b, 0x06, 0x00, 0x41, 0x05, 0x10,
	0x04, 0x0b, 0x07, 0x00, 0x10, 0x02, 0x41, 0x01, 0x6a, 0x0b, 0x06, 0x00, 0x20, 0x00, 0x40, 0x00,
	0x0b, 0x09, 0x00, 0x20, 0x00, 0x20, 0x01, 0x36, 0x02, 0x00, 0x0b, 0x07, 0x00, 0x20, 0x00, 0x28,
	0x02, 0x00, 0x0b, 0x07, 0x00, 0x03, 0x40, 0x0c, 0x00, 0x0b, 0x0b, 0x04, 0x00, 0x10, 0x0b, 0x0b,
	0x07, 0x00, 0x20, 0x00, 0x20, 0x01, 0x6d, 0x0b,
}

// benchWasm is compiled from:
//
//	;; Benchmark kernels.
//	(module
//	  (func $fib (export "fib") (param i32) (result i32)
//	    (if (result i32) (i32.lt_u (local.get 0) (i32.const 2))
//	      (then (local.get 0))
//	      (else (i32.add (call $fib (i32.sub (local.get 0) (i32.const 1)))
//	                     (call $fib (i32.sub (local.get 0) (i32.const 2)))))))
//	  ;; xorshift mixing loop, typical of hash and cipher code.
//	  (func (export "mix") (param $n i32) (result i32) (local $x i32)
//	    (local.set $x (i32.const 0x9e3779b9))
//	    (block $done
//	      (loop $l
//	        (br_if $done (i32.eqz (local.get $n)))
//	        (local.set $x (i32.xor (local.get $x) (i32.shl (local.get $x) (i32.const 13))))
//	        (local.set $x (i32.xor (local.get $x) (i32.shr_u (local.get $x) (i32.const 17))))
//	        (local.set $x (i32.xor (local.get $x) (i32.shl (local.get $x) (i32.const 5))))
//	        (local.set $n (i32.sub (local.get $n) (i32.const 1)))
//	        (br $l)))
//	    (local.get $x))
//	)
var benchWasm = []byte{
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x06, 0x01, 0x60, 0x01, 0x7f, 0x01, 0x7f,
	0x03, 0x03, 0x02, 0x00, 0x00, 0x07, 0x0d, 0x02, 0x03, 0x66, 0x69, 0x62, 0x00, 0x00, 0x03, 0x6d,
	0x69, 0x78, 0x00, 0x01, 0x0a, 0x5f, 0x02, 0x1c, 0x00, 0x20, 0x00, 0x41, 0x02, 0x49, 0x04, 0x7f,
	0x20, 0x00, 0x05, 0x20, 0x00, 0x41, 0x01, 0x6b, 0x10, 0x00, 0x20, 0x00, 0x41, 0x02, 0x6b, 0x10,
	0x00, 0x6a, 0x0b, 0x0b, 0x40, 0x01, 0x01, 0x7f, 0x41, 0xb9, 0xf3, 0xdd, 0xf1, 0x79, 0x21, 0x01,
	0x02, 0x40, 0x03, 0x40, 0x20, 0x00, 0x45, 0x0d, 0x01, 0x20, 0x01, 0x20, 0x01, 0x41, 0x0d, 0x74,
	0x73, 0x21, 0x01, 0x20, 0x01, 0x20, 0x01, 0x41, 0x11, 0x76, 0x73, 0x21, 0x01, 0x20, 0x01, 0x20,
	0x01, 0x41, 0x05, 0x74, 0x73, 0x21, 0x01, 0x20, 0x00, 0x41, 0x01, 0x6b, 0x21, 0x00, 0x0c, 0x00,
	0x0b, 0x0b, 0x20, 0x01, 0x0b,
}

type fixture struct {
	store *Store
	inst  *Instance
	// Host behavior, set per test.
	fact  func(n uint64) ([]uint64, error)
	fail  error
	catch func() ([]uint64, error)
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	m, err := Compile(reentryWasm)
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{store: NewStore()}
	i32 := types.I32
	host := map[string]Extern{
		"fact": f.store.NewHostFunc("host.fact", types.FuncType{Params: []types.ValueType{i32}, Results: []types.ValueType{i32}},
			func(_ *Caller, args []uint64) ([]uint64, error) { return f.fact(args[0]) }),
		"fail": f.store.NewHostFunc("host.fail", types.FuncType{},
			func(*Caller, []uint64) ([]uint64, error) { return nil, f.fail }),
		"catch": f.store.NewHostFunc("host.catch", types.FuncType{Results: []types.ValueType{i32}},
			func(*Caller, []uint64) ([]uint64, error) { return f.catch() }),
	}
	f.inst, err = f.store.Instantiate(m, Imports{"host": host})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *fixture) call(t *testing.T, name string, args ...uint64) []uint64 {
	t.Helper()
	res, err := f.inst.Func(name).Call(args...)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return res
}

func TestReentrantCalls(t *testing.T) {
	f := newFixture(t)
	depth, maxDepth := 0, 0
	f.fact = func(n uint64) ([]uint64, error) {
		depth++
		maxDepth = max(maxDepth, depth)
		defer func() { depth-- }()
		return f.inst.Func("fact").Call(n)
	}
	if got := f.call(t, "fact", 10)[0]; got != 3628800 {
		t.Fatalf("fact(10) = %d", got)
	}
	if maxDepth != 10 {
		t.Fatalf("host nesting %d, want 10", maxDepth)
	}
	if f.store.sp != 0 || len(f.store.frames) != 0 {
		t.Fatalf("stack not unwound: sp=%d frames=%d", f.store.sp, len(f.store.frames))
	}
}

func TestReentrantDepthLimit(t *testing.T) {
	f := newFixture(t)
	f.store.MaxCallDepth = 100
	f.fact = func(n uint64) ([]uint64, error) { return f.inst.Func("fact").Call(n) }
	_, err := f.inst.Func("fact").Call(1000)
	var trap *Trap
	if !errors.As(err, &trap) || trap.Code != TrapCallStackExhausted {
		t.Fatalf("got %v, want call stack exhausted", err)
	}
	if f.store.sp != 0 || len(f.store.frames) != 0 {
		t.Fatalf("stack not unwound: sp=%d frames=%d", f.store.sp, len(f.store.frames))
	}
}

func TestHostErrorPropagates(t *testing.T) {
	f := newFixture(t)
	sentinel := errors.New("thrown from JS")
	f.fail = sentinel
	_, err := f.inst.Func("outer").Call()
	if err != sentinel {
		t.Fatalf("got %v, want the host's error value itself", err)
	}
	if f.store.sp != 0 || len(f.store.frames) != 0 {
		t.Fatalf("stack not unwound: sp=%d frames=%d", f.store.sp, len(f.store.frames))
	}
}

func TestHostCatchesNestedError(t *testing.T) {
	f := newFixture(t)
	f.fail = errors.New("inner")
	var spInside int
	f.catch = func() ([]uint64, error) {
		spInside = f.store.sp
		if _, err := f.inst.Func("outer").Call(); err == nil {
			t.Error("outer did not fail")
		}
		if f.store.sp != spInside {
			t.Errorf("nested failure left sp=%d, want %d", f.store.sp, spInside)
		}
		return []uint64{7}, nil
	}
	if got := f.call(t, "catcher")[0]; got != 8 {
		t.Fatalf("catcher() = %d, want 8", got)
	}
	// The store still works afterwards.
	f.fact = func(n uint64) ([]uint64, error) { return f.inst.Func("fact").Call(n) }
	if got := f.call(t, "fact", 5)[0]; got != 120 {
		t.Fatalf("fact(5) = %d", got)
	}
}

func TestInterrupt(t *testing.T) {
	f := newFixture(t)
	go func() {
		time.Sleep(20 * time.Millisecond)
		f.store.Interrupt()
	}()
	_, err := f.inst.Func("spin").Call()
	var trap *Trap
	if !errors.As(err, &trap) || trap.Code != TrapInterrupted {
		t.Fatalf("got %v, want interrupted", err)
	}
	f.store.ClearInterrupt()
	if got := f.call(t, "load", 0); len(got) != 1 {
		t.Fatal("store unusable after interrupt")
	}
}

func TestTraps(t *testing.T) {
	f := newFixture(t)
	_, err := f.inst.Func("runaway").Call()
	var trap *Trap
	if !errors.As(err, &trap) || trap.Code != TrapCallStackExhausted {
		t.Fatalf("got %v, want call stack exhausted", err)
	}
	if len(trap.Stack) != DefaultMaxCallDepth+1 {
		t.Fatalf("trap stack has %d frames", len(trap.Stack))
	}

	_, err = f.inst.Func("div").Call(I32(1), I32(0))
	if !errors.As(err, &trap) || trap.Code != TrapIntegerDivideByZero {
		t.Fatalf("got %v, want divide by zero", err)
	}
	if fn := trap.Stack[0].Func; fn.Name() != "div" {
		t.Fatalf("trap in %s, want div", fn)
	}
	if got := AsI32(f.call(t, "div", I32(-7), I32(2))[0]); got != -3 {
		t.Fatalf("div(-7, 2) = %d", got)
	}
}

func benchInstance(b *testing.B) *Instance {
	m, err := Compile(benchWasm)
	if err != nil {
		b.Fatal(err)
	}
	inst, err := NewStore().Instantiate(m, nil)
	if err != nil {
		b.Fatal(err)
	}
	return inst
}

// BenchmarkFib is call-heavy: fib(25) makes 242785 calls.
func BenchmarkFib(b *testing.B) {
	fib := benchInstance(b).Func("fib")
	for b.Loop() {
		if res, err := fib.Call(25); err != nil || res[0] != 75025 {
			b.Fatal(res, err)
		}
	}
}

// BenchmarkMix runs 1M iterations of a 22-instruction loop body.
func BenchmarkMix(b *testing.B) {
	mix := benchInstance(b).Func("mix")
	for b.Loop() {
		if _, err := mix.Call(1_000_000); err != nil {
			b.Fatal(err)
		}
	}
}
