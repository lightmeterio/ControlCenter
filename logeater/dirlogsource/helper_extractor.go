package dirlogsource

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"log"
	"os"
	"path"
)

// Adapted from https://stackoverflow.com/a/57640231/1721672
//nolint:deadcode,unused
func extractTarGz(gzipStream io.Reader, outDir string) {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	errorutil.MustSucceed(err)

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if errors.Is(err, io.EOF) {
			break
		}

		errorutil.MustSucceed(err)

		//nolint:gosec
		outName := path.Join(outDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			err := os.MkdirAll(outName, os.ModePerm)
			errorutil.MustSucceed(err)
		case tar.TypeReg:
			err := os.MkdirAll(path.Dir(outName), os.ModePerm)
			errorutil.MustSucceed(err)
			outFile, err := os.Create(outName)
			errorutil.MustSucceed(err)
			//nolint:gosec
			_, err = io.Copy(outFile, tarReader)
			errorutil.MustSucceed(err)
			errorutil.MustSucceed(outFile.Close())
			errorutil.MustSucceed(os.Chtimes(outName, header.ModTime, header.ModTime))
		default:
			log.Fatalf(
				"ExtractTarGz: uknown type: %v in %v",
				header.Typeflag,
				header.Name)
		}
	}
}
