// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dirwatcher

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"strings"
)

const bufferedReaderBufferSize = 1 * 1024 * 1024

func ensureReaderIsDecompressed(plainReader io.ReadCloser, filename string) (io.ReadCloser, error) {
	type readCloser struct {
		closeutil.Closers
		io.Reader

		reader io.ReadCloser
	}

	bufferedReader := bufio.NewReaderSize(plainReader, bufferedReaderBufferSize)
	reader := &readCloser{Reader: bufferedReader, reader: plainReader, Closers: closeutil.New(plainReader)}

	if strings.HasSuffix(filename, ".gz") {
		compressedReader, err := gzip.NewReader(reader)

		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		return &readCloser{reader: reader, Reader: compressedReader, Closers: closeutil.New(reader, compressedReader)}, nil
	}

	if strings.HasSuffix(filename, ".bz2") {
		compressedReader := bzip2.NewReader(reader)
		return &readCloser{reader: reader, Reader: compressedReader, Closers: closeutil.New(reader)}, nil
	}

	return reader, nil
}
