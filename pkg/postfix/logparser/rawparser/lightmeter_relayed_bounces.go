// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawparser

func init() {
	registerHandler("lightmeter", "relayed-bounce", parseRelayedBouncePayload)
}

type LightmeterRelayedBounce struct {
	Queue           string `json:"queue"`
	Sender          string `json:"sender"`
	Recipient       string `json:"recipient"`
	DeliveryCode    string `json:"delivery_code"`
	DeliveryMessage string `json:"delivery_message"`
	ReportingMTA    string `json:"reporting_mta"`
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
