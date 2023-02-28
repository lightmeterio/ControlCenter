// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
)

func init() {
	registerHandler(rawparser.PayloadTypeLightmeterDumpedHeader, convertDumpedHeader)
}

type LightmeterDumpedHeader rawparser.LightmeterDumpedHeader

func (LightmeterDumpedHeader) isPayload() {
	// required by interface Payload
}

func convertDumpedHeader(r rawparser.RawPayload) (Payload, error) {
	return LightmeterDumpedHeader(r.LightmeterDumpedHeader), nil
}
