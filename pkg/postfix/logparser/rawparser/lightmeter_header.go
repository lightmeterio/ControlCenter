// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawparser

func init() {
	registerHandler("lightmeter", "headers", parseDumpHeaderPayload)
}

type LightmeterDumpedHeader struct {
	Key   string
	Value string
	Queue string
}

func parseDumpHeaderPayload(payloadLine string) (RawPayload, error) {
	if p, parsed := parseDumpedHeader(payloadLine); parsed {
		return RawPayload{
			PayloadType:            PayloadTypeLightmeterDumpedHeader,
			LightmeterDumpedHeader: p,
		}, nil
	}

	return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
}
