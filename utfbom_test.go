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
			enc := utfbom.DetectEncoding(tc.input)
			if enc != tc.expected {
				t.Errorf("unexpected encoding %s, expected is %s\n", enc, tc.expected)
			}
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

	output := utfbom.Trim(input, enc)
	fmt.Printf("output string: %q\n", output)
	fmt.Printf("output bytes:%#x\n", output)

	// output:
	// input string: "\ufeffhey"
	// input bytes: 0xefbbbf686579
	// detected encoding: UTF8
	// is UTF16:false
	// is UTF8:true
	// output string: "hey"
	// output bytes:0x686579
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

var testCases = []struct {
	name       string
	input      []byte
	inputError error
	encoding   utfbom.Encoding
	output     []byte
}{
	{"1", []byte{}, nil, utfbom.Unknown, []byte{}},
	{"2", []byte("hello"), nil, utfbom.Unknown, []byte("hello")},
	{"3", []byte("\xEF\xBB\xBF"), nil, utfbom.UTF8, []byte{}},
	{"4", []byte("\xEF\xBB\xBFhello"), nil, utfbom.UTF8, []byte("hello")},
	{"5", []byte("\xFE\xFF"), nil, utfbom.UTF16BigEndian, []byte{}},
	{"6", []byte("\xFF\xFE"), nil, utfbom.UTF16LittleEndian, []byte{}},
	{"7", []byte("\x00\x00\xFE\xFF"), nil, utfbom.UTF32BigEndian, []byte{}},
	{"8", []byte("\xFF\xFE\x00\x00"), nil, utfbom.UTF32LittleEndian, []byte{}},
	{
		"5", []byte("\xFE\xFF\x00\x68\x00\x65\x00\x6C\x00\x6C\x00\x6F"), nil,
		utfbom.UTF16BigEndian,
		[]byte{0x00, 0x68, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F},
	},
	{
		"6", []byte("\xFF\xFE\x68\x00\x65\x00\x6C\x00\x6C\x00\x6F\x00"), nil,
		utfbom.UTF16LittleEndian,
		[]byte{0x68, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00},
	},
	{
		"7", []byte("\x00\x00\xFE\xFF\x00\x00\x00\x68\x00\x00\x00\x65\x00\x00\x00\x6C\x00\x00\x00\x6C\x00\x00\x00\x6F"), nil,
		utfbom.UTF32BigEndian,
		[]byte{0x00, 0x00, 0x00, 0x68, 0x00, 0x00, 0x00, 0x65, 0x00, 0x00, 0x00, 0x6C, 0x00, 0x00, 0x00, 0x6C, 0x00, 0x00, 0x00, 0x6F},
	},
	{
		"8", []byte("\xFF\xFE\x00\x00\x68\x00\x00\x00\x65\x00\x00\x00\x6C\x00\x00\x00\x6C\x00\x00\x00\x6F\x00\x00\x00"), nil,
		utfbom.UTF32LittleEndian,
		[]byte{0x68, 0x00, 0x00, 0x00, 0x65, 0x00, 0x00, 0x00, 0x6C, 0x00, 0x00, 0x00, 0x6C, 0x00, 0x00, 0x00, 0x6F, 0x00, 0x00, 0x00},
	},
	{"9", []byte("\xEF"), nil, utfbom.Unknown, []byte("\xEF")},
	{"10", []byte("\xEF\xBB"), nil, utfbom.Unknown, []byte("\xEF\xBB")},
	{"11", []byte("\xEF\xBB\xBF"), io.ErrClosedPipe, utfbom.UTF8, []byte{}},
	{"12", []byte("\xFE\xFF"), io.ErrClosedPipe, utfbom.Unknown, []byte("\xFE\xFF")},
	{"13", []byte("\xFE"), io.ErrClosedPipe, utfbom.Unknown, []byte("\xFE")},
	{"14", []byte("\xFF\xFE"), io.ErrClosedPipe, utfbom.Unknown, []byte("\xFF\xFE")},
	{"15", []byte("\x00\x00\xFE\xFF"), io.ErrClosedPipe, utfbom.UTF32BigEndian, []byte{}},
	{"16", []byte("\x00\x00\xFE"), io.ErrClosedPipe, utfbom.Unknown, []byte{0x00, 0x00, 0xFE}},
	{"17", []byte("\x00\x00"), io.ErrClosedPipe, utfbom.Unknown, []byte{0x00, 0x00}},
	{"18", []byte("\x00"), io.ErrClosedPipe, utfbom.Unknown, []byte{0x00}},
	{"19", []byte("\xFF\xFE\x00\x00"), io.ErrClosedPipe, utfbom.UTF32LittleEndian, []byte{}},
	{"20", []byte("\xFF\xFE\x00"), io.ErrClosedPipe, utfbom.Unknown, []byte{0xFF, 0xFE, 0x00}},
	{"21", []byte("\xFF\xFE"), io.ErrClosedPipe, utfbom.Unknown, []byte{0xFF, 0xFE}},
	{"22", []byte("\xFF"), io.ErrClosedPipe, utfbom.Unknown, []byte{0xFF}},
	{"23", []byte("\x68\x65"), nil, utfbom.Unknown, []byte{0x68, 0x65}},
}

