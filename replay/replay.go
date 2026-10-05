// Package replay checks the runtime against a recording of the same module
// running in a JS engine. The recording is every crossing of the JS/WASM
// boundary: export calls with their arguments and the memory JS wrote
// beforehand, import calls with their arguments, and what each returned.
// Replaying feeds the runtime the same inputs and checks that it makes the
// same import calls with the same arguments, returns the same results, and
// leaves memory with the same SHA-256 at every crossing. The first mismatch
// is the first point where execution diverged.
//
// A recording is a JSON object {"events": [...]}. Each event has a kind k:
//
//	call    JS calls export f with args, after writing mem into memory
//	done    the export returns r or throws, leaving memory with hash
//	import  wasm calls import f ("module.name") with args, memory has hash
//	ret     the import returns r or throws, after JS wrote mem
//
// Between an import and its ret come the export calls JS made from inside
// the import. Values are encoded as in Value, mem is a list of MemSpan, and
// hash is the hex SHA-256 of the whole memory. A recording of several calls
// in a row lists the event index where each starts in marks, and input is
// the last call's input.
package replay

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"

	"github.com/t14raptor/go-fwasm/runtime"
	"github.com/t14raptor/go-fwasm/types"
)

// Log is a recording.
type Log struct {
	Events []Event `json:"events"`
	Marks  []int   `json:"marks"`
	Input  []byte  `json:"input"`
}

// Event is one boundary crossing.
type Event struct {
	K     string    `json:"k"` // call, done, import, ret
	F     string    `json:"f"`
	Args  []Value   `json:"args"`
	R     []Value   `json:"r"`
	Throw *string   `json:"throw"`
	Mem   []MemSpan `json:"mem"`
	Hash  string    `json:"hash"`
}

// Value is a JS value after ToNumber: N holds float64 bits in hex, or B a
// BigInt in decimal.
type Value struct {
	N string `json:"n"`
	B string `json:"b"`
}

// MemSpan is bytes JS wrote at an offset, as [offset, base64].
type MemSpan struct {
	Offset uint64
	Data   []byte
}

func (s *MemSpan) UnmarshalJSON(b []byte) error {
	var raw [2]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if err := json.Unmarshal(raw[0], &s.Offset); err != nil {
		return err
	}
	var data string
	if err := json.Unmarshal(raw[1], &data); err != nil {
		return err
	}
	var err error
	s.Data, err = base64.StdEncoding.DecodeString(data)
	return err
}

// Load reads a recording.
func Load(path string) (*Log, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var l Log
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &l, nil
}

// Stats summarizes a replay.
type Stats struct {
	Calls, Imports, Hashes int
}

// JSError stands in for an exception thrown by a recorded import.
type JSError struct{ Message string }

func (e *JSError) Error() string { return "JS exception: " + e.Message }

// Mismatch reports where the runtime diverged from the recording.
type Mismatch struct {
	Event int
	Msg   string
}

func (m *Mismatch) Error() string { return fmt.Sprintf("event %d: %s", m.Event, m.Msg) }

// Options tune a replay.
type Options struct {
	// Memory names the exported memory. Default "memory".
	Memory string
	// HashEvery checks the memory hash at every Nth export return. Import
	// calls and the last return are always checked. Default 1.
	HashEvery int
	// FinalHashOnly checks memory only at the last return, so hashing
	// doesn't swamp a benchmark.
	FinalHashOnly bool
}

// Replayer replays a recording one top-level export call at a time.
type Replayer struct {
	opts  Options
	log   *Log
	pos   int
	dones int
	inst  *runtime.Instance
	mem   *runtime.Memory
	stats Stats
}

// Run instantiates m with imports that replay log, then makes every
// top-level export call in the log.
func Run(m *runtime.Module, log *Log, opts Options) (Stats, error) {
	r, err := New(m, log, opts)
	if err != nil {
		return Stats{}, err
	}
	err = r.RunTo(len(log.Events))
	return r.stats, err
}

