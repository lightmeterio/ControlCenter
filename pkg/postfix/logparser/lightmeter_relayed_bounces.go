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

type LightmeterRelayedBounce rawparser.LightmeterRelayedBounce

func (LightmeterRelayedBounce) isPayload() {
	// required by interface Payload
}

func convertRelayedBounce(r rawparser.RawPayload) (Payload, error) {
	return LightmeterRelayedBounce(r.LightmeterRelayedBounce), nil
}
