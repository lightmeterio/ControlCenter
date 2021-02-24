// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package filelogsource

import (
	"gitlab.com/lightmeter/controlcenter/logeater"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"io"
	"time"
)

type Source struct {
	file        io.Reader
	initialTime time.Time
	year        int
}

func New(file io.Reader, initialTime time.Time, year int) (*Source, error) {
	return &Source{
		file:        file,
		initialTime: initialTime,
		year:        year,
	}, nil
}

func (s *Source) PublishLogs(p postfix.Publisher) error {
	initialLogsTime := logeater.BuildInitialLogsTime(s.initialTime, s.year)
	logeater.ParseLogsFromReader(p, initialLogsTime, s.file)

	return nil
}