// New instantiates m with imports that replay log.
func New(m *runtime.Module, log *Log, opts Options) (*Replayer, error) {
	if opts.Memory == "" {
		opts.Memory = "memory"
	}
	opts.HashEvery = max(opts.HashEvery, 1)
	r := &Replayer{opts: opts, log: log}
	store := runtime.NewStore()
	imports := runtime.Imports{}
	for _, imp := range m.Imports() {
		if imp.Kind != types.ImportFunc {
			return nil, fmt.Errorf("import %s.%s: only function imports can be replayed", imp.Module, imp.Name)
		}
		if imports[imp.Module] == nil {
			imports[imp.Module] = map[string]runtime.Extern{}
		}
		name := imp.Module + "." + imp.Name
		ft := imp.Func
		imports[imp.Module][imp.Name] = store.NewHostFunc(name, *ft, func(_ *runtime.Caller, args []uint64) ([]uint64, error) {
			return r.importCall(name, ft, args)
		})
	}
	inst, err := store.Instantiate(m, imports)
	if err != nil {
		return nil, err
	}
	r.inst = inst
	r.mem = inst.Memory(opts.Memory)
	if r.mem == nil {
		return nil, fmt.Errorf("no exported memory %q", opts.Memory)
	}
	return r, nil
}

// RunTo makes top-level export calls until the replay reaches event pos,
// which should be the start of a call (see Log.Marks) or the end.
func (r *Replayer) RunTo(pos int) error {
	for r.pos < min(pos, len(r.log.Events)) {
		if err := r.exportCall(); err != nil {
			return err
		}
	}
	return nil
}

// Stats returns what has been replayed so far.
func (r *Replayer) Stats() Stats { return r.stats }

func (r *Replayer) next(kind string) (int, *Event, error) {
	if r.pos >= len(r.log.Events) {
		return r.pos, nil, &Mismatch{r.pos, fmt.Sprintf("log ended, runtime wants a %q event", kind)}
	}
	i := r.pos
	e := &r.log.Events[i]
	r.pos++
	if e.K != kind {
		return i, nil, &Mismatch{i, fmt.Sprintf("runtime is at a %q event, log has %q %s", kind, e.K, e.F)}
	}
	return i, e, nil
}

func (r *Replayer) applyMem(e *Event) {
	buf := r.mem.Bytes()
	for _, s := range e.Mem {
		copy(buf[s.Offset:], s.Data)
	}
}

func (r *Replayer) checkHash(i int, e *Event, where string) error {
	if e.Hash == "" || r.opts.FinalHashOnly && r.pos < len(r.log.Events) {
		return nil
	}
	sum := sha256.Sum256(r.mem.Bytes())
	r.stats.Hashes++
	if got := hex.EncodeToString(sum[:]); got != e.Hash {
		return &Mismatch{i, fmt.Sprintf("%s: memory differs (sha256 %s…, recorded %s…)", where, got[:12], e.Hash[:12])}
	}
	return nil
}

// exportCall replays one export call and everything nested in it.
func (r *Replayer) exportCall() error {
	i, e, err := r.next("call")
	if err != nil {
		return err
	}
	r.stats.Calls++
	fn := r.inst.Func(e.F)
	if fn == nil {
		return &Mismatch{i, fmt.Sprintf("no export %q", e.F)}
	}
	ft := fn.Type()
	if len(e.Args) != len(ft.Params) {
		return &Mismatch{i, fmt.Sprintf("%s: %d recorded args for %d params", e.F, len(e.Args), len(ft.Params))}
	}
	args := make([]uint64, len(e.Args))
	for j, v := range e.Args {
		if args[j], err = fromJS(v, ft.Params[j]); err != nil {
			return &Mismatch{i, err.Error()}
		}
	}
	r.applyMem(e)
	res, callErr := fn.Call(args...)

	var mm *Mismatch
	if errors.As(callErr, &mm) {
		return callErr
	}
	j, done, err := r.next("done")
	if err != nil {
		return err
	}
	where := fmt.Sprintf("%s returned", e.F)
	switch {
	case done.Throw != nil && callErr == nil:
		return &Mismatch{j, fmt.Sprintf("%s: recording threw %q, runtime returned %v", e.F, *done.Throw, res)}
	case done.Throw == nil && callErr != nil:
		return &Mismatch{j, fmt.Sprintf("%s: runtime failed: %v", e.F, callErr)}
	case callErr == nil:
		if err := matchResults(done.R, res, ft.Results); err != nil {
			return &Mismatch{j, fmt.Sprintf("%s: %v", where, err)}
		}
	}
	if r.dones++; r.dones%r.opts.HashEvery != 0 && r.pos < len(r.log.Events) {
		return nil
	}
	return r.checkHash(j, done, where)
}

