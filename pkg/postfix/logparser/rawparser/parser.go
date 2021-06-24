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
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/timeutil"
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
	Time      timeutil.RawTime
	Host      []byte
	Process   []byte
	Daemon    []byte
	ProcessIP []byte
	ProcessID []byte
}

type TimeFormat = timeutil.TimeFormat

func tryToGetHeaderAndPayloadContent(logLine []byte, format TimeFormat) (RawHeader, []byte, error) {
	t, remainingHeader, l, err := format.ExtractRaw(logLine)
	if err != nil {
		return RawHeader{}, nil, err
	}

	h := RawHeader{Time: t}

	n, succeed := parseHeaderPostfixPart(&h, remainingHeader)

	if !succeed {
		return RawHeader{}, nil, ErrInvalidHeaderLine
	}

	payloadLine := logLine[l+n+1:]

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

func ParseHeaderWithCustomTimeFormat(logLine []byte, format TimeFormat) (RawHeader, []byte, error) {
	// Remove leading 0x0
	start := bytes.IndexFunc(logLine, func(r rune) bool {
		return r != 0
	})

	if start != -1 {
		logLine = logLine[start:]
	}

	header, payloadLine, err := tryToGetHeaderAndPayloadContent(logLine, format)
	if errors.Is(err, ErrInvalidHeaderLine) {
		return RawHeader{}, nil, err
	}

	if errors.Is(err, timeutil.ErrInvalidTimeFormat) {
		return RawHeader{}, nil, ErrInvalidHeaderLine
	}

	return header, payloadLine, nil
}

func ParseHeader(logLine []byte) (RawHeader, []byte, error) {
	return ParseHeaderWithCustomTimeFormat(logLine, timeutil.DefaultTimeFormat{})
}

func ParsePayload(payloadLine []byte, daemon, process string) (RawPayload, error) {
	handler, found := payloadHandlers[payloadHandlerKey{daemon: daemon, process: process}]
	if !found {
		return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
	}

	return handler(payloadLine)
}
