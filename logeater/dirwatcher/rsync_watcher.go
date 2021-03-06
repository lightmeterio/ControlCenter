// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dirwatcher

import (
	"bufio"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/logeater/rsyncwatcher"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	parsertimeutil "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/timeutil"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type rsyncedFileWatcher struct {
	filename string
	offset   int64
	format   parsertimeutil.TimeFormat
}

type rsyncedFileWatcherRunner = runner.CancellableRunner

func newRsyncedFileWatcherRunner(watcher *rsyncedFileWatcher, onRecord func(parser.Header, string, int)) rsyncedFileWatcherRunner {
	rw := rsyncwatcher.ReadWriter()

	w, err := rsyncwatcher.New(watcher.filename, watcher.offset, rw)
	errorutil.MustSucceed(err)

	return runner.NewCancellableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
		wDone, wCancel := runner.Run(w)

		go func() {
			<-cancel
			wCancel()
		}()

		go func() {
			scanner := bufio.NewScanner(rw)

			for scanner.Scan() {
				line := scanner.Text()
				h, payloadOffset, err := parser.ParseHeaderWithCustomTimeFormat(line, watcher.format)

				if !parser.IsRecoverableError(err) {
					log.Error().Msgf("parsing line on file: %v", watcher.filename)
					continue
				}

				onRecord(h, line, payloadOffset)
			}

			if err := scanner.Err(); err != nil {
				done <- errorutil.Wrap(err)
				return
			}

			done <- wDone()
		}()
	})
}

func (watcher *rsyncedFileWatcher) run(onRecord func(parser.Header, string, int)) {
	done, _ := runner.Run(newRsyncedFileWatcherRunner(watcher, onRecord))

	// never cancel, wait forever, no error handling
	errorutil.MustSucceed(done())
}
