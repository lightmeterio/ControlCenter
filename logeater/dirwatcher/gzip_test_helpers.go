// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dirwatcher

import (
	"bytes"
	"compress/gzip"
	"github.com/rs/zerolog/log"
	"io"
)

// nolint:unused,deadcode
func compressGzip(content string) []byte {
	var buf bytes.Buffer

	w := gzip.NewWriter(&buf)

	_, err := w.Write([]byte(content))

	if err != nil {
		log.Fatal().Msg("compressing data")
	}

	if err := w.Close(); err != nil {
		log.Fatal().Msg("compressing data")
	}

	return buf.Bytes()
}

// nolint:unused,deadcode
func gzipedDataReaderFromBytes(data []byte) fakeFileReader {
	plainReader := plainDataReaderFromBytes(data)

	reader, err := ensureReaderIsDecompressed(plainReader, "something.gz")

	if err != nil {
		panic("Failed on decompressing file!!!! FIX IT!")
	}

	return fakeFileReader{reader}
}

// nolint:unused,deadcode
func gzipedDataReader(s string) io.ReadCloser {
	return gzipedDataReaderFromBytes(compressGzip(s))
}

// nolint:unused,deadcode
func gzippedDataFile(s string) fakeFileDataBytes {
	return fakeFileDataBytes{compressGzip(s)}
}
