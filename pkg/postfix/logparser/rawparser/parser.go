package rawparser

import (
	"bytes"
	"errors"
	"regexp"
)

const (
	hostRegexpFormat = `(?P<Host>[\w\.]+)`

	processRegexpFormat = `(?P<ProcessName>[\w]+)(-(?P<ProcessIP>[^/+]+))?(/(?P<DaemonName>[^[]+))?(\[(?P<ProcessID>\d+)\])?`

	headerRegexpFormat = `^` + hostRegexpFormat + ` ` + processRegexpFormat + `:\s`

	mailLocalPartRegexpFormat = `[^@]+`
)

func normalizeMailLocalPart(s []byte) []byte {
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

func indexForGroup(r *regexp.Regexp, name string) int {
	e := r.SubexpNames()
	for i, v := range e {
		if v == name {
			return i
		}
	}

	panic("Wrong Group Name: " + name + "!")
}

var (
	headerRegexp *regexp.Regexp

	hostIndex          int
	processNameIndex   int
	processDaemonIndex int
	processIPIndex     int
	processIDIndex     int
)

func init() {
	headerRegexp = regexp.MustCompile(headerRegexpFormat)

	hostIndex = indexForGroup(headerRegexp, "Host")

	processDaemonIndex = indexForGroup(headerRegexp, "DaemonName")
	processIPIndex = indexForGroup(headerRegexp, "ProcessIP")
	processNameIndex = indexForGroup(headerRegexp, "ProcessName")
	processIDIndex = indexForGroup(headerRegexp, "ProcessID")
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

	headerMatches := headerRegexp.FindSubmatch(remainingHeader)

	if len(headerMatches) == 0 {
		return RawHeader{}, nil, ErrInvalidHeaderLine
	}

	payloadLine := logLine[len(sampleLogDateTime)+len(headerMatches[0]):]

	return RawHeader{
		Time:      logLine[:len(sampleLogDateTime)-1],
		Month:     logLine[0:3],
		Day:       logLine[4:6],
		Hour:      logLine[7:9],
		Minute:    logLine[10:12],
		Second:    logLine[13:15],
		Host:      headerMatches[hostIndex],
		Process:   headerMatches[processNameIndex],
		ProcessIP: headerMatches[processIPIndex],
		Daemon:    headerMatches[processDaemonIndex],
		ProcessID: headerMatches[processIDIndex],
	}, payloadLine, nil
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
