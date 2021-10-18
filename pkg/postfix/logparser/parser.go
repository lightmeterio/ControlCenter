// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/timeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"strconv"
)

type Payload interface {
	isPayload()
}

func atoi(s string) (int, error) {
	return strconv.Atoi(s)
}

func atof(s string) (float32, error) {
	r, err := strconv.ParseFloat(s, 32)

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return float32(r), nil
}

var (
	handlers = map[rawparser.PayloadType]func(rawparser.RawPayload) (Payload, error){}
)

func registerHandler(payloadType rawparser.PayloadType, handler func(rawparser.RawPayload) (Payload, error)) {
	handlers[payloadType] = handler
}

func ParseHeaderWithCustomTimeFormat(line string, format rawparser.TimeFormat) (h Header, payloadOffset int, err error) {
	rawHeader, payloadOffset, err := rawparser.ParseHeaderWithCustomTimeFormat(line, format)
	if err != nil {
		// TODO: unify parser and rawparser packages in a single one, for the sake of simplicity
		//nolint:wrapcheck
		return Header{}, payloadOffset, err
	}

	header, err := parseHeader(rawHeader, format)
	if err != nil {
		return Header{}, payloadOffset, err
	}

	return header, payloadOffset, nil
}

func ParseHeader(line string) (h Header, payloadOffset int, err error) {
	return ParseHeaderWithCustomTimeFormat(line, timeutil.DefaultTimeFormat{})
}

func ParsePayload(h Header, payloadLine string) (Payload, error) {
	p, err := rawparser.ParsePayload(payloadLine, h.Daemon, h.Process)
	if err != nil {
		return nil, rawparser.ErrUnsupportedLogLine
	}

	handler, found := handlers[p.PayloadType]
	if !found {
		return nil, rawparser.ErrUnsupportedLogLine
	}

	parsed, err := handler(p)
	if err != nil {
		return nil, err
	}

	return parsed, nil
}

func ParseWithCustomTimeFormat(line string, format rawparser.TimeFormat) (Header, Payload, error) {
	h, payloadOffset, err := ParseHeaderWithCustomTimeFormat(line, format)
	if err != nil {
		return Header{}, nil, err
	}

	payload, err := ParsePayload(h, line[payloadOffset:])
	if err != nil {
		return h, nil, err
	}

	return h, payload, nil
}

func Parse(line string) (Header, Payload, error) {
	return ParseWithCustomTimeFormat(line, timeutil.DefaultTimeFormat{})
}
