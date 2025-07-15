// Package utfbom provides utilities for handling the Unicode Byte Order Mark (BOM).
//
// It detects the type of BOM present in data,
// offers functions to strip the BOM from strings or byte slices,
// and includes an io.Reader wrapper that automatically detects and removes the BOM during reading.
package utfbom

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

var (
	_          io.Reader = (*Reader)(nil)
	bom        rune      = '\uFEFF'
	utf8BOM              = [3]byte{0xef, 0xbb, 0xbf}
	utf16BEBOM           = [2]byte{0xfe, 0xff}
	utf16LEBOM           = [2]byte{0xff, 0xfe}
	utf32BEBOM           = [4]byte{0x00, 0x00, 0xfe, 0xff}
	utf32LEBOM           = [4]byte{0xff, 0xfe, 0x00, 0x00}
)

// Encoding is a character encoding standard.
type Encoding int

const (
	Unknown Encoding = iota
	UTF8
	UTF16BigEndian
	UTF16LittleEndian
	UTF32BigEndian
	UTF32LittleEndian
)

// DetectEncoding inspects the initial bytes of a string or byte slice (T)
// and returns the detected text encoding based on the presence of known BOMs (Byte Order Marks).
// If no known BOM is found, it returns Unknown.
//
// Supported encodings:
//   - UTF-8 (BOM: 0xef 0xbb 0xbf)
//   - UTF-16 Big Endian (BOM: 0xfe 0xff)
//   - UTF-16 Little Endian (BOM: 0xff 0xfe)
//   - UTF-32 Big Endian (BOM: 0x00 0x00 0xfe 0xff)
//   - UTF-32 Little Endian (BOM: 0xff 0xfe 0x00 0x00)
func DetectEncoding[T string | []byte](b T) Encoding {
	i := []byte(b)

	if len(i) < 2 {
		return Unknown
	}

	if len(i) >= 4 {
		if utf32BEBOM[0] == i[0] &&
			utf32BEBOM[1] == i[1] &&
			utf32BEBOM[2] == i[2] &&
			utf32BEBOM[3] == i[3] {
			return UTF32BigEndian
		}

		if utf32LEBOM[0] == i[0] &&
			utf32LEBOM[1] == i[1] &&
			utf32LEBOM[2] == i[2] &&
			utf32LEBOM[3] == i[3] {
			return UTF32LittleEndian
		}
	}

	if len(i) >= 3 {
		if utf8BOM[0] == i[0] && utf8BOM[1] == i[1] && utf8BOM[2] == i[2] {
			return UTF8
		}
	}

	if utf16BEBOM[0] == i[0] && utf16BEBOM[1] == i[1] {
		return UTF16BigEndian
	}

	if utf16LEBOM[0] == i[0] && utf16LEBOM[1] == i[1] {
		return UTF16LittleEndian
	}

	return Unknown
}

func (e Encoding) AnyOf(es ...Encoding) bool {
	for _, enc := range es {
		if enc == e {
			return true
		}
	}

	return false
}

func (e Encoding) String() string {
	switch e {
	case UTF8:
		return "UTF8"
	case UTF16BigEndian:
		return "UTF16BigEndian"
	case UTF16LittleEndian:
		return "UTF16LittleEndian"
	case UTF32BigEndian:
		return "UTF32BigEndian"
	case UTF32LittleEndian:
		return "UTF32LittleEndian"
	default:
		return "Unknown"
	}
}

func (e Encoding) Len() int {
	switch e {
	default:
		return 0
	case UTF8:
		return 3
	case UTF16BigEndian, UTF16LittleEndian:
		return 2
	case UTF32BigEndian, UTF32LittleEndian:
		return 4
	}
}

// Trim removes the BOM prefix from the input `s` based on the encoding `enc`.
// Supports string or []byte inputs and returns the same type without the BOM.
func Trim[T string | []byte](s T, enc Encoding) T {
	b := []byte(s)

	switch enc {
	case UTF8:
		b = b[3:]
	case UTF16BigEndian, UTF16LittleEndian:
		b = b[2:]
	case UTF32BigEndian, UTF32LittleEndian:
		b = b[4:]
	}

	return T(b)
}

func NewReader(rd io.Reader) *Reader {
	return &Reader{
		rd: rd,
	}
}

// Reader implements automatic BOM (Unicode Byte Order Mark) checking and
// removing as necessary for an io.Reader object.
type Reader struct {
	rd   io.Reader
	once sync.Once
	// Enc will be available after first read
	Enc Encoding
}

// Read is an implementation of io.Reader interface.
// The bytes are taken from the underlying Reader, but it checks for BOMs, removing them as necessary.
// todo: rewrite this, tries to be concurrently safe but depends totally on underlying Reader implementation.
func (r *Reader) Read(p []byte) (int, error) {
	const maxBOMLen = 4

	if len(p) == 0 {
		return 0, io.ErrShortBuffer
	}

	var (
		n   int
		err error
	)

	r.once.Do(func() {
		if len(p) < maxBOMLen {
			err = errors.Join(fmt.Errorf("min buffer lenght required: %d", maxBOMLen), io.ErrShortBuffer)
		}

		s := make([]byte, len(p))
		n, err = r.rd.Read(s)
		if err != nil {
			return
		}
		r.Enc = DetectEncoding(s)
		s = Trim(s, r.Enc)
		n = n - r.Enc.Len()

		copy(p, s[:n])
	})

	if n > 0 || err != nil {
		return n, err
	}

	n, err = r.rd.Read(p)

	return n, err
}
