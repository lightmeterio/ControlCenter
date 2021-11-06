// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

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
		Queue:      p.Queue,
		ChildQueue: p.ChildQueue,
	}, nil
}
