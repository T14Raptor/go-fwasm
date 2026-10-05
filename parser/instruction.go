package parser

import (
	"fmt"
	"math"

	"github.com/t14raptor/go-fwasm/instruction"
	"github.com/t14raptor/go-fwasm/types"
)

type parseResult struct {
	instr  instruction.Instruction
	isEnd  bool
	isElse bool
}

func parseExpression(r *Reader) ([]instruction.Instruction, error) {
	var instrs []instruction.Instruction

	for {
		result, err := parseInstructionInternal(r)
		if err != nil {
			return nil, err
		}

		if result.isEnd {
			break
		}
		if result.isElse {
			return nil, fmt.Errorf("%w: else outside if", ErrInvalidOpcode)
		}

		instrs = append(instrs, result.instr)
	}

	return instrs, nil
}

func parseInstructionInternal(r *Reader) (parseResult, error) {
	op, err := r.ReadByte()
	if err != nil {
		return parseResult{}, err
	}

	opcode := instruction.Opcode(op)

	switch opcode {
	// Control instructions
	case instruction.OpUnreachable:
		return ctrl(instruction.Unreachable{}), nil

	case instruction.OpNop:
		return ctrl(instruction.Nop{}), nil

	case instruction.OpBlock:
		blockType, err := parseBlockType(r)
		if err != nil {
			return parseResult{}, err
		}
		body, err := parseBlockBody(r)
		if err != nil {
			return parseResult{}, err
		}
		return ctrl(instruction.Block{BlockType: blockType, Body: body}), nil

	case instruction.OpLoop:
		blockType, err := parseBlockType(r)
		if err != nil {
			return parseResult{}, err
		}
		body, err := parseBlockBody(r)
		if err != nil {
			return parseResult{}, err
		}
		return ctrl(instruction.Loop{BlockType: blockType, Body: body}), nil

	case instruction.OpIf:
		blockType, err := parseBlockType(r)
		if err != nil {
			return parseResult{}, err
		}
		thenBody, elseBody, err := parseIfBody(r)
		if err != nil {
			return parseResult{}, err
		}
		return ctrl(instruction.If{BlockType: blockType, Then: thenBody, Else: elseBody}), nil

	case instruction.OpElse:
		return parseResult{isElse: true}, nil

	case instruction.OpEnd:
		return parseResult{isEnd: true}, nil

	case instruction.OpBr:
		labelIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return ctrl(instruction.Br{LabelIdx: labelIdx}), nil

	case instruction.OpBrIf:
		labelIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return ctrl(instruction.BrIf{LabelIdx: labelIdx}), nil

	case instruction.OpBrTable:
		count, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		labels := make([]uint32, count)
		for i := uint32(0); i < count; i++ {
			label, err := r.ReadU32()
			if err != nil {
				return parseResult{}, err
			}
			labels[i] = label
		}
		defaultLabel, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return ctrl(instruction.BrTable{Labels: labels, DefaultLabel: defaultLabel}), nil

	case instruction.OpReturn:
		return ctrl(instruction.Return{}), nil

	case instruction.OpCall:
		funcIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return ctrl(instruction.Call{FuncIdx: funcIdx}), nil

	case instruction.OpCallIndirect:
		typeIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		tableIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return ctrl(instruction.CallIndirect{TypeIdx: typeIdx, TableIdx: tableIdx}), nil

	// Reference instructions
	case instruction.OpRefNull:
		refType, err := readRefType(r)
		if err != nil {
			return parseResult{}, err
		}
		return ref(instruction.RefNull{Type: refType}), nil

	case instruction.OpRefIsNull:
		return ref(instruction.RefIsNull{}), nil

	case instruction.OpRefFunc:
		funcIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return ref(instruction.RefFunc{FuncIdx: funcIdx}), nil

	// Parametric instructions
	case instruction.OpDrop:
		return param(instruction.Drop{}), nil

	case instruction.OpSelect:
		return param(instruction.Select{}), nil

	case instruction.OpSelectT:
		valTypes, err := parseValueTypes(r)
		if err != nil {
			return parseResult{}, err
		}
		return param(instruction.SelectT{Types: valTypes}), nil

	// Variable instructions
	case instruction.OpLocalGet:
		localIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return variable(instruction.LocalGet{LocalIdx: localIdx}), nil

	case instruction.OpLocalSet:
		localIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return variable(instruction.LocalSet{LocalIdx: localIdx}), nil

	case instruction.OpLocalTee:
		localIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return variable(instruction.LocalTee{LocalIdx: localIdx}), nil

	case instruction.OpGlobalGet:
		globalIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return variable(instruction.GlobalGet{GlobalIdx: globalIdx}), nil

	case instruction.OpGlobalSet:
		globalIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return variable(instruction.GlobalSet{GlobalIdx: globalIdx}), nil

	case instruction.OpTableGet:
		tableIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return variable(instruction.TableGet{TableIdx: tableIdx}), nil

	case instruction.OpTableSet:
		tableIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return variable(instruction.TableSet{TableIdx: tableIdx}), nil

	// Memory instructions
	case instruction.OpI32Load:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I32Load{MemArg: memArg}), nil

	case instruction.OpI64Load:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I64Load{MemArg: memArg}), nil

	case instruction.OpF32Load:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.F32Load{MemArg: memArg}), nil

	case instruction.OpF64Load:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.F64Load{MemArg: memArg}), nil

	case instruction.OpI32Load8S:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I32Load8S{MemArg: memArg}), nil

	case instruction.OpI32Load8U:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I32Load8U{MemArg: memArg}), nil

	case instruction.OpI32Load16S:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I32Load16S{MemArg: memArg}), nil

	case instruction.OpI32Load16U:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I32Load16U{MemArg: memArg}), nil

	case instruction.OpI64Load8S:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I64Load8S{MemArg: memArg}), nil

	case instruction.OpI64Load8U:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I64Load8U{MemArg: memArg}), nil

	case instruction.OpI64Load16S:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I64Load16S{MemArg: memArg}), nil

	case instruction.OpI64Load16U:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I64Load16U{MemArg: memArg}), nil

	case instruction.OpI64Load32S:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I64Load32S{MemArg: memArg}), nil

	case instruction.OpI64Load32U:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I64Load32U{MemArg: memArg}), nil

	case instruction.OpI32Store:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I32Store{MemArg: memArg}), nil

	case instruction.OpI64Store:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I64Store{MemArg: memArg}), nil

	case instruction.OpF32Store:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.F32Store{MemArg: memArg}), nil

	case instruction.OpF64Store:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.F64Store{MemArg: memArg}), nil

	case instruction.OpI32Store8:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I32Store8{MemArg: memArg}), nil

	case instruction.OpI32Store16:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I32Store16{MemArg: memArg}), nil

	case instruction.OpI64Store8:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I64Store8{MemArg: memArg}), nil

	case instruction.OpI64Store16:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I64Store16{MemArg: memArg}), nil

	case instruction.OpI64Store32:
		memArg, err := parseMemArg(r)
		if err != nil {
			return parseResult{}, err
		}
		return mem(instruction.I64Store32{MemArg: memArg}), nil

	case instruction.OpMemorySize:
		if err := readZeroByte(r); err != nil {
			return parseResult{}, err
		}
		return mem(instruction.MemorySize{}), nil

	case instruction.OpMemoryGrow:
		if err := readZeroByte(r); err != nil {
			return parseResult{}, err
		}
		return mem(instruction.MemoryGrow{}), nil

	// Numeric constants
	case instruction.OpI32Const:
		value, err := r.ReadI32()
		if err != nil {
			return parseResult{}, err
		}
		return num(instruction.I32Const{Value: value}), nil

	case instruction.OpI64Const:
		value, err := r.ReadI64()
		if err != nil {
			return parseResult{}, err
		}
		return num(instruction.I64Const{Value: value}), nil

	case instruction.OpF32Const:
		value, err := r.ReadF32()
		if err != nil {
			return parseResult{}, err
		}
		return num(instruction.F32Const{Value: value}), nil

	case instruction.OpF64Const:
		value, err := r.ReadF64()
		if err != nil {
			return parseResult{}, err
		}
		return num(instruction.F64Const{Value: value}), nil

	// i32 comparison
	case instruction.OpI32Eqz:
		return num(instruction.I32Eqz{}), nil
	case instruction.OpI32Eq:
		return num(instruction.I32Eq{}), nil
	case instruction.OpI32Ne:
		return num(instruction.I32Ne{}), nil
	case instruction.OpI32LtS:
		return num(instruction.I32LtS{}), nil
	case instruction.OpI32LtU:
		return num(instruction.I32LtU{}), nil
	case instruction.OpI32GtS:
		return num(instruction.I32GtS{}), nil
	case instruction.OpI32GtU:
		return num(instruction.I32GtU{}), nil
	case instruction.OpI32LeS:
		return num(instruction.I32LeS{}), nil
	case instruction.OpI32LeU:
		return num(instruction.I32LeU{}), nil
	case instruction.OpI32GeS:
		return num(instruction.I32GeS{}), nil
	case instruction.OpI32GeU:
		return num(instruction.I32GeU{}), nil

	// i64 comparison
	case instruction.OpI64Eqz:
		return num(instruction.I64Eqz{}), nil
	case instruction.OpI64Eq:
		return num(instruction.I64Eq{}), nil
	case instruction.OpI64Ne:
		return num(instruction.I64Ne{}), nil
	case instruction.OpI64LtS:
		return num(instruction.I64LtS{}), nil
	case instruction.OpI64LtU:
		return num(instruction.I64LtU{}), nil
	case instruction.OpI64GtS:
		return num(instruction.I64GtS{}), nil
	case instruction.OpI64GtU:
		return num(instruction.I64GtU{}), nil
	case instruction.OpI64LeS:
		return num(instruction.I64LeS{}), nil
	case instruction.OpI64LeU:
		return num(instruction.I64LeU{}), nil
	case instruction.OpI64GeS:
		return num(instruction.I64GeS{}), nil
	case instruction.OpI64GeU:
		return num(instruction.I64GeU{}), nil

	// f32 comparison
	case instruction.OpF32Eq:
		return num(instruction.F32Eq{}), nil
	case instruction.OpF32Ne:
		return num(instruction.F32Ne{}), nil
	case instruction.OpF32Lt:
		return num(instruction.F32Lt{}), nil
	case instruction.OpF32Gt:
		return num(instruction.F32Gt{}), nil
	case instruction.OpF32Le:
		return num(instruction.F32Le{}), nil
	case instruction.OpF32Ge:
		return num(instruction.F32Ge{}), nil

	// f64 comparison
	case instruction.OpF64Eq:
		return num(instruction.F64Eq{}), nil
	case instruction.OpF64Ne:
		return num(instruction.F64Ne{}), nil
	case instruction.OpF64Lt:
		return num(instruction.F64Lt{}), nil
	case instruction.OpF64Gt:
		return num(instruction.F64Gt{}), nil
	case instruction.OpF64Le:
		return num(instruction.F64Le{}), nil
	case instruction.OpF64Ge:
		return num(instruction.F64Ge{}), nil

	// i32 arithmetic
	case instruction.OpI32Clz:
		return num(instruction.I32Clz{}), nil
	case instruction.OpI32Ctz:
		return num(instruction.I32Ctz{}), nil
	case instruction.OpI32Popcnt:
		return num(instruction.I32Popcnt{}), nil
	case instruction.OpI32Add:
		return num(instruction.I32Add{}), nil
	case instruction.OpI32Sub:
		return num(instruction.I32Sub{}), nil
	case instruction.OpI32Mul:
		return num(instruction.I32Mul{}), nil
	case instruction.OpI32DivS:
		return num(instruction.I32DivS{}), nil
	case instruction.OpI32DivU:
		return num(instruction.I32DivU{}), nil
	case instruction.OpI32RemS:
		return num(instruction.I32RemS{}), nil
	case instruction.OpI32RemU:
		return num(instruction.I32RemU{}), nil
	case instruction.OpI32And:
		return num(instruction.I32And{}), nil
	case instruction.OpI32Or:
		return num(instruction.I32Or{}), nil
	case instruction.OpI32Xor:
		return num(instruction.I32Xor{}), nil
	case instruction.OpI32Shl:
		return num(instruction.I32Shl{}), nil
	case instruction.OpI32ShrS:
		return num(instruction.I32ShrS{}), nil
	case instruction.OpI32ShrU:
		return num(instruction.I32ShrU{}), nil
	case instruction.OpI32Rotl:
		return num(instruction.I32Rotl{}), nil
	case instruction.OpI32Rotr:
		return num(instruction.I32Rotr{}), nil

	// i64 arithmetic
	case instruction.OpI64Clz:
		return num(instruction.I64Clz{}), nil
	case instruction.OpI64Ctz:
		return num(instruction.I64Ctz{}), nil
	case instruction.OpI64Popcnt:
		return num(instruction.I64Popcnt{}), nil
	case instruction.OpI64Add:
		return num(instruction.I64Add{}), nil
	case instruction.OpI64Sub:
		return num(instruction.I64Sub{}), nil
	case instruction.OpI64Mul:
		return num(instruction.I64Mul{}), nil
	case instruction.OpI64DivS:
		return num(instruction.I64DivS{}), nil
	case instruction.OpI64DivU:
		return num(instruction.I64DivU{}), nil
	case instruction.OpI64RemS:
		return num(instruction.I64RemS{}), nil
	case instruction.OpI64RemU:
		return num(instruction.I64RemU{}), nil
	case instruction.OpI64And:
		return num(instruction.I64And{}), nil
	case instruction.OpI64Or:
		return num(instruction.I64Or{}), nil
	case instruction.OpI64Xor:
		return num(instruction.I64Xor{}), nil
	case instruction.OpI64Shl:
		return num(instruction.I64Shl{}), nil
	case instruction.OpI64ShrS:
		return num(instruction.I64ShrS{}), nil
	case instruction.OpI64ShrU:
		return num(instruction.I64ShrU{}), nil
	case instruction.OpI64Rotl:
		return num(instruction.I64Rotl{}), nil
	case instruction.OpI64Rotr:
		return num(instruction.I64Rotr{}), nil

	// f32 arithmetic
	case instruction.OpF32Abs:
		return num(instruction.F32Abs{}), nil
	case instruction.OpF32Neg:
		return num(instruction.F32Neg{}), nil
	case instruction.OpF32Ceil:
		return num(instruction.F32Ceil{}), nil
	case instruction.OpF32Floor:
		return num(instruction.F32Floor{}), nil
	case instruction.OpF32Trunc:
		return num(instruction.F32Trunc{}), nil
	case instruction.OpF32Nearest:
		return num(instruction.F32Nearest{}), nil
	case instruction.OpF32Sqrt:
		return num(instruction.F32Sqrt{}), nil
	case instruction.OpF32Add:
		return num(instruction.F32Add{}), nil
	case instruction.OpF32Sub:
		return num(instruction.F32Sub{}), nil
	case instruction.OpF32Mul:
		return num(instruction.F32Mul{}), nil
	case instruction.OpF32Div:
		return num(instruction.F32Div{}), nil
	case instruction.OpF32Min:
		return num(instruction.F32Min{}), nil
	case instruction.OpF32Max:
		return num(instruction.F32Max{}), nil
	case instruction.OpF32Copysign:
		return num(instruction.F32Copysign{}), nil

	// f64 arithmetic
	case instruction.OpF64Abs:
		return num(instruction.F64Abs{}), nil
	case instruction.OpF64Neg:
		return num(instruction.F64Neg{}), nil
	case instruction.OpF64Ceil:
		return num(instruction.F64Ceil{}), nil
	case instruction.OpF64Floor:
		return num(instruction.F64Floor{}), nil
	case instruction.OpF64Trunc:
		return num(instruction.F64Trunc{}), nil
	case instruction.OpF64Nearest:
		return num(instruction.F64Nearest{}), nil
	case instruction.OpF64Sqrt:
		return num(instruction.F64Sqrt{}), nil
	case instruction.OpF64Add:
		return num(instruction.F64Add{}), nil
	case instruction.OpF64Sub:
		return num(instruction.F64Sub{}), nil
	case instruction.OpF64Mul:
		return num(instruction.F64Mul{}), nil
	case instruction.OpF64Div:
		return num(instruction.F64Div{}), nil
	case instruction.OpF64Min:
		return num(instruction.F64Min{}), nil
	case instruction.OpF64Max:
		return num(instruction.F64Max{}), nil
	case instruction.OpF64Copysign:
		return num(instruction.F64Copysign{}), nil

	// Conversions
	case instruction.OpI32WrapI64:
		return num(instruction.I32WrapI64{}), nil
	case instruction.OpI32TruncF32S:
		return num(instruction.I32TruncF32S{}), nil
	case instruction.OpI32TruncF32U:
		return num(instruction.I32TruncF32U{}), nil
	case instruction.OpI32TruncF64S:
		return num(instruction.I32TruncF64S{}), nil
	case instruction.OpI32TruncF64U:
		return num(instruction.I32TruncF64U{}), nil
	case instruction.OpI64ExtendI32S:
		return num(instruction.I64ExtendI32S{}), nil
	case instruction.OpI64ExtendI32U:
		return num(instruction.I64ExtendI32U{}), nil
	case instruction.OpI64TruncF32S:
		return num(instruction.I64TruncF32S{}), nil
	case instruction.OpI64TruncF32U:
		return num(instruction.I64TruncF32U{}), nil
	case instruction.OpI64TruncF64S:
		return num(instruction.I64TruncF64S{}), nil
	case instruction.OpI64TruncF64U:
		return num(instruction.I64TruncF64U{}), nil
	case instruction.OpF32ConvertI32S:
		return num(instruction.F32ConvertI32S{}), nil
	case instruction.OpF32ConvertI32U:
		return num(instruction.F32ConvertI32U{}), nil
	case instruction.OpF32ConvertI64S:
		return num(instruction.F32ConvertI64S{}), nil
	case instruction.OpF32ConvertI64U:
		return num(instruction.F32ConvertI64U{}), nil
	case instruction.OpF32DemoteF64:
		return num(instruction.F32DemoteF64{}), nil
	case instruction.OpF64ConvertI32S:
		return num(instruction.F64ConvertI32S{}), nil
	case instruction.OpF64ConvertI32U:
		return num(instruction.F64ConvertI32U{}), nil
	case instruction.OpF64ConvertI64S:
		return num(instruction.F64ConvertI64S{}), nil
	case instruction.OpF64ConvertI64U:
		return num(instruction.F64ConvertI64U{}), nil
	case instruction.OpF64PromoteF32:
		return num(instruction.F64PromoteF32{}), nil
	case instruction.OpI32ReinterpretF32:
		return num(instruction.I32ReinterpretF32{}), nil
	case instruction.OpI64ReinterpretF64:
		return num(instruction.I64ReinterpretF64{}), nil
	case instruction.OpF32ReinterpretI32:
		return num(instruction.F32ReinterpretI32{}), nil
	case instruction.OpF64ReinterpretI64:
		return num(instruction.F64ReinterpretI64{}), nil

	// Sign extension
	case instruction.OpI32Extend8S:
		return num(instruction.I32Extend8S{}), nil
	case instruction.OpI32Extend16S:
		return num(instruction.I32Extend16S{}), nil
	case instruction.OpI64Extend8S:
		return num(instruction.I64Extend8S{}), nil
	case instruction.OpI64Extend16S:
		return num(instruction.I64Extend16S{}), nil
	case instruction.OpI64Extend32S:
		return num(instruction.I64Extend32S{}), nil

	// Extended instructions (0xFC prefix)
	case instruction.OpPrefix:
		return parseExtendedInstruction(r)

	default:
		return parseResult{}, fmt.Errorf("%w: 0x%02X", ErrInvalidOpcode, op)
	}
}

