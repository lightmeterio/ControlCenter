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

type Header struct {
	Time    Time
	Host    string
	Process string
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

	process := string(h.Process)

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

func tryToParserHeaderOnly(header rawparser.RawHeader, err error) (Header, Payload, error) {
	if err == rawparser.InvalidHeaderLineError {
		return Header{}, nil, err
	}

	if err != rawparser.UnsupportedLogLineError {
		panic("This is a bug; maybe more error types have been added, but not handled. Who knows?!")
	}

	h, headerParsingError := parseHeader(header)

	if headerParsingError != nil {
		return h, nil, rawparser.UnsupportedLogLineError
	}

	return h, nil, err
}

var (
	handlers = map[rawparser.PayloadType]func(rawparser.RawPayload) (Payload, error){}
)

func registerHandler(payloadType rawparser.PayloadType, handler func(rawparser.RawPayload) (Payload, error)) {
	handlers[payloadType] = handler
}

func Parse(line []byte) (Header, Payload, error) {
	rawHeader, p, err := rawparser.Parse(line)

	if err != nil {
		return tryToParserHeaderOnly(rawHeader, err)
	}

	h, err := parseHeader(rawHeader)

	if err != nil {
		return Header{}, nil, err
	}

	handler, found := handlers[p.PayloadType]

	if !found {
		return h, nil, rawparser.UnsupportedLogLineError
	}

	parsed, err := handler(p)

	if err != nil {
		return h, nil, err
	}

	return h, parsed, nil
}
