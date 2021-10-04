// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
)

func init() {
	registerHandler(rawparser.PayloadTypeVersion, convertVersion)
}

type Version string

func (Version) isPayload() {
	// required by interface Payload
}

func convertVersion(r rawparser.RawPayload) (Payload, error) {
	return Version(r.Version), nil
}
