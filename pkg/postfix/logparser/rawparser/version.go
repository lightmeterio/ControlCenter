// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawparser

func init() {
	registerHandler("postfix", "master", parseVersionPayload)
}

type Version []byte

func parseVersionPayload(payloadLine []byte) (RawPayload, error) {
	if version, parsed := parseVersion(payloadLine); parsed {
		return RawPayload{
			PayloadType: PayloadTypeVersion,
			Version:     version,
		}, nil
	}

	return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
}
