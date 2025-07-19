package utfbom_test

import (
	"bytes"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/nalgeon/be"
	"github.com/slash3b/utfbom"
)

func TestDetectBom(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected utfbom.Encoding
	}{
		{
			name:     "empty",
			input:    nil,
			expected: utfbom.Unknown,
		},
		{
			name:     "no_encoding",
			input:    []byte("hello"),
			expected: utfbom.Unknown,
		},
		{
			name:     "bom_located_in_unexpected_place",
			input:    []byte("hello\ufeff"),
			expected: utfbom.Unknown,
		},
		{
			name:     "utf8_detected_from_string_literal",
			input:    []byte("\ufeffhello"),
			expected: utfbom.UTF8,
		},
		{
			name:     "utf16_big_endian_detected_and_some_text",
			input:    []byte{0xfe, 0xff, 'h', 'e', 'y'},
			expected: utfbom.UTF16BigEndian,
		},
		{
			name:     "utf16_big_endian_detected",
			input:    []byte{0xfe, 0xff},
			expected: utfbom.UTF16BigEndian,
		},
		{
			name:     "utf16_little_endian_detected_and_some_text",
			input:    []byte{0xff, 0xfe, 'h', 'e', 'y'},
			expected: utfbom.UTF16LittleEndian,
		},
		{
			name:     "utf16_little_endian_detected",
			input:    []byte{0xff, 0xfe},
			expected: utfbom.UTF16LittleEndian,
		},
		{
			name:     "utf32_big_endian_detected_and_some_text",
			input:    []byte{0x0, 0x0, 0xfe, 0xff, 'h', 'e', 'y'},
			expected: utfbom.UTF32BigEndian,
		},
		{
			name:     "utf32_big_endian_detected",
			input:    []byte{0x0, 0x0, 0xfe, 0xff},
			expected: utfbom.UTF32BigEndian,
		},
		{
			name:     "utf32_little_endian_detected_and_some_text",
			input:    []byte{0xff, 0xfe, 0x0, 0x0, 'h', 'e', 'y'},
			expected: utfbom.UTF32LittleEndian,
		},
		{
			name:     "utf32_little_endian_detected",
			input:    []byte{0xff, 0xfe, 0x0, 0x0},
			expected: utfbom.UTF32LittleEndian,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			be.Equal(t, utfbom.DetectEncoding(tc.input), tc.expected)
		})
	}
}

func ExampleDetectEncoding() {
	input := "\ufeffhey"
	fmt.Printf("input string: %q\n", input)
	fmt.Printf("input bytes: %#x\n", input)

	enc := utfbom.DetectEncoding(input)
	fmt.Printf("detected encoding: %s\n", enc)

	fmt.Printf("is UTF16:%v\n", enc.AnyOf(utfbom.UTF16BigEndian, utfbom.UTF16LittleEndian))
	fmt.Printf("is UTF8:%v\n", enc.AnyOf(utfbom.UTF8))

	// output:
	// input string: "\ufeffhey"
	// input bytes: 0xefbbbf686579
	// detected encoding: UTF8
	// is UTF16:false
	// is UTF8:true
}

func ExampleTrim() {
	input := "\ufeffhello"
	fmt.Printf("input string: %q\n", input)
	fmt.Printf("input bytes: %#x\n", input)

	output, enc := utfbom.Trim(input)

	fmt.Printf("detected encoding: %s\n", enc)
	fmt.Printf("output string: %q\n", output)
	fmt.Printf("output bytes:%#x\n", output)

	// output:
	// input string: "\ufeffhello"
	// input bytes: 0xefbbbf68656c6c6f
	// detected encoding: UTF8
	// output string: "hello"
	// output bytes:0x68656c6c6f
}

func ExampleReader() {
	csvFile := "\uFEFFIndex,Customer Id,First Name\n" +
		"1,DD37Cf93aecA6Dc,Sheryl"

	urd := utfbom.NewReader(bytes.NewReader([]byte(csvFile)))
	crd := csv.NewReader(urd)

	out := ""
	for {
		row, err := crd.Read()
		if err != nil {
			break
		}

		out += strings.Join(row, ",")
	}

	fmt.Println("detected encoding:", urd.Enc)
	fmt.Println("before")
	fmt.Println(hex.Dump([]byte(csvFile)))
	fmt.Println("after")
	fmt.Println(hex.Dump([]byte(out)))

	// output:
	//detected encoding: UTF8
	//before
	//00000000  ef bb bf 49 6e 64 65 78  2c 43 75 73 74 6f 6d 65  |...Index,Custome|
	//00000010  72 20 49 64 2c 46 69 72  73 74 20 4e 61 6d 65 0a  |r Id,First Name.|
	//00000020  31 2c 44 44 33 37 43 66  39 33 61 65 63 41 36 44  |1,DD37Cf93aecA6D|
	//00000030  63 2c 53 68 65 72 79 6c                           |c,Sheryl|
	//
	//after
	//00000000  49 6e 64 65 78 2c 43 75  73 74 6f 6d 65 72 20 49  |Index,Customer I|
	//00000010  64 2c 46 69 72 73 74 20  4e 61 6d 65 31 2c 44 44  |d,First Name1,DD|
	//00000020  33 37 43 66 39 33 61 65  63 41 36 44 63 2c 53 68  |37Cf93aecA6Dc,Sh|
	//00000030  65 72 79 6c                                       |eryl|
}