// importCall replays an import: it checks the call against the log, runs any
// export calls JS made from inside it, and returns what JS returned.
func (r *Replayer) importCall(name string, ft *types.FuncType, args []uint64) ([]uint64, error) {
	i, e, err := r.next("import")
	if err != nil {
		return nil, err
	}
	r.stats.Imports++
	if e.F != name {
		return nil, &Mismatch{i, fmt.Sprintf("runtime called %s, recording called %s", name, e.F)}
	}
	if err := matchResults(e.Args, args, ft.Params); err != nil {
		return nil, &Mismatch{i, fmt.Sprintf("%s arguments: %v", name, err)}
	}
	if err := r.checkHash(i, e, "calling "+name); err != nil {
		return nil, err
	}
	for r.pos < len(r.log.Events) && r.log.Events[r.pos].K == "call" {
		if err := r.exportCall(); err != nil {
			return nil, err
		}
	}
	j, ret, err := r.next("ret")
	if err != nil {
		return nil, err
	}
	r.applyMem(ret)
	if ret.Throw != nil {
		return nil, &JSError{*ret.Throw}
	}
	if len(ft.Results) == 0 {
		return nil, nil
	}
	if len(ret.R) != len(ft.Results) {
		// JS returned undefined for a non-void import: ToNumber gives NaN.
		if len(ret.R) == 0 && len(ft.Results) == 1 && ft.Results[0] != types.I64 {
			return []uint64{fromNumber(math.NaN(), ft.Results[0])}, nil
		}
		return nil, &Mismatch{j, fmt.Sprintf("%s returned %d values, import has %d results", name, len(ret.R), len(ft.Results))}
	}
	out := make([]uint64, len(ft.Results))
	for k, v := range ret.R {
		if out[k], err = fromJS(v, ft.Results[k]); err != nil {
			return nil, &Mismatch{j, err.Error()}
		}
	}
	return out, nil
}

// fromJS applies the JS API's ToWebAssemblyValue.
func fromJS(v Value, t types.ValueType) (uint64, error) {
	if v.B != "" {
		n, ok := new(big.Int).SetString(v.B, 10)
		if !ok {
			return 0, fmt.Errorf("bad BigInt %q", v.B)
		}
		if t != types.I64 {
			return 0, fmt.Errorf("BigInt for %s", t)
		}
		// ToBigInt64: wrap modulo 2^64.
		return new(big.Int).And(n, new(big.Int).SetUint64(math.MaxUint64)).Uint64(), nil
	}
	bits, err := strconv.ParseUint(v.N, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("bad number %q", v.N)
	}
	if t == types.I64 {
		return 0, errors.New("number for i64 (JS would throw a TypeError)")
	}
	return fromNumber(math.Float64frombits(bits), t), nil
}

func fromNumber(f float64, t types.ValueType) uint64 {
	switch t {
	case types.I32:
		return uint64(toUint32(f))
	case types.F32:
		return uint64(math.Float32bits(float32(f)))
	case types.F64:
		return math.Float64bits(f)
	}
	return 0
}

// toUint32 is ECMAScript ToInt32, as unsigned bits.
func toUint32(f float64) uint32 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	m := math.Mod(math.Trunc(f), 4294967296)
	if m < 0 {
		m += 4294967296
	}
	return uint32(m)
}

// toJS applies ToJSValue, encoded like the log.
func toJS(v uint64, t types.ValueType) Value {
	var f float64
	switch t {
	case types.I64:
		return Value{B: strconv.FormatInt(int64(v), 10)}
	case types.I32:
		f = float64(int32(v))
	case types.F32:
		f = float64(math.Float32frombits(uint32(v)))
	case types.F64:
		f = math.Float64frombits(v)
	}
	return Value{N: strconv.FormatUint(math.Float64bits(f), 16)}
}

func matchResults(want []Value, got []uint64, ts []types.ValueType) error {
	if len(want) != len(got) {
		return fmt.Errorf("%d values, recording had %d", len(got), len(want))
	}
	for k := range got {
		g := toJS(got[k], ts[k])
		if g == want[k] {
			continue
		}
		// NaN payloads may differ once they reach JS.
		gb, err1 := strconv.ParseUint(g.N, 16, 64)
		wb, err2 := strconv.ParseUint(want[k].N, 16, 64)
		if err1 == nil && err2 == nil && math.IsNaN(math.Float64frombits(gb)) && math.IsNaN(math.Float64frombits(wb)) {
			continue
		}
		return fmt.Errorf("value %d is %+v, recording had %+v", k, g, want[k])
	}
	return nil
}
