package parser

import (
	"fmt"
	"math"

	"github.com/t14raptor/go-fwasm/instruction"
	"github.com/t14raptor/go-fwasm/module"
	"github.com/t14raptor/go-fwasm/types"
)

func Parse(data []byte) (*module.Module, error) {
	r := NewReader(data)

	magic, err := readU32LE(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read magic number: %w", err)
	}
	if magic != module.Magic {
		return nil, ErrInvalidMagic
	}

	version, err := readU32LE(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read version: %w", err)
	}
	if version != module.Version {
		return nil, ErrInvalidVersion
	}

	mod := &module.Module{}

	// Read sections. Non-custom sections appear at most once, in order.
	lastRank := 0
	for !r.EOF() {
		sectionID, err := r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("failed to read section ID: %w", err)
		}
		if sectionID > byte(types.SectionDataCount) {
			return nil, fmt.Errorf("%w: malformed section id %d", ErrInvalidSection, sectionID)
		}
		if rank := sectionRank[sectionID]; rank != 0 {
			if rank <= lastRank {
				return nil, fmt.Errorf("%w: section %s", ErrSectionOrder, types.SectionID(sectionID))
			}
			lastRank = rank
		}

		sectionSize, err := r.ReadU32()
		if err != nil {
			return nil, fmt.Errorf("failed to read section size: %w", err)
		}

		sectionReader, err := r.Slice(int(sectionSize))
		if err != nil {
			return nil, fmt.Errorf("failed to read section data: %w", err)
		}

		if err := parseSection(mod, types.SectionID(sectionID), sectionReader); err != nil {
			return nil, fmt.Errorf("failed to parse section %s: %w", types.SectionID(sectionID), err)
		}
		if !sectionReader.EOF() {
			return nil, fmt.Errorf("%w: section %s", ErrSectionSize, types.SectionID(sectionID))
		}
		if sectionReader.usesDataIdx && mod.DataCount == nil {
			return nil, fmt.Errorf("%w: data count section required", ErrMalformedSection)
		}
	}

	if len(mod.Functions) != len(mod.Codes) {
		return nil, fmt.Errorf("%w: function and code section have inconsistent lengths", ErrMalformedSection)
	}
	if mod.DataCount != nil && int(*mod.DataCount) != len(mod.Datas) {
		return nil, fmt.Errorf("%w: data count and data section have inconsistent lengths", ErrMalformedSection)
	}
	return mod, nil
}

// sectionRank orders the non-custom sections. The data count section sits
// between the element and code sections.
var sectionRank = [...]int{
	types.SectionType:      1,
	types.SectionImport:    2,
	types.SectionFunction:  3,
	types.SectionTable:     4,
	types.SectionMemory:    5,
	types.SectionGlobal:    6,
	types.SectionExport:    7,
	types.SectionStart:     8,
	types.SectionElement:   9,
	types.SectionDataCount: 10,
	types.SectionCode:      11,
	types.SectionData:      12,
}

func readU32LE(r *Reader) (uint32, error) {
	b, err := r.ReadBytes(4)
	if err != nil {
		return 0, err
	}
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24, nil
}

func parseSection(mod *module.Module, id types.SectionID, r *Reader) error {
	switch id {
	case types.SectionCustom:
		return parseCustomSection(mod, r)
	case types.SectionType:
		return parseTypeSection(mod, r)
	case types.SectionImport:
		return parseImportSection(mod, r)
	case types.SectionFunction:
		return parseFunctionSection(mod, r)
	case types.SectionTable:
		return parseTableSection(mod, r)
	case types.SectionMemory:
		return parseMemorySection(mod, r)
	case types.SectionGlobal:
		return parseGlobalSection(mod, r)
	case types.SectionExport:
		return parseExportSection(mod, r)
	case types.SectionStart:
		return parseStartSection(mod, r)
	case types.SectionElement:
		return parseElementSection(mod, r)
	case types.SectionCode:
		return parseCodeSection(mod, r)
	case types.SectionData:
		return parseDataSection(mod, r)
	case types.SectionDataCount:
		return parseDataCountSection(mod, r)
	default:
		// Skip unknown sections
		return nil
	}
}

