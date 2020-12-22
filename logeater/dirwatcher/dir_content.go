package dirwatcher

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/hpcloud/tail"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
)

type localDirectoryContent struct {
	dir     string
	entries fileEntryList
}

func NewDirectoryContent(dir string) (DirectoryContent, error) {
	infos, err := ioutil.ReadDir(dir)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	entries := fileEntryList{}

	for _, i := range infos {
		name := path.Join(dir, i.Name())
		entries = append(entries, fileEntry{filename: name, modificationTime: i.ModTime()})
	}

	return &localDirectoryContent{dir: dir, entries: entries}, nil
}

func (f *localDirectoryContent) dirName() string {
	return f.dir
}

func (f *localDirectoryContent) fileEntries() fileEntryList {
	return f.entries
}

func (f *localDirectoryContent) readerForEntry(filename string) (fileReader, error) {
	reader, err := os.Open(filename)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return ensureReaderIsDecompressed(reader, filename)
}

func (f *localDirectoryContent) readSeekerForEntry(filename string) (fileReadSeeker, error) {
	return os.Open(filename)
}

type localFileWatcher struct {
	t        *tail.Tail
	filename string
}

func (w *localFileWatcher) run(onNewRecord func(parser.Header, parser.Payload)) {
	for line := range w.t.Lines {
		h, p, err := parser.Parse([]byte(line.Text))

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
		ReOpen:    false,
		Logger:    tail.DefaultLogger,
		MustExist: true,
		Location:  &tail.SeekInfo{Offset: offset, Whence: io.SeekStart},
	})

	if err != nil {
		return &localFileWatcher{}, errorutil.Wrap(err)
	}

	return &localFileWatcher{t, filename}, nil
}

func (f *localDirectoryContent) modificationTimeForEntry(filename string) (time.Time, error) {
	for _, e := range f.entries {
		if filename == e.filename {
			return e.modificationTime, nil
		}
	}

	return time.Time{}, errorutil.Wrap(fmt.Errorf("File not found: %v", filename))
}
