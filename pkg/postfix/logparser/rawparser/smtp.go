package rawparser

import (
	"regexp"
)

func init() {
	registerHandler("postfix", "smtp", parseSmtpPayload)
}

const (
	queueIdRawSmtpSentStatusRegexpFormat = `(?P<Queue>[0-9A-F]+)`

	anythingExceptCommaRegexpFormat = `[^,]+`

	// NOTE: Relay name might be absent, having only "none"
	relayComponentsRegexpFormat = `((?P<RelayName>[^\,[]+)` + `\[(?P<RelayIp>[^\],]+)\]` + `:` + `(?P<RelayPort>[\d]+)|` + `none)`

	mailRecipientPartRegexpFormat = `(?P<RecipientLocalPart>` + mailLocalPartRegexpFormat + `)`

	messageSentWithStatusRawSmtpSentStatusRegexpFormat = `(?P<MessageSentWithStatus>` +
		`to=<` + mailRecipientPartRegexpFormat + `@(?P<RecipientDomainPart>[^>]+)>` + `,\s` +
		`relay=` + relayComponentsRegexpFormat + `,\s` +
		`delay=(?P<Delay>` + anythingExceptCommaRegexpFormat + `)` + `,\s` +
		`delays=(?P<Delays>(?P<Delays0>[^/]+)/(?P<Delays1>[^/]+)/(?P<Delays2>[^/]+)/(?P<Delays3>[^/]+))` + `,\s` +
		`dsn=(?P<Dsn>` + anythingExceptCommaRegexpFormat + `)` + `,\s` +
		`status=(?P<Status>(deferred|bounced|sent))` + `\s` +
		`(?P<ExtraMessage>.*)` +
		`)`

	smtpPossiblePayloadsFormat = messageSentWithStatusRawSmtpSentStatusRegexpFormat

	smtpPayloadsRegexpFormat = `^` + queueIdRawSmtpSentStatusRegexpFormat + `:\s` +
		`(` + smtpPossiblePayloadsFormat + `)$`
)

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

var (
	smtpPossiblePayloadsRegexp *regexp.Regexp

	smtpMessageSentWithStatusIndex int
	smtpQueueIndex                 int
	smtpRecipientLocalPartIndex    int
	smtpRecipientDomainPartIndex   int
	smtpRelayNameIndex             int
	smtpRelayIpIndex               int
	smtpRelayPortIndex             int
	smtpDelayIndex                 int
	smtpDelaysIndex                int
	smtpDelays0Index               int
	smtpDelays1Index               int
	smtpDelays2Index               int
	smtpDelays3Index               int
	smtpDsnIndex                   int
	smtpStatusIndex                int
	smtpExtraMessageIndex          int
)

func init() {
	smtpPossiblePayloadsRegexp = regexp.MustCompile(smtpPayloadsRegexpFormat)

	smtpMessageSentWithStatusIndex = indexForGroup(smtpPossiblePayloadsRegexp, "MessageSentWithStatus")
	smtpQueueIndex = indexForGroup(smtpPossiblePayloadsRegexp, "Queue")
	smtpRecipientLocalPartIndex = indexForGroup(smtpPossiblePayloadsRegexp, "RecipientLocalPart")
	smtpRecipientDomainPartIndex = indexForGroup(smtpPossiblePayloadsRegexp, "RecipientDomainPart")
	smtpRelayNameIndex = indexForGroup(smtpPossiblePayloadsRegexp, "RelayName")
	smtpRelayIpIndex = indexForGroup(smtpPossiblePayloadsRegexp, "RelayIp")
	smtpRelayPortIndex = indexForGroup(smtpPossiblePayloadsRegexp, "RelayPort")
	smtpDelayIndex = indexForGroup(smtpPossiblePayloadsRegexp, "Delay")
	smtpDelaysIndex = indexForGroup(smtpPossiblePayloadsRegexp, "Delays")
	smtpDelays0Index = indexForGroup(smtpPossiblePayloadsRegexp, "Delays0")
	smtpDelays1Index = indexForGroup(smtpPossiblePayloadsRegexp, "Delays1")
	smtpDelays2Index = indexForGroup(smtpPossiblePayloadsRegexp, "Delays2")
	smtpDelays3Index = indexForGroup(smtpPossiblePayloadsRegexp, "Delays3")
	smtpDsnIndex = indexForGroup(smtpPossiblePayloadsRegexp, "Dsn")
	smtpStatusIndex = indexForGroup(smtpPossiblePayloadsRegexp, "Status")
	smtpExtraMessageIndex = indexForGroup(smtpPossiblePayloadsRegexp, "ExtraMessage")
}

func parseSmtpPayload(header RawHeader, payloadLine []byte) (RawPayload, error) {
	payloadMatches := smtpPossiblePayloadsRegexp.FindSubmatch(payloadLine)

	if len(payloadMatches) == 0 {
		return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
	}

	if len(payloadMatches[smtpMessageSentWithStatusIndex]) == 0 {
		// TODO: implement other stuff done by the "smtp" process
		return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
	}

	recipientLocalPart := normalizeMailLocalPart(payloadMatches[smtpRecipientLocalPartIndex])

	s := RawSmtpSentStatus{
		Queue:               payloadMatches[smtpQueueIndex],
		RecipientLocalPart:  recipientLocalPart,
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

	return RawPayload{
		PayloadType:       PayloadTypeSmtpMessageStatus,
		RawSmtpSentStatus: s,
	}, nil
}
