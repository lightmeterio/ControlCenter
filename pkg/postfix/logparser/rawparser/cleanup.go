// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawparser

func init() {
	registerHandler("postfix", "sender-cleanup/cleanup", parseCleanup)
	registerHandler("postfix", "cleanup", parseCleanup)
}

type CleanupMessageAccepted struct {
	Queue     []byte
	MessageId []byte
	Corrupted bool
}

func parseCleanup(header RawHeader, payloadLine []byte) (RawPayload, error) {
	if s, parsed := parseCleanupMessageAccepted(payloadLine); parsed {
		return RawPayload{
			PayloadType:           PayloadTypeCleanupMessageAccepted,
			CleanupMesageAccepted: s,
		}, nil
	}

	if s, parsed := parseCleanupMilterReject(payloadLine); parsed {
		return RawPayload{
			PayloadType:         PayloadTypeCleanupMilterReject,
			CleanupMilterReject: s,
		}, nil
	}

	return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine

}

type CleanupMilterReject struct {
	Queue        []byte
	ExtraMessage []byte
}
