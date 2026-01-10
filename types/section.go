package types

type SectionID byte

const (
	SectionCustom    SectionID = 0
	SectionType      SectionID = 1
	SectionImport    SectionID = 2
	SectionFunction  SectionID = 3
	SectionTable     SectionID = 4
	SectionMemory    SectionID = 5
	SectionGlobal    SectionID = 6
	SectionExport    SectionID = 7
	SectionStart     SectionID = 8
	SectionElement   SectionID = 9
	SectionCode      SectionID = 10
	SectionData      SectionID = 11
	SectionDataCount SectionID = 12
)

func (s SectionID) String() string {
	switch s {
	case SectionCustom:
		return "custom"
	case SectionType:
		return "type"
	case SectionImport:
		return "import"
	case SectionFunction:
		return "function"
	case SectionTable:
		return "table"
	case SectionMemory:
		return "memory"
	case SectionGlobal:
		return "global"
	case SectionExport:
		return "export"
	case SectionStart:
		return "start"
	case SectionElement:
		return "element"
	case SectionCode:
		return "code"
	case SectionData:
		return "data"
	case SectionDataCount:
		return "datacount"
	default:
		return "unknown"
	}
}