func parseExtendedInstruction(r *Reader) (parseResult, error) {
	extOp, err := r.ReadU32()
	if err != nil {
		return parseResult{}, err
	}

	switch instruction.ExtOpcode(extOp) {
	case instruction.ExtOpI32TruncSatF32S:
		return num(instruction.I32TruncSatF32S{}), nil
	case instruction.ExtOpI32TruncSatF32U:
		return num(instruction.I32TruncSatF32U{}), nil
	case instruction.ExtOpI32TruncSatF64S:
		return num(instruction.I32TruncSatF64S{}), nil
	case instruction.ExtOpI32TruncSatF64U:
		return num(instruction.I32TruncSatF64U{}), nil
	case instruction.ExtOpI64TruncSatF32S:
		return num(instruction.I64TruncSatF32S{}), nil
	case instruction.ExtOpI64TruncSatF32U:
		return num(instruction.I64TruncSatF32U{}), nil
	case instruction.ExtOpI64TruncSatF64S:
		return num(instruction.I64TruncSatF64S{}), nil
	case instruction.ExtOpI64TruncSatF64U:
		return num(instruction.I64TruncSatF64U{}), nil

	case instruction.ExtOpMemoryInit:
		dataIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		if err := readZeroByte(r); err != nil {
			return parseResult{}, err
		}
		r.usesDataIdx = true
		return mem(instruction.MemoryInit{DataIdx: dataIdx}), nil

	case instruction.ExtOpDataDrop:
		dataIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		r.usesDataIdx = true
		return mem(instruction.DataDrop{DataIdx: dataIdx}), nil

	case instruction.ExtOpMemoryCopy:
		if err := readZeroByte(r); err != nil {
			return parseResult{}, err
		}
		if err := readZeroByte(r); err != nil {
			return parseResult{}, err
		}
		return mem(instruction.MemoryCopy{}), nil

	case instruction.ExtOpMemoryFill:
		if err := readZeroByte(r); err != nil {
			return parseResult{}, err
		}
		return mem(instruction.MemoryFill{}), nil

	case instruction.ExtOpTableInit:
		elemIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		tableIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return variable(instruction.TableInit{ElemIdx: elemIdx, TableIdx: tableIdx}), nil

	case instruction.ExtOpElemDrop:
		elemIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return variable(instruction.ElemDrop{ElemIdx: elemIdx}), nil

	case instruction.ExtOpTableCopy:
		dstTableIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		srcTableIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return variable(instruction.TableCopy{DstTableIdx: dstTableIdx, SrcTableIdx: srcTableIdx}), nil

	case instruction.ExtOpTableGrow:
		tableIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return variable(instruction.TableGrow{TableIdx: tableIdx}), nil

	case instruction.ExtOpTableSize:
		tableIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return variable(instruction.TableSize{TableIdx: tableIdx}), nil

	case instruction.ExtOpTableFill:
		tableIdx, err := r.ReadU32()
		if err != nil {
			return parseResult{}, err
		}
		return variable(instruction.TableFill{TableIdx: tableIdx}), nil

	default:
		return parseResult{}, fmt.Errorf("%w: 0xFC 0x%02X", ErrInvalidOpcode, extOp)
	}
}

