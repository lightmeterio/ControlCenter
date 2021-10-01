// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rsyncwatcher

import (
	"bufio"
	"github.com/fsnotify/fsnotify"
	"github.com/hlubek/readercomp"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"os"
	"path"
)

// NOTE: heavily inspired by https://github.com/latitov/milkthisbuffer
type readWriter struct {
	// Meh, working on characters can be freaking inneficient!
	content chan byte
}

func (rw *readWriter) Close() error {
	close(rw.content)
	return nil
}

func (rw *readWriter) Read(p []byte) (n int, err error) {
	handle := func(b byte, ok bool) bool {
		if !ok {
			// input ended
			err = io.EOF
			return true
		}

		p[n] = b
		n++

		return false
	}

	for n = 0; n < len(p); {
		if n == 0 {
			b, ok := <-rw.content

			if handle(b, ok) {
				return
			}

			continue
		}

		select {
		case b, ok := <-rw.content:
			if handle(b, ok) {
				return
			}
		default:
			return
		}
	}

	return n, err
}

func (rw *readWriter) Write(p []byte) (n int, err error) {
	for n = 0; n < len(p); n++ {
		rw.content <- p[n]
	}

	return n, nil
}

func ReadWriter() io.ReadWriteCloser {
	return &readWriter{content: make(chan byte, 4096)}
}

// Watcher is able to observe changes in files updated by rsync and potentially managed by logrotate
type Watcher = runner.CancellableRunner

func New(filename string, offset int64, w io.WriteCloser) (Watcher, error) {
	filename = path.Clean(filename)

	dirname := path.Dir(filename)
	basename := path.Base(filename)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err := watcher.Add(dirname); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return runner.NewCancellableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
		go func() {
			<-cancel
			watcher.Close()
		}()

		go func() {
			if err := watchForEvents(watcher, dirname, basename, offset, w); err != nil {
				w.Close()
				done <- errorutil.Wrap(err)
				return
			}

			if err := w.Close(); err != nil {
				done <- errorutil.Wrap(err)
				return
			}

			done <- nil
		}()
	}), nil
}

func notifyFromReader(r io.Reader, w io.Writer) error {
	if _, err := io.Copy(w, r); err != nil {
		return errorutil.Wrap(err)
	}

	// FIXME: is a somehow ugly workaround and assumes we are working break line terminated files,
	// and we do partial line reads. Which is valid for our log files.
	// it might generate some 0-length lines, which are properly ignored by the parser.
	if _, err := w.Write([]byte("\n")); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func eventHas(op, t fsnotify.Op) bool {
	return op&t == t
}

func flushFile(f *os.File, w io.Writer) error {
	// read anything in the current file
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		// TODO: handle case where not all bytes are written in a single call?
		// I don't think this will be a problem for us right now, but you know how it works, right?
		if _, err := w.Write(scanner.Bytes()); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

func watchForEvents(watcher *fsnotify.Watcher, dirname, filename string, offset int64, w io.Writer) error {
	origFile, err := os.Open(path.Join(dirname, filename))
	if err != nil {
		return errorutil.Wrap(err)
	}

	if _, err := origFile.Seek(offset, io.SeekStart); err != nil {
		return errorutil.Wrap(err)
	}

	if err := flushFile(origFile, w); err != nil {
		return errorutil.Wrap(err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				// watcher closed, ends watching
				return nil
			}

			if event.Name == path.Join(dirname, filename) && eventHas(event.Op, fsnotify.Create) {
				if origFile, err = handleMove(origFile, path.Join(dirname, filename), w); err != nil {
					return errorutil.Wrap(err)
				}

				continue
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}

			return errorutil.Wrap(err)
		}
	}
}

func fileSize(f *os.File) (int64, error) {
	s, err := f.Stat()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return s.Size(), nil
}

func handleMove(origFile *os.File, filename string, w io.Writer) (*os.File, error) {
	defer func() {
		// TODO: handle this error properly!
		errorutil.MustSucceed(origFile.Close())
	}()

	newFile, err := os.Open(filename)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	sizeOfNewFile, err := fileSize(newFile)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	sizeOfOrigFile, err := fileSize(origFile)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	if sizeOfNewFile < sizeOfOrigFile {
		// it's definitely a new file, probably logrotated
		if err = notifyFromReader(newFile, w); err != nil {
			return nil, errorutil.Wrap(err)
		}

		return newFile, nil
	}

	if _, err = newFile.Seek(0, io.SeekStart); err != nil {
		return nil, errorutil.Wrap(err)
	}

	if _, err = origFile.Seek(0, io.SeekStart); err != nil {
		return nil, errorutil.Wrap(err)
	}

	// here newFile is either bigger or same size as origFile
	commonAreaOfNewFile := io.LimitReader(newFile, sizeOfOrigFile)

	equal, err := readercomp.Equal(commonAreaOfNewFile, origFile, 4096)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	// temp file is a totally new file, probably logrotated
	if !equal {
		if _, err = newFile.Seek(0, io.SeekStart); err != nil {
			return nil, errorutil.Wrap(err)
		}

		if err = notifyFromReader(newFile, w); err != nil {
			return nil, errorutil.Wrap(err)
		}

		return newFile, nil
	}

	// newFile content is basically origFile + some new content
	// then notify only the new content!
	if err = notifyFromReader(newFile, w); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return newFile, nil
}
