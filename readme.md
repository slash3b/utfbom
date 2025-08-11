# utfbom
[![Godoc](https://godoc.org/github.com/slash3b/utfbom?status.png)](https://godoc.org/github.com/slash3b/utfbom) 
[![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0) 
[![Go Report Card](https://goreportcard.com/badge/github.com/slash3b/utfbom)](https://goreportcard.com/report/github.com/slash3b/utfbom) 
![Build Status](https://github.com/slash3b/utfbom/actions/workflows/go-ci.yml/badge.svg)
[![Sourcegraph](https://sourcegraph.com/github.com/slash3b/utfbom/-/badge.svg)](https://sourcegraph.com/github.com/slash3b/utfbom?badge)

Package `utfbom` is able to detect and remove the Unicode Byte Order Mark (BOM) from input streams.

## Installation
```shell
    go get -u github.com/slash3b/utfbom
```

## What is `\uFEFF`?
`\uFEFF` is the Unicode Byte Order Mark (BOM), it indicates text encoding and byte order.  
Go source code is defined to be UTF-8 text, so **all string literals in Go source files are by default UTF-8 encoded sequences**, making Go a UTF-8 compliant language at its core.   
Note, BOM is a zero width character.

This library supports detection and trimming of the following BOM prefixes:

| Encoding         | BOM Hex Values               |
|:-----------------|:-----------------------------|
| UTF-8            | 0xef 0xbb 0xbf               |
| UTF-16 (BE)      | 0xfe 0xff                    |
| UTF-16 (LE)      | 0xff 0xfe                    |
| UTF-32 (BE)      | 0x00 0x00 0xfe 0xff          |
| UTF-32 (LE)      | 0xff 0xfe 0x00 0x00          |

[go.dev/play](https://go.dev/play/p/-VVI1k8UEnI)

```golang
    package main

    import (
        "bytes"
        "encoding/hex"
        "fmt"
    )

    func main() {
        a := "\ufefehey"
        b := "hey"

        fmt.Println(a == b)
        fmt.Println(bytes.Equal([]byte(a), []byte(b)))

        fmt.Print(hex.Dump([]byte(a)))
        fmt.Print(hex.Dump([]byte(b)))
    }

    // Output:
    // false
    // false
    // 00000000  ef bb be 68 65 79                                 |...hey|
    // 00000000  68 65 79                                          |hey|
```

Links:
- https://en.wikipedia.org/wiki/Byte_order_mark

## Examples

### Encoding detection
[go.dev/play](https://go.dev/play/p/yjj__F_fcEE)
```golang
    package main

    import (
        "fmt"

        "github.com/slash3b/utfbom"
    )

    func main() {
        input := "\ufeffhey"
        fmt.Printf("input string: %q\n", input)
        fmt.Printf("input bytes: %#x\n", input)

        enc := utfbom.DetectEncoding(input)
        fmt.Printf("detected encoding: %s\n", enc)

        fmt.Printf("is UTF16:%v\n", enc.AnyOf(utfbom.UTF16BigEndian, utfbom.UTF16LittleEndian))
        fmt.Printf("is UTF8:%v\n", enc.AnyOf(utfbom.UTF8))
    }

    // Output: 
    // input string: "\ufeffhey"
    // input bytes: 0xefbbbf686579
    // detected encoding: UTF8
    // is UTF16:false
    // is UTF8:true
```

### BOM trimming
[go.dev/play](https://go.dev/play/p/lAp1A-42qJN)
```golang
    package main

    import (
        "fmt"

        "github.com/slash3b/utfbom"
    )

    func main() {
        input := "\ufeffhey"
        fmt.Printf("input string: %q\n", input)
        fmt.Printf("input bytes: %#x\n", input)

		output, enc := utfbom.Trim(input)
		fmt.Printf("detected encoding: %s\n", enc)
		fmt.Printf("output string: %q\n", output)
		fmt.Printf("output bytes:%#x\n", output)
	}

    // Output: 
    // input string: "\ufeffhey"
    // input bytes: 0xefbbbf686579
    // detected encoding: UTF8
    // output string: "hey"
    // output bytes:0x686579
```

### Reading CSV file with BOM:
[go.dev/play](https://go.dev/play/p/aWOq-0GKQY7)
```golang
    package main

    import (
        "bytes"
        "encoding/csv"
        "encoding/hex"
        "fmt"
        "strings"

        "github.com/slash3b/utfbom"
    )

    func main() {
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
    }

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
```
