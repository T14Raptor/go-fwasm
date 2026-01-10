package generator

import (
	"fmt"
	"io"
	"strings"

	"github.com/t14raptor/go-fwasm/instruction"
	"github.com/t14raptor/go-fwasm/module"
	"github.com/t14raptor/go-fwasm/types"
)

type Generator struct {
	w      io.Writer
	indent int
	mod    *module.Module
}

func New(w io.Writer) *Generator {
	return &Generator{w: w}
}

func (g *Generator) Generate(mod *module.Module) error {
	g.mod = mod
	g.indent = 0

	g.writeLine("(module")
	g.indent++

	// Types
	for i, ft := range mod.Types {
		g.writeType(i, ft)
	}

	// Imports
	for _, imp := range mod.Imports {
		g.writeImport(imp)
	}

	// Tables
	for i, table := range mod.Tables {
		g.writeTable(i, table)
	}

	// Memories
	for i, mem := range mod.Memories {
		g.writeMemory(i, mem)
	}

	// Globals
	for i, global := range mod.Globals {
		g.writeGlobal(i, global)
	}

	// Exports
	for _, exp := range mod.Exports {
		g.writeExport(exp)
	}

	// Start
	if mod.Start != nil {
		g.writeLine(fmt.Sprintf("(start %d)", *mod.Start))
	}

	// Elements
	for i, elem := range mod.Elements {
		g.writeElement(i, elem)
	}

	// Functions
	numImportedFuncs := mod.NumImportedFunctions()
	for i, code := range mod.Codes {
		funcIdx := numImportedFuncs + i
		var funcType *types.FuncType
		if i < len(mod.Functions) {
			typeIdx := mod.Functions[i]
			if int(typeIdx) < len(mod.Types) {
				funcType = &mod.Types[typeIdx]
			}
		}
		g.writeFunc(funcIdx, i, funcType, code)
	}

	// Data
	for i, data := range mod.Datas {
		g.writeData(i, data)
	}

	g.indent--
	g.writeLine(")")

	return nil
}

func (g *Generator) write(s string) {
	fmt.Fprint(g.w, s)
}

func (g *Generator) writeLine(s string) {
	fmt.Fprintf(g.w, "%s%s\n", strings.Repeat("  ", g.indent), s)
}

func (g *Generator) writeType(idx int, ft types.FuncType) {
	params := formatValueTypes(ft.Params)
	results := formatValueTypes(ft.Results)

	line := fmt.Sprintf("(type (;%d;) (func", idx)
	if len(ft.Params) > 0 {
		line += fmt.Sprintf(" (param%s)", params)
	}
	if len(ft.Results) > 0 {
		line += fmt.Sprintf(" (result%s)", results)
	}
	line += "))"
	g.writeLine(line)
}

func (g *Generator) writeImport(imp types.Import) {
	var desc string
	switch imp.Desc.Kind {
	case types.ImportFunc:
		desc = fmt.Sprintf("(func (type %d))", imp.Desc.TypeIdx)
	case types.ImportTable:
		desc = fmt.Sprintf("(table %s)", formatTableType(imp.Desc.TableType))
	case types.ImportMem:
		desc = fmt.Sprintf("(memory %s)", formatLimits(imp.Desc.MemType.Limits))
	case types.ImportGlobal:
		desc = fmt.Sprintf("(global %s)", formatGlobalType(imp.Desc.GlobalType))
	}
	g.writeLine(fmt.Sprintf("(import %q %q %s)", imp.Module, imp.Name, desc))
}

func (g *Generator) writeTable(idx int, table types.TableType) {
	g.writeLine(fmt.Sprintf("(table (;%d;) %s)", idx, formatTableType(table)))
}

func (g *Generator) writeMemory(idx int, mem types.MemoryType) {
	g.writeLine(fmt.Sprintf("(memory (;%d;) %s)", idx, formatLimits(mem.Limits)))
}

func (g *Generator) writeGlobal(idx int, global module.Global) {
	init := g.formatExpressionInline(global.Init)
	g.writeLine(fmt.Sprintf("(global (;%d;) %s %s)", idx, formatGlobalType(global.Type), init))
}

