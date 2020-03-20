package rawparser

import (
	"regexp"
)

const (
	possibleMonths = `Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec`

	timeRawSmtpSentStatusRegexpFormat = `(?P<Time>(?P<Month>(` + possibleMonths + `))\s\s?(?P<Day>[0-9]{1,2}) (?P<Hour>[0-9]{2}):(?P<Minute>[0-9]{2}):(?P<Second>[0-9]{2}))`

	hostRawSmtpSentStatusRegexpFormat = `(?P<Host>[0-9A-Za-z\.]+)`

	postfixProcessRawSmtpRegexpFormat = `^postfix(?P<PostfixSuffix>-[^/]+)?/` + `(?P<ProcessName>.*)`

	processRegexpFormat = `(?P<ProcessAndMaybePid>(?P<Process>[^[\s:]+)(\[(?P<ProcessId>[0-9]{1,5})\])?)`

	queueIdRawSmtpSentStatusRegexpFormat = `(?P<Queue>[0-9A-F]+)`

	headerRegexpFormat = `^` + timeRawSmtpSentStatusRegexpFormat + ` ` + hostRawSmtpSentStatusRegexpFormat +
		` ` + processRegexpFormat + `: `

	anythingExceptCommaRegexpFormat = `[^,]+`

	// NOTE: Relay name might be absent, having only "none"
	relayComponentsRegexpFormat = `((?P<RelayName>[^\,[]+)` + `\[(?P<RelayIp>[^\],]+)\]` + `:` + `(?P<RelayPort>[\d]+)|` + `none)`

	messageSentWithStatusRawSmtpSentStatusRegexpFormat = `(?P<MessageSentWithStatus>` +
		`to=<(?P<RecipientLocalPart>[^@]+)@(?P<RecipientDomainPart>[^>]+)>` + `, ` +
		`relay=` + relayComponentsRegexpFormat + `, ` +
		`delay=(?P<Delay>` + anythingExceptCommaRegexpFormat + `)` + `, ` +
		`delays=(?P<Delays>(?P<Delays0>[^/]+)/(?P<Delays1>[^/]+)/(?P<Delays2>[^/]+)/(?P<Delays3>[^/]+))` + `, ` +
		`dsn=(?P<Dsn>` + anythingExceptCommaRegexpFormat + `)` + `, ` +
		`status=(?P<Status>(deferred|bounced|sent))` + ` ` +
		`(?P<ExtraMessage>.*)` +
		`)`

	possibleSmtpPayloadsFormat = messageSentWithStatusRawSmtpSentStatusRegexpFormat

	smtpPayloadsRegexpFormat = `^` + queueIdRawSmtpSentStatusRegexpFormat + `: ` +
		`(` + possibleSmtpPayloadsFormat + `)$`
)

type PayloadType int

