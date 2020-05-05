package rawparser

import (
	"regexp"
)

const (
	possibleMonths = `Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec`

	timeRegexpFormat = `(?P<Time>(?P<Month>(` + possibleMonths + `))\s\s?(?P<Day>[0-9]{1,2})\s(?P<Hour>[0-9]{2}):(?P<Minute>[0-9]{2}):(?P<Second>[0-9]{2}))`

	hostRegexpFormat = `(?P<Host>[0-9A-Za-z\.]+)`

	postfixProcessRegexpFormat = `^postfix(?P<PostfixSuffix>-[^/]+)?/` + `(?P<ProcessName>.*)`

	processRegexpFormat = `(?P<ProcessAndMaybePid>(?P<Process>[^[\s:]+)(\[(?P<ProcessId>[0-9]{1,5})\])?)`

	headerRegexpFormat = `^` + timeRegexpFormat + `\s` + hostRegexpFormat +
		` ` + processRegexpFormat + `:\s`
)

type RawHeader struct {
	Time    []byte
	Month   []byte
	Day     []byte
	Hour    []byte
	Minute  []byte
	Second  []byte
	Host    []byte
	Process []byte
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
	headerRegexp         *regexp.Regexp
	postfixProcessRegexp *regexp.Regexp

	timeIndex   int
	monthIndex  int
	dayIndex    int
	hourIndex   int
	minuteIndex int
	secondIndex int
	hostIndex   int

	processIndex        int
	postfixProcessIndex int
)

func init() {
	headerRegexp = regexp.MustCompile(headerRegexpFormat)

	postfixProcessRegexp = regexp.MustCompile(postfixProcessRegexpFormat)

	timeIndex = indexForGroup(headerRegexp, "Time")
	monthIndex = indexForGroup(headerRegexp, "Month")
	dayIndex = indexForGroup(headerRegexp, "Day")
	hourIndex = indexForGroup(headerRegexp, "Hour")
	minuteIndex = indexForGroup(headerRegexp, "Minute")
	secondIndex = indexForGroup(headerRegexp, "Second")
	hostIndex = indexForGroup(headerRegexp, "Host")

	processIndex = indexForGroup(headerRegexp, "Process")

	postfixProcessIndex = indexForGroup(postfixProcessRegexp, "ProcessName")
}

func tryToGetHeaderAndPayloadContent(logLine []byte) (RawHeader, []byte, error) {
	headerMatches := headerRegexp.FindSubmatch(logLine)

	if len(headerMatches) == 0 {
		return RawHeader{}, nil, InvalidHeaderLineError
	}

	buildHeader := func(process []byte) RawHeader {
		return RawHeader{
			Time:    headerMatches[timeIndex],
			Month:   headerMatches[monthIndex],
			Day:     headerMatches[dayIndex],
			Hour:    headerMatches[hourIndex],
			Minute:  headerMatches[minuteIndex],
			Second:  headerMatches[secondIndex],
			Host:    headerMatches[hostIndex],
			Process: process,
		}
	}

	payloadLine := logLine[len(headerMatches[0]):]

	if len(headerMatches[processIndex]) == 0 {
		return buildHeader(nil), nil, UnsupportedLogLineError
	}

	postfixProcessMatches := postfixProcessRegexp.FindSubmatch(headerMatches[processIndex])

	if len(postfixProcessMatches) == 0 {
		return buildHeader(nil), nil, UnsupportedLogLineError
	}

	return buildHeader(postfixProcessMatches[postfixProcessIndex]), payloadLine, nil
}

var (
	payloadHandlers = map[string]func(RawHeader, []byte) (RawPayload, error){}
)

func registerHandler(process string, handler func(RawHeader, []byte) (RawPayload, error)) {
	payloadHandlers[process] = handler
}

func Parse(logLine []byte) (RawHeader, RawPayload, error) {
	header, payloadLine, err := tryToGetHeaderAndPayloadContent(logLine)

	if err == InvalidHeaderLineError {
		return RawHeader{}, RawPayload{}, err
	}

	if err != nil {
		return header, RawPayload{}, err
	}

	handler, found := payloadHandlers[string(header.Process)]

	if !found {
		return header, RawPayload{PayloadType: PayloadTypeUnsupported}, UnsupportedLogLineError
	}

	p, err := handler(header, payloadLine)

	return header, p, err
}
