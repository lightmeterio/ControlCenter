package parser

import (
	"gitlab.com/lightmeter/postfix-log-parser/rawparser"
)

type QmgrReturnedToSender struct {
	Queue            string
	SenderLocalPart  string
	SenderDomainPart string
}

func (QmgrReturnedToSender) isPayload() {
}

func convertQmgrReturnedToSender(p rawparser.QmgrReturnedToSender) (QmgrReturnedToSender, error) {
	return QmgrReturnedToSender{
		Queue:            string(p.Queue),
		SenderLocalPart:  string(p.SenderLocalPart),
		SenderDomainPart: string(p.SenderDomainPart),
	}, nil
}