func parseBlockType(r *Reader) (types.BlockType, error) {
	b, err := r.PeekByte()
	if err != nil {
		return types.BlockType{}, err
	}

	// Empty block type
	if b == 0x40 {
		_, _ = r.ReadByte()
		return types.BlockType{Kind: types.BlockTypeEmpty}, nil
	}

	// Value type (single result)
	if b >= 0x7B && b <= 0x7F || b == 0x70 || b == 0x6F {
		_, _ = r.ReadByte()
		return types.BlockType{
			Kind:    types.BlockTypeValue,
			ValType: types.ValueType(b),
		}, nil
	}

	// Type index (s33 encoded)
	typeIdx, err := r.ReadS33()
	if err != nil {
		return types.BlockType{}, err
	}
	if typeIdx < 0 || typeIdx > math.MaxUint32 {
		return types.BlockType{}, ErrInvalidBlockType
	}

	return types.BlockType{
		Kind:    types.BlockTypeIndex,
		TypeIdx: uint32(typeIdx),
	}, nil
}

func parseBlockBody(r *Reader) ([]instruction.Instruction, error) {
	var body []instruction.Instruction

	for {
		result, err := parseInstructionInternal(r)
		if err != nil {
			return nil, err
		}

		if result.isEnd {
			break
		}
		if result.isElse {
			return nil, fmt.Errorf("%w: else outside if", ErrInvalidOpcode)
		}

		body = append(body, result.instr)
	}

	return body, nil
}

