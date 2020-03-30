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

// NOTE: Go does not have unions, and using interfaces implies on virtual calls
// (which are being done in the higher level parsing interface, anyways),
// so we add all the possible payloads inlined in the struct, with a field describing which
// payload the whole record refers to.
// This is ok as all payloads here store basically byte slices only, which are trivially constructible and copyable
// so, although this struct will grow as newer payloads are supported,
// copying will perform better than using virtual calls
type RawPayload struct {
	PayloadType          PayloadType
	RawSmtpSentStatus    RawSmtpSentStatus
	QmgrReturnedToSender QmgrReturnedToSender
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

	processAndMaybePidIndex int
	processIndex            int
	processIdIndex          int
	postfixProcessIndex     int
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

	processAndMaybePidIndex = indexForGroup(headerRegexp, "ProcessAndMaybePid")
	processIndex = indexForGroup(headerRegexp, "Process")
	processIdIndex = indexForGroup(headerRegexp, "ProcessId")

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
		return RawHeader{}, nil, UnsupportedLogLineError
	}

	return buildHeader(postfixProcessMatches[postfixProcessIndex]), payloadLine, nil
}

var (
	payloadHandlers = map[string]func(RawHeader, []byte) (RawPayload, error){
		"smtp": parseSmtpPayload,
		"qmgr": parseQmgrPayload,
	}
)

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
