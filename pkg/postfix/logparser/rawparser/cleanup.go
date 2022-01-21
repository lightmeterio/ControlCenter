// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawparser

func init() {
	registerHandler("postfix", "sender-cleanup/cleanup", parseCleanup)
	registerHandler("postfix", "cleanup", parseCleanup)
	registerHandler("postfix", "submission/cleanup", parseCleanup)
	registerHandler("postfix", "cleanupspam/cleanup", parseCleanup)
	registerHandler("postfix", "authclean/cleanup", parseCleanup)
}

type CleanupMessageAccepted struct {
	Queue     string
	MessageId string
	Corrupted bool
}

func parseCleanup(payloadLine string) (RawPayload, error) {
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
	Queue        string
	ExtraMessage string
}
