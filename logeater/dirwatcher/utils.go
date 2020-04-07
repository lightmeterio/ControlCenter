package dirwatcher

import (
	"compress/gzip"
	"strings"
)

type gzippedFileReader struct {
	gzipReader *gzip.Reader
	fileReader fileReader
}

func (r *gzippedFileReader) Close() error {
	if err := r.gzipReader.Close(); err != nil {
		return err
	}

	return r.fileReader.Close()
}

func (r *gzippedFileReader) Read(p []byte) (int, error) {
	return r.gzipReader.Read(p)
}

func ensureReaderIsDecompressed(reader fileReader, filename string) (fileReader, error) {
	if strings.HasSuffix(filename, ".gz") {
		gzipReader, err := gzip.NewReader(reader)

		if err != nil {
			return nil, err
		}

		return &gzippedFileReader{fileReader: reader, gzipReader: gzipReader}, nil
	}

	return reader, nil
}
