package rawparser

import (
	"errors"
	"regexp"
)

const (
	timeRegexpFormat = `(?P<Time>(?P<Month>([\w]{3}))\s\s?(?P<Day>[0-9]{1,2})\s(?P<Hour>[0-9]{2}):(?P<Minute>[\d]{2}):(?P<Second>[0-9]{2}))`

	hostRegexpFormat = `(?P<Host>[\w\.]+)`

	processRegexpFormat = `(?P<ProcessName>[\w]+)(-(?P<ProcessIP>[^/+]+))?(/(?P<DaemonName>[^[]+))?(\[(?P<ProcessID>\d+)\])?`

	headerRegexpFormat = `^` + timeRegexpFormat + `\s` + hostRegexpFormat + ` ` + processRegexpFormat + `:\s`
)

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

	timeIndex   int
	monthIndex  int
	dayIndex    int
	hourIndex   int
	minuteIndex int
	secondIndex int
	hostIndex   int

	processNameIndex   int
	processDaemonIndex int
	processIPIndex     int
	processIDIndex     int
)

func init() {
	headerRegexp = regexp.MustCompile(headerRegexpFormat)

	timeIndex = indexForGroup(headerRegexp, "Time")
	monthIndex = indexForGroup(headerRegexp, "Month")
	dayIndex = indexForGroup(headerRegexp, "Day")
	hourIndex = indexForGroup(headerRegexp, "Hour")
	minuteIndex = indexForGroup(headerRegexp, "Minute")
	secondIndex = indexForGroup(headerRegexp, "Second")
	hostIndex = indexForGroup(headerRegexp, "Host")

	processDaemonIndex = indexForGroup(headerRegexp, "DaemonName")
	processIPIndex = indexForGroup(headerRegexp, "ProcessIP")
	processNameIndex = indexForGroup(headerRegexp, "ProcessName")
	processIDIndex = indexForGroup(headerRegexp, "ProcessID")
}

func tryToGetHeaderAndPayloadContent(logLine []byte) (RawHeader, []byte, error) {
	headerMatches := headerRegexp.FindSubmatch(logLine)

	if len(headerMatches) == 0 {
		return RawHeader{}, nil, ErrInvalidHeaderLine
	}

	payloadLine := logLine[len(headerMatches[0]):]

	return RawHeader{
		Time:      headerMatches[timeIndex],
		Month:     headerMatches[monthIndex],
		Day:       headerMatches[dayIndex],
		Hour:      headerMatches[hourIndex],
		Minute:    headerMatches[minuteIndex],
		Second:    headerMatches[secondIndex],
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
