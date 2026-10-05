// Package spectest runs the official WebAssembly spec tests (converted by
// wast2json, see fetch.sh) against the parser, compiler and runtime.
package spectest

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/t14raptor/go-fwasm/compile"
	"github.com/t14raptor/go-fwasm/runtime"
	"github.com/t14raptor/go-fwasm/types"
)

type script struct {
	SourceFilename string    `json:"source_filename"`
	Commands       []command `json:"commands"`
}

type command struct {
	Type       string  `json:"type"`
	Line       int     `json:"line"`
	Filename   string  `json:"filename"`
	Name       string  `json:"name"`
	As         string  `json:"as"`
	Text       string  `json:"text"`
	ModuleType string  `json:"module_type"`
	Action     *action `json:"action"`
	Expected   []value `json:"expected"`
}

type action struct {
	Type   string  `json:"type"`
	Module string  `json:"module"`
	Field  string  `json:"field"`
	Args   []value `json:"args"`
}

type value struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Counts tallies outcomes per command type, indexed by Pass, Fail and Skip.
type Counts map[string]*[3]int

func (c Counts) add(kind string, outcome int) {
	if c[kind] == nil {
		c[kind] = new([3]int)
	}
	c[kind][outcome]++
}

// Command outcomes.
const (
	Pass = iota
	Fail
	Skip
)

// Failure is a command that did not behave as its script expects.
type Failure struct {
	Line    int
	Command string
	Err     error
}

func (f Failure) Error() string { return fmt.Sprintf("line %d: %s: %v", f.Line, f.Command, f.Err) }

type runner struct {
	dir     string
	store   *runtime.Store
	current *runtime.Instance
	named   map[string]*runtime.Instance
	imports runtime.Imports
}

func newRunner(hooks *runtime.Hooks) *runner {
	r := &runner{store: runtime.NewStore(), named: map[string]*runtime.Instance{}}
	r.store.SetHooks(hooks)
	r.imports = runtime.Imports{"spectest": spectestModule(r.store)}
	return r
}

// spectestModule is the host module the spec tests import from.
func spectestModule(s *runtime.Store) map[string]runtime.Extern {
	print := func(params ...types.ValueType) *runtime.Function {
		return s.NewHostFunc("spectest.print", types.FuncType{Params: params},
			func(*runtime.Caller, []uint64) ([]uint64, error) { return nil, nil })
	}
	table, _ := s.NewTable(types.TableType{ElemType: types.FuncRef, Limits: types.Limits{Min: 10, Max: 20, HasMax: true}})
	memory, _ := s.NewMemory(types.MemoryType{Limits: types.Limits{Min: 1, Max: 2, HasMax: true}})
	return map[string]runtime.Extern{
		"print":         print(),
		"print_i32":     print(types.I32),
		"print_i64":     print(types.I64),
		"print_f32":     print(types.F32),
		"print_f64":     print(types.F64),
		"print_i32_f32": print(types.I32, types.F32),
		"print_f64_f64": print(types.F64, types.F64),
		"global_i32":    s.NewGlobal(types.GlobalType{ValType: types.I32}, 666),
		"global_i64":    s.NewGlobal(types.GlobalType{ValType: types.I64}, 666),
		"global_f32":    s.NewGlobal(types.GlobalType{ValType: types.F32}, runtime.F32(666.6)),
		"global_f64":    s.NewGlobal(types.GlobalType{ValType: types.F64}, runtime.F64(666.6)),
		"table":         table,
		"memory":        memory,
	}
}

// RunScript runs one wast2json script in a fresh store, adding each
// command's outcome to counts. The error is for a script that cannot be
// read; failing commands are returned as Failures.
func RunScript(path string, counts Counts) ([]Failure, error) {
	return runScript(path, counts, nil)
}

// runScript is RunScript with hooks installed in the store.
func runScript(path string, counts Counts, hooks *runtime.Hooks) ([]Failure, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sc script
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	r := newRunner(hooks)
	r.dir = filepath.Dir(path)
	var failures []Failure
	for _, cmd := range sc.Commands {
		outcome, err := r.command(cmd)
		counts.add(cmd.Type, outcome)
		if outcome == Fail {
			failures = append(failures, Failure{cmd.Line, cmd.Type, err})
		}
	}
	return failures, nil
}

func (r *runner) load(file string) (*runtime.Module, error) {
	wasm, err := os.ReadFile(filepath.Join(r.dir, file))
	if err != nil {
		return nil, err
	}
	return runtime.Compile(wasm)
}

