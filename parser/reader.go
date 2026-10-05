package parser

import (
	"math"
	"unicode/utf8"
)

type Reader struct {
	data   []byte
	offset int
	// usesDataIdx records a memory.init or data.drop, which need the data
	// count section.
	usesDataIdx bool
}

func NewReader(data []byte) *Reader {
	return &Reader{data: data, offset: 0}
}

func (r *Reader) Offset() int {
	return r.offset
}

func (r *Reader) Remaining() int {
	return len(r.data) - r.offset
}

func (r *Reader) EOF() bool {
	return r.offset >= len(r.data)
}

func (r *Reader) ReadByte() (byte, error) {
	if r.offset >= len(r.data) {
		return 0, ErrUnexpectedEOF
	}
	b := r.data[r.offset]
	r.offset++
	return b, nil
}

func (r *Reader) ReadBytes(n int) ([]byte, error) {
	if r.offset+n > len(r.data) {
		return nil, ErrUnexpectedEOF
	}
	b := r.data[r.offset : r.offset+n]
	r.offset += n
	return b, nil
}

func (r *Reader) PeekByte() (byte, error) {
	if r.offset >= len(r.data) {
		return 0, ErrUnexpectedEOF
	}
	return r.data[r.offset], nil
}

func (r *Reader) ReadU32() (uint32, error) {
	v, err := r.readUnsigned(32)
	return uint32(v), err
}

func (r *Reader) ReadU64() (uint64, error) {
	return r.readUnsigned(64)
}

func (r *Reader) ReadI32() (int32, error) {
	v, err := r.readSigned(32)
	return int32(v), err
}

func (r *Reader) ReadI64() (int64, error) {
	return r.readSigned(64)
}

// ReadS33 reads the signed 33-bit LEB128 used for block type indices.
func (r *Reader) ReadS33() (int64, error) {
	return r.readSigned(33)
}

// readUnsigned decodes an unsigned LEB128 of at most ceil(bits/7) bytes
// whose unused high bits are zero.
func (r *Reader) readUnsigned(bits uint) (uint64, error) {
	var result uint64
	for shift := uint(0); ; shift += 7 {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		last := shift+7 >= bits
		if last && b&0x80 != 0 {
			return 0, ErrIntegerTooLong
		}
		if last && uint64(b&0x7f)>>(bits-shift) != 0 {
			return 0, ErrIntegerOverflow
		}
		result |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return result, nil
		}
	}
}

// readSigned decodes a signed LEB128 of at most ceil(bits/7) bytes whose
// unused high bits repeat the sign bit.
func (r *Reader) readSigned(bits uint) (int64, error) {
	var result int64
	var shift uint
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		last := shift+7 >= bits
		if last && b&0x80 != 0 {
			return 0, ErrIntegerTooLong
		}
		if last {
			// The sign bit and every unused bit above it must agree.
			high := (b & 0x7f) >> (bits - shift - 1)
			if high != 0 && high != 0x7f>>(bits-shift-1) {
				return 0, ErrIntegerOverflow
			}
		}
		result |= int64(b&0x7f) << shift
		shift += 7
		if b&0x80 == 0 {
			if shift < 64 && b&0x40 != 0 {
				result |= -1 << shift
			}
			return result, nil
		}
	}
}

func (r *Reader) ReadF32() (float32, error) {
	if r.offset+4 > len(r.data) {
		return 0, ErrUnexpectedEOF
	}
	bits := uint32(r.data[r.offset]) |
		uint32(r.data[r.offset+1])<<8 |
		uint32(r.data[r.offset+2])<<16 |
		uint32(r.data[r.offset+3])<<24
	r.offset += 4

	return math.Float32frombits(bits), nil
}

func (r *Reader) ReadF64() (float64, error) {
	if r.offset+8 > len(r.data) {
		return 0, ErrUnexpectedEOF
	}
	bits := uint64(r.data[r.offset]) |
		uint64(r.data[r.offset+1])<<8 |
		uint64(r.data[r.offset+2])<<16 |
		uint64(r.data[r.offset+3])<<24 |
		uint64(r.data[r.offset+4])<<32 |
		uint64(r.data[r.offset+5])<<40 |
		uint64(r.data[r.offset+6])<<48 |
		uint64(r.data[r.offset+7])<<56
	r.offset += 8

	return math.Float64frombits(bits), nil
}

func (r *Reader) ReadName() (string, error) {
	length, err := r.ReadU32()
	if err != nil {
		return "", err
	}
	bytes, err := r.ReadBytes(int(length))
	if err != nil {
		return "", err
	}
	if !utf8.Valid(bytes) {
		return "", ErrMalformedUTF8
	}
	return string(bytes), nil
}

func ReadVec[T any](r *Reader, readElem func(*Reader) (T, error)) ([]T, error) {
	count, err := r.ReadU32()
	if err != nil {
		return nil, err
	}
	result := make([]T, count)
	for i := uint32(0); i < count; i++ {
		elem, err := readElem(r)
		if err != nil {
			return nil, err
		}
		result[i] = elem
	}
	return result, nil
}

func (r *Reader) Skip(n int) error {
	if r.offset+n > len(r.data) {
		return ErrUnexpectedEOF
	}
	r.offset += n
	return nil
}

func (r *Reader) Slice(length int) (*Reader, error) {
	if r.offset+length > len(r.data) {
		return nil, ErrUnexpectedEOF
	}
	sub := &Reader{
		data:   r.data[r.offset : r.offset+length],
		offset: 0,
	}
	r.offset += length
	return sub, nil
}