func parseCustomSection(mod *module.Module, r *Reader) error {
	name, err := r.ReadName()
	if err != nil {
		return err
	}

	// Read remaining bytes as data
	remaining := r.Remaining()
	data, err := r.ReadBytes(remaining)
	if err != nil {
		return err
	}

	mod.Customs = append(mod.Customs, module.Custom{
		Name: name,
		Data: data,
	})
	return nil
}

func parseTypeSection(mod *module.Module, r *Reader) error {
	count, err := r.ReadU32()
	if err != nil {
		return err
	}

	mod.Types = make([]types.FuncType, count)
	for i := uint32(0); i < count; i++ {
		funcType, err := parseFuncType(r)
		if err != nil {
			return err
		}
		mod.Types[i] = funcType
	}
	return nil
}

func parseFuncType(r *Reader) (types.FuncType, error) {
	// Function type starts with 0x60
	marker, err := r.ReadByte()
	if err != nil {
		return types.FuncType{}, err
	}
	if marker != 0x60 {
		return types.FuncType{}, ErrInvalidType
	}

	params, err := parseValueTypes(r)
	if err != nil {
		return types.FuncType{}, err
	}

	results, err := parseValueTypes(r)
	if err != nil {
		return types.FuncType{}, err
	}

	return types.FuncType{Params: params, Results: results}, nil
}

func parseValueTypes(r *Reader) ([]types.ValueType, error) {
	count, err := r.ReadU32()
	if err != nil {
		return nil, err
	}

	result := make([]types.ValueType, count)
	for i := uint32(0); i < count; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		result[i] = types.ValueType(b)
	}
	return result, nil
}

func parseImportSection(mod *module.Module, r *Reader) error {
	count, err := r.ReadU32()
	if err != nil {
		return err
	}

	mod.Imports = make([]types.Import, count)
	for i := uint32(0); i < count; i++ {
		imp, err := parseImport(r)
		if err != nil {
			return err
		}
		mod.Imports[i] = imp
	}
	return nil
}

func parseImport(r *Reader) (types.Import, error) {
	moduleName, err := r.ReadName()
	if err != nil {
		return types.Import{}, err
	}

	name, err := r.ReadName()
	if err != nil {
		return types.Import{}, err
	}

	kind, err := r.ReadByte()
	if err != nil {
		return types.Import{}, err
	}

	var desc types.ImportDesc
	desc.Kind = types.ImportDescKind(kind)

	switch desc.Kind {
	case types.ImportFunc:
		typeIdx, err := r.ReadU32()
		if err != nil {
			return types.Import{}, err
		}
		desc.TypeIdx = typeIdx
	case types.ImportTable:
		tableType, err := parseTableType(r)
		if err != nil {
			return types.Import{}, err
		}
		desc.TableType = tableType
	case types.ImportMem:
		memType, err := parseMemoryType(r)
		if err != nil {
			return types.Import{}, err
		}
		desc.MemType = memType
	case types.ImportGlobal:
		globalType, err := parseGlobalType(r)
		if err != nil {
			return types.Import{}, err
		}
		desc.GlobalType = globalType
	default:
		return types.Import{}, ErrInvalidType
	}

	return types.Import{
		Module: moduleName,
		Name:   name,
		Desc:   desc,
	}, nil
}

func parseTableType(r *Reader) (types.TableType, error) {
	elemType, err := r.ReadByte()
	if err != nil {
		return types.TableType{}, err
	}

	limits, err := parseLimits(r)
	if err != nil {
		return types.TableType{}, err
	}

	return types.TableType{
		ElemType: types.ValueType(elemType),
		Limits:   limits,
	}, nil
}

func parseMemoryType(r *Reader) (types.MemoryType, error) {
	limits, err := parseLimits(r)
	if err != nil {
		return types.MemoryType{}, err
	}
	return types.MemoryType{Limits: limits}, nil
}

func parseLimits(r *Reader) (types.Limits, error) {
	flags, err := r.ReadByte()
	if err != nil {
		return types.Limits{}, err
	}

	min, err := r.ReadU32()
	if err != nil {
		return types.Limits{}, err
	}

	if flags > 1 {
		return types.Limits{}, fmt.Errorf("%w: limits flag 0x%02x", ErrIntegerOverflow, flags)
	}
	limits := types.Limits{Min: min, HasMax: flags&0x01 != 0}
	if limits.HasMax {
		max, err := r.ReadU32()
		if err != nil {
			return types.Limits{}, err
		}
		limits.Max = max
	}
	return limits, nil
}

