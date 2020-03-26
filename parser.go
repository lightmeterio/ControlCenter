package parser

import (
	"gitlab.com/lightmeter/postfix-log-parser/rawparser"
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
	case "qmgr":
		return QMgrProcess, nil
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
	QMgrProcess
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
	case rawparser.PayloadTypeQmgrReturnedToSender:
		p, err := convertQmgrReturnedToSender(p.QmgrReturnedToSender)
		if err != nil {
			return Record{}, err
		}
		return Record{Header: h, Payload: p}, nil
	}

	return Record{}, rawparser.UnsupportedLogLineError
}
