// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package parser

import (
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
	"net"
)

func init() {
	registerHandler(rawparser.PayloadTypeSmtpdConnect, convertSmtpdConnect)
	registerHandler(rawparser.PayloadTypeSmtpdDisconnect, convertSmtpdDisconnect)
	registerHandler(rawparser.PayloadTypeSmtpdMailAccepted, convertSmtpdMailAccepted)
}

type SmtpdConnect struct {
	Host string
	IP   net.IP
}

func (SmtpdConnect) isPayload() {
	// required by Payload interface
}

func convertSmtpdConnect(r rawparser.RawPayload) (Payload, error) {
	p := r.SmtpdConnect

	ip, err := parseIP(p.IP)
	if err != nil {
		return SmtpSentStatus{}, err
	}

	return SmtpdConnect{
		Host: string(p.Host),
		IP:   ip,
	}, nil
}

type SmtpdDisconnect struct {
	Host string
	IP   net.IP
	// TODO: disconnect can have lots of optional extra data
	// that could be represented as a map[string]string
}

func (SmtpdDisconnect) isPayload() {
	// required by Payload interface
}

func convertSmtpdDisconnect(r rawparser.RawPayload) (Payload, error) {
	p := r.SmtpdDisconnect

	ip, err := parseIP(p.IP)
	if err != nil {
		return nil, err
	}

	return SmtpdDisconnect{
		Host: string(p.Host),
		IP:   ip,
	}, nil
}

type SmtpdMailAccepted struct {
	Queue string
	Host  string
	IP    net.IP
}

func (SmtpdMailAccepted) isPayload() {
	// required by Payload interface
}

func convertSmtpdMailAccepted(r rawparser.RawPayload) (Payload, error) {
	p := r.SmtpdMailAccepted

	ip, err := parseIP(p.IP)
	if err != nil {
		return SmtpSentStatus{}, err
	}

	return SmtpdMailAccepted{
		Host:  string(p.Host),
		IP:    ip,
		Queue: string(p.Queue),
	}, nil
}
