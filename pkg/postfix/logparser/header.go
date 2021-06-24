// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"net"

	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/timeutil"
)

type Time = timeutil.Time

type Header struct {
	Time      Time
	Host      string
	Process   string
	Daemon    string
	PID       int
	ProcessIP net.IP
}

func parseHeader(h rawparser.RawHeader, format rawparser.TimeFormat) (Header, error) {
	time, err := format.Convert(h.Time)
	if err != nil {
		return Header{}, err
	}

	pid, err := func() (int, error) {
		if len(h.ProcessID) == 0 {
			return 0, nil
		}

		return atoi(h.ProcessID)
	}()

	if err != nil {
		return Header{}, err
	}

	processIP, err := parseIP(h.ProcessIP)
	if err != nil {
		return Header{}, err
	}

	return Header{
		Time:      time,
		Host:      string(h.Host),
		Process:   string(h.Process),
		Daemon:    string(h.Daemon),
		PID:       pid,
		ProcessIP: processIP,
	}, nil
}
