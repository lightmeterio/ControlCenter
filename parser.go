package parser

import (
	"encoding/hex"
	"gitlab.com/lightmeter/postfix-logs-parser/rawparser"
	"net"
	"strconv"
	"time"
)

func atoi(s []byte) int {
	if r, e := strconv.Atoi(string(s)); e == nil {
		return r
	}

	panic("atoi error! " + string(s))
}

func atof(s []byte) float32 {
	if r, e := strconv.ParseFloat(string(s), 32); e == nil {
		return float32(r)
	}

	panic("atoi error! " + string(s))
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

func parseProcess(p []byte) Process {
	switch string(p) {
	case "smtp":
		return SmtpProcess
	}

	panic("Failed to parse process")
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

	h := Header{
		Time: Time{
			Day:    uint8(atoi(p.Header.Day)),
			Month:  parseMonth(p.Header.Month),
			Hour:   uint8(atoi(p.Header.Hour)),
			Minute: uint8(atoi(p.Header.Minute)),
			Second: uint8(atoi(p.Header.Second)),
		},
		Host:    string(p.Header.Host),
		Process: parseProcess(p.Header.Process),
	}

	switch p.Payload.(type) {
	case rawparser.RawSmtpSentStatus:
		return Record{Header: h,
			Payload: convertSmtpSentStatus(p.Payload.(rawparser.RawSmtpSentStatus))}, nil
	}

	panic("Not implemented!")
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

func convertSmtpSentStatus(p rawparser.RawSmtpSentStatus) *SmtpSentStatus {
	q, err := hex.DecodeString(string(p.Queue))

	if err != nil {
		// TODO: handle error on decoding the queue. Maybe even pre-allocate the destination buffer
		panic("Ahhh, invalid queue")
	}

	ip := net.ParseIP(string(p.RelayIp))

	if ip == nil {
		// TODO: handle error on invalid ip address
		panic("Ahhh, invalid IP address")
	}

	return &SmtpSentStatus{
		Queue:               q,
		RecipientLocalPart:  string(p.RecipientLocalPart),
		RecipientDomainPart: string(p.RecipientDomainPart),
		RelayName:           string(p.RelayName),
		RelayIP:             ip,
		RelayPort:           uint16(atoi(p.RelayPort)),
		Delay:               atof(p.Delay),
		Delays: Delays{
			Smtpd:   atof(p.Delays[1]),
			Cleanup: atof(p.Delays[2]),
			Qmgr:    atof(p.Delays[3]),
			Smtp:    atof(p.Delays[4]),
		},
		Dsn:          string(p.Dsn),
		Status:       parseStatus(p.Status),
		ExtraMessage: string(p.ExtraMessage),
	}
}
