// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package parser

import (
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func init() {
	registerHandler(rawparser.PayloadTypeQmgrReturnedToSender, convertQmgrReturnedToSender)
	registerHandler(rawparser.PayloadTypeQmgrMailQueued, convertQmgrMailQueued)
	registerHandler(rawparser.PayloadTypeQmgrRemoved, convertQmgrRemoved)
}

type QmgrReturnedToSender struct {
	Queue            string
	SenderLocalPart  string
	SenderDomainPart string
}

func (QmgrReturnedToSender) isPayload() {
	// required by interface Payload
}

func convertQmgrReturnedToSender(r rawparser.RawPayload) (Payload, error) {
	p := r.QmgrReturnedToSender

	return QmgrReturnedToSender{
		Queue:            string(p.Queue),
		SenderLocalPart:  string(p.SenderLocalPart),
		SenderDomainPart: string(p.SenderDomainPart),
	}, nil
}

type QmgrMailQueued struct {
	Queue            string
	SenderLocalPart  string
	SenderDomainPart string
	Size             int
	Nrcpt            int
}

func (QmgrMailQueued) isPayload() {
	// required by interface Payload
}

func convertQmgrMailQueued(r rawparser.RawPayload) (Payload, error) {
	p := r.QmgrMailQueued

	size, err := atoi(p.Size)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	nrcpt, err := atoi(p.Nrcpt)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return QmgrMailQueued{
		Queue:            string(p.Queue),
		SenderLocalPart:  string(p.SenderLocalPart),
		SenderDomainPart: string(p.SenderDomainPart),
		Size:             size,
		Nrcpt:            nrcpt,
	}, nil
}

type QmgrRemoved struct {
	Queue string
}

func (QmgrRemoved) isPayload() {
	// required by interface Payload
}

func convertQmgrRemoved(r rawparser.RawPayload) (Payload, error) {
	p := r.QmgrRemoved

	return QmgrRemoved{
		Queue: string(p.Queue),
	}, nil
}
