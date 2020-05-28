package dirwatcher

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/hpcloud/tail"
	parser "gitlab.com/lightmeter/postfix-log-parser"
)

type localDirectoryContent struct {
	entries fileEntryList
}

func NewDirectoryContent(dir string) (localDirectoryContent, error) {
	infos, err := ioutil.ReadDir(dir)

	if err != nil {
		return localDirectoryContent{}, err
	}

	entries := fileEntryList{}

	for _, i := range infos {
		name := path.Join(dir, i.Name())
		entries = append(entries, fileEntry{filename: name, modificationTime: i.ModTime()})
	}

	return localDirectoryContent{entries: entries}, nil
}

func (f *localDirectoryContent) fileEntries() fileEntryList {
	return f.entries
}

func (f *localDirectoryContent) readerForEntry(filename string) (fileReader, error) {
	reader, err := os.Open(filename)

	if err != nil {
		return nil, err
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
			log.Println("Error parsing line on file", w.filename)
			continue
		}

		onNewRecord(h, p)
	}

	// It never reaches here, actually
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
		return &localFileWatcher{}, err
	}

	return &localFileWatcher{t, filename}, nil
}

func (f *localDirectoryContent) modificationTimeForEntry(filename string) (time.Time, error) {
	for _, e := range f.entries {
		if filename == e.filename {
			return e.modificationTime, nil
		}
	}

	return time.Time{}, fmt.Errorf("File not found: %v", filename)
}
