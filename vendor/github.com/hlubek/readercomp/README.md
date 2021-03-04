# readercomp : Go package for comparing two finite io.Reader (e.g. files)

[![GoDoc](https://godoc.org/github.com/hlubek/readercomp?status.svg)](https://godoc.org/github.com/hlubek/readercomp)
[![Build Status](https://github.com/hlubek/readercomp/workflows/run%20tests/badge.svg)](https://github.com/hlubek/readercomp/actions?workflow=run%20tests)
[![Coverage Status](https://coveralls.io/repos/github/hlubek/readercomp/badge.svg?branch=main)](https://coveralls.io/github/hlubek/readercomp?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/hlubek/readercomp)](https://goreportcard.com/report/github.com/hlubek/readercomp)

## Why?

Comparing files (or more generally two finite `io.Reader`) is not as easy due to edge-cases like short reads and how errors (including `EOF`) are returned by calls to `Read` according to the [interface specification](https://pkg.go.dev/io/#Reader).

This packages (tries) to deliver a solid implementation for these cases.

## Install

```
go get github.com/hlubek/readercomp
```

## Example

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hlubek/readercomp"
)

func main() {
	result, err := readercomp.FilesEqual(os.Args[1], os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
```

## License

MIT.