type sliceReader struct {
	input      []byte
	inputError error
}

func (r *sliceReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	if err = r.getError(); err != nil {
		return
	}

	n = copy(p, r.input)
	r.input = r.input[n:]
	err = r.getError()
	return
}

func (r *sliceReader) getError() (err error) {
	if len(r.input) == 0 {
		if r.inputError == nil {
			err = io.EOF
		} else {
			err = r.inputError
		}
	}
	return
}

var readMakers = []struct {
	name string
	fn   func(io.Reader) io.Reader
}{
	{"full", func(r io.Reader) io.Reader { return r }},
	{"byte", iotest.OneByteReader},
}

/*
func TestSkip(t *testing.T) {
	t.Parallel()

	for _, tc := range testCases {
		tc := tc
		t.Run("test "+tc.name, func(t *testing.T) {
			t.Parallel()

			for _, readMaker := range readMakers {
				readMaker := readMaker
				t.Run("reader="+readMaker.name, func(t *testing.T) {
					t.Parallel()

					r := readMaker.fn(&sliceReader{tc.input, tc.inputError})

					sr, enc := utfbom.Skip(r)
					if enc != tc.encoding {
						t.Fatalf("expected encoding %v, but got %v", tc.encoding, enc)
					}

					output, err := io.ReadAll(sr)
					if !reflect.DeepEqual(output, tc.output) {
						t.Fatalf("expected to read %+#v, but got %+#v", tc.output, output)
					}
					if !errors.Is(err, tc.inputError) {
						t.Fatalf("expected to get %+#v error, but got %+#v", tc.inputError, err)
					}
				})
			}
		})
	}
}

func TestSkipSkip(t *testing.T) {
	t.Parallel()

	for _, tc := range testCases {
		tc := tc
		t.Run("test "+tc.name, func(t *testing.T) {
			t.Parallel()

			for _, readMaker := range readMakers {
				readMaker := readMaker
				t.Run("reader="+readMaker.name, func(t *testing.T) {
					t.Parallel()

					r := readMaker.fn(&sliceReader{tc.input, tc.inputError})

					sr0, _ := utfbom.Skip(r)
					sr, enc := utfbom.Skip(sr0)
					if enc != tc.encoding {
						t.Fatalf("expected encoding %v, but got %v", tc.encoding, enc)
					}

					output, err := io.ReadAll(sr)
					if !reflect.DeepEqual(output, tc.output) {
						t.Fatalf("expected to read %+#v, but got %+#v", tc.output, output)
					}
					if !errors.Is(err, tc.inputError) {
						t.Fatalf("expected to get %+#v error, but got %+#v", tc.inputError, err)
					}
				})
			}
		})
	}
}

func TestSkipOnly(t *testing.T) {
	t.Parallel()

	for _, tc := range testCases {
		tc := tc
		t.Run("test "+tc.name, func(t *testing.T) {
			t.Parallel()

			for _, readMaker := range readMakers {
				readMaker := readMaker
				t.Run("reader="+readMaker.name, func(t *testing.T) {
					t.Parallel()

					r := readMaker.fn(&sliceReader{tc.input, tc.inputError})

					sr := utfbom.SkipOnly(r)

					output, err := io.ReadAll(sr)
					if !reflect.DeepEqual(output, tc.output) {
						t.Fatalf("expected to read %+#v, but got %+#v", tc.output, output)
					}
					if !errors.Is(err, tc.inputError) {
						t.Fatalf("expected to get %+#v error, but got %+#v", tc.inputError, err)
					}
				})
			}
		})
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	return 0, nil
}

type readerEncoding struct {
	Rd  *utfbom.Reader
	Enc utfbom.Encoding
}

func TestSkipZeroReader(t *testing.T) {
	t.Parallel()

	var z zeroReader

	c := make(chan readerEncoding)
	go func() {
		r, enc := utfbom.Skip(z)
		c <- readerEncoding{r, enc}
	}()

	select {
	case re := <-c:
		if re.Enc != utfbom.Unknown {
			t.Error("utfbom.Unknown encoding expected")
		} else {
			var b [1]byte
			n, err := re.Rd.Read(b[:])
			if n != 0 {
				t.Error("unexpected bytes count:", n)
			}
			if err != io.ErrNoProgress {
				t.Error("unexpected error:", err)
			}
		}
	case <-time.After(time.Second):
		t.Error("test timed out (endless loop in utfbom.Skip?)")
	}
}

func TestSkipOnlyZeroReader(t *testing.T) {
	t.Parallel()

	var z zeroReader

	c := make(chan *utfbom.Reader)
	go func() {
		r := utfbom.SkipOnly(z)
		c <- r
	}()

	select {
	case r := <-c:
		var b [1]byte
		n, err := r.Read(b[:])
		if n != 0 {
			t.Error("unexpected bytes count:", n)
		}
		if err != io.ErrNoProgress {
			t.Error("unexpected error:", err)
		}
	case <-time.After(time.Second):
		t.Error("test timed out (endless loop in utfbom.Skip?)")
	}
}

func TestReader_ReadEmpty(t *testing.T) {
	t.Parallel()

	for _, tc := range testCases {
		tc := tc
		t.Run("test "+tc.name, func(t *testing.T) {
			t.Parallel()

			for _, readMaker := range readMakers {
				readMaker := readMaker
				t.Run("reader="+readMaker.name, func(t *testing.T) {
					t.Parallel()

					r := readMaker.fn(&sliceReader{tc.input, tc.inputError})

					sr := utfbom.SkipOnly(r)

					n, err := sr.Read(nil)
					if n != 0 {
						t.Fatalf("test %v reader=%s: expected to read zero bytes, but got %v", tc.name, readMaker.name, n)
					}
					if err != nil {
						t.Fatalf("test %v reader=%s: expected to get <nil> error, but got %+#v", tc.name, readMaker.name, err)
					}
				})
			}
		})
	}
}

func TestEncoding_String(t *testing.T) {
	t.Parallel()

	for e := utfbom.Unknown; e <= utfbom.UTF32LittleEndian; e++ {
		s := e.String()
		if s == "" {
			t.Errorf("no string for %#v", e)
		}
	}

	s := utfbom.Encoding(999).String()
	if s != "Unknown" {
		t.Errorf("wrong string '%s' for invalid encoding", s)
	}
}

func ExampleSkipOnly() {
	byteData := []byte("\xEF\xBB\xBFhello")

	fmt.Println("Input:", byteData)

	output, err := io.ReadAll(utfbom.SkipOnly(bytes.NewReader(byteData)))
	if err != nil {
		panic(err)
	}

	fmt.Println("ReadAll with BOM skipping", output)

	// Output:
	// Input: [239 187 191 104 101 108 108 111]
	// ReadAll with BOM skipping [104 101 108 108 111]
}

func ExampleSkip() {
	byteData := []byte("\xEF\xBB\xBFhello")

	sr, enc := utfbom.Skip(bytes.NewReader(byteData))

	fmt.Printf("Detected encoding: %s\n", enc)

	output, err := io.ReadAll(sr)
	if err != nil {
		panic(err)
	}

	fmt.Println("ReadAll with BOM detection and skipping", output)

	// Output:
	// Detected encoding: UTF8
	// ReadAll with BOM detection and skipping [104 101 108 108 111]
}


*/
