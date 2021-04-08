// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package filelogsource

import (
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/logeater/reader"
	"gitlab.com/lightmeter/controlcenter/logeater/transform"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"io"
	"time"
)

type Source struct {
	file      io.Reader
	announcer announcer.ImportAnnouncer
	builder   transform.Builder
	clock     timeutil.Clock
}

func New(file io.Reader, builder transform.Builder, announcer announcer.ImportAnnouncer) (*Source, error) {
	return newWithClock(file, builder, announcer, &timeutil.RealClock{})
}

func newWithClock(file io.Reader, builder transform.Builder, announcer announcer.ImportAnnouncer, clock timeutil.Clock) (*Source, error) {
	return &Source{
		file:      file,
		announcer: announcer,
		builder:   builder,
		clock:     clock,
	}, nil
}

func (s *Source) PublishLogs(p postfix.Publisher) error {
	if err := reader.ReadFromReader(s.file, p, s.builder, s.announcer, s.clock, time.Second*10); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
