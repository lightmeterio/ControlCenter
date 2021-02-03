// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package parser

import (
	"errors"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"strconv"
)

type Payload interface {
	isPayload()
}

func atoi(s []byte) (int, error) {
	return strconv.Atoi(string(s))
}

func atof(s []byte) (float32, error) {
	r, err := strconv.ParseFloat(string(s), 32)

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return float32(r), nil
}

func tryToParserHeaderOnly(header rawparser.RawHeader, err error) (Header, Payload, error) {
	if errors.Is(err, rawparser.ErrInvalidHeaderLine) {
		return Header{}, nil, err
	}

	if !errors.Is(err, rawparser.ErrUnsupportedLogLine) {
		panic("This is a bug; maybe more error types have been added, but not handled. Who knows?!")
	}

	h, headerParsingError := parseHeader(header)

	if headerParsingError != nil {
		return Header{}, nil, rawparser.ErrUnsupportedLogLine
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
		return h, nil, rawparser.ErrUnsupportedLogLine
	}

	parsed, err := handler(p)

	if err != nil {
		return h, nil, err
	}

	return h, parsed, nil
}
