// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawparser

func init() {
	registerHandler("postfix", "qmgr", parseQmgrPayload)
}

type QmgrMessageExpired struct {
	Queue            string
	SenderLocalPart  string
	SenderDomainPart string
	Message          string
}

type QmgrMailQueued struct {
	Queue            string
	SenderLocalPart  string
	SenderDomainPart string
	Size             string
	Nrcpt            string
}

type QmgrRemoved struct {
	Queue string
}

func parseQmgrPayload(payloadLine string) (RawPayload, error) {
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
