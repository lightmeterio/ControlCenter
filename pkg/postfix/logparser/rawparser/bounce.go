package rawparser

func init() {
	registerHandler("postfix", "bounce", parseBounce)
}

type BounceCreated struct {
	Queue      []byte
	ChildQueue []byte
}

func parseBounce(header RawHeader, payloadLine []byte) (RawPayload, error) {
	if s, parsed := parseBounceCreated(payloadLine); parsed {
		return RawPayload{
			PayloadType:   PayloadTypeBounceCreated,
			BounceCreated: s,
		}, nil
	}

	return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
}