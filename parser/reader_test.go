package parser

import (
	"errors"
	"math"
	"testing"
)

type lebCase struct {
	name string
	in   []byte
	want int64
	err  error
}

func runLEB(t *testing.T, cases []lebCase, read func(*Reader) (int64, error)) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := read(NewReader(tc.in))
			if tc.err != nil {
				if !errors.Is(err, tc.err) {
					t.Fatalf("got %d, %v; want error %v", got, err, tc.err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("got %d, %v; want %d", got, err, tc.want)
			}
		})
	}
}

func TestReadU32(t *testing.T) {
	runLEB(t, []lebCase{
		{"zero", []byte{0x00}, 0, nil},
		{"padded zero", []byte{0x80, 0x00}, 0, nil},
		{"max", []byte{0xff, 0xff, 0xff, 0xff, 0x0f}, math.MaxUint32, nil},
		{"unused bits set", []byte{0xff, 0xff, 0xff, 0xff, 0x1f}, 0, ErrIntegerOverflow},
		{"six bytes", []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x00}, 0, ErrIntegerTooLong},
		{"truncated", []byte{0x80}, 0, ErrUnexpectedEOF},
	}, func(r *Reader) (int64, error) {
		v, err := r.ReadU32()
		return int64(v), err
	})
}

func TestReadI32(t *testing.T) {
	runLEB(t, []lebCase{
		{"minus one", []byte{0x7f}, -1, nil},
		{"max", []byte{0xff, 0xff, 0xff, 0xff, 0x07}, math.MaxInt32, nil},
		{"min", []byte{0x80, 0x80, 0x80, 0x80, 0x78}, math.MinInt32, nil},
		{"padded minus one", []byte{0xff, 0x7f}, -1, nil},
		{"positive with unused bits", []byte{0xff, 0xff, 0xff, 0xff, 0x0f}, 0, ErrIntegerOverflow},
		{"negative with unused bits clear", []byte{0x80, 0x80, 0x80, 0x80, 0x70}, 0, ErrIntegerOverflow},
		{"six bytes", []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}, 0, ErrIntegerTooLong},
	}, func(r *Reader) (int64, error) {
		v, err := r.ReadI32()
		return int64(v), err
	})
}

func TestReadI64(t *testing.T) {
	runLEB(t, []lebCase{
		{"max", []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00}, math.MaxInt64, nil},
		{"min", []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x7f}, math.MinInt64, nil},
		{"unused bits", []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01}, 0, ErrIntegerOverflow},
		{"eleven bytes", []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x00}, 0, ErrIntegerTooLong},
	}, (*Reader).ReadI64)
}

func TestReadS33(t *testing.T) {
	runLEB(t, []lebCase{
		{"empty block type", []byte{0x40}, -64, nil},
		{"max type index", []byte{0xff, 0xff, 0xff, 0xff, 0x0f}, math.MaxUint32, nil},
		{"unused bits", []byte{0xff, 0xff, 0xff, 0xff, 0x1f}, 0, ErrIntegerOverflow},
	}, (*Reader).ReadS33)
}

func TestReadName(t *testing.T) {
	if got, err := NewReader([]byte{0x03, 'a', 0xc3, 0xa9}).ReadName(); err != nil || got != "aé" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := NewReader([]byte{0x02, 0xc3, 0x28}).ReadName(); !errors.Is(err, ErrMalformedUTF8) {
		t.Fatalf("got %v, want %v", err, ErrMalformedUTF8)
	}
}
