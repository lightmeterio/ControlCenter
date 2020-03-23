package rawparser

import (
	"regexp"
)

const (
	PayloadTypeSmtpMessageStatus PayloadType = iota
)

const (
	queueIdRawSmtpSentStatusRegexpFormat = `(?P<Queue>[0-9A-F]+)`

	anythingExceptCommaRegexpFormat = `[^,]+`

	// NOTE: Relay name might be absent, having only "none"
	relayComponentsRegexpFormat = `((?P<RelayName>[^\,[]+)` + `\[(?P<RelayIp>[^\],]+)\]` + `:` + `(?P<RelayPort>[\d]+)|` + `none)`

	// TODO: I have the feeling this expression can be simplified a lot,
	// and started seeing that using a grammar based syntax instead of regexp would make it easier to write as well
	// But I don't know how it's be performance-wise
	mailRecipientPartRegexpFormat = `((?P<NonQuotedRecipientLocalPart>[^@"]+)|"(?P<QuotedRecipientLocalPart>[^@"]+)")`

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

	smtpMessageSentWithStatusIndex       int
	smtpQueueIndex                       int
	smtpNonQuotedRecipientLocalPartIndex int
	smtpQuotedRecipientLocalPartIndex    int
	smtpRecipientDomainPartIndex         int
	smtpRelayNameIndex                   int
	smtpRelayIpIndex                     int
	smtpRelayPortIndex                   int
	smtpDelayIndex                       int
	smtpDelaysIndex                      int
	smtpDelays0Index                     int
	smtpDelays1Index                     int
	smtpDelays2Index                     int
	smtpDelays3Index                     int
	smtpDsnIndex                         int
	smtpStatusIndex                      int
	smtpExtraMessageIndex                int
)

func init() {
	smtpPossiblePayloadsRegexp = regexp.MustCompile(smtpPayloadsRegexpFormat)

	smtpMessageSentWithStatusIndex = indexForGroup(smtpPossiblePayloadsRegexp, "MessageSentWithStatus")
	smtpQueueIndex = indexForGroup(smtpPossiblePayloadsRegexp, "Queue")
	smtpNonQuotedRecipientLocalPartIndex = indexForGroup(smtpPossiblePayloadsRegexp, "NonQuotedRecipientLocalPart")
	smtpQuotedRecipientLocalPartIndex = indexForGroup(smtpPossiblePayloadsRegexp, "QuotedRecipientLocalPart")
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

func parseSmtpPayload(header RawHeader, payloadLine []byte) (RawRecord, error) {
	payloadMatches := smtpPossiblePayloadsRegexp.FindSubmatch(payloadLine)

	if len(payloadMatches) == 0 {
		return RawRecord{}, UnsupportedLogLineError
	}

	if len(payloadMatches[smtpMessageSentWithStatusIndex]) == 0 {
		// TODO: implement other stuff done by the "smtp" process
		return RawRecord{}, UnsupportedLogLineError
	}

	recipientLocalPart := func() []byte {
		if len(payloadMatches[smtpNonQuotedRecipientLocalPartIndex]) > 0 {
			return payloadMatches[smtpNonQuotedRecipientLocalPartIndex]
		}

		return payloadMatches[smtpQuotedRecipientLocalPartIndex]
	}()

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

	return RawRecord{
		Header:            header,
		PayloadType:       PayloadTypeSmtpMessageStatus,
		RawSmtpSentStatus: s,
	}, nil
}
