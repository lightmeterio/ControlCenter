// Package readercomp provides comparison functions for readers or more specific for comparing file contents
package readercomp

import (
	"bytes"
	"io"
	"os"
)

// Equal tests two finite reader streams for equality by comparing batches up to bufSize
func Equal(r1, r2 io.Reader, bufSize int) (bool, error) {
	b1 := make([]byte, bufSize)
	b2 := make([]byte, bufSize)

	for {
		n1, err1 := r1.Read(b1)
		n2, err2 := r2.Read(b2)

		if err1 != nil && err1 != io.EOF {
			return false, err1
		}
		if err2 != nil && err2 != io.EOF {
			return false, err2
		}

		// Catch up with r2
		for n1 < n2 {
			var more int
			more, err1 = r1.Read(b1[n1:n2])
			n1 += more
			if err1 == io.EOF {
				break
			} else if err1 != nil {
				return false, err1
			}
		}
		// Catch up with r1
		for n2 < n1 {
			var more int
			more, err2 = r2.Read(b2[n2:n1])
			n2 += more
			if err2 == io.EOF {
				break
			} else if err2 != nil {
				return false, err2
			}
		}

		if !bytes.Equal(b1[:n1], b2[:n2]) {
			return false, nil
		}

		// We asserted before that err1 / err2 are either nil or io.EOF - if both are io.EOF we are finished
		if err1 == io.EOF && err2 == io.EOF {
			return true, nil
		}

		// Continue anyway, either err1 or err2 could be io.EOF and the other nil, which is allowed and could settle
		// on the next call to Read
	}
}

// FilesEqual tests two files for equality (same content)
func FilesEqual(name1, name2 string) (bool, error) {
	f1Info, err := os.Stat(name1)
	if err != nil {
		return false, err
	}
	f2Info, err := os.Stat(name2)
	if err != nil {
		return false, err
	}

	if f1Info.Size() != f2Info.Size() {
		return false, nil
	}

	f1, err := os.Open(name1)
	if err != nil {
		return false, err
	}
	defer f1.Close()

	f2, err := os.Open(name2)
	if err != nil {
		return false, err
	}
	defer f2.Close()

	// 64 KiB buffers seem to be most performant with larger files
	return Equal(f1, f2, 64*1024)
}
