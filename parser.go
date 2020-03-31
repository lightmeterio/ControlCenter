package parser

import (
	"gitlab.com/lightmeter/postfix-log-parser/rawparser"
	"strconv"
)

type Payload interface {
	isPayload()
}

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
