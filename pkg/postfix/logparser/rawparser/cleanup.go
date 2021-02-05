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
}

func parseCleanup(header RawHeader, payloadLine []byte) (RawPayload, error) {
	s, parsed := parseCleanupMessageAccepted(payloadLine)

	if !parsed {
		return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
	}

	return RawPayload{
		PayloadType:           PayloadTypeCleanupMessageAccepted,
		CleanupMesageAccepted: s,
	}, nil
}
