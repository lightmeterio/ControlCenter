// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dirwatcher

import (
	"fmt"
	"github.com/hpcloud/tail"
	"github.com/rs/zerolog/log"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	parsertimeutil "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/timeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"os"
	"path"
	"time"
)

type localDirectoryContent struct {
	dir     string
	entries fileEntryList
	format  parsertimeutil.TimeFormat
}

func entriesForDir(dir string) (fileEntryList, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	entries := fileEntryList{}

	for _, i := range dirEntries {
		name := path.Join(dir, i.Name())

		info, err := i.Info()
		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		entry := fileEntry{filename: name, modificationTime: info.ModTime()}
		entries = append(entries, entry)
	}

	return entries, nil
}

func NewDirectoryContent(dir string, format parsertimeutil.TimeFormat) (DirectoryContent, error) {
	entries, err := entriesForDir(dir)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &localDirectoryContent{dir: dir, entries: entries, format: format}, nil
}

func (f *localDirectoryContent) dirName() string {
	return f.dir
}

func (f *localDirectoryContent) fileEntries() (fileEntryList, error) {
	return f.entries, nil
}

func localReaderForEntry(filename string) (fileReader, error) {
	reader, err := os.Open(filename)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return ensureReaderIsDecompressed(reader, filename)
}

func (f *localDirectoryContent) readerForEntry(filename string) (fileReader, error) {
	return localReaderForEntry(filename)
}

func localReadSeekerForEntry(filename string) (fileReadSeeker, error) {
	return os.Open(filename)
}

func (f *localDirectoryContent) readSeekerForEntry(filename string) (fileReadSeeker, error) {
	return localReadSeekerForEntry(filename)
}

type localFileWatcher struct {
	t        *tail.Tail
	filename string
	format   parsertimeutil.TimeFormat
}

func (w *localFileWatcher) run(onNewRecord func(parser.Header, parser.Payload)) {
	for line := range w.t.Lines {
		h, p, err := parser.ParseWithCustomTimeFormat([]byte(line.Text), w.format)

		if !parser.IsRecoverableError(err) {
			log.Error().Msgf("parsing line on file: %v", w.filename)
			continue
		}

		onNewRecord(h, p)
	}
}

func (f *localDirectoryContent) watcherForEntry(filename string, offset int64) (fileWatcher, error) {
	t, err := tail.TailFile(filename, tail.Config{
		Follow:    true,
		ReOpen:    true,
		Logger:    tail.DefaultLogger,
		MustExist: true,
		Location:  &tail.SeekInfo{Offset: offset, Whence: io.SeekStart},
	})

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &localFileWatcher{t, filename, f.format}, nil
}

func localModificationTimeForEntry(entries fileEntryList, filename string) (time.Time, error) {
	for _, e := range entries {
		if filename == e.filename {
			return e.modificationTime, nil
		}
	}

	return time.Time{}, errorutil.Wrap(fmt.Errorf("File not found: %v", filename))
}

func (f *localDirectoryContent) modificationTimeForEntry(filename string) (time.Time, error) {
	t, err := localModificationTimeForEntry(f.entries, filename)
	if err != nil {
		return time.Time{}, errorutil.Wrap(err)
	}

	return t, nil
}
