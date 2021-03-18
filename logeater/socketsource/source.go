// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package socketsource

import (
	"fmt"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/logeater/transform"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net"
	"os"
	"strings"
)

type Source struct {
	announcer announcer.ImportAnnouncer
	listener  net.Listener
	builder   transform.Builder
	closed    bool
	closeErr  error
}

func New(socketDesc string, builder transform.Builder, announcer announcer.ImportAnnouncer) (*Source, error) {
	c := strings.Split(socketDesc, "=")

	if len(c) != 2 {
		return nil, fmt.Errorf(`Invalid socket description: %v. It should have the form "unix=/path/to/socket_file" or "tcp=:9999"`, socketDesc)
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
		listener:  l,
		builder:   builder,
		announcer: announcer,
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
	announcer.Skip(s.announcer)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return errorutil.Wrap(err)
		}

		go func() {
			errorutil.MustSucceed(transform.ReadFromReader(conn, p, s.builder))
		}()
	}
}
