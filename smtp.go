package parser

import (
	"encoding/hex"
	"gitlab.com/lightmeter/postfix-log-parser/rawparser"
	"net"
)

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
	}
)

func (this SmtpStatus) String() string {
	return smtpStatusHumanForm[this]
}

const (
	SentStatus     SmtpStatus = 0
	BouncedStatus  SmtpStatus = 1
	DeferredStatus SmtpStatus = 2
)

type SmtpSentStatus struct {
	Queue               []byte
	RecipientLocalPart  string
	RecipientDomainPart string
	RelayName           string
	RelayIP             net.IP
	RelayPort           uint16
	Delay               float32
	Delays              Delays
	Dsn                 string
	Status              SmtpStatus
	ExtraMessage        string
}

func (SmtpSentStatus) isPayload() {
}

func parseStatus(s []byte) SmtpStatus {
	switch string(s) {
	case "deferred":
		return DeferredStatus
	case "sent":
		return SentStatus
	case "bounced":
		return BouncedStatus
	}

	panic("Ahhh, invalid status!!!" + string(s))
}

func convertSmtpSentStatus(p rawparser.RawSmtpSentStatus) (SmtpSentStatus, error) {
	q, err := hex.DecodeString(string(p.Queue))

	if err != nil {
		return SmtpSentStatus{}, err
	}

	ip, err := func() (net.IP, error) {
		if len(p.RelayIp) == 0 {
			return nil, nil
		}

		ip := net.ParseIP(string(p.RelayIp))

		if ip == nil {
			return nil, &net.ParseError{Type: "IP Address", Text: "Invalid Relay IP"}
		}

		return ip, nil
	}()

	if err != nil {
		return SmtpSentStatus{}, err
	}

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

		return string(p.RelayName)
	}()

	return SmtpSentStatus{
		Queue:               q,
		RecipientLocalPart:  string(p.RecipientLocalPart),
		RecipientDomainPart: string(p.RecipientDomainPart),
		RelayName:           relayName,
		RelayIP:             ip,
		RelayPort:           uint16(relayPort),
		Delay:               delay,
		Delays: Delays{
			Smtpd:   smtpdDelay,
			Cleanup: cleanupDelay,
			Qmgr:    qmgrDelay,
			Smtp:    smtpDelay,
		},
		Dsn:          string(p.Dsn),
		Status:       parseStatus(p.Status),
		ExtraMessage: string(p.ExtraMessage),
	}, nil
}
