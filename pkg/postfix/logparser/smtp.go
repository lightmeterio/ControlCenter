// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"errors"
	"net"

	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
)

func init() {
	registerHandler(rawparser.PayloadTypeSmtpMessageStatus, convertSmtpSentStatus)
}

type Delays struct {
	Smtpd   float32
	Cleanup float32
	Qmgr    float32
	Smtp    float32
}

type SmtpStatus int

var (
	smtpStatusHumanForm = map[SmtpStatus]string{
		DeferredStatus: "deferred",
		BouncedStatus:  "bounced",
		SentStatus:     "sent",
		ExpiredStatus:  "expired",
		ReturnedStatus: "returned",
	}
)

func (s SmtpStatus) String() string {
	return smtpStatusHumanForm[s]
}

const (
	SentStatus     SmtpStatus = 0
	BouncedStatus  SmtpStatus = 1
	DeferredStatus SmtpStatus = 2
	ExpiredStatus  SmtpStatus = 3
	ReturnedStatus SmtpStatus = 4
	ReceivedStatus SmtpStatus = 5
)

type SmtpSentStatus struct {
	Queue                   string
	RecipientLocalPart      string
	RecipientDomainPart     string
	OrigRecipientLocalPart  string
	OrigRecipientDomainPart string
	RelayName               string
	RelayPath               string
	RelayIP                 net.IP
	RelayPort               uint16
	Delay                   float32
	Delays                  Delays
	Dsn                     string
	Status                  SmtpStatus
	ExtraMessage            string
	ExtraMessagePayload     Payload
}

func (SmtpSentStatus) isPayload() {
	// required by Payload interface
}

type SmtpSentStatusExtraMessageSentQueued struct {
	SmtpCode int
	Dsn      string
	IP       net.IP
	Port     int
	Queue    string
}

func (SmtpSentStatusExtraMessageSentQueued) isPayload() {
	// required by Payload interface
}

var ErrInvalidStatus = errors.New(`Invalid Status`)

func ParseStatus(s string) (SmtpStatus, error) {
	switch s {
	case "deferred":
		return DeferredStatus, nil
	case "sent":
		return SentStatus, nil
	case "bounced":
		return BouncedStatus, nil
	case "expired":
		return ExpiredStatus, nil
	case "returned":
		return ReturnedStatus, nil
	}

	return 0, ErrInvalidStatus
}

func convertSmtpSentStatus(r rawparser.RawPayload) (Payload, error) {
	p := r.RawSmtpSentStatus

	relayIp, relayPath := func() (net.IP, string) {
		ip, err := parseIP(p.RelayIpOrPath)
		if err == nil {
			return ip, ""
		}

		if len(p.RelayIpOrPath) == 0 {
			return nil, ""
		}

		return nil, p.RelayIpOrPath
	}()

	relayPort, err := func() (int, error) {
		if len(p.RelayPort) == 0 {
			return 0, nil
		}

		return atoi(p.RelayPort)
	}()

	if err != nil {
		return SmtpSentStatus{}, err
	}

	delay, err := atof(p.Delay)

	if err != nil {
		return SmtpSentStatus{}, err
	}

	smtpdDelay, err := atof(p.Delays[1])

	if err != nil {
		return SmtpSentStatus{}, err
	}

	cleanupDelay, err := atof(p.Delays[2])

	if err != nil {
		return SmtpSentStatus{}, err
	}

	qmgrDelay, err := atof(p.Delays[3])

	if err != nil {
		return SmtpSentStatus{}, err
	}

	smtpDelay, err := atof(p.Delays[4])

	if err != nil {
		return SmtpSentStatus{}, err
	}

	relayName := func() string {
		if len(p.RelayName) == 0 {
			return ""
		}

		r := p.RelayName

		if r == "none" {
			return ""
		}

		return r
	}()

	parsedExtraMessage, err := parseSmtpSentStatusExtraMessage(p)
	if err != nil {
		return SmtpSentStatus{}, err
	}

	status, err := ParseStatus(p.Status)
	if err != nil {
		return SmtpSentStatus{}, err
	}

	return SmtpSentStatus{
		Queue:                   p.Queue,
		RecipientLocalPart:      p.RecipientLocalPart,
		RecipientDomainPart:     p.RecipientDomainPart,
		OrigRecipientLocalPart:  p.OrigRecipientLocalPart,
		OrigRecipientDomainPart: p.OrigRecipientDomainPart,
		RelayName:               relayName,
		RelayPath:               relayPath,
		RelayIP:                 relayIp,
		RelayPort:               uint16(relayPort),
		Delay:                   delay,
		Delays: Delays{
			Smtpd:   smtpdDelay,
			Cleanup: cleanupDelay,
			Qmgr:    qmgrDelay,
			Smtp:    smtpDelay,
		},
		Dsn:                 p.Dsn,
		Status:              status,
		ExtraMessage:        p.ExtraMessage,
		ExtraMessagePayload: parsedExtraMessage,
	}, nil
}

func parseSmtpSentStatusExtraMessage(s rawparser.RawSmtpSentStatus) (Payload, error) {
	if s.ExtraMessagePayloadType != rawparser.PayloadTypeSmtpMessageStatusSentQueued {
		return nil, nil
	}

	p := s.ExtraMessageSmtpSentStatusSentQueued

	optionalAtoi := func(v string) (int, error) {
		if len(v) == 0 {
			return 0, nil
		}

		return atoi(v)
	}

	smtpCode, err := optionalAtoi(p.SmtpCode)
	if err != nil {
		return nil, err
	}

	port, err := optionalAtoi(p.Port)
	if err != nil {
		return nil, err
	}

	ip, err := parseIP(p.IP)
	if err != nil {
		return nil, err
	}

	return SmtpSentStatusExtraMessageSentQueued{
		Dsn:      p.Dsn,
		IP:       ip,
		Port:     port,
		Queue:    p.Queue,
		SmtpCode: smtpCode,
	}, nil
}
