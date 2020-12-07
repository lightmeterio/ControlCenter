package rawparser

func init() {
	registerHandler("postfix", "smtp", parseSmtpPayload)
}

type RawSmtpSentStatus struct {
	Queue                   []byte
	RecipientLocalPart      []byte
	RecipientDomainPart     []byte
	OrigRecipientLocalPart  []byte
	OrigRecipientDomainPart []byte
	RelayName               []byte
	RelayIp                 []byte
	RelayPort               []byte
	Delay                   []byte
	Delays                  [5][]byte
	Dsn                     []byte
	Status                  []byte
	ExtraMessage            []byte
}

func parseSmtpPayload(header RawHeader, payloadLine []byte) (RawPayload, error) {
	r, parsed := parseSmtpSentStatus(payloadLine)

	if !parsed {
		return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
	}

	return RawPayload{
		PayloadType:       PayloadTypeSmtpMessageStatus,
		RawSmtpSentStatus: r,
	}, nil
}
