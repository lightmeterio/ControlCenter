// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func init() {
	registerHandler(rawparser.PayloadTypePickup, convertPickup)
}

type Pickup struct {
	Queue  string
	Uid    int
	Sender string
}

func (Pickup) isPayload() {
	// required by interface Payload
}

func convertPickup(r rawparser.RawPayload) (Payload, error) {
	p := r.Pickup

	uid, err := atoi(p.Uid)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return Pickup{
		Queue:  p.Queue,
		Uid:    uid,
		Sender: p.Sender,
	}, nil
}