func (r *runner) command(cmd command) (int, error) {
	switch cmd.Type {
	case "module":
		m, err := r.load(cmd.Filename)
		if err != nil {
			return Fail, err
		}
		inst, err := r.store.Instantiate(m, r.imports)
		if err != nil {
			return Fail, err
		}
		r.current = inst
		if cmd.Name != "" {
			r.named[cmd.Name] = inst
		}
		return Pass, nil

	case "register":
		inst := r.current
		if cmd.Name != "" {
			inst = r.named[cmd.Name]
		}
		if inst == nil {
			return Fail, errors.New("no module to register")
		}
		r.imports[cmd.As] = inst.Exports()
		return Pass, nil

	case "action", "assert_return":
		got, err := r.act(cmd.Action)
		if err != nil {
			return Fail, err
		}
		if len(got) != len(cmd.Expected) {
			return Fail, fmt.Errorf("got %d results, want %d", len(got), len(cmd.Expected))
		}
		for i, want := range cmd.Expected {
			if err := match(want, got[i]); err != nil {
				return Fail, fmt.Errorf("%s: result %d: %w", cmd.Action.Field, i, err)
			}
		}
		return Pass, nil

	case "assert_trap", "assert_exhaustion":
		_, err := r.act(cmd.Action)
		var trap *runtime.Trap
		if !errors.As(err, &trap) {
			return Fail, fmt.Errorf("%s: want trap %q, got %v", cmd.Action.Field, cmd.Text, err)
		}
		if !strings.Contains(trap.Code.String(), cmd.Text) && !strings.Contains(cmd.Text, trap.Code.String()) {
			return Fail, fmt.Errorf("%s: want trap %q, got %q", cmd.Action.Field, cmd.Text, trap.Code)
		}
		return Pass, nil

	case "assert_invalid", "assert_malformed":
		if cmd.ModuleType == "text" {
			return Skip, nil
		}
		_, err := r.load(cmd.Filename)
		if err == nil {
			return Fail, fmt.Errorf("%s: module accepted, want %q", cmd.Filename, cmd.Text)
		}
		// Invalid modules must get past the parser and fail validation;
		// malformed ones must fail in the parser. The exception: wast2json
		// omits the data count section from modules without data segments,
		// so an invalid data.drop or memory.init there decodes as malformed.
		invalid := errors.Is(err, compile.ErrInvalid)
		dataCount := strings.Contains(err.Error(), "data count section required")
		if invalid != (cmd.Type == "assert_invalid") && !(cmd.Type == "assert_invalid" && dataCount) {
			return Fail, fmt.Errorf("%s: rejected by the wrong stage, want %q: %v", cmd.Filename, cmd.Text, err)
		}
		return Pass, nil

	case "assert_unlinkable":
		m, err := r.load(cmd.Filename)
		if err != nil {
			return Fail, err
		}
		_, err = r.store.Instantiate(m, r.imports)
		var le *runtime.LinkError
		if !errors.As(err, &le) {
			return Fail, fmt.Errorf("want link error %q, got %v", cmd.Text, err)
		}
		return Pass, nil

	case "assert_uninstantiable":
		m, err := r.load(cmd.Filename)
		if err != nil {
			return Fail, err
		}
		_, err = r.store.Instantiate(m, r.imports)
		var trap *runtime.Trap
		if !errors.As(err, &trap) {
			return Fail, fmt.Errorf("want trap %q, got %v", cmd.Text, err)
		}
		return Pass, nil
	}
	return Skip, nil
}

func (r *runner) act(a *action) ([]uint64, error) {
	inst := r.current
	if a.Module != "" {
		inst = r.named[a.Module]
	}
	if inst == nil {
		return nil, errors.New("no module instance")
	}
	switch a.Type {
	case "invoke":
		f := inst.Func(a.Field)
		if f == nil {
			return nil, fmt.Errorf("no exported function %q", a.Field)
		}
		args := make([]uint64, len(a.Args))
		for i, v := range a.Args {
			x, err := parse(v)
			if err != nil {
				return nil, err
			}
			args[i] = x
		}
		return f.Call(args...)
	case "get":
		g := inst.Global(a.Field)
		if g == nil {
			return nil, fmt.Errorf("no exported global %q", a.Field)
		}
		return []uint64{g.Get()}, nil
	}
	return nil, fmt.Errorf("unknown action %q", a.Type)
}

// parse decodes an argument. externref N is passed as N+1, keeping 0 for null.
func parse(v value) (uint64, error) {
	switch v.Type {
	case "funcref", "externref":
		if v.Value == "null" {
			return 0, nil
		}
		n, err := strconv.ParseUint(v.Value, 10, 64)
		return n + 1, err
	case "i32", "i64", "f32", "f64":
		return strconv.ParseUint(v.Value, 10, 64)
	}
	return 0, fmt.Errorf("unsupported value type %q", v.Type)
}

func match(want value, got uint64) error {
	switch want.Type {
	case "f32":
		bits := uint32(got)
		isNaN := bits&0x7f800000 == 0x7f800000 && bits&0x007fffff != 0
		switch want.Value {
		case "nan:canonical":
			if bits&0x7fffffff != 0x7fc00000 {
				return fmt.Errorf("got %#08x, want canonical NaN", bits)
			}
			return nil
		case "nan:arithmetic":
			if !isNaN || bits&0x00400000 == 0 {
				return fmt.Errorf("got %#08x, want arithmetic NaN", bits)
			}
			return nil
		}
	case "f64":
		isNaN := got&0x7ff0000000000000 == 0x7ff0000000000000 && got&0x000fffffffffffff != 0
		switch want.Value {
		case "nan:canonical":
			if got&math.MaxInt64 != 0x7ff8000000000000 {
				return fmt.Errorf("got %#016x, want canonical NaN", got)
			}
			return nil
		case "nan:arithmetic":
			if !isNaN || got&(1<<51) == 0 {
				return fmt.Errorf("got %#016x, want arithmetic NaN", got)
			}
			return nil
		}
	case "funcref", "externref":
		if want.Value == "" {
			if got == 0 {
				return errors.New("got null, want a reference")
			}
			return nil
		}
	}
	w, err := parse(want)
	if err != nil {
		return err
	}
	if want.Type == "i32" || want.Type == "f32" {
		w, got = uint64(uint32(w)), uint64(uint32(got))
	}
	if w != got {
		return fmt.Errorf("got %#x, want %#x (%s)", got, w, want.Type)
	}
	return nil
}
