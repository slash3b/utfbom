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

var (
	utf8BOM    = []byte{0xef, 0xbb, 0xbf}
	utf16BEBOM = []byte{0xfe, 0xff}
	utf16LEBOM = []byte{0xff, 0xfe}
	utf32BEBOM = []byte{0x00, 0x00, 0xfe, 0xff}
	utf32LEBOM = []byte{0xff, 0xfe, 0x00, 0x00}
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

func ExamplePrepend() {
	// Prepend a UTF-8 BOM to a simple string.
	// The UTF-8 BOM is represented by the rune \ufeff.
	withBOM := utfbom.Prepend("hello", utfbom.UTF8)
	fmt.Printf("String with UTF-8 BOM: %q\n", withBOM)
	fmt.Printf("Bytes: %#x\n\n", withBOM)

	// Prepend a UTF-16LE BOM to a byte slice that is also UTF-16LE encoded.
	// This represents the word "world" in UTF-16 Little Endian.
	data := []byte{0x77, 0x00, 0x6f, 0x00, 0x72, 0x00, 0x6c, 0x00, 0x64, 0x00}
	withBOMBytes := utfbom.Prepend(data, utfbom.UTF16LittleEndian)
	fmt.Printf("Bytes with UTF-16LE BOM: %#x\n\n", withBOMBytes)

	// The Prepend function is idempotent.
	// If a BOM already exists, it will not add another one.
	alreadyHasBOM := "\ufeffhello"
	idempotentResult := utfbom.Prepend(alreadyHasBOM, utfbom.UTF8)
	fmt.Printf("Idempotent result: %q\n", idempotentResult)
	fmt.Printf("Bytes are unchanged: %#x\n", idempotentResult)

	// output:
	// String with UTF-8 BOM: "\ufeffhello"
	// Bytes: 0xefbbbf68656c6c6f
	//
	// Bytes with UTF-16LE BOM: 0xfffe77006f0072006c006400
	//
	// Idempotent result: "\ufeffhello"
	// Bytes are unchanged: 0xefbbbf68656c6c6f
}

func ExampleReader() {
	csvFile := "\uFEFFIndex,Customer Id,First Name\n" +
		"1,DD37Cf93aecA6Dc,Sheryl"

	urd := utfbom.NewReader(bytes.NewReader([]byte(csvFile)))
	crd := csv.NewReader(urd)

	var out strings.Builder
	for {
		row, err := crd.Read()
		if err != nil {
			break
		}

		out.WriteString(strings.Join(row, ","))
	}

	fmt.Println("detected encoding:", urd.Enc)
	fmt.Println("before")
	fmt.Println(hex.Dump([]byte(csvFile)))
	fmt.Println("after")
	fmt.Println(hex.Dump([]byte(out.String())))

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
		name     string
		enc      utfbom.Encoding
		expected int
	}{
		{"Unknown", utfbom.Unknown, 0},
		{"UTF8", utfbom.UTF8, 3},
		{"UTF16BigEndian", utfbom.UTF16BigEndian, 2},
		{"UTF16LittleEndian", utfbom.UTF16LittleEndian, 2},
		{"UTF32BigEndian", utfbom.UTF32BigEndian, 4},
		{"UTF32LittleEndian", utfbom.UTF32LittleEndian, 4},
		{"InvalidEncoding", 999, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			be.Equal(t, tc.enc.Len(), tc.expected)
		})
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

// TestReader_WrappeeReaderHasTinyPayload tests that bufio.Reader is able to
// read on the first Read without failing.
func TestReader_WrappeeReaderHasTinyPayload_EnoughBuffer(t *testing.T) {
	t.Parallel()

	wrappee := bytes.NewReader([]byte{0xff, 0xfe, 0x01, 0x02, 0x03})
	wrapped := utfbom.NewReader(wrappee)

	buf := make([]byte, 100)
	n, err := wrapped.Read(buf)
	be.Equal(t, 3, n)
	t.Logf("have read %q", string(buf[:n]))
	t.Logf("detected enc: %s", wrapped.Enc)
	t.Logf("err: %v", err)
	be.Equal(t, buf[:n], []byte{0x01, 0x02, 0x03})
	be.Err(t, err, nil)

	n, err = wrapped.Read(buf)
	t.Logf("second read returns err: %v", err)
	be.Err(t, err, io.EOF)
	be.Equal(t, 0, n)
}

func TestReader_WrappeeReaderHasTinyPayload_OneByteBuffer(t *testing.T) {
	t.Parallel()

	wrappee := bytes.NewReader([]byte{0xff, 0xfe, 0x01, 0x02, 0x03})
	rd := iotest.OneByteReader(utfbom.NewReader(wrappee))

	buf := make([]byte, 1)
	for i := range 3 {
		n, err := rd.Read(buf)
		be.Err(t, err, nil)
		be.Equal(t, 1, n)
		be.Equal(t, []byte{0x01 + byte(i)}, buf[:n])
	}

	n, err := rd.Read(buf)
	be.Err(t, err, io.EOF)
	be.Equal(t, 0, n)
}

func TestEncoding_Bytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		enc      utfbom.Encoding
		expected []byte
	}{
		{
			name:     "Unknown",
			enc:      utfbom.Unknown,
			expected: nil,
		},
		{
			name:     "UTF8",
			enc:      utfbom.UTF8,
			expected: []byte{0xef, 0xbb, 0xbf},
		},
		{
			name:     "UTF16BigEndian",
			enc:      utfbom.UTF16BigEndian,
			expected: []byte{0xfe, 0xff},
		},
		{
			name:     "UTF16LittleEndian",
			enc:      utfbom.UTF16LittleEndian,
			expected: []byte{0xff, 0xfe},
		},
		{
			name:     "UTF32BigEndian",
			enc:      utfbom.UTF32BigEndian,
			expected: []byte{0x00, 0x00, 0xfe, 0xff},
		},
		{
			name:     "UTF32LittleEndian",
			enc:      utfbom.UTF32LittleEndian,
			expected: []byte{0xff, 0xfe, 0x00, 0x00},
		},
		{
			name:     "InvalidEncoding",
			enc:      utfbom.Encoding(999),
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.enc.Bytes()
			be.Equal(t, got, tc.expected)
		})
	}
}

