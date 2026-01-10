package parser

import (
	"math"
)

type Reader struct {
	data   []byte
	offset int
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
	var result uint32
	var shift uint

	for {
		if r.offset >= len(r.data) {
			return 0, ErrUnexpectedEOF
		}
		b := r.data[r.offset]
		r.offset++

		result |= uint32(b&0x7F) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
		if shift >= 35 {
			return 0, ErrIntegerOverflow
		}
	}

	return result, nil
}

func (r *Reader) ReadU64() (uint64, error) {
	var result uint64
	var shift uint

	for {
		if r.offset >= len(r.data) {
			return 0, ErrUnexpectedEOF
		}
		b := r.data[r.offset]
		r.offset++

		result |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
		if shift >= 70 {
			return 0, ErrIntegerOverflow
		}
	}

	return result, nil
}

func (r *Reader) ReadI32() (int32, error) {
	var result int32
	var shift uint

	for {
		if r.offset >= len(r.data) {
			return 0, ErrUnexpectedEOF
		}
		b := r.data[r.offset]
		r.offset++

		result |= int32(b&0x7F) << shift
		shift += 7

		if b&0x80 == 0 {
			if shift < 32 && (b&0x40) != 0 {
				result |= ^int32(0) << shift
			}
			break
		}
		if shift >= 35 {
			return 0, ErrIntegerOverflow
		}
	}

	return result, nil
}

func (r *Reader) ReadI64() (int64, error) {
	var result int64
	var shift uint

	for {
		if r.offset >= len(r.data) {
			return 0, ErrUnexpectedEOF
		}
		b := r.data[r.offset]
		r.offset++

		result |= int64(b&0x7F) << shift
		shift += 7

		if b&0x80 == 0 {
			if shift < 64 && (b&0x40) != 0 {
				result |= ^int64(0) << shift
			}
			break
		}
		if shift >= 70 {
			return 0, ErrIntegerOverflow
		}
	}

	return result, nil
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
