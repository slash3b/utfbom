// Package utfbom provides utilities for handling the Unicode Byte Order Mark (BOM).
//
// It detects the type of BOM present in data,
// offers functions to strip the BOM from strings or byte slices,
// and includes an io.Reader wrapper that automatically detects and removes the BOM during reading.
package utfbom

import (
	"errors"
	"io"
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
//   - UTF-8 (BOM: 0xEF 0xBB 0xBF)
//   - UTF-16 Big Endian (BOM: 0xFE 0xFF)
//   - UTF-16 Little Endian (BOM: 0xFF 0xFE)
//   - UTF-32 Big Endian (BOM: 0x00 0x00 0xFE 0xFF)
//   - UTF-32 Little Endian (BOM: 0xFF 0xFE 0x00 0x00)
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

// func (e Encoding) RemoveBOM(s string) string {
//  if e == UTF8 {
//   return strings.TrimPrefix(s, unicodeBOM)
//  }

//  return s
// }

const maxConsecutiveEmptyReads = 100

// Skip creates Reader which automatically detects BOM (Unicode Byte Order Mark) and removes it as necessary.
// It also returns the encoding detected by the BOM.
// If the detected encoding is not needed, you can call the SkipOnly function.
func Skip(rd io.Reader) (*Reader, Encoding) {
	// Is it already a Reader?
	b, ok := rd.(*Reader)
	if ok {
		return b, b.enc
	}

	enc, left, err := detectUtf(rd)
	return &Reader{
		rd:  rd,
		buf: left,
		err: err,
		enc: enc,
	}, enc
}

// SkipOnly creates Reader which automatically detects BOM (Unicode Byte Order Mark) and removes it as necessary.
func SkipOnly(rd io.Reader) *Reader {
	r, _ := Skip(rd)
	return r
}

// Reader implements automatic BOM (Unicode Byte Order Mark) checking and
// removing as necessary for an io.Reader object.
type Reader struct {
	rd  io.Reader // reader provided by the client
	buf []byte    // buffered data
	err error     // last error
	enc Encoding  // encoding
}

// Read is an implementation of io.Reader interface.
// The bytes are taken from the underlying Reader, but it checks for BOMs, removing them as necessary.
func (r *Reader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	if r.buf == nil {
		if r.err != nil {
			return 0, r.readErr()
		}

		return r.rd.Read(p)
	}

	// copy as much as we can
	n = copy(p, r.buf)
	r.buf = nilIfEmpty(r.buf[n:])
	return n, nil
}

func (r *Reader) readErr() error {
	err := r.err
	r.err = nil
	return err
}

var errNegativeRead = errors.New("utfbom: reader returned negative count from Read")

func detectUtf(rd io.Reader) (enc Encoding, buf []byte, err error) {
	buf, err = readBOM(rd)

	if len(buf) >= 4 {
		if isUTF32BigEndianBOM4(buf) {
			return UTF32BigEndian, nilIfEmpty(buf[4:]), err
		}
		if isUTF32LittleEndianBOM4(buf) {
			return UTF32LittleEndian, nilIfEmpty(buf[4:]), err
		}
	}

	if len(buf) > 2 && isUTF8BOM3(buf) {
		return UTF8, nilIfEmpty(buf[3:]), err
	}

	if (err != nil && err != io.EOF) || (len(buf) < 2) {
		return Unknown, nilIfEmpty(buf), err
	}

	if isUTF16BigEndianBOM2(buf) {
		return UTF16BigEndian, nilIfEmpty(buf[2:]), err
	}
	if isUTF16LittleEndianBOM2(buf) {
		return UTF16LittleEndian, nilIfEmpty(buf[2:]), err
	}

	return Unknown, nilIfEmpty(buf), err
}

func readBOM(rd io.Reader) (buf []byte, err error) {
	const maxBOMSize = 4
	var bom [maxBOMSize]byte // used to read BOM

	// read as many bytes as possible
	for nEmpty, n := 0, 0; err == nil && len(buf) < maxBOMSize; buf = bom[:len(buf)+n] {
		if n, err = rd.Read(bom[len(buf):]); n < 0 {
			panic(errNegativeRead)
		}
		if n > 0 {
			nEmpty = 0
		} else {
			nEmpty++
			if nEmpty >= maxConsecutiveEmptyReads {
				err = io.ErrNoProgress
			}
		}
	}
	return
}

func isUTF32BigEndianBOM4(buf []byte) bool {
	return buf[0] == 0x00 && buf[1] == 0x00 && buf[2] == 0xFE && buf[3] == 0xFF
}

func isUTF32LittleEndianBOM4(buf []byte) bool {
	return buf[0] == 0xFF && buf[1] == 0xFE && buf[2] == 0x00 && buf[3] == 0x00
}

func isUTF8BOM3(buf []byte) bool {
	return buf[0] == 0xEF && buf[1] == 0xBB && buf[2] == 0xBF
}

func isUTF16BigEndianBOM2(buf []byte) bool {
	return buf[0] == 0xFE && buf[1] == 0xFF
}

func isUTF16LittleEndianBOM2(buf []byte) bool {
	return buf[0] == 0xFF && buf[1] == 0xFE
}

func nilIfEmpty(buf []byte) (res []byte) {
	if len(buf) > 0 {
		res = buf
	}
	return
}