func (g *Generator) writeExport(exp types.Export) {
	var kind string
	switch exp.Desc.Kind {
	case types.ExportFunc:
		kind = "func"
	case types.ExportTable:
		kind = "table"
	case types.ExportMem:
		kind = "memory"
	case types.ExportGlobal:
		kind = "global"
	}
	g.writeLine(fmt.Sprintf("(export %q (%s %d))", exp.Name, kind, exp.Desc.Idx))
}

func (g *Generator) writeElement(idx int, elem module.Element) {
	var mode string
	switch elem.Mode {
	case module.ElementModePassive:
		mode = ""
	case module.ElementModeActive:
		offset := g.formatExpressionInline(elem.Offset)
		if elem.TableIdx == 0 {
			mode = fmt.Sprintf("(offset %s)", offset)
		} else {
			mode = fmt.Sprintf("(table %d) (offset %s)", elem.TableIdx, offset)
		}
	case module.ElementModeDeclarative:
		mode = "declare"
	}

	g.writeLine(fmt.Sprintf("(elem (;%d;) %s %s ...)", idx, mode, elem.Type.String()))
}

func (g *Generator) writeFunc(funcIdx, localIdx int, funcType *types.FuncType, code module.Code) {
	// Function signature
	sig := ""
	if funcType != nil {
		if len(funcType.Params) > 0 {
			sig += fmt.Sprintf(" (param%s)", formatValueTypes(funcType.Params))
		}
		if len(funcType.Results) > 0 {
			sig += fmt.Sprintf(" (result%s)", formatValueTypes(funcType.Results))
		}
	}

	// Locals
	locals := ""
	for _, local := range code.Locals {
		for i := uint32(0); i < local.Count; i++ {
			locals += fmt.Sprintf(" (local %s)", local.Type.String())
		}
	}

	typeIdx := uint32(0)
	if localIdx < len(g.mod.Functions) {
		typeIdx = g.mod.Functions[localIdx]
	}

	g.writeLine(fmt.Sprintf("(func (;%d;) (type %d)%s%s", funcIdx, typeIdx, sig, locals))
	g.indent++

	// Body
	g.writeInstructions(code.Body)

	g.indent--
	g.writeLine(")")
}

func (g *Generator) writeData(idx int, data module.Data) {
	var mode string
	switch data.Mode {
	case module.DataModePassive:
		mode = ""
	case module.DataModeActive:
		offset := g.formatExpressionInline(data.Offset)
		if data.MemIdx == 0 {
			mode = fmt.Sprintf("(offset %s)", offset)
		} else {
			mode = fmt.Sprintf("(memory %d) (offset %s)", data.MemIdx, offset)
		}
	}

	// Format init data as string
	initStr := formatDataString(data.Init)
	g.writeLine(fmt.Sprintf("(data (;%d;) %s %s)", idx, mode, initStr))
}

func (g *Generator) writeInstructions(instrs []instruction.Instruction) {
	for _, instr := range instrs {
		g.writeInstruction(instr)
	}
}

