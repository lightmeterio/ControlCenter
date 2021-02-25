// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
)

func init() {
	registerHandler(rawparser.PayloadTypeCleanupMessageAccepted, convertCleanupMessageAccepted)
	registerHandler(rawparser.PayloadTypeCleanupMilterReject, convertCleanupMilterReject)
}

type CleanupMessageAccepted struct {
	Queue     string
	Corrupted bool
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
		Corrupted: p.Corrupted,
	}, nil
}

type CleanupMilterReject struct {
	Queue        string
	ExtraMessage string
}

func (CleanupMilterReject) isPayload() {
	// required by interface Payload
}

func convertCleanupMilterReject(r rawparser.RawPayload) (Payload, error) {
	p := r.CleanupMilterReject

	return CleanupMilterReject{
		Queue:        string(p.Queue),
		ExtraMessage: string(p.ExtraMessage),
	}, nil
}
