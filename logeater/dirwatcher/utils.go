package dirwatcher

import (
	"bufio"
	"compress/gzip"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"strings"
)

type gzippedFileReader struct {
	gzipReader *gzip.Reader
	fileReader fileReader
}

func (r *gzippedFileReader) Close() error {
	if err := r.gzipReader.Close(); err != nil {
		return errorutil.Wrap(err)
	}

	return r.fileReader.Close()
}

func (r *gzippedFileReader) Read(p []byte) (int, error) {
	return r.gzipReader.Read(p)
}

type bufferedReader struct {
	b *bufio.Reader
	r fileReader
}

func (b *bufferedReader) Close() error {
	return b.r.Close()
}

func (b *bufferedReader) Read(p []byte) (int, error) {
	return b.b.Read(p)
}

const bufferedReaderBufferSize = 1 * 1024 * 1024

func ensureReaderIsDecompressed(plainReader fileReader, filename string) (fileReader, error) {
	reader := &bufferedReader{bufio.NewReaderSize(plainReader, bufferedReaderBufferSize), plainReader}

	if strings.HasSuffix(filename, ".gz") {
		gzipReader, err := gzip.NewReader(reader)

		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		return &gzippedFileReader{fileReader: reader, gzipReader: gzipReader}, nil
	}

	return reader, nil
}
