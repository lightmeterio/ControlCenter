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

type LightmeterDumpedHeader struct {
	Queue string
	Key   string
	Value string
}

func (LightmeterDumpedHeader) isPayload() {
	// required by interface Payload
}

func convertDumpedHeader(r rawparser.RawPayload) (Payload, error) {
	return LightmeterDumpedHeader{
		Key:   r.LightmeterDumpedHeader.Key,
		Value: r.LightmeterDumpedHeader.Value,
		Queue: r.LightmeterDumpedHeader.Queue,
	}, nil
}
