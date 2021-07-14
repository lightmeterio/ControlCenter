// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dirwatcher

import (
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
	parsertimeutil "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/timeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"path"
	"sync"
	"time"
)

func NewDirectoryContentForRsync(dir string, format parsertimeutil.TimeFormat, patterns LogPatterns) (DirectoryContent, error) {
	return contentForRsyncManagedDirectory(dir, format, patterns, time.Second*10)
}

type rsyncedFileEntryList struct {
	sync.Mutex
	list fileEntryList
}

type rsyncdDirectoryContent struct {
	dir                 string
	format              parsertimeutil.TimeFormat
	errorChan           <-chan error
	filesEntriesChan    <-chan fileEntryList
	cachedFileEntryList rsyncedFileEntryList
}

func (d *rsyncdDirectoryContent) dirName() string {
	return d.dir
}

func (d *rsyncdDirectoryContent) fileEntries() (fileEntryList, error) {
	d.cachedFileEntryList.Lock()

	defer d.cachedFileEntryList.Unlock()

	// a list is already prepared and cached. Use a copy of it.
	if len(d.cachedFileEntryList.list) > 0 {
		c := make(fileEntryList, len(d.cachedFileEntryList.list))
		copy(c, d.cachedFileEntryList.list)

		return c, nil
	}

	// The list is not ready yet. Just wait until it is.

	select {
	case err := <-d.errorChan:
		return nil, errorutil.Wrap(err)

	case list, ok := <-d.filesEntriesChan:
		if !ok {
			return nil, nil
		}

		d.cachedFileEntryList.list = list

		return list, nil
	}
}

func (d *rsyncdDirectoryContent) modificationTimeForEntry(filename string) (time.Time, error) {
	entries, err := d.fileEntries()
	if err != nil {
		return time.Time{}, errorutil.Wrap(err)
	}

	t, err := localModificationTimeForEntry(entries, filename)
	if err != nil {
		return time.Time{}, errorutil.Wrap(err)
	}

	return t, nil
}

func (d *rsyncdDirectoryContent) readerForEntry(filename string) (fileReader, error) {
	return localReaderForEntry(filename)
}

func (d *rsyncdDirectoryContent) watcherForEntry(filename string, offset int64) (fileWatcher, error) {
	return &rsyncedFileWatcher{filename, offset, d.format}, nil
}

func (d *rsyncdDirectoryContent) readSeekerForEntry(filename string) (fileReadSeeker, error) {
	return localReadSeekerForEntry(filename)
}

func contentForRsyncManagedDirectory(dir string, format parsertimeutil.TimeFormat, patterns LogPatterns, timeout time.Duration) (DirectoryContent, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	log.Info().Msgf("Waiting for contents in the directory %s", dir)

	if err := watcher.Add(dir); err != nil {
		return nil, errorutil.Wrap(err)
	}

	errorChan := make(chan error, 1)
	filesEntriesChan := make(chan fileEntryList, 1)

	// an rsync ends with a file being created (due renaming, followed by a chomd for the whole directory)

	var prevOp fsnotify.Op = 0

	go func() {
		defer watcher.Close()

		ticker := time.NewTicker(timeout)

		reportError := func(ok bool, err error) {
			log.Warn().Msgf("rsync waiter has received an error: %v", err)

			if ok {
				errorChan <- errorutil.Wrap(err)
			}

			close(errorChan)

			ticker.Stop()
		}

		for {
			select {
			case e, ok := <-watcher.Events:
				if !ok {
					log.Debug().Msgf("rsync waiter closed!")
					return
				}

				log.Debug().Msgf("rsync waiter has received a new event: %v", e)

				// the last operation in a file is it being renamed to its final name.
				if !(prevOp == fsnotify.Create && eventHas(e.Op, fsnotify.Chmod)) {
					prevOp = e.Op

					ticker.Reset(timeout)
				}
			case err, ok := <-watcher.Errors:
				reportError(ok, err)
				return
			case <-ticker.C:
				// build file entries and send it to any callers, if there are any files.
				// in case the directory is empty, wait more...
				entries, err := entriesForDir(dir)
				if err != nil {
					reportError(true, err)
					return
				}

				// here we don't bother to filter files by time. Just list them all
				if entriesAreComplete(entries, patterns, time.Time{}) {
					filesEntriesChan <- entries
					return
				}

				// otherwise, we should wait more...
				ticker.Reset(timeout)
			}
		}
	}()

	return &rsyncdDirectoryContent{dir: dir, format: format, errorChan: errorChan, filesEntriesChan: filesEntriesChan}, nil
}

func entriesAreComplete(entries fileEntryList, patterns LogPatterns, initialTime time.Time) bool {
	queues := buildFilesToImport(entries, patterns, initialTime)

	hasSomeContent := func() bool {
		for _, q := range queues {
			if len(q) > 0 {
				return true
			}
		}

		return false
	}()

	if !hasSomeContent {
		return false
	}

	// all queues must have the "current" file, which is a non-archived one.
	for p, q := range queues {
		if len(q) == 0 {
			// empty queue, allowed.
			continue
		}

		lastEntry := q[len(q)-1]
		if path.Base(lastEntry.filename) != p {
			return false
		}
	}

	return true
}

func eventHas(op, t fsnotify.Op) bool {
	return op&t == t
}
