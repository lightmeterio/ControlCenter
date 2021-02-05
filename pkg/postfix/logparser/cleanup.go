// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
)

func init() {
	registerHandler(rawparser.PayloadTypeCleanupMessageAccepted, convertCleanupMessageAccepted)
}

type CleanupMessageAccepted struct {
	Queue     string
	MessageId string
}

func (CleanupMessageAccepted) isPayload() {
	// required by interface Payload
}

func convertCleanupMessageAccepted(r rawparser.RawPayload) (Payload, error) {
	p := r.CleanupMesageAccepted

	return CleanupMessageAccepted{
		Queue:     string(p.Queue),
		MessageId: string(p.MessageId),
	}, nil
}
