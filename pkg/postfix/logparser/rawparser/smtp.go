package rawparser

func init() {
	registerHandler("postfix", "smtp", parseSmtpPayload)
	registerHandler("postfix", "lmtp", parseSmtpPayload)
}

type RawSmtpSentStatus struct {
	Queue                   []byte
	RecipientLocalPart      []byte
	RecipientDomainPart     []byte
	OrigRecipientLocalPart  []byte
	OrigRecipientDomainPart []byte
	RelayName               []byte
	RelayIpOrPath           []byte
	RelayPort               []byte
	Delay                   []byte
	Delays                  [5][]byte
	Dsn                     []byte
	Status                  []byte
	ExtraMessage            []byte

	// parsed extra message
	ExtraMessagePayloadType              PayloadType
	ExtraMessageSmtpSentStatusSentQueued SmtpSentStatusExtraMessageSentQueued
}

type SmtpSentStatusExtraMessageSentQueued struct {
	SmtpCode []byte
	Dsn      []byte
	IP       []byte
	Port     []byte
	Queue    []byte
}

func parseSmtpPayload(header RawHeader, payloadLine []byte) (RawPayload, error) {
	r, parsed := parseSmtpSentStatus(payloadLine)

	if !parsed {
		return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
	}

	// TODO: refactor this code when mode kinds of extra messages are parsed
	if extraMessage, parsed := parseSmtpSentStatusExtraMessageSentQueued(r.ExtraMessage); parsed {
		r.ExtraMessageSmtpSentStatusSentQueued = extraMessage
		r.ExtraMessagePayloadType = PayloadTypeSmtpMessageStatusSentQueued
	}

	return RawPayload{
		PayloadType:       PayloadTypeSmtpMessageStatus,
		RawSmtpSentStatus: r,
	}, nil
}
