package parser

import (
	"encoding/hex"
	"gitlab.com/lightmeter/postfix-log-parser/rawparser"
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

func parsePostfixProcess(p []byte) (Process, error) {
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

func parseHeader(h rawparser.RawHeader) (Header, error) {
	day, err := atoi(h.Day)

	if err != nil {
		return Header{}, err
	}

	hour, err := atoi(h.Hour)

	if err != nil {
		return Header{}, err
	}

	minute, err := atoi(h.Minute)

	if err != nil {
		return Header{}, err
	}

	second, err := atoi(h.Second)

	if err != nil {
		return Header{}, err
	}

	process, err := parsePostfixProcess(h.Process)

	if err != nil {
		return Header{}, err
	}

	return Header{
		Time: Time{
			Day:    uint8(day),
			Month:  parseMonth(h.Month),
			Hour:   uint8(hour),
			Minute: uint8(minute),
			Second: uint8(second),
		},
		Host:    string(h.Host),
		Process: process,
	}, nil
}

func Parse(line []byte) (Record, error) {
	p, err := rawparser.ParseLogLine(line)

	if err != nil {
		return Record{}, err
	}

	h, err := parseHeader(p.Header)

	if err != nil {
		return Record{}, err
	}

	switch p.PayloadType {
	case rawparser.PayloadTypeSmtpMessageStatus:
		p, err := convertSmtpSentStatus(p.RawSmtpSentStatus)
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
	SentStatus     SmtpStatus = 0
	BouncedStatus  SmtpStatus = 1
	DeferredStatus SmtpStatus = 2
)

func (this SmtpStatus) String() string {
	return strconv.FormatInt(int64(this), 10)
}

var (
	smtpStatusHumanForm = map[SmtpStatus]string{
		DeferredStatus: "deferred",
		BouncedStatus:  "bounced",
		SentStatus:     "sent",
	}
)

func (this SmtpStatus) HumanForm() string {
	return smtpStatusHumanForm[this]
}

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
