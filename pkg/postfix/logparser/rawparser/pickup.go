// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawparser

func init() {
	registerHandler("postfix", "pickup", parsePickupPayload)
}

type Pickup struct {
	Queue  string
	Uid    string
	Sender string
}

func parsePickupPayload(payloadLine string) (RawPayload, error) {
	if s, parsed := parsePickup(payloadLine); parsed {
		return RawPayload{
			PayloadType: PayloadTypePickup,
			Pickup:      s,
		}, nil
	}

	return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
}
