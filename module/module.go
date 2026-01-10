package module

import (
	"github.com/t14raptor/go-fwasm/instruction"
	"github.com/t14raptor/go-fwasm/types"
)

const (
	Magic   uint32 = 0x6D736100 // \0asm
	Version uint32 = 1
)

type Module struct {
	Customs   []Custom
	Types     []types.FuncType
	Imports   []types.Import
	Functions []uint32
	Tables    []types.TableType
	Memories  []types.MemoryType
	Globals   []Global
	Exports   []types.Export
	Start     *uint32
	Elements  []Element
	DataCount *uint32
	Codes     []Code
	Datas     []Data
}

type Custom struct {
	Name string
	Data []byte
}

type Global struct {
	Type types.GlobalType
	Init []instruction.Instruction
}

type Element struct {
	Type     types.ValueType
	Init     [][]instruction.Instruction
	Mode     ElementMode
	TableIdx uint32
	Offset   []instruction.Instruction
}

type ElementMode byte

const (
	ElementModePassive     ElementMode = 0
	ElementModeActive      ElementMode = 1
	ElementModeDeclarative ElementMode = 2
)

type Code struct {
	Locals []Local
	Body   []instruction.Instruction
}

type Local struct {
	Count uint32
	Type  types.ValueType
}

type Data struct {
	Init   []byte
	Mode   DataMode
	MemIdx uint32
	Offset []instruction.Instruction
}

type DataMode byte

const (
	DataModePassive DataMode = 0
	DataModeActive  DataMode = 1
)

func (m *Module) NumImportedFunctions() int {
	count := 0
	for _, imp := range m.Imports {
		if imp.Desc.Kind == types.ImportFunc {
			count++
		}
	}
	return count
}

func (m *Module) NumImportedTables() int {
	count := 0
	for _, imp := range m.Imports {
		if imp.Desc.Kind == types.ImportTable {
			count++
		}
	}
	return count
}

func (m *Module) NumImportedMemories() int {
	count := 0
	for _, imp := range m.Imports {
		if imp.Desc.Kind == types.ImportMem {
			count++
		}
	}
	return count
}

func (m *Module) NumImportedGlobals() int {
	count := 0
	for _, imp := range m.Imports {
		if imp.Desc.Kind == types.ImportGlobal {
			count++
		}
	}
	return count
}

func (m *Module) GetFunctionType(funcIdx uint32) *types.FuncType {
	numImported := uint32(m.NumImportedFunctions())
	if funcIdx < numImported {
		importIdx := uint32(0)
		for _, imp := range m.Imports {
			if imp.Desc.Kind == types.ImportFunc {
				if importIdx == funcIdx {
					if imp.Desc.TypeIdx < uint32(len(m.Types)) {
						return &m.Types[imp.Desc.TypeIdx]
					}
					return nil
				}
				importIdx++
			}
		}
		return nil
	}

	localIdx := funcIdx - numImported
	if localIdx < uint32(len(m.Functions)) {
		typeIdx := m.Functions[localIdx]
		if typeIdx < uint32(len(m.Types)) {
			return &m.Types[typeIdx]
		}
	}
	return nil
}