// TestBytes_NoAliasing checks that bytes returned by Bytes() are immutable.
func TestBytes_NoAliasing(t *testing.T) {
	t.Parallel()

	encodings := []utfbom.Encoding{
		utfbom.UTF8,
		utfbom.UTF16BigEndian,
		utfbom.UTF16LittleEndian,
		utfbom.UTF32BigEndian,
		utfbom.UTF32LittleEndian,
	}

	for _, enc := range encodings {
		t.Run(enc.String(), func(t *testing.T) {
			t.Parallel()

			original := enc.Bytes()
			originalCopy := make([]byte, len(original))
			copy(originalCopy, original)

			original[0] = 0x00

			fresh := enc.Bytes()
			be.Equal(t, fresh, originalCopy)
		})
	}
}

func TestPrepend(t *testing.T) {
	t.Parallel()

	t.Run("byte_slice", func(t *testing.T) {
		data := []byte("data")

		testCases := []struct {
			name     string
			input    []byte
			enc      utfbom.Encoding
			expected []byte
		}{
			{"unknown_on_data", data, utfbom.Unknown, data},
			{"unknown_on_empty", []byte{}, utfbom.Unknown, []byte{}},
			{"unknown_on_nil", nil, utfbom.Unknown, nil},
			{"utf8_on_data", data, utfbom.UTF8, append(utf8BOM, data...)},
			{"utf8_on_empty", []byte{}, utfbom.UTF8, utf8BOM},
			{"utf8_on_nil", nil, utfbom.UTF8, utf8BOM},
			{"utf16be_on_data", data, utfbom.UTF16BigEndian, append(utf16BEBOM, data...)},
			{"utf16be_on_empty", []byte{}, utfbom.UTF16BigEndian, utf16BEBOM},
			{"utf16be_on_nil", nil, utfbom.UTF16BigEndian, utf16BEBOM},
			{"utf16le_on_data", data, utfbom.UTF16LittleEndian, append(utf16LEBOM, data...)},
			{"utf16le_on_empty", []byte{}, utfbom.UTF16LittleEndian, utf16LEBOM},
			{"utf16le_on_nil", nil, utfbom.UTF16LittleEndian, utf16LEBOM},
			{"utf32be_on_data", data, utfbom.UTF32BigEndian, append(utf32BEBOM, data...)},
			{"utf32be_on_empty", []byte{}, utfbom.UTF32BigEndian, utf32BEBOM},
			{"utf32be_on_nil", nil, utfbom.UTF32BigEndian, utf32BEBOM},
			{"utf32le_on_data", data, utfbom.UTF32LittleEndian, append(utf32LEBOM, data...)},
			{"utf32le_on_empty", []byte{}, utfbom.UTF32LittleEndian, utf32LEBOM},
			{"utf32le_on_nil", nil, utfbom.UTF32LittleEndian, utf32LEBOM},
			{"idempotent_when_bom_exists", append(utf8BOM, data...), utfbom.UTF8, append(utf8BOM, data...)},
			{"idempotent_when_different_bom_exists", append(utf32LEBOM, data...), utfbom.UTF16BigEndian, append(utf32LEBOM, data...)},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				got := utfbom.Prepend(tc.input, tc.enc)
				be.Equal(t, got, tc.expected)
			})
		}
	})
}