func TestEncoding_String(t *testing.T) {
	t.Parallel()

	for e := utfbom.Unknown; e <= utfbom.UTF32LittleEndian; e++ {
		be.True(t, e.String() != "")
	}

	s := utfbom.Encoding(999).String()
	be.True(t, s == "Unknown")
}

func TestEncoding_Len(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		enc      utfbom.Encoding
		expected int
	}{
		{
			enc:      utfbom.Unknown,
			expected: 0,
		},
		{
			enc:      utfbom.UTF8,
			expected: 3,
		},
		{
			enc:      utfbom.UTF16BigEndian,
			expected: 2,
		},
		{
			enc:      utfbom.UTF16LittleEndian,
			expected: 2,
		},
		{
			enc:      utfbom.UTF32BigEndian,
			expected: 4,
		},
		{
			enc:      utfbom.UTF32LittleEndian,
			expected: 4,
		},
		{
			enc:      999,
			expected: 0,
		},
	}

	for _, tc := range testCases {
		be.Equal(t, tc.enc.Len(), tc.expected)
	}
}

func TestEncoding_Trim(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    []byte
		encoding utfbom.Encoding
		output   []byte
	}{
		{"empty", nil, utfbom.Unknown, nil},
		{"no_bom", []byte("hello"), utfbom.Unknown, []byte("hello")},
		{"only_utf8_bom", []byte("\ufeff"), utfbom.UTF8, []byte{}},
		{"utf8_bom_with_string", []byte("\ufeffhello"), utfbom.UTF8, []byte("hello")},
		{"incomplete_payload_left_intact", []byte{0xef}, utfbom.Unknown, []byte{0xef}},
		{"utf16_be_empty", []byte{0xfe, 0xff}, utfbom.UTF16BigEndian, []byte{}},
		{"utf16_be", []byte{0xfe, 0xff, 0x00, 0x68, 0x00, 0x65}, utfbom.UTF16BigEndian, []byte{0x00, 0x68, 0x00, 0x65}},
		{"utf16_le_empty", []byte{0xff, 0xfe}, utfbom.UTF16LittleEndian, []byte{}},
		{"utf16_le", []byte{0xff, 0xfe, 0x68, 0x00, 0x65}, utfbom.UTF16LittleEndian, []byte{0x68, 0x00, 0x65}},
		{"utf32_be_empty", []byte{0x00, 0x00, 0xfe, 0xff}, utfbom.UTF32BigEndian, []byte{}},
		{"utf32_be", []byte{0x00, 0x00, 0xfe, 0xff, 0x00, 0x00, 0x00, 0x68}, utfbom.UTF32BigEndian, []byte{0x00, 0x00, 0x00, 0x68}},
		{"utf32_le_empty", []byte{0xff, 0xfe, 0x00, 0x00}, utfbom.UTF32LittleEndian, []byte{}},
		{"utf32_le", []byte{0xff, 0xfe, 0x00, 0x00, 0x68, 0x00, 0x65}, utfbom.UTF32LittleEndian, []byte{0x68, 0x00, 0x65}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out, enc := utfbom.Trim(tc.input)

			be.Equal(t, enc, tc.encoding)
			be.Equal(t, out, tc.output)
		})
	}
}

var teststring = "\ufeff" + `Lorem ipsum dolor sit amet consectetur adipiscing elit
Quisque faucibus ex sapien vitae pellentesque sem placerat.`

func TestReader_BOM_DeleteSuccess(t *testing.T) {
	t.Parallel()

	bomPrefixedStringReader := strings.NewReader(teststring)

	rd := utfbom.NewReader(bomPrefixedStringReader)

	be.Err(t, iotest.TestReader(rd, []byte(teststring[3:])), nil)
}

func TestReader_StringWithoutBOM(t *testing.T) {
	t.Parallel()

	nobomstring, _ := utfbom.Trim(teststring)

	rd := utfbom.NewReader(strings.NewReader(nobomstring))

	be.Err(t, iotest.TestReader(rd, []byte(nobomstring)), nil)
}

func TestReader_UsualReader(t *testing.T) {
	t.Parallel()

	bomPrefixedStringReader := strings.NewReader(teststring)

	rd := utfbom.NewReader(bomPrefixedStringReader)

	be.Err(t, iotest.TestReader(rd, []byte(teststring[3:])), nil)
}

func TestReader_OneByteReader(t *testing.T) {
	t.Parallel()

	bomPrefixedStringReader := strings.NewReader(teststring)

	rd := iotest.OneByteReader(utfbom.NewReader(bomPrefixedStringReader))

	be.Err(t, iotest.TestReader(rd, []byte(teststring[3:])), nil)
}

func TestReader_EmptyBuffer(t *testing.T) {
	t.Parallel()

	rd := utfbom.NewReader(nil)

	// in case buf length is 0, io.Reader implementations
	// should always return 0, nil
	// see: https://pkg.go.dev/io#Reader
	for range 10 {
		n, err := rd.Read(nil)
		be.Equal(t, 0, n)
		be.Err(t, err, nil)
	}
}

func TestReader_WrappeeReaderIsTooSmall(t *testing.T) {
	t.Parallel()

	wrappee := strings.NewReader("a")
	wrapped := utfbom.NewReader(wrappee)

	buf := make([]byte, 100)
	n, err := wrapped.Read(buf)
	be.Equal(t, 0, n)
	be.Err(t, err, io.EOF)
	be.Err(t, err, utfbom.ErrRead)

	// you might proceed reading if you want
	n, err = wrapped.Read(buf)
	be.Err(t, err, nil)
	be.Equal(t, 1, n)
	be.Equal(t, string(buf[:n]), "a")
}