func parseGlobalType(r *Reader) (types.GlobalType, error) {
	valType, err := r.ReadByte()
	if err != nil {
		return types.GlobalType{}, err
	}

	mut, err := r.ReadByte()
	if err != nil {
		return types.GlobalType{}, err
	}
	if mut > 1 {
		return types.GlobalType{}, fmt.Errorf("%w: malformed mutability 0x%02x", ErrInvalidType, mut)
	}

	return types.GlobalType{
		ValType: types.ValueType(valType),
		Mutable: mut == 0x01,
	}, nil
}

func parseFunctionSection(mod *module.Module, r *Reader) error {
	count, err := r.ReadU32()
	if err != nil {
		return err
	}

	mod.Functions = make([]uint32, count)
	for i := uint32(0); i < count; i++ {
		typeIdx, err := r.ReadU32()
		if err != nil {
			return err
		}
		mod.Functions[i] = typeIdx
	}
	return nil
}

func parseTableSection(mod *module.Module, r *Reader) error {
	count, err := r.ReadU32()
	if err != nil {
		return err
	}

	mod.Tables = make([]types.TableType, count)
	for i := uint32(0); i < count; i++ {
		table, err := parseTableType(r)
		if err != nil {
			return err
		}
		mod.Tables[i] = table
	}
	return nil
}

func parseMemorySection(mod *module.Module, r *Reader) error {
	count, err := r.ReadU32()
	if err != nil {
		return err
	}

	mod.Memories = make([]types.MemoryType, count)
	for i := uint32(0); i < count; i++ {
		mem, err := parseMemoryType(r)
		if err != nil {
			return err
		}
		mod.Memories[i] = mem
	}
	return nil
}

func parseGlobalSection(mod *module.Module, r *Reader) error {
	count, err := r.ReadU32()
	if err != nil {
		return err
	}

	mod.Globals = make([]module.Global, count)
	for i := uint32(0); i < count; i++ {
		globalType, err := parseGlobalType(r)
		if err != nil {
			return err
		}

		init, err := parseExpression(r)
		if err != nil {
			return err
		}

		mod.Globals[i] = module.Global{
			Type: globalType,
			Init: init,
		}
	}
	return nil
}

func parseExportSection(mod *module.Module, r *Reader) error {
	count, err := r.ReadU32()
	if err != nil {
		return err
	}

	mod.Exports = make([]types.Export, count)
	for i := uint32(0); i < count; i++ {
		name, err := r.ReadName()
		if err != nil {
			return err
		}

		kind, err := r.ReadByte()
		if err != nil {
			return err
		}

		idx, err := r.ReadU32()
		if err != nil {
			return err
		}

		mod.Exports[i] = types.Export{
			Name: name,
			Desc: types.ExportDesc{
				Kind: types.ExportDescKind(kind),
				Idx:  idx,
			},
		}
	}
	return nil
}

func parseStartSection(mod *module.Module, r *Reader) error {
	funcIdx, err := r.ReadU32()
	if err != nil {
		return err
	}
	mod.Start = &funcIdx
	return nil
}

func parseElementSection(mod *module.Module, r *Reader) error {
	count, err := r.ReadU32()
	if err != nil {
		return err
	}

	mod.Elements = make([]module.Element, count)
	for i := uint32(0); i < count; i++ {
		elem, err := parseElement(r)
		if err != nil {
			return err
		}
		mod.Elements[i] = elem
	}
	return nil
}

