// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
)

func init() {
	registerHandler(rawparser.PayloadTypeLightmeterRelayedBounce, convertRelayedBounce)
}

type LightmeterRelayedBounce struct {
	Queue           string
	Sender          string
	Recipient       string
	DeliveryCode    string
	DeliveryMessage string
	ReportingMTA    string
}

func (LightmeterRelayedBounce) isPayload() {
	// required by interface Payload
}

func convertRelayedBounce(r rawparser.RawPayload) (Payload, error) {
	return LightmeterRelayedBounce{
		Queue:           r.LightmeterRelayedBounce.Queue,
		Sender:          r.LightmeterRelayedBounce.Sender,
		Recipient:       r.LightmeterRelayedBounce.Recipient,
		DeliveryCode:    r.LightmeterRelayedBounce.DeliveryCode,
		DeliveryMessage: r.LightmeterRelayedBounce.DeliveryMessage,
		ReportingMTA:    r.LightmeterRelayedBounce.ReportingMTA,
	}, nil
}
