// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dirlogsource

import (
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/logeater/dirwatcher"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type Source struct {
	initialTime time.Time
	dir         dirwatcher.DirectoryContent
	announcer   announcer.ImportAnnouncer

	// should continue waiting for new results (tail -f)?
	follow bool
}

func New(dirname string, initialTime time.Time, announcer announcer.ImportAnnouncer, follow bool, rsynced bool) (*Source, error) {
	dir, err := dirwatcher.NewDirectoryContent(dirname, rsynced)

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
	}, nil
}

func (s *Source) PublishLogs(p postfix.Publisher) error {
	watcher := dirwatcher.NewDirectoryImporter(s.dir, p, s.announcer, s.initialTime)

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
