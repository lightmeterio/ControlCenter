package parser

import (
	"gitlab.com/lightmeter/postfix-log-parser/rawparser"
)

func init() {
	registerHandler(rawparser.PayloadTypeQmgrReturnedToSender, convertQmgrReturnedToSender)
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
