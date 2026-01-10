package parser

import "errors"

var (
	ErrUnexpectedEOF    = errors.New("unexpected end of data")
	ErrIntegerOverflow  = errors.New("integer overflow in LEB128 encoding")
	ErrInvalidMagic     = errors.New("invalid WASM magic number")
	ErrInvalidVersion   = errors.New("unsupported WASM version")
	ErrInvalidSection   = errors.New("invalid section ID")
	ErrInvalidOpcode    = errors.New("invalid opcode")
	ErrInvalidType      = errors.New("invalid type encoding")
	ErrInvalidBlockType = errors.New("invalid block type")
	ErrMalformedSection = errors.New("malformed section")
)
