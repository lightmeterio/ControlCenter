// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package parser

import (
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
)

func init() {
	registerHandler(rawparser.PayloadTypeBounceCreated, convertBounceCreated)
}

type BounceCreated struct {
	Queue      string
	ChildQueue string
}

func (BounceCreated) isPayload() {
	// required by interface Payload
}

func convertBounceCreated(r rawparser.RawPayload) (Payload, error) {
	p := r.BounceCreated

	return BounceCreated{
		Queue:      string(p.Queue),
		ChildQueue: string(p.ChildQueue),
	}, nil
}