func (g *Generator) writeInstruction(instr instruction.Instruction) {
	switch i := instr.(type) {
	// Unwrap category wrappers
	case instruction.Control:
		g.writeInstruction(i.Instr)
		return
	case instruction.Reference:
		g.writeInstruction(i.Instr)
		return
	case instruction.Parametric:
		g.writeInstruction(i.Instr)
		return
	case instruction.Variable:
		g.writeInstruction(i.Instr)
		return
	case instruction.Memory:
		g.writeInstruction(i.Instr)
		return
	case instruction.Numeric:
		g.writeInstruction(i.Instr)
		return

	// Control instructions with blocks
	case instruction.Block:
		g.writeLine(fmt.Sprintf("block%s", formatBlockType(i.BlockType)))
		g.indent++
		g.writeInstructions(i.Body)
		g.indent--
		g.writeLine("end")

	case instruction.Loop:
		g.writeLine(fmt.Sprintf("loop%s", formatBlockType(i.BlockType)))
		g.indent++
		g.writeInstructions(i.Body)
		g.indent--
		g.writeLine("end")

	case instruction.If:
		g.writeLine(fmt.Sprintf("if%s", formatBlockType(i.BlockType)))
		g.indent++
		g.writeInstructions(i.Then)
		if len(i.Else) > 0 {
			g.indent--
			g.writeLine("else")
			g.indent++
			g.writeInstructions(i.Else)
		}
		g.indent--
		g.writeLine("end")

	// Simple control instructions
	case instruction.Unreachable:
		g.writeLine("unreachable")
	case instruction.Nop:
		g.writeLine("nop")
	case instruction.Br:
		g.writeLine(fmt.Sprintf("br %d", i.LabelIdx))
	case instruction.BrIf:
		g.writeLine(fmt.Sprintf("br_if %d", i.LabelIdx))
	case instruction.BrTable:
		labels := make([]string, len(i.Labels))
		for j, l := range i.Labels {
			labels[j] = fmt.Sprintf("%d", l)
		}
		g.writeLine(fmt.Sprintf("br_table %s %d", strings.Join(labels, " "), i.DefaultLabel))
	case instruction.Return:
		g.writeLine("return")
	case instruction.Call:
		g.writeLine(fmt.Sprintf("call %d", i.FuncIdx))
	case instruction.CallIndirect:
		g.writeLine(fmt.Sprintf("call_indirect (type %d)", i.TypeIdx))

	// Reference instructions
	case instruction.RefNull:
		g.writeLine(fmt.Sprintf("ref.null %s", i.Type.String()))
	case instruction.RefIsNull:
		g.writeLine("ref.is_null")
	case instruction.RefFunc:
		g.writeLine(fmt.Sprintf("ref.func %d", i.FuncIdx))

	// Parametric instructions
	case instruction.Drop:
		g.writeLine("drop")
	case instruction.Select:
		g.writeLine("select")
	case instruction.SelectT:
		g.writeLine(fmt.Sprintf("select%s", formatValueTypes(i.Types)))

	// Variable instructions
	case instruction.LocalGet:
		g.writeLine(fmt.Sprintf("local.get %d", i.LocalIdx))
	case instruction.LocalSet:
		g.writeLine(fmt.Sprintf("local.set %d", i.LocalIdx))
	case instruction.LocalTee:
		g.writeLine(fmt.Sprintf("local.tee %d", i.LocalIdx))
	case instruction.GlobalGet:
		g.writeLine(fmt.Sprintf("global.get %d", i.GlobalIdx))
	case instruction.GlobalSet:
		g.writeLine(fmt.Sprintf("global.set %d", i.GlobalIdx))
	case instruction.TableGet:
		g.writeLine(fmt.Sprintf("table.get %d", i.TableIdx))
	case instruction.TableSet:
		g.writeLine(fmt.Sprintf("table.set %d", i.TableIdx))

	// Memory instructions
	case instruction.I32Load:
		g.writeLine(formatMemInstr("i32.load", i.MemArg, 4))
	case instruction.I64Load:
		g.writeLine(formatMemInstr("i64.load", i.MemArg, 8))
	case instruction.F32Load:
		g.writeLine(formatMemInstr("f32.load", i.MemArg, 4))
	case instruction.F64Load:
		g.writeLine(formatMemInstr("f64.load", i.MemArg, 8))
	case instruction.I32Load8S:
		g.writeLine(formatMemInstr("i32.load8_s", i.MemArg, 1))
	case instruction.I32Load8U:
		g.writeLine(formatMemInstr("i32.load8_u", i.MemArg, 1))
	case instruction.I32Load16S:
		g.writeLine(formatMemInstr("i32.load16_s", i.MemArg, 2))
	case instruction.I32Load16U:
		g.writeLine(formatMemInstr("i32.load16_u", i.MemArg, 2))
	case instruction.I64Load8S:
		g.writeLine(formatMemInstr("i64.load8_s", i.MemArg, 1))
	case instruction.I64Load8U:
		g.writeLine(formatMemInstr("i64.load8_u", i.MemArg, 1))
	case instruction.I64Load16S:
		g.writeLine(formatMemInstr("i64.load16_s", i.MemArg, 2))
	case instruction.I64Load16U:
		g.writeLine(formatMemInstr("i64.load16_u", i.MemArg, 2))
	case instruction.I64Load32S:
		g.writeLine(formatMemInstr("i64.load32_s", i.MemArg, 4))
	case instruction.I64Load32U:
		g.writeLine(formatMemInstr("i64.load32_u", i.MemArg, 4))
	case instruction.I32Store:
		g.writeLine(formatMemInstr("i32.store", i.MemArg, 4))
	case instruction.I64Store:
		g.writeLine(formatMemInstr("i64.store", i.MemArg, 8))
	case instruction.F32Store:
		g.writeLine(formatMemInstr("f32.store", i.MemArg, 4))
	case instruction.F64Store:
		g.writeLine(formatMemInstr("f64.store", i.MemArg, 8))
	case instruction.I32Store8:
		g.writeLine(formatMemInstr("i32.store8", i.MemArg, 1))
	case instruction.I32Store16:
		g.writeLine(formatMemInstr("i32.store16", i.MemArg, 2))
	case instruction.I64Store8:
		g.writeLine(formatMemInstr("i64.store8", i.MemArg, 1))
	case instruction.I64Store16:
		g.writeLine(formatMemInstr("i64.store16", i.MemArg, 2))
	case instruction.I64Store32:
		g.writeLine(formatMemInstr("i64.store32", i.MemArg, 4))
	case instruction.MemorySize:
		g.writeLine("memory.size")
	case instruction.MemoryGrow:
		g.writeLine("memory.grow")

	// Numeric constants
	case instruction.I32Const:
		g.writeLine(fmt.Sprintf("i32.const %d", i.Value))
	case instruction.I64Const:
		g.writeLine(fmt.Sprintf("i64.const %d", i.Value))
	case instruction.F32Const:
		g.writeLine(fmt.Sprintf("f32.const %g", i.Value))
	case instruction.F64Const:
		g.writeLine(fmt.Sprintf("f64.const %g", i.Value))

	// i32 comparison
	case instruction.I32Eqz:
		g.writeLine("i32.eqz")
	case instruction.I32Eq:
		g.writeLine("i32.eq")
	case instruction.I32Ne:
		g.writeLine("i32.ne")
	case instruction.I32LtS:
		g.writeLine("i32.lt_s")
	case instruction.I32LtU:
		g.writeLine("i32.lt_u")
	case instruction.I32GtS:
		g.writeLine("i32.gt_s")
	case instruction.I32GtU:
		g.writeLine("i32.gt_u")
	case instruction.I32LeS:
		g.writeLine("i32.le_s")
	case instruction.I32LeU:
		g.writeLine("i32.le_u")
	case instruction.I32GeS:
		g.writeLine("i32.ge_s")
	case instruction.I32GeU:
		g.writeLine("i32.ge_u")

	// i64 comparison
	case instruction.I64Eqz:
		g.writeLine("i64.eqz")
	case instruction.I64Eq:
		g.writeLine("i64.eq")
	case instruction.I64Ne:
		g.writeLine("i64.ne")
	case instruction.I64LtS:
		g.writeLine("i64.lt_s")
	case instruction.I64LtU:
		g.writeLine("i64.lt_u")
	case instruction.I64GtS:
		g.writeLine("i64.gt_s")
	case instruction.I64GtU:
		g.writeLine("i64.gt_u")
	case instruction.I64LeS:
		g.writeLine("i64.le_s")
	case instruction.I64LeU:
		g.writeLine("i64.le_u")
	case instruction.I64GeS:
		g.writeLine("i64.ge_s")
	case instruction.I64GeU:
		g.writeLine("i64.ge_u")

	// f32 comparison
	case instruction.F32Eq:
		g.writeLine("f32.eq")
	case instruction.F32Ne:
		g.writeLine("f32.ne")
	case instruction.F32Lt:
		g.writeLine("f32.lt")
	case instruction.F32Gt:
		g.writeLine("f32.gt")
	case instruction.F32Le:
		g.writeLine("f32.le")
	case instruction.F32Ge:
		g.writeLine("f32.ge")

	// f64 comparison
	case instruction.F64Eq:
		g.writeLine("f64.eq")
	case instruction.F64Ne:
		g.writeLine("f64.ne")
	case instruction.F64Lt:
		g.writeLine("f64.lt")
	case instruction.F64Gt:
		g.writeLine("f64.gt")
	case instruction.F64Le:
		g.writeLine("f64.le")
	case instruction.F64Ge:
		g.writeLine("f64.ge")

	// i32 arithmetic
	case instruction.I32Clz:
		g.writeLine("i32.clz")
	case instruction.I32Ctz:
		g.writeLine("i32.ctz")
	case instruction.I32Popcnt:
		g.writeLine("i32.popcnt")
	case instruction.I32Add:
		g.writeLine("i32.add")
	case instruction.I32Sub:
		g.writeLine("i32.sub")
	case instruction.I32Mul:
		g.writeLine("i32.mul")
	case instruction.I32DivS:
		g.writeLine("i32.div_s")
	case instruction.I32DivU:
		g.writeLine("i32.div_u")
	case instruction.I32RemS:
		g.writeLine("i32.rem_s")
	case instruction.I32RemU:
		g.writeLine("i32.rem_u")
	case instruction.I32And:
		g.writeLine("i32.and")
	case instruction.I32Or:
		g.writeLine("i32.or")
	case instruction.I32Xor:
		g.writeLine("i32.xor")
	case instruction.I32Shl:
		g.writeLine("i32.shl")
	case instruction.I32ShrS:
		g.writeLine("i32.shr_s")
	case instruction.I32ShrU:
		g.writeLine("i32.shr_u")
	case instruction.I32Rotl:
		g.writeLine("i32.rotl")
	case instruction.I32Rotr:
		g.writeLine("i32.rotr")

	// i64 arithmetic
	case instruction.I64Clz:
		g.writeLine("i64.clz")
	case instruction.I64Ctz:
		g.writeLine("i64.ctz")
	case instruction.I64Popcnt:
		g.writeLine("i64.popcnt")
	case instruction.I64Add:
		g.writeLine("i64.add")
	case instruction.I64Sub:
		g.writeLine("i64.sub")
	case instruction.I64Mul:
		g.writeLine("i64.mul")
	case instruction.I64DivS:
		g.writeLine("i64.div_s")
	case instruction.I64DivU:
		g.writeLine("i64.div_u")
	case instruction.I64RemS:
		g.writeLine("i64.rem_s")
	case instruction.I64RemU:
		g.writeLine("i64.rem_u")
	case instruction.I64And:
		g.writeLine("i64.and")
	case instruction.I64Or:
		g.writeLine("i64.or")
	case instruction.I64Xor:
		g.writeLine("i64.xor")
	case instruction.I64Shl:
		g.writeLine("i64.shl")
	case instruction.I64ShrS:
		g.writeLine("i64.shr_s")
	case instruction.I64ShrU:
		g.writeLine("i64.shr_u")
	case instruction.I64Rotl:
		g.writeLine("i64.rotl")
	case instruction.I64Rotr:
		g.writeLine("i64.rotr")

	// f32 arithmetic
	case instruction.F32Abs:
		g.writeLine("f32.abs")
	case instruction.F32Neg:
		g.writeLine("f32.neg")
	case instruction.F32Ceil:
		g.writeLine("f32.ceil")
	case instruction.F32Floor:
		g.writeLine("f32.floor")
	case instruction.F32Trunc:
		g.writeLine("f32.trunc")
	case instruction.F32Nearest:
		g.writeLine("f32.nearest")
	case instruction.F32Sqrt:
		g.writeLine("f32.sqrt")
	case instruction.F32Add:
		g.writeLine("f32.add")
	case instruction.F32Sub:
		g.writeLine("f32.sub")
	case instruction.F32Mul:
		g.writeLine("f32.mul")
	case instruction.F32Div:
		g.writeLine("f32.div")
	case instruction.F32Min:
		g.writeLine("f32.min")
	case instruction.F32Max:
		g.writeLine("f32.max")
	case instruction.F32Copysign:
		g.writeLine("f32.copysign")

	// f64 arithmetic
	case instruction.F64Abs:
		g.writeLine("f64.abs")
	case instruction.F64Neg:
		g.writeLine("f64.neg")
	case instruction.F64Ceil:
		g.writeLine("f64.ceil")
	case instruction.F64Floor:
		g.writeLine("f64.floor")
	case instruction.F64Trunc:
		g.writeLine("f64.trunc")
	case instruction.F64Nearest:
		g.writeLine("f64.nearest")
	case instruction.F64Sqrt:
		g.writeLine("f64.sqrt")
	case instruction.F64Add:
		g.writeLine("f64.add")
	case instruction.F64Sub:
		g.writeLine("f64.sub")
	case instruction.F64Mul:
		g.writeLine("f64.mul")
	case instruction.F64Div:
		g.writeLine("f64.div")
	case instruction.F64Min:
		g.writeLine("f64.min")
	case instruction.F64Max:
		g.writeLine("f64.max")
	case instruction.F64Copysign:
		g.writeLine("f64.copysign")

	// Conversions
	case instruction.I32WrapI64:
		g.writeLine("i32.wrap_i64")
	case instruction.I32TruncF32S:
		g.writeLine("i32.trunc_f32_s")
	case instruction.I32TruncF32U:
		g.writeLine("i32.trunc_f32_u")
	case instruction.I32TruncF64S:
		g.writeLine("i32.trunc_f64_s")
	case instruction.I32TruncF64U:
		g.writeLine("i32.trunc_f64_u")
	case instruction.I64ExtendI32S:
		g.writeLine("i64.extend_i32_s")
	case instruction.I64ExtendI32U:
		g.writeLine("i64.extend_i32_u")
	case instruction.I64TruncF32S:
		g.writeLine("i64.trunc_f32_s")
	case instruction.I64TruncF32U:
		g.writeLine("i64.trunc_f32_u")
	case instruction.I64TruncF64S:
		g.writeLine("i64.trunc_f64_s")
	case instruction.I64TruncF64U:
		g.writeLine("i64.trunc_f64_u")
	case instruction.F32ConvertI32S:
		g.writeLine("f32.convert_i32_s")
	case instruction.F32ConvertI32U:
		g.writeLine("f32.convert_i32_u")
	case instruction.F32ConvertI64S:
		g.writeLine("f32.convert_i64_s")
	case instruction.F32ConvertI64U:
		g.writeLine("f32.convert_i64_u")
	case instruction.F32DemoteF64:
		g.writeLine("f32.demote_f64")
	case instruction.F64ConvertI32S:
		g.writeLine("f64.convert_i32_s")
	case instruction.F64ConvertI32U:
		g.writeLine("f64.convert_i32_u")
	case instruction.F64ConvertI64S:
		g.writeLine("f64.convert_i64_s")
	case instruction.F64ConvertI64U:
		g.writeLine("f64.convert_i64_u")
	case instruction.F64PromoteF32:
		g.writeLine("f64.promote_f32")
	case instruction.I32ReinterpretF32:
		g.writeLine("i32.reinterpret_f32")
	case instruction.I64ReinterpretF64:
		g.writeLine("i64.reinterpret_f64")
	case instruction.F32ReinterpretI32:
		g.writeLine("f32.reinterpret_i32")
	case instruction.F64ReinterpretI64:
		g.writeLine("f64.reinterpret_i64")

	// Sign extension
	case instruction.I32Extend8S:
		g.writeLine("i32.extend8_s")
	case instruction.I32Extend16S:
		g.writeLine("i32.extend16_s")
	case instruction.I64Extend8S:
		g.writeLine("i64.extend8_s")
	case instruction.I64Extend16S:
		g.writeLine("i64.extend16_s")
	case instruction.I64Extend32S:
		g.writeLine("i64.extend32_s")

	// Extended instructions
	case instruction.I32TruncSatF32S:
		g.writeLine("i32.trunc_sat_f32_s")
	case instruction.I32TruncSatF32U:
		g.writeLine("i32.trunc_sat_f32_u")
	case instruction.I32TruncSatF64S:
		g.writeLine("i32.trunc_sat_f64_s")
	case instruction.I32TruncSatF64U:
		g.writeLine("i32.trunc_sat_f64_u")
	case instruction.I64TruncSatF32S:
		g.writeLine("i64.trunc_sat_f32_s")
	case instruction.I64TruncSatF32U:
		g.writeLine("i64.trunc_sat_f32_u")
	case instruction.I64TruncSatF64S:
		g.writeLine("i64.trunc_sat_f64_s")
	case instruction.I64TruncSatF64U:
		g.writeLine("i64.trunc_sat_f64_u")
	case instruction.MemoryInit:
		g.writeLine(fmt.Sprintf("memory.init %d", i.DataIdx))
	case instruction.DataDrop:
		g.writeLine(fmt.Sprintf("data.drop %d", i.DataIdx))
	case instruction.MemoryCopy:
		g.writeLine("memory.copy")
	case instruction.MemoryFill:
		g.writeLine("memory.fill")
	case instruction.TableInit:
		g.writeLine(fmt.Sprintf("table.init %d %d", i.ElemIdx, i.TableIdx))
	case instruction.ElemDrop:
		g.writeLine(fmt.Sprintf("elem.drop %d", i.ElemIdx))
	case instruction.TableCopy:
		g.writeLine(fmt.Sprintf("table.copy %d %d", i.DstTableIdx, i.SrcTableIdx))
	case instruction.TableGrow:
		g.writeLine(fmt.Sprintf("table.grow %d", i.TableIdx))
	case instruction.TableSize:
		g.writeLine(fmt.Sprintf("table.size %d", i.TableIdx))
	case instruction.TableFill:
		g.writeLine(fmt.Sprintf("table.fill %d", i.TableIdx))

	default:
		g.writeLine(fmt.Sprintf(";; unknown instruction: %T", instr))
	}
}