func parseIfBody(r *Reader) (thenBody, elseBody []instruction.Instruction, err error) {
	for {
		result, err := parseInstructionInternal(r)
		if err != nil {
			return nil, nil, err
		}

		if result.isEnd {
			break
		}

		if result.isElse {
			// Parse else branch
			elseBody, err = parseBlockBody(r)
			if err != nil {
				return nil, nil, err
			}
			break
		}

		thenBody = append(thenBody, result.instr)
	}

	return thenBody, elseBody, nil
}

func parseMemArg(r *Reader) (instruction.MemArg, error) {
	align, err := r.ReadU32()
	if err != nil {
		return instruction.MemArg{}, err
	}
	if align >= 32 {
		return instruction.MemArg{}, fmt.Errorf("%w: malformed memop flags", ErrInvalidOpcode)
	}
	offset, err := r.ReadU32()
	if err != nil {
		return instruction.MemArg{}, err
	}
	return instruction.MemArg{Align: align, Offset: offset}, nil
}

// readZeroByte reads the reserved memory-index byte, which is 0 in a
// single-memory module.
func readZeroByte(r *Reader) error {
	b, err := r.ReadByte()
	if err != nil {
		return err
	}
	if b != 0 {
		return fmt.Errorf("%w: zero byte expected", ErrInvalidOpcode)
	}
	return nil
}

// readRefType reads funcref or externref.
func readRefType(r *Reader) (types.ValueType, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	if t := types.ValueType(b); t == types.FuncRef || t == types.ExternRef {
		return t, nil
	}
	return 0, fmt.Errorf("%w: malformed reference type 0x%02x", ErrInvalidType, b)
}

// Wrapper helpers for category types
func ctrl(instr instruction.ControlInstr) parseResult {
	return parseResult{instr: instruction.Control{Instr: instr}}
}

func ref(instr instruction.ReferenceInstr) parseResult {
	return parseResult{instr: instruction.Reference{Instr: instr}}
}

func param(instr instruction.ParametricInstr) parseResult {
	return parseResult{instr: instruction.Parametric{Instr: instr}}
}

func variable(instr instruction.VariableInstr) parseResult {
	return parseResult{instr: instruction.Variable{Instr: instr}}
}

func mem(instr instruction.MemoryInstr) parseResult {
	return parseResult{instr: instruction.Memory{Instr: instr}}
}

func num(instr instruction.NumericInstr) parseResult {
	return parseResult{instr: instruction.Numeric{Instr: instr}}
}
