package parser

import (
	"encoding/hex"
	"gitlab.com/lightmeter/postfix-logs-parser/rawparser"
	"net"
	"strconv"
	"time"
)

func atoi(s []byte) (int, error) {
	return strconv.Atoi(string(s))
}

func atof(s []byte) (float32, error) {
	r, e := strconv.ParseFloat(string(s), 32)

	if e == nil {
		return float32(r), nil
	}

	return 0, e
}

func parseMonth(m []byte) time.Month {
	switch string(m) {
	case "Jan":
		return 1
	case "Feb":
		return 2
	case "Mar":
		return 3
	case "Apr":
		return 4
	case "May":
		return 5
	case "Jun":
		return 6
	case "Jul":
		return 7
	case "Aug":
		return 8
	case "Sep":
		return 9
	case "Oct":
		return 10
	case "Nov":
		return 11
	case "Dec":
		return 12
	}

	panic("Invalid Month! " + string(m))
}

func parseProcess(p []byte) (Process, error) {
	switch string(p) {
	case "smtp":
		return SmtpProcess, nil
	}

	return 0, rawparser.UnsupportedLogLineError
}

type Payload interface {
	isPayload()
}

type Time struct {
	Day    uint8
	Month  time.Month
	Hour   uint8
	Minute uint8
	Second uint8
}

const (
	SmtpProcess = iota
)

type Process int

type Header struct {
	Time    Time
	Host    string
	Process Process
}

type Record struct {
	Header  Header
	Payload Payload
}

func Parse(line []byte) (Record, error) {
	p, err := rawparser.ParseLogLine(line)

	if err != nil {
		return Record{}, err
	}

	day, err := atoi(p.Header.Day)

	if err != nil {
		return Record{}, err
	}

	hour, err := atoi(p.Header.Hour)

	if err != nil {
		return Record{}, err
	}

	minute, err := atoi(p.Header.Minute)

	if err != nil {
		return Record{}, err
	}

	second, err := atoi(p.Header.Second)

	if err != nil {
		return Record{}, err
	}

	process, err := parseProcess(p.Header.Process)

	if err != nil {
		return Record{}, err
	}

	h := Header{
		Time: Time{
			Day:    uint8(day),
			Month:  parseMonth(p.Header.Month),
			Hour:   uint8(hour),
			Minute: uint8(minute),
			Second: uint8(second),
		},
		Host:    string(p.Header.Host),
		Process: process,
	}

	switch p.Payload.(type) {
	case rawparser.RawSmtpSentStatus:
		p, err := convertSmtpSentStatus(p.Payload.(rawparser.RawSmtpSentStatus))
		if err != nil {
			return Record{}, err
		}
		return Record{Header: h, Payload: p}, nil
	}

	return Record{}, rawparser.UnsupportedLogLineError
}

type Delays struct {
	Smtpd   float32
	Cleanup float32
	Qmgr    float32
	Smtp    float32
}

type SmtpStatus int

const (
	SentStatus = iota
	BouncedStatus
	DeferredStatus
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

	ip := net.ParseIP(string(p.RelayIp))

	if ip == nil {
		return SmtpSentStatus{}, &net.ParseError{}
	}

	relayPort, err := atoi(p.RelayPort)

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

	return SmtpSentStatus{
		Queue:               q,
		RecipientLocalPart:  string(p.RecipientLocalPart),
		RecipientDomainPart: string(p.RecipientDomainPart),
		RelayName:           string(p.RelayName),
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
