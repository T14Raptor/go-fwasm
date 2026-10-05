package runtime

import (
	"fmt"
	"slices"
	"strings"

	"github.com/t14raptor/go-fwasm/compile"
)

// TrapCode identifies why execution trapped.
type TrapCode uint8

const (
	TrapUnreachable TrapCode = iota + 1
	TrapIntegerDivideByZero
	TrapIntegerOverflow
	TrapInvalidConversion
	TrapMemoryOutOfBounds
	TrapTableOutOfBounds
	TrapUndefinedElement
	TrapUninitializedElement
	TrapIndirectCallTypeMismatch
	TrapCallStackExhausted
	TrapInterrupted
)

// The messages match the spec interpreter's, which the spec tests check.
var trapMessages = [...]string{
	TrapUnreachable:              "unreachable",
	TrapIntegerDivideByZero:      "integer divide by zero",
	TrapIntegerOverflow:          "integer overflow",
	TrapInvalidConversion:        "invalid conversion to integer",
	TrapMemoryOutOfBounds:        "out of bounds memory access",
	TrapTableOutOfBounds:         "out of bounds table access",
	TrapUndefinedElement:         "undefined element",
	TrapUninitializedElement:     "uninitialized element",
	TrapIndirectCallTypeMismatch: "indirect call type mismatch",
	TrapCallStackExhausted:       "call stack exhausted",
	TrapInterrupted:              "interrupted",
}

func (c TrapCode) String() string {
	if int(c) < len(trapMessages) && trapMessages[c] != "" {
		return trapMessages[c]
	}
	return fmt.Sprintf("trap(%d)", uint8(c))
}

// Trap is a wasm trap.
type Trap struct {
	Code TrapCode
	// Stack lists the wasm frames active at the trap, innermost first.
	Stack []TrapFrame
}

// TrapFrame is one frame of a trap's stack.
type TrapFrame struct {
	Func *Function
	PC   int // bytecode index of the trapping or calling instruction
}

func (t *Trap) Error() string {
	var b strings.Builder
	b.WriteString("wasm trap: ")
	b.WriteString(t.Code.String())
	for i, f := range t.Stack {
		if i == 8 {
			fmt.Fprintf(&b, " ... (%d more)", len(t.Stack)-i)
			break
		}
		fmt.Fprintf(&b, "\n\tat %s pc=%d", f.Func, f.PC)
	}
	return b.String()
}

// trap builds a Trap for the instruction before pc in cf, the code fn is
// running. An instruction inlined from another function shows as a frame
// of its own.
func (s *Store) trap(code TrapCode, cf *compile.Func, fn *Function, pc int) error {
	t := &Trap{Code: code}
	pc--
	for {
		site := inlineSite(cf, pc)
		if site == nil {
			break
		}
		t.Stack = append(t.Stack, TrapFrame{fn, int(site.Start)})
		fn = fn.inst.funcs[site.Func]
		cf, pc = fn.code, site.CalleePC(pc)
	}
	t.Stack = append(t.Stack, TrapFrame{fn, pc})
	slices.Reverse(t.Stack)
	for i := len(s.frames) - 1; i >= 0; i-- {
		f := s.frames[i]
		t.Stack = append(t.Stack, TrapFrame{s.funcs[f.fn], int(f.pc) - 1})
	}
	return t
}

func inlineSite(cf *compile.Func, pc int) *compile.InlineSite {
	for i := range cf.Inlined {
		if site := &cf.Inlined[i]; pc >= int(site.Start) && pc < int(site.End) {
			return site
		}
	}
	return nil
}
