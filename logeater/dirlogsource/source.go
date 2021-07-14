// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dirlogsource

import (
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/logeater/dirwatcher"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parsertimeutil "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/timeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type Source struct {
	initialTime time.Time
	dir         dirwatcher.DirectoryContent
	announcer   announcer.ImportAnnouncer
	patterns    dirwatcher.LogPatterns
	format      parsertimeutil.TimeFormat

	// should continue waiting for new results (tail -f)?
	follow bool
}

func New(dirname string, initialTime time.Time, announcer announcer.ImportAnnouncer, follow bool, rsynced bool, logFormat string, patterns dirwatcher.LogPatterns) (*Source, error) {
	timeFormat, err := parsertimeutil.Get(logFormat)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	dir, err := func() (dirwatcher.DirectoryContent, error) {
		if rsynced {
			return dirwatcher.NewDirectoryContentForRsync(dirname, timeFormat, patterns)
		}

		return dirwatcher.NewDirectoryContent(dirname, timeFormat)
	}()

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	format, err := parsertimeutil.Get(logFormat)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	func() {
		if initialTime.IsZero() {
			log.Info().Msg("Start importing Postfix logs directory into a new workspace")
			return
		}

		log.Info().Msgf("Importing Postfix logs directory from time %v", initialTime)
	}()

	return &Source{
		initialTime: initialTime,
		dir:         dir,
		follow:      follow,
		announcer:   announcer,
		patterns:    patterns,
		format:      format,
	}, nil
}

func (s *Source) PublishLogs(p postfix.Publisher) error {
	watcher := dirwatcher.NewDirectoryImporter(s.dir, p, s.announcer, s.initialTime, s.format, s.patterns)

	f := func() func() error {
		if s.follow {
			return watcher.Run
		}

		return watcher.ImportOnly
	}()

	if err := f(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