func parseElement(r *Reader) (module.Element, error) {
	flags, err := r.ReadU32()
	if err != nil {
		return module.Element{}, err
	}

	elem := module.Element{
		Type: types.FuncRef, // Default type
	}

	// Determine mode and parse accordingly based on flags
	switch flags {
	case 0:
		// Active, funcref, table 0, expr offset, vec of funcidx
		elem.Mode = module.ElementModeActive
		elem.TableIdx = 0
		offset, err := parseExpression(r)
		if err != nil {
			return module.Element{}, err
		}
		elem.Offset = offset
		funcIdxs, err := parseFuncIdxVec(r)
		if err != nil {
			return module.Element{}, err
		}
		elem.Init = funcIdxsToExprs(funcIdxs)

	case 1:
		// Passive, elemkind, vec of funcidx
		elem.Mode = module.ElementModePassive
		if err := readElemKind(r); err != nil {
			return module.Element{}, err
		}
		funcIdxs, err := parseFuncIdxVec(r)
		if err != nil {
			return module.Element{}, err
		}
		elem.Init = funcIdxsToExprs(funcIdxs)

	case 2:
		// Active, table idx, expr offset, elemkind, vec of funcidx
		elem.Mode = module.ElementModeActive
		tableIdx, err := r.ReadU32()
		if err != nil {
			return module.Element{}, err
		}
		elem.TableIdx = tableIdx
		offset, err := parseExpression(r)
		if err != nil {
			return module.Element{}, err
		}
		elem.Offset = offset
		if err := readElemKind(r); err != nil {
			return module.Element{}, err
		}
		funcIdxs, err := parseFuncIdxVec(r)
		if err != nil {
			return module.Element{}, err
		}
		elem.Init = funcIdxsToExprs(funcIdxs)

	case 3:
		// Declarative, elemkind, vec of funcidx
		elem.Mode = module.ElementModeDeclarative
		if err := readElemKind(r); err != nil {
			return module.Element{}, err
		}
		funcIdxs, err := parseFuncIdxVec(r)
		if err != nil {
			return module.Element{}, err
		}
		elem.Init = funcIdxsToExprs(funcIdxs)

	case 4:
		// Active, funcref, table 0, expr offset, vec of expr
		elem.Mode = module.ElementModeActive
		elem.TableIdx = 0
		offset, err := parseExpression(r)
		if err != nil {
			return module.Element{}, err
		}
		elem.Offset = offset
		exprs, err := parseExprVec(r)
		if err != nil {
			return module.Element{}, err
		}
		elem.Init = exprs

	case 5:
		// Passive, reftype, vec of expr
		elem.Mode = module.ElementModePassive
		refType, err := readRefType(r)
		if err != nil {
			return module.Element{}, err
		}
		elem.Type = refType
		exprs, err := parseExprVec(r)
		if err != nil {
			return module.Element{}, err
		}
		elem.Init = exprs

	case 6:
		// Active, table idx, expr offset, reftype, vec of expr
		elem.Mode = module.ElementModeActive
		tableIdx, err := r.ReadU32()
		if err != nil {
			return module.Element{}, err
		}
		elem.TableIdx = tableIdx
		offset, err := parseExpression(r)
		if err != nil {
			return module.Element{}, err
		}
		elem.Offset = offset
		refType, err := readRefType(r)
		if err != nil {
			return module.Element{}, err
		}
		elem.Type = refType
		exprs, err := parseExprVec(r)
		if err != nil {
			return module.Element{}, err
		}
		elem.Init = exprs

	case 7:
		// Declarative, reftype, vec of expr
		elem.Mode = module.ElementModeDeclarative
		refType, err := readRefType(r)
		if err != nil {
			return module.Element{}, err
		}
		elem.Type = refType
		exprs, err := parseExprVec(r)
		if err != nil {
			return module.Element{}, err
		}
		elem.Init = exprs

	default:
		return module.Element{}, fmt.Errorf("unknown element flags: %d", flags)
	}

	return elem, nil
}

// readElemKind reads an element kind, of which 0x00 (funcref) is the only one.
func readElemKind(r *Reader) error {
	kind, err := r.ReadByte()
	if err != nil {
		return err
	}
	if kind != 0x00 {
		return fmt.Errorf("%w: malformed element kind 0x%02x", ErrInvalidType, kind)
	}
	return nil
}

func parseFuncIdxVec(r *Reader) ([]uint32, error) {
	count, err := r.ReadU32()
	if err != nil {
		return nil, err
	}

	result := make([]uint32, count)
	for i := uint32(0); i < count; i++ {
		idx, err := r.ReadU32()
		if err != nil {
			return nil, err
		}
		result[i] = idx
	}
	return result, nil
}

