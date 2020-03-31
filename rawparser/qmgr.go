package rawparser

import (
	"regexp"
)

func init() {
	registerHandler("qmgr", parseQmgrPayload)
}

const (
	// TODO: I have the feeling this expression can be simplified a lot,
	// and started seeing that using a grammar based syntax instead of regexp would make it easier to write as well,
	// But I don't know how it'd be performance-wise
	mailSenderPartRegexpFormat = `((?P<NonQuotedSenderLocalPart>[^@"]+)|"(?P<QuotedSenderLocalPart>[^@"]+)")`

	qmgrPossiblePayloadsFormat = `(?P<MessageReturnedToSenderStatus>` +
		`from=<` + mailSenderPartRegexpFormat + `@(?P<SenderDomainPart>[^>]+)>` + `,\s` +
		`status=expired, returned to sender)`

	queueIdRawQmgrSentStatusRegexpFormat = `(?P<Queue>[0-9A-F]+)`

	qmgrPayloadsRegexpFormat = `^` + queueIdRawQmgrSentStatusRegexpFormat + `:\s` +
		`(` + qmgrPossiblePayloadsFormat + `)$`
)

type QmgrReturnedToSender struct {
	Queue            []byte
	SenderLocalPart  []byte
	SenderDomainPart []byte
}

var (
	qmgrPossiblePayloadsRegexp *regexp.Regexp

	qmgrMessageSentWithStatusIndex    int
	qmgrQueueIndex                    int
	qmgrNonQuotedSenderLocalPartIndex int
	qmgrQuotedSenderLocalPartIndex    int
	qmgrSenderDomainPartIndex         int
	qmgrStatusIndex                   int
)

func init() {
	qmgrPossiblePayloadsRegexp = regexp.MustCompile(qmgrPayloadsRegexpFormat)

	qmgrMessageSentWithStatusIndex = indexForGroup(qmgrPossiblePayloadsRegexp, "MessageReturnedToSenderStatus")
	qmgrQueueIndex = indexForGroup(qmgrPossiblePayloadsRegexp, "Queue")
	qmgrNonQuotedSenderLocalPartIndex = indexForGroup(qmgrPossiblePayloadsRegexp, "NonQuotedSenderLocalPart")
	qmgrQuotedSenderLocalPartIndex = indexForGroup(qmgrPossiblePayloadsRegexp, "QuotedSenderLocalPart")
	qmgrSenderDomainPartIndex = indexForGroup(qmgrPossiblePayloadsRegexp, "SenderDomainPart")
}

func parseQmgrPayload(header RawHeader, payloadLine []byte) (RawPayload, error) {
	payloadMatches := qmgrPossiblePayloadsRegexp.FindSubmatch(payloadLine)

	if len(payloadMatches) == 0 {
		return RawPayload{PayloadType: PayloadTypeUnsupported}, UnsupportedLogLineError
	}

	if len(payloadMatches[qmgrMessageSentWithStatusIndex]) == 0 {
		// TODO: implement other stuff done by the "qmgr" process
		return RawPayload{PayloadType: PayloadTypeUnsupported}, UnsupportedLogLineError
	}

	senderLocalPart := func() []byte {
		if len(payloadMatches[qmgrNonQuotedSenderLocalPartIndex]) > 0 {
			return payloadMatches[qmgrNonQuotedSenderLocalPartIndex]
		}

		return payloadMatches[qmgrQuotedSenderLocalPartIndex]
	}()

	s := QmgrReturnedToSender{
		Queue:            payloadMatches[qmgrQueueIndex],
		SenderLocalPart:  senderLocalPart,
		SenderDomainPart: payloadMatches[qmgrSenderDomainPartIndex],
	}

	return RawPayload{
		PayloadType:          PayloadTypeQmgrReturnedToSender,
		QmgrReturnedToSender: s,
	}, nil
}
