# utfbom
[![Godoc](https://godoc.org/github.com/slash3b/utfbom?status.png)](https://godoc.org/github.com/slash3b/utfbom) 
[![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0) 
[![Build Status](https://travis-ci.org/slash3b/utfbom.svg?branch=master)](https://travis-ci.org/slash3b/utfbom) 
[![Go Report Card](https://goreportcard.com/badge/github.com/slash3b/utfbom)](https://goreportcard.com/report/github.com/slash3b/utfbom) 
[![Coverage Status](https://coveralls.io/repos/github/slash3b/utfbom/badge.svg?branch=master)](https://coveralls.io/github/slash3b/utfbom?branch=master)

Package `utfbom` is able to detect and remove the Unicode Byte Order Mark (BOM) from input streams.
// todo add more

## Installation
```shell
    go get -u github.com/slash3b/utfbom
```

## Example
```go
package main

import (
	"bytes"
	"fmt"
	"io"

	"github.com/slash3b/utfbom"
)

func main() {
	trySkip([]byte("\xEF\xBB\xBFhello"))
	trySkip([]byte("hello"))
}

func trySkip(byteData []byte) {
	fmt.Println("Input:", byteData)

	// just skip BOM
	output, err := io.ReadAll(utfbom.SkipOnly(bytes.NewReader(byteData)))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("ReadAll with BOM skipping", output)

	// skip BOM and detect encoding
	sr, enc := utfbom.Skip(bytes.NewReader(byteData))
	fmt.Printf("Detected encoding: %s\n", enc)
	output, err = io.ReadAll(sr)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("ReadAll with BOM detection and skipping", output)
	fmt.Println()
}
```

Output:

```
$ go run main.go
Input: [239 187 191 104 101 108 108 111]
ReadAll with BOM skipping [104 101 108 108 111]
Detected encoding: UTF8
ReadAll with BOM detection and skipping [104 101 108 108 111]

Input: [104 101 108 108 111]
ReadAll with BOM skipping [104 101 108 108 111]
Detected encoding: Unknown
ReadAll with BOM detection and skipping [104 101 108 108 111]
```


