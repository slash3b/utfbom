// Package utfbom provides utilities for handling the Unicode Byte Order Mark.
//
// It detects the type of BOM present in data,
// offers functions to strip the BOM from strings or byte slices,
// and includes an io.Reader wrapper that automatically detects and removes the BOM during reading.
package utfbom

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"sync"
)

var (
	_          io.Reader = (*Reader)(nil)
	utf8BOM              = []byte{0xef, 0xbb, 0xbf}
	utf16BEBOM           = []byte{0xfe, 0xff}
	utf16LEBOM           = []byte{0xff, 0xfe}
	utf32BEBOM           = []byte{0x00, 0x00, 0xfe, 0xff}
	utf32LEBOM           = []byte{0xff, 0xfe, 0x00, 0x00}
)

// ErrRead helps to trace error origin.
var ErrRead = errors.New("utfbom library unable to detect BOM")

// Encoding is a character encoding standard.
type Encoding int

const (
	// Unknown represents an unknown encoding that does not affect the incoming byte stream.
	// It has no associated Byte Order Mark.
	Unknown Encoding = iota

	// UTF8 represents UTF-8 encoding.
	// Its Byte Order Mark (BOM) is 0xef 0xbb 0xbf.
	UTF8

	// UTF16BigEndian represents UTF-16 encoding with big-endian byte order.
	// Its Byte Order Mark (BOM) is 0xfe 0xff.
	UTF16BigEndian

	// UTF16LittleEndian represents UTF-16 encoding with little-endian byte order.
	// Its Byte Order Mark (BOM) is 0xff 0xfe.
	UTF16LittleEndian

	// UTF32BigEndian represents UTF-32 encoding with big-endian byte order.
	// Its Byte Order Mark (BOM) is 0x00 0x00 0xfe 0xff.
	UTF32BigEndian

	// UTF32LittleEndian represents UTF-32 encoding with little-endian byte order.
	// Its Byte Order Mark (BOM) is 0xff 0xfe 0x00 0x00.
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
func DetectEncoding[T string | []byte](input T) Encoding {
	ibs := []byte(input)

	if len(ibs) < 2 {
		return Unknown
	}

	if len(ibs) >= 4 {
		if bytes.HasPrefix(ibs, utf32BEBOM) {
			return UTF32BigEndian
		}

		if bytes.HasPrefix(ibs, utf32LEBOM) {
			return UTF32LittleEndian
		}
	}

	if len(ibs) >= 3 && bytes.HasPrefix(ibs, utf8BOM) {
		return UTF8
	}

	if bytes.HasPrefix(ibs, utf16BEBOM) {
		return UTF16BigEndian
	}

	if bytes.HasPrefix(ibs, utf16LEBOM) {
		return UTF16LittleEndian
	}

	return Unknown
}

// AnyOf reports whether the Encoding value equals any of the given Encoding values.
// It returns true if a match is found, otherwise false.
func (e Encoding) AnyOf(es ...Encoding) bool {
	for _, enc := range es {
		if enc == e {
			return true
		}
	}

	return false
}

// Strings returns human-readable name of encoding.
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

// Len returns number of bytes specific for Encoding.
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
func Trim[T string | []byte](input T) (T, Encoding) {
	bytes := []byte(input)
	enc := DetectEncoding(bytes)

	if enc == Unknown {
		return input, enc
	}

	return T(bytes[enc.Len():]), enc
}

// Reader implements automatic BOM (Unicode Byte Order Mark) checking and
// removing as necessary for an io.Reader object.
type Reader struct {
	rd   *bufio.Reader
	once sync.Once
	// Enc will be available after first read
	Enc Encoding
}

// NewReader wraps an incoming reader.
func NewReader(rd io.Reader) *Reader {
	return &Reader{
		rd:   bufio.NewReader(rd),
		once: sync.Once{},
		Enc:  Unknown,
	}
}

// Read implements the io.Reader interface.
// On the first read call, it reads from the underlying Reader, detects and removes any Byte Order Mark (BOM).
// Subsequent calls delegate directly to the underlying Reader without BOM handling.
// Read is only safe for concurrent use during the first call due to sync.Once; after that, thread-safety
// depends on the underlying Reader. It is best to assume unsafe concurrent use.
func (r *Reader) Read(buf []byte) (int, error) {
	const maxBOMLen = 4

	if len(buf) == 0 {
		return 0, nil
	}

	var bomErr error

	r.once.Do(func() {
		bytes, err := r.rd.Peek(maxBOMLen)
		// do not error out in case underlying payload is too small
		// still attempt to read fewer than n bytes.
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			bomErr = errors.Join(ErrRead, err)

			return
		}

		r.Enc = DetectEncoding(bytes)
		if r.Enc != Unknown {
			_, err = r.rd.Discard(r.Enc.Len())
			if err != nil {
				bomErr = errors.Join(ErrRead, err)
			}
		}
	})

	if bomErr != nil {
		return 0, bomErr
	}

	return r.rd.Read(buf)
}
