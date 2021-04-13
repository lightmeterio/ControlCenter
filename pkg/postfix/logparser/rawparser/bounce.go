// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawparser

func init() {
	registerHandler("postfix", "bounce", parseBounce)
}

type BounceCreated struct {
	Queue      []byte
	ChildQueue []byte
}

func parseBounce(payloadLine []byte) (RawPayload, error) {
	if s, parsed := parseBounceCreated(payloadLine); parsed {
		return RawPayload{
			PayloadType:   PayloadTypeBounceCreated,
			BounceCreated: s,
		}, nil
	}

	return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
}