func (g *Generator) formatExpressionInline(instrs []instruction.Instruction) string {
	var parts []string
	for _, instr := range instrs {
		parts = append(parts, g.formatInstructionInline(instr))
	}
	return "(" + strings.Join(parts, " ") + ")"
}

func (g *Generator) formatInstructionInline(instr instruction.Instruction) string {
	switch i := instr.(type) {
	// Unwrap category wrappers
	case instruction.Control:
		return g.formatInstructionInline(i.Instr)
	case instruction.Reference:
		return g.formatInstructionInline(i.Instr)
	case instruction.Parametric:
		return g.formatInstructionInline(i.Instr)
	case instruction.Variable:
		return g.formatInstructionInline(i.Instr)
	case instruction.Memory:
		return g.formatInstructionInline(i.Instr)
	case instruction.Numeric:
		return g.formatInstructionInline(i.Instr)

	case instruction.I32Const:
		return fmt.Sprintf("i32.const %d", i.Value)
	case instruction.I64Const:
		return fmt.Sprintf("i64.const %d", i.Value)
	case instruction.F32Const:
		return fmt.Sprintf("f32.const %g", i.Value)
	case instruction.F64Const:
		return fmt.Sprintf("f64.const %g", i.Value)
	case instruction.GlobalGet:
		return fmt.Sprintf("global.get %d", i.GlobalIdx)
	case instruction.RefNull:
		return fmt.Sprintf("ref.null %s", i.Type.String())
	case instruction.RefFunc:
		return fmt.Sprintf("ref.func %d", i.FuncIdx)
	default:
		return fmt.Sprintf("%s", instr.Opcode().String())
	}
}