type CustomString string

type CustomBytes []byte

func TestDetectEncoding_TypeAliases(t *testing.T) {
	t.Parallel()

	t.Run("custom_string", func(t *testing.T) {
		input := CustomString("\ufeffhello")
		enc := utfbom.DetectEncoding(input)
		be.Equal(t, enc, utfbom.UTF8)
	})

	t.Run("custom_bytes", func(t *testing.T) {
		input := CustomBytes([]byte{0xfe, 0xff, 'h', 'i'})
		enc := utfbom.DetectEncoding(input)
		be.Equal(t, enc, utfbom.UTF16BigEndian)
	})
}

func TestTrim_TypeAliases(t *testing.T) {
	t.Parallel()

	t.Run("custom_string", func(t *testing.T) {
		input := CustomString("\ufeffhello")
		out, enc := utfbom.Trim(input)
		be.Equal(t, enc, utfbom.UTF8)
		be.Equal(t, out, CustomString("hello"))
	})

	t.Run("custom_bytes", func(t *testing.T) {
		input := CustomBytes([]byte{0xfe, 0xff, 'h', 'i'})
		out, enc := utfbom.Trim(input)
		be.Equal(t, enc, utfbom.UTF16BigEndian)
		be.Equal(t, out, CustomBytes([]byte{'h', 'i'}))
	})
}

func TestPrepend_TypeAliases(t *testing.T) {
	t.Parallel()

	t.Run("custom_string", func(t *testing.T) {
		input := CustomString("hello")
		out := utfbom.Prepend(input, utfbom.UTF8)
		be.Equal(t, out, CustomString("\ufeffhello"))
	})

	t.Run("custom_bytes", func(t *testing.T) {
		input := CustomBytes([]byte{'h', 'i'})
		out := utfbom.Prepend(input, utfbom.UTF16BigEndian)
		be.Equal(t, out, CustomBytes([]byte{0xfe, 0xff, 'h', 'i'}))
	})
}

func TestNewReader_NilPanics(t *testing.T) {
	t.Parallel()

	rd := utfbom.NewReader(nil)

	defer func() {
		r := recover()
		be.True(t, r != nil)
	}()

	buf := make([]byte, 10)
	_, _ = rd.Read(buf)
}
