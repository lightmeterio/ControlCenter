//go:generate ragel -Z -G2 header.rl -o header.gen.go
//go:generate ragel -Z -G2 smtp.rl -o smtp.gen.go
//go:generate ragel -Z -G2 qmgr.rl -o qmgr.gen.go

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
	payloadHandlers = map[payloadHandlerKey]func(RawHeader, []byte) (RawPayload, error){}
)

func registerHandler(process, daemon string, handler func(RawHeader, []byte) (RawPayload, error)) {
	payloadHandlers[payloadHandlerKey{process: process, daemon: daemon}] = handler
}

func Parse(logLine []byte) (RawHeader, RawPayload, error) {
	// Remove leading 0x0
	start := bytes.IndexFunc(logLine, func(r rune) bool {
		return r != 0
	})

	if start != -1 {
		logLine = logLine[start:]
	}

	header, payloadLine, err := tryToGetHeaderAndPayloadContent(logLine)

	if errors.Is(err, ErrInvalidHeaderLine) {
		return RawHeader{}, RawPayload{}, err
	}

	handler, found := payloadHandlers[payloadHandlerKey{
		daemon:  string(header.Daemon),
		process: string(header.Process),
	}]

	if !found {
		return header, RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
	}

	p, err := handler(header, payloadLine)

	return header, p, err
}