func formatValueTypes(vts []types.ValueType) string {
	if len(vts) == 0 {
		return ""
	}
	var parts []string
	for _, vt := range vts {
		parts = append(parts, vt.String())
	}
	return " " + strings.Join(parts, " ")
}

func formatBlockType(bt types.BlockType) string {
	switch bt.Kind {
	case types.BlockTypeEmpty:
		return ""
	case types.BlockTypeValue:
		return fmt.Sprintf(" (result %s)", bt.ValType.String())
	case types.BlockTypeIndex:
		return fmt.Sprintf(" (type %d)", bt.TypeIdx)
	default:
		return ""
	}
}

func formatLimits(l types.Limits) string {
	if l.HasMax {
		return fmt.Sprintf("%d %d", l.Min, l.Max)
	}
	return fmt.Sprintf("%d", l.Min)
}

func formatTableType(t types.TableType) string {
	return fmt.Sprintf("%s %s", formatLimits(t.Limits), t.ElemType.String())
}

func formatGlobalType(g types.GlobalType) string {
	if g.Mutable {
		return fmt.Sprintf("(mut %s)", g.ValType.String())
	}
	return g.ValType.String()
}

func formatMemInstr(name string, memArg instruction.MemArg, naturalAlign uint32) string {
	var parts []string
	parts = append(parts, name)

	// Only show offset if non-zero
	if memArg.Offset != 0 {
		parts = append(parts, fmt.Sprintf("offset=%d", memArg.Offset))
	}

	// Only show align if not natural
	align := uint32(1) << memArg.Align
	if align != naturalAlign {
		parts = append(parts, fmt.Sprintf("align=%d", align))
	}

	return strings.Join(parts, " ")
}

func formatDataString(data []byte) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, b := range data {
		if b >= 32 && b < 127 && b != '"' && b != '\\' {
			sb.WriteByte(b)
		} else {
			sb.WriteString(fmt.Sprintf("\\%02x", b))
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

func Generate(mod *module.Module) (string, error) {
	var sb strings.Builder
	g := New(&sb)
	if err := g.Generate(mod); err != nil {
		return "", err
	}
	return sb.String(), nil
}
