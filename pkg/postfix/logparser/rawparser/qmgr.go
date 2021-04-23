// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawparser

func init() {
	registerHandler("postfix", "qmgr", parseQmgrPayload)
}

type QmgrMessageExpired struct {
	Queue            []byte
	SenderLocalPart  []byte
	SenderDomainPart []byte
	Message          []byte
}

type QmgrMailQueued struct {
	Queue            []byte
	SenderLocalPart  []byte
	SenderDomainPart []byte
	Size             []byte
	Nrcpt            []byte
}

type QmgrRemoved struct {
	Queue []byte
}

func parseQmgrPayload(payloadLine []byte) (RawPayload, error) {
	if s, parsed := parseQmgrMailQueued(payloadLine); parsed {
		return RawPayload{
			PayloadType:    PayloadTypeQmgrMailQueued,
			QmgrMailQueued: s,
		}, nil
	}

	if s, parsed := parseQmgrRemoved(payloadLine); parsed {
		return RawPayload{
			PayloadType: PayloadTypeQmgrRemoved,
			QmgrRemoved: s,
		}, nil
	}

	if s, parsed := parseQmgrMessageExpired(payloadLine); parsed {
		return RawPayload{
			PayloadType:        PayloadTypeQmgrMessageExpired,
			QmgrMessageExpired: s,
		}, nil
	}

	return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
}
