// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:generate ragel -Z -G2 header.rl -o header.gen.go
//go:generate ragel -Z -G2 smtp.rl -o smtp.gen.go
//go:generate ragel -Z -G2 qmgr.rl -o qmgr.gen.go
//go:generate ragel -Z -G2 cleanup.rl -o cleanup.gen.go
//go:generate ragel -Z -G2 bounce.rl -o bounce.gen.go
//go:generate ragel -Z -G2 pickup.rl -o pickup.gen.go

// TODO: move the go:generate comments to their respective go files
// TODO: create a wrapper command to allows us to use ragel-7, which has a different interface.

package rawparser

import (
	"bytes"
	"errors"
)

//nolint:deadcode,unused
// this function is used by the Ragel generated code (.rl files)
// and the linters are not able to see that.
func normalizeMailLocalPart(s []byte) []byte {
	// email local part can contain quotes, in case it contains spaces, like in: from=<"some email"@example.com>.
	// this function removes the trailing quotes
	return bytes.Trim(s, `"`)
}

type RawHeader struct {
	Time      []byte
	Month     []byte
	Day       []byte
	Hour      []byte
	Minute    []byte
	Second    []byte
	Host      []byte
	Process   []byte
	Daemon    []byte
	ProcessIP []byte
	ProcessID []byte
}

const (
	// A line starts with a time, with fixed length
	// the `day` field is always trailed with a space, if needed
	// so it's always two characters long
	sampleLogDateTime = `Mar 22 06:28:55 `
)

func tryToGetHeaderAndPayloadContent(logLine []byte) (RawHeader, []byte, error) {
	if len(logLine) < len(sampleLogDateTime) {
		return RawHeader{}, nil, ErrInvalidHeaderLine
	}

	remainingHeader := logLine[len(sampleLogDateTime):]

	if len(remainingHeader) == 0 {
		return RawHeader{}, nil, ErrInvalidHeaderLine
	}

	h := RawHeader{
		Time:   logLine[:len(sampleLogDateTime)-1],
		Month:  logLine[0:3],
		Day:    logLine[4:6],
		Hour:   logLine[7:9],
		Minute: logLine[10:12],
		Second: logLine[13:15],
		// Other fields intentionally left empty
	}

	n, succeed := parseHeaderPostfixPart(&h, remainingHeader)

	if !succeed {
		return RawHeader{}, nil, ErrInvalidHeaderLine
	}

	payloadLine := logLine[len(sampleLogDateTime)+n+1:]

	return h, payloadLine, nil
}

type payloadHandlerKey struct {
	process string
	daemon  string
}

var (
	payloadHandlers = map[payloadHandlerKey]func([]byte) (RawPayload, error){}
)

func registerHandler(process, daemon string, handler func([]byte) (RawPayload, error)) {
	payloadHandlers[payloadHandlerKey{process: process, daemon: daemon}] = handler
}

func ParseHeader(logLine []byte) (RawHeader, []byte, error) {
	// Remove leading 0x0
	start := bytes.IndexFunc(logLine, func(r rune) bool {
		return r != 0
	})

	if start != -1 {
		logLine = logLine[start:]
	}

	header, payloadLine, err := tryToGetHeaderAndPayloadContent(logLine)

	if errors.Is(err, ErrInvalidHeaderLine) {
		return RawHeader{}, nil, err
	}

	return header, payloadLine, nil
}

func ParsePayload(payloadLine []byte, daemon, process string) (RawPayload, error) {
	handler, found := payloadHandlers[payloadHandlerKey{daemon: daemon, process: process}]
	if !found {
		return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
	}

	return handler(payloadLine)
}
