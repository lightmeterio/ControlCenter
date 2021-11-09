// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
	"net"
)

func init() {
	registerHandler(rawparser.PayloadTypeSmtpdConnect, convertSmtpdConnect)
	registerHandler(rawparser.PayloadTypeSmtpdDisconnect, convertSmtpdDisconnect)
	registerHandler(rawparser.PayloadTypeSmtpdMailAccepted, convertSmtpdMailAccepted)
	registerHandler(rawparser.PayloadTypeSmtpdReject, convertSmtpdReject)
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
		Host: p.Host,
		IP:   ip,
	}, nil
}

type SmtpdDisconnect struct {
	Host  string
	IP    net.IP
	Stats map[string]SmtpdDisconnectStat
}

type SmtpdDisconnectStat = rawparser.SmtpdDisconnectStat

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
		Host:  p.Host,
		IP:    ip,
		Stats: p.Stats,
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
		Host:  p.Host,
		IP:    ip,
		Queue: p.Queue,
	}, nil
}

type SmtpdReject struct {
	Queue        string
	ExtraMessage string
}

func (SmtpdReject) isPayload() {
	// required by Payload interface
}

func convertSmtpdReject(r rawparser.RawPayload) (Payload, error) {
	p := r.SmtpdReject

	return SmtpdReject{
		Queue:        p.Queue,
		ExtraMessage: p.ExtraMessage,
	}, nil
}
