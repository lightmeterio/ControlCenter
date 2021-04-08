// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package socketsource

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/logeater/reader"
	"gitlab.com/lightmeter/controlcenter/logeater/transform"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"net"
	"os"
	"strings"
	"time"
)

type Source struct {
	announcer announcer.ImportAnnouncer
	listener  net.Listener
	builder   transform.Builder
	clock     timeutil.Clock
	closed    bool
	closeErr  error
}

func New(socketDesc string, builder transform.Builder, announcer announcer.ImportAnnouncer) (*Source, error) {
	return newWithClock(socketDesc, builder, announcer, &timeutil.RealClock{})
}

func newWithClock(socketDesc string, builder transform.Builder, announcer announcer.ImportAnnouncer, clock timeutil.Clock) (*Source, error) {
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
		clock:     clock,
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

type emptyAnnouncer = announcer.EmptyImportAnnouncer

func (s *Source) PublishLogs(p postfix.Publisher) error {
	// only the first execution can potentially to notify import progress
	firstExecution := true

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return errorutil.Wrap(err)
		}

		log.Info().Msgf("New log socket connection from: %v", conn.RemoteAddr())

		announcer := func() announcer.ImportAnnouncer {
			if firstExecution {
				firstExecution = false
				return s.announcer
			}

			// as if we might already have finished the import, we cannot do it again
			return &emptyAnnouncer{}
		}()

		go func() {
			defer conn.Close()

			// FIXME: Handling multiple connections that feed times in similar intervals would mess up with the import/progress logic...
			errorutil.MustSucceed(reader.ReadFromReader(conn, p, s.builder, announcer, s.clock, time.Second*10))
		}()
	}
}