const (
	PayloadTypeSmtpMessageStatus PayloadType = iota
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

// NOTE: Go does not have unions and using interfaces implies on virtual calls
// (which are being done in the higher level parsing interface, anyways),
// so we add all the possible payloads inlined in the struct, with a field describing which
// payload the whole record refers to.
// This is ok as all payloads here store basically byte slices only, which are trivially constructible and copyable
// so, although this struct will grow as newer payloads are supported,
// copying will perform better than using virtual calls
type RawRecord struct {
	Header            RawHeader
	PayloadType       PayloadType
	RawSmtpSentStatus RawSmtpSentStatus
}

type RawSmtpSentStatus struct {
	Queue               []byte
	RecipientLocalPart  []byte
	RecipientDomainPart []byte
	RelayName           []byte
	RelayIp             []byte
	RelayPort           []byte
	Delay               []byte
	Delays              [5][]byte
	Dsn                 []byte
	Status              []byte
	ExtraMessage        []byte
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
	possibleSmtpPayloadsRegexp *regexp.Regexp
	headerRegexp               *regexp.Regexp
	postfixProcessRegexp       *regexp.Regexp

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

	messageSentWithStatusIndex   int
	smtpQueueIndex               int
	smtpRecipientLocalPartIndex  int
	smtpRecipientDomainPartIndex int
	smtpRelayNameIndex           int
	smtpRelayIpIndex             int
	smtpRelayPortIndex           int
	smtpDelayIndex               int
	smtpDelaysIndex              int
	smtpDelays0Index             int
	smtpDelays1Index             int
	smtpDelays2Index             int
	smtpDelays3Index             int
	smtpDsnIndex                 int
	smtpStatusIndex              int
	smtpExtraMessageIndex        int
)

func init() {
	possibleSmtpPayloadsRegexp = regexp.MustCompile(smtpPayloadsRegexpFormat)

	headerRegexp = regexp.MustCompile(headerRegexpFormat)

	postfixProcessRegexp = regexp.MustCompile(postfixProcessRawSmtpRegexpFormat)

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

	messageSentWithStatusIndex = indexForGroup(possibleSmtpPayloadsRegexp, "MessageSentWithStatus")
	smtpQueueIndex = indexForGroup(possibleSmtpPayloadsRegexp, "Queue")
	smtpRecipientLocalPartIndex = indexForGroup(possibleSmtpPayloadsRegexp, "RecipientLocalPart")
	smtpRecipientDomainPartIndex = indexForGroup(possibleSmtpPayloadsRegexp, "RecipientDomainPart")
	smtpRelayNameIndex = indexForGroup(possibleSmtpPayloadsRegexp, "RelayName")
	smtpRelayIpIndex = indexForGroup(possibleSmtpPayloadsRegexp, "RelayIp")
	smtpRelayPortIndex = indexForGroup(possibleSmtpPayloadsRegexp, "RelayPort")
	smtpDelayIndex = indexForGroup(possibleSmtpPayloadsRegexp, "Delay")
	smtpDelaysIndex = indexForGroup(possibleSmtpPayloadsRegexp, "Delays")
	smtpDelays0Index = indexForGroup(possibleSmtpPayloadsRegexp, "Delays0")
	smtpDelays1Index = indexForGroup(possibleSmtpPayloadsRegexp, "Delays1")
	smtpDelays2Index = indexForGroup(possibleSmtpPayloadsRegexp, "Delays2")
	smtpDelays3Index = indexForGroup(possibleSmtpPayloadsRegexp, "Delays3")
	smtpDsnIndex = indexForGroup(possibleSmtpPayloadsRegexp, "Dsn")
	smtpStatusIndex = indexForGroup(possibleSmtpPayloadsRegexp, "Status")
	smtpExtraMessageIndex = indexForGroup(possibleSmtpPayloadsRegexp, "ExtraMessage")
}

func ParseLogLine(logLine []byte) (RawRecord, error) {
	headerMatches := headerRegexp.FindSubmatch(logLine)

	if len(headerMatches) == 0 {
		return RawRecord{}, InvalidHeaderLineError
	}

	processLine := headerMatches[processAndMaybePidIndex]

	if len(processLine) == 0 {
		panic("There is an error in the header Regex! Fix it!")
	}

	linePayload := logLine[len(headerMatches[0]):]

	if len(headerMatches[processIndex]) == 0 {
		return RawRecord{}, UnsupportedLogLineError
	}

	postfixProcessMatches := postfixProcessRegexp.FindSubmatch(headerMatches[processIndex])

	if len(postfixProcessMatches) == 0 {
		return RawRecord{}, UnsupportedLogLineError
	}

	postfixProcess := postfixProcessMatches[postfixProcessIndex]

	header := RawHeader{
		Time:    headerMatches[timeIndex],
		Month:   headerMatches[monthIndex],
		Day:     headerMatches[dayIndex],
		Hour:    headerMatches[hourIndex],
		Minute:  headerMatches[minuteIndex],
		Second:  headerMatches[secondIndex],
		Host:    headerMatches[hostIndex],
		Process: postfixProcess,
	}

	switch string(postfixProcess) {
	case "smtp":
		return parseSmtpPayload(header, linePayload)
	default:
		// TODO: implement support for other non-smtp processes
		return RawRecord{Header: header}, UnsupportedLogLineError
	}
}

func parseSmtpPayload(header RawHeader, linePayload []byte) (RawRecord, error) {
	payloadMatches := possibleSmtpPayloadsRegexp.FindSubmatch(linePayload)

	if len(payloadMatches) == 0 {
		return RawRecord{}, UnsupportedLogLineError
	}

	if len(payloadMatches[messageSentWithStatusIndex]) == 0 {
		// TODO: implement other stuff done by the "smtp" process
		return RawRecord{}, UnsupportedLogLineError
	}

	s := RawSmtpSentStatus{
		Queue:               payloadMatches[smtpQueueIndex],
		RecipientLocalPart:  payloadMatches[smtpRecipientLocalPartIndex],
		RecipientDomainPart: payloadMatches[smtpRecipientDomainPartIndex],
		RelayName:           payloadMatches[smtpRelayNameIndex],
		RelayIp:             payloadMatches[smtpRelayIpIndex],
		RelayPort:           payloadMatches[smtpRelayPortIndex],
		Delay:               payloadMatches[smtpDelayIndex],
		Delays: [5][]byte{payloadMatches[smtpDelaysIndex],
			payloadMatches[smtpDelays0Index],
			payloadMatches[smtpDelays1Index],
			payloadMatches[smtpDelays2Index],
			payloadMatches[smtpDelays3Index]},
		Dsn:          payloadMatches[smtpDsnIndex],
		Status:       payloadMatches[smtpStatusIndex],
		ExtraMessage: payloadMatches[smtpExtraMessageIndex],
	}

	return RawRecord{
		Header:            header,
		PayloadType:       PayloadTypeSmtpMessageStatus,
		RawSmtpSentStatus: s,
	}, nil
}
