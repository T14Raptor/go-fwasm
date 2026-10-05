package parser

import "errors"

var (
	ErrUnexpectedEOF    = errors.New("unexpected end of data")
	ErrIntegerOverflow  = errors.New("integer too large")
	ErrIntegerTooLong   = errors.New("integer representation too long")
	ErrMalformedUTF8    = errors.New("malformed UTF-8 encoding")
	ErrSectionOrder     = errors.New("unexpected content after last section")
	ErrSectionSize      = errors.New("section size mismatch")
	ErrEndExpected      = errors.New("END opcode expected")
	ErrInvalidMagic     = errors.New("invalid WASM magic number")
	ErrInvalidVersion   = errors.New("unsupported WASM version")
	ErrInvalidSection   = errors.New("invalid section ID")
	ErrInvalidOpcode    = errors.New("invalid opcode")
	ErrInvalidType      = errors.New("invalid type encoding")
	ErrInvalidBlockType = errors.New("invalid block type")
	ErrMalformedSection = errors.New("malformed section")
)
