// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package socketsource

import (
	"fmt"
	"gitlab.com/lightmeter/controlcenter/logeater"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net"
	"os"
	"strings"
	"time"
)

type Source struct {
	listener    net.Listener
	initialTime time.Time
	year        int
	closed      bool
	closeErr    error
}

func New(socketDesc string, initialTime time.Time, year int) (*Source, error) {
	c := strings.Split(socketDesc, ";")

	if len(c) != 2 {
		return nil, fmt.Errorf(`Invalid socket description: %v. It should have the form "unix;/path/to/socket_file" or "tcp;:9999"`, socketDesc)
	}

	network := c[0]
	address := c[1]

	if network == "unix" {
		if err := os.RemoveAll(address); err != nil {
			return nil, errorutil.Wrap(err)
		}
	}

	l, err := net.Listen(network, address)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &Source{
		listener:    l,
		initialTime: initialTime,
		year:        year,
	}, nil
}

func (s *Source) Close() error {
	if s.closed {
		return s.closeErr
	}

	s.closed = true

	s.closeErr = s.listener.Close()

	if s.closeErr != nil {
		return errorutil.Wrap(s.closeErr)
	}

	return nil
}

func (s *Source) PublishLogs(p postfix.Publisher) error {
	initialLogsTime := logeater.BuildInitialLogsTime(s.initialTime, s.year)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return errorutil.Wrap(err)
		}

		go func() {
			logeater.ParseLogsFromReader(p, initialLogsTime, conn)
		}()
	}
}
