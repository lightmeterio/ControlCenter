// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawparser

func init() {
	registerHandler("lightmeter", "relayed-bounce", parseRelayedBouncePayload)
}

type LightmeterRelayedBounce struct {
	Queue           string
	Sender          string
	Recipient       string
	DeliveryCode    string
	DeliveryMessage string
	ReportingMTA    string
}

func parseRelayedBouncePayload(payloadLine string) (RawPayload, error) {
	if p, parsed := parseRelayedBounce(payloadLine); parsed {
		return RawPayload{
			PayloadType:             PayloadTypeLightmeterRelayedBounce,
			LightmeterRelayedBounce: p,
		}, nil
	}

	return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
}
