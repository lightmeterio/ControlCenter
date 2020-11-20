package rawparser

func init() {
	registerHandler("postfix", "qmgr", parseQmgrPayload)
}

type QmgrReturnedToSender struct {
	Queue            []byte
	SenderLocalPart  []byte
	SenderDomainPart []byte
}

func parseQmgrPayload(header RawHeader, payloadLine []byte) (RawPayload, error) {
	s, parsed := parseQmgrReturnedToSender(payloadLine)

	if !parsed {
		return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
	}

	return RawPayload{
		PayloadType:          PayloadTypeQmgrReturnedToSender,
		QmgrReturnedToSender: s,
	}, nil
}
