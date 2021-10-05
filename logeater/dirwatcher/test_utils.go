// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dirwatcher

import (
	"bufio"
	"bytes"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"io"
	"strings"
)

// nolint:unused,deadcode
func readFromReader(reader io.Reader,
	filename string,
	onNewRecord func(parser.Header, []byte)) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Bytes()

		h, p, err := parser.ParseHeader(line)

		if parser.IsRecoverableError(err) {
			onNewRecord(h, p)
		}
	}
}

// nolint:unused,deadcode
type fakePublisher struct {
	logs []postfix.Record
}

// nolint:unused,deadcode
func (pub *fakePublisher) Publish(r postfix.Record) {
	pub.logs = append(pub.logs, r)
}

// nolint:unused,deadcode
type fakeFileReader struct {
	io.Reader
}

// nolint:unused,deadcode
func plainDataReaderFromBytes(data []byte) fakeFileReader {
	buf := bytes.NewBuffer(data)
	return fakeFileReader{Reader: strings.NewReader(buf.String())}
}

// nolint:unused,deadcode
func plainDataReader(s string) io.ReadCloser {
	return plainDataReaderFromBytes([]byte(s))
}

// nolint:unused,deadcode
func (fakeFileReader) Close() error {
	return nil
}

// nolint:unused,deadcode
type fakeFileData interface {
	hasFakeContent()
}

// nolint:unused,deadcode
type fakeFileDataBytes struct {
	content []byte
}

// nolint:unused,deadcode
func (fakeFileDataBytes) hasFakeContent() {
}

// nolint:unused,deadcode
func plainDataFile(s string) fakeFileDataBytes {
	return fakeFileDataBytes{[]byte(s)}
}

// nolint:unused,deadcode
type fakePlainCurrentFileData struct {
	content []byte
	offset  int64
}

// nolint:unused,deadcode
func (fakePlainCurrentFileData) hasFakeContent() {
}

// nolint:unused,deadcode
func plainCurrentDataFile(s, c string) fakePlainCurrentFileData {
	return fakePlainCurrentFileData{[]byte(s + c), int64(len(s))}
}
