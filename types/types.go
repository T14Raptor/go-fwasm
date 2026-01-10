package types

type ValueType byte

const (
	I32       ValueType = 0x7F
	I64       ValueType = 0x7E
	F32       ValueType = 0x7D
	F64       ValueType = 0x7C
	V128      ValueType = 0x7B
	FuncRef   ValueType = 0x70
	ExternRef ValueType = 0x6F
)

func (v ValueType) String() string {
	switch v {
	case I32:
		return "i32"
	case I64:
		return "i64"
	case F32:
		return "f32"
	case F64:
		return "f64"
	case V128:
		return "v128"
	case FuncRef:
		return "funcref"
	case ExternRef:
		return "externref"
	default:
		return "unknown"
	}
}

type BlockType struct {
	Kind    BlockTypeKind
	ValType ValueType
	TypeIdx uint32
}

type BlockTypeKind byte

const (
	BlockTypeEmpty BlockTypeKind = iota
	BlockTypeValue
	BlockTypeIndex
)

type FuncType struct {
	Params  []ValueType
	Results []ValueType
}

type Limits struct {
	Min    uint32
	Max    uint32
	HasMax bool
}

type MemoryType struct {
	Limits Limits
}

type TableType struct {
	ElemType ValueType
	Limits   Limits
}

type GlobalType struct {
	ValType ValueType
	Mutable bool
}

type Import struct {
	Module string
	Name   string
	Desc   ImportDesc
}

type ImportDescKind byte

const (
	ImportFunc   ImportDescKind = 0x00
	ImportTable  ImportDescKind = 0x01
	ImportMem    ImportDescKind = 0x02
	ImportGlobal ImportDescKind = 0x03
)

type ImportDesc struct {
	Kind       ImportDescKind
	TypeIdx    uint32
	TableType  TableType
	MemType    MemoryType
	GlobalType GlobalType
}

type Export struct {
	Name string
	Desc ExportDesc
}

type ExportDescKind byte

const (
	ExportFunc   ExportDescKind = 0x00
	ExportTable  ExportDescKind = 0x01
	ExportMem    ExportDescKind = 0x02
	ExportGlobal ExportDescKind = 0x03
)

type ExportDesc struct {
	Kind ExportDescKind
	Idx  uint32
}

type Global struct {
	Type GlobalType
	Init []byte
}

type Element struct {
	Type     ValueType
	Init     []uint32
	Mode     ElemMode
	TableIdx uint32
	Offset   []byte
}

type ElemMode byte

const (
	ElemModePassive     ElemMode = 0
	ElemModeActive      ElemMode = 1
	ElemModeDeclarative ElemMode = 2
)

type Data struct {
	Init   []byte
	Mode   DataMode
	MemIdx uint32
	Offset []byte
}

type DataMode byte

const (
	DataModePassive DataMode = 0
	DataModeActive  DataMode = 1
)

type Local struct {
	Count uint32
	Type  ValueType
}

type Code struct {
	Locals []Local
	Body   []byte
}