func parseExprVec(r *Reader) ([][]instruction.Instruction, error) {
	count, err := r.ReadU32()
	if err != nil {
		return nil, err
	}

	result := make([][]instruction.Instruction, count)
	for i := uint32(0); i < count; i++ {
		expr, err := parseExpression(r)
		if err != nil {
			return nil, err
		}
		result[i] = expr
	}
	return result, nil
}

func funcIdxsToExprs(funcIdxs []uint32) [][]instruction.Instruction {
	result := make([][]instruction.Instruction, len(funcIdxs))
	for i, idx := range funcIdxs {
		result[i] = []instruction.Instruction{instruction.Reference{Instr: instruction.RefFunc{FuncIdx: idx}}}
	}
	return result
}

func parseCodeSection(mod *module.Module, r *Reader) error {
	count, err := r.ReadU32()
	if err != nil {
		return err
	}

	mod.Codes = make([]module.Code, count)
	for i := uint32(0); i < count; i++ {
		code, err := parseCode(r)
		if err != nil {
			return err
		}
		mod.Codes[i] = code
	}
	return nil
}

func parseCode(r *Reader) (module.Code, error) {
	// Read code size
	size, err := r.ReadU32()
	if err != nil {
		return module.Code{}, err
	}

	codeReader, err := r.Slice(int(size))
	if err != nil {
		return module.Code{}, err
	}

	// Read locals
	localCount, err := codeReader.ReadU32()
	if err != nil {
		return module.Code{}, err
	}

	locals := make([]module.Local, localCount)
	var total uint64
	for i := uint32(0); i < localCount; i++ {
		count, err := codeReader.ReadU32()
		if err != nil {
			return module.Code{}, err
		}
		if total += uint64(count); total > math.MaxUint32 {
			return module.Code{}, fmt.Errorf("%w: too many locals", ErrMalformedSection)
		}
		valType, err := codeReader.ReadByte()
		if err != nil {
			return module.Code{}, err
		}
		locals[i] = module.Local{
			Count: count,
			Type:  types.ValueType(valType),
		}
	}

	// The body is one expression whose end is the last byte.
	body, err := parseExpression(codeReader)
	if err != nil {
		return module.Code{}, err
	}
	if !codeReader.EOF() {
		return module.Code{}, ErrEndExpected
	}
	r.usesDataIdx = r.usesDataIdx || codeReader.usesDataIdx

	return module.Code{
		Locals: locals,
		Body:   body,
	}, nil
}

func parseDataSection(mod *module.Module, r *Reader) error {
	count, err := r.ReadU32()
	if err != nil {
		return err
	}

	mod.Datas = make([]module.Data, count)
	for i := uint32(0); i < count; i++ {
		data, err := parseData(r)
		if err != nil {
			return err
		}
		mod.Datas[i] = data
	}
	return nil
}

func parseData(r *Reader) (module.Data, error) {
	flags, err := r.ReadU32()
	if err != nil {
		return module.Data{}, err
	}

	data := module.Data{}

	switch flags {
	case 0:
		// Active, memory 0
		data.Mode = module.DataModeActive
		data.MemIdx = 0
		offset, err := parseExpression(r)
		if err != nil {
			return module.Data{}, err
		}
		data.Offset = offset

	case 1:
		// Passive
		data.Mode = module.DataModePassive

	case 2:
		// Active, explicit memory index
		data.Mode = module.DataModeActive
		memIdx, err := r.ReadU32()
		if err != nil {
			return module.Data{}, err
		}
		data.MemIdx = memIdx
		offset, err := parseExpression(r)
		if err != nil {
			return module.Data{}, err
		}
		data.Offset = offset

	default:
		return module.Data{}, fmt.Errorf("unknown data flags: %d", flags)
	}

	// Read init bytes
	initLen, err := r.ReadU32()
	if err != nil {
		return module.Data{}, err
	}
	init, err := r.ReadBytes(int(initLen))
	if err != nil {
		return module.Data{}, err
	}
	data.Init = init

	return data, nil
}

func parseDataCountSection(mod *module.Module, r *Reader) error {
	count, err := r.ReadU32()
	if err != nil {
		return err
	}
	mod.DataCount = &count
	return nil
}
