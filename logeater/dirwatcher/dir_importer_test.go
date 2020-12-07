package dirwatcher

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
)

func readFromReader(reader io.Reader,
	filename string,
	onNewRecord func(parser.Header, parser.Payload)) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Bytes()

		h, p, err := parser.Parse(line)

		if parser.IsRecoverableError(err) {
			onNewRecord(h, p)
		}
	}
}

type fakePublisher struct {
	logs []data.Record
}

func (this *fakePublisher) Publish(r data.Record) {
	this.logs = append(this.logs, r)
}

func (fakePublisher) Close() {
}

func compress(content []byte) []byte {
	var buf bytes.Buffer

	w := gzip.NewWriter(&buf)

	_, err := w.Write(content)

	if err != nil {
		log.Fatalln("Error compressing data")
	}

	w.Close()

	return buf.Bytes()
}

type fakeFileReader struct {
	io.Reader
}

func plainDataReaderFromBytes(data []byte) fakeFileReader {
	buf := bytes.NewBuffer(data)
	return fakeFileReader{Reader: strings.NewReader(string(buf.Bytes()))}
}

func gzipedDataReaderFromBytes(data []byte) fakeFileReader {
	plainReader := plainDataReaderFromBytes(data)

	reader, err := ensureReaderIsDecompressed(plainReader, "something.gz")

	if err != nil {
		panic("Failed on decompressing file!!!! FIX IT!")
	}

	return fakeFileReader{reader}
}

func plainDataReader(s string) fileReader {
	return plainDataReaderFromBytes([]byte(s))
}

func gzipedDataReader(s string) fileReader {
	return gzipedDataReaderFromBytes(compress([]byte(s)))
}

func (fakeFileReader) Close() error {
	return nil
}

type fakeFileData interface {
	hasFakeContent()
}

type fakeFileDataBytes struct {
	content []byte
}

func (fakeFileDataBytes) hasFakeContent() {
}

func gzippedDataFile(s string) fakeFileDataBytes {
	return fakeFileDataBytes{compress([]byte(s))}
}

func plainDataFile(s string) fakeFileDataBytes {
	return fakeFileDataBytes{[]byte(s)}
}

type fakePlainCurrentFileData struct {
	content []byte
	offset  int64
}

func (fakePlainCurrentFileData) hasFakeContent() {
}

func plainCurrentDataFile(s, c string) fakePlainCurrentFileData {
	return fakePlainCurrentFileData{[]byte(s + c), int64(len(s))}
}

type FakeDirectoryContent struct {
	entries  fileEntryList
	contents map[string]fakeFileData
}

func (f FakeDirectoryContent) fileEntries() fileEntryList {
	return f.entries
}

func (f FakeDirectoryContent) readerForEntry(filename string) (fileReader, error) {
	content, ok := f.contents[filename]

	if !ok {
		log.Fatalln("Missing filename: " + filename)
	}

	if data, ok := content.(fakePlainCurrentFileData); ok {
		// It will get here when the the only file in the queue is the current one
		// Meaning there's no archived file to import
		return plainDataReaderFromBytes(data.content[:data.offset]), nil
	}

	if data, ok := content.(fakeFileDataBytes); ok {
		return ensureReaderIsDecompressed(plainDataReaderFromBytes(data.content), filename)
	}

	panic("Should never reach here!!!")
}

type fakeFileReadSeeker struct {
	content    []byte
	currentPos int64
	offset     int64
}

// implements io.Seeker
func (s *fakeFileReadSeeker) Seek(offset int64, whence int) (int64, error) {
	checkPos := func() {
		if s.currentPos < 0 || s.currentPos > s.offset {
			panic("Seek went to invalid file position!")
		}
	}

	if whence == io.SeekEnd && offset == 0 {
		s.currentPos = s.offset
		checkPos()
		return s.currentPos, nil
	}

	if whence == io.SeekStart && offset == 0 {
		s.currentPos = 0
		checkPos()
		return s.currentPos, nil
	}

	log.Fatalln("Unexpected fakeFileReadSeeker seeking arguments")

	return 0, nil
}

func (s *fakeFileReadSeeker) Read(p []byte) (n int, err error) {
	reader := bytes.NewReader(s.content[s.currentPos:])

	n, err = reader.Read(p)

	if err != nil {
		s.currentPos += int64(n)
	}

	return
}

func (s *fakeFileReadSeeker) Close() error {
	return nil
}

func (f FakeDirectoryContent) readSeekerForEntry(filename string) (fileReadSeeker, error) {
	content, ok := f.contents[filename]

	if !ok {
		log.Fatalln("Missing filename: " + filename)
	}

	data, ok := content.(fakePlainCurrentFileData)

	if !ok {
		log.Fatalln("Could not find a entry for current log file:", filename)
	}

	return &fakeFileReadSeeker{content: data.content, offset: data.offset, currentPos: 0}, nil
}

type fakeFileWatcher struct {
	filename string
	reader   io.Reader
}

func (f fakeFileWatcher) run(onNewRecord func(parser.Header, parser.Payload)) {
	readFromReader(f.reader, f.filename, onNewRecord)
}

func (f FakeDirectoryContent) watcherForEntry(filename string, offset int64) (fileWatcher, error) {
	content, ok := f.contents[filename]

	if !ok {
		log.Fatalln("Missing filename: " + filename)
	}

	data, ok := content.(fakePlainCurrentFileData)

	if !ok {
		log.Fatalln("Could not find a entry for current log file:", filename)
	}

	reader := bytes.NewReader(data.content[offset:])

	return fakeFileWatcher{filename, reader}, nil
}

func (f FakeDirectoryContent) modificationTimeForEntry(filename string) (time.Time, error) {
	for _, e := range f.fileEntries() {
		if filename == e.filename {
			return e.modificationTime, nil
		}
	}

	panic("File not Found!")
}

func TestGuessingYearWhenFileStarts(t *testing.T) {
	Convey("Guess Based on file content and modification date", t, func() {
		Convey("Empty file uses modification date directly", func() {
			date, err := guessInitialDateForFile(plainDataReader(``), testutil.MustParseTime(`2020-04-03 19:01:53 +0000`))
			So(err, ShouldBeNil)
			So(date, ShouldEqual, testutil.MustParseTime(`2020-04-03 19:01:53 +0000`))
		})

		Convey("Fail to read single invalid line file", func() {
			reader := plainDataReader(`Invalid Log Line`)
			_, err := guessInitialDateForFile(reader, testutil.MustParseTime(`2020-04-03 19:01:53 +0000`))
			So(err, ShouldNotBeNil)
		})

		Convey("Fail if the last line in the file is invalid", func() {
			reader := plainDataReader(
				`Mar 22 06:28:55 mail dovecot: Useless Payload
Invalid Line`)
			_, err := guessInitialDateForFile(reader, testutil.MustParseTime(`2020-04-03 19:01:53 +0000`))
			So(err, ShouldNotBeNil)
		})

		Convey("Single line, no change in year", func() {
			reader := plainDataReader(`Mar 22 06:28:55 mail dovecot: Useless Payload`)
			date, err := guessInitialDateForFile(reader, testutil.MustParseTime(`2020-04-03 19:01:53 +0000`))
			So(err, ShouldBeNil)
			So(date, ShouldEqual, testutil.MustParseTime(`2020-03-22 06:28:55 +0000`))
		})

		Convey("Single line, no change in year, same second", func() {
			reader := plainDataReader(`Mar 22 06:28:55 mail dovecot: Useless Payload`)
			date, err := guessInitialDateForFile(reader, testutil.MustParseTime(`2020-03-22 06:28:55 +0000`))
			So(err, ShouldBeNil)
			So(date, ShouldEqual, testutil.MustParseTime(`2020-03-22 06:28:55 +0000`))
		})

		Convey("Single line, year changes", func() {
			reader := plainDataReader(`Dec 31 23:59:58 mail dovecot: Useless Payload`)
			date, err := guessInitialDateForFile(reader, testutil.MustParseTime(`2020-01-01 00:00:01 +0000`))
			So(err, ShouldBeNil)
			So(date, ShouldEqual, testutil.MustParseTime(`2019-12-31 23:59:58 +0000`))
		})

		// Calendar days (ignoring year):
		// B = Time for first Line
		// E = Time for last Line
		// M = File modification time

		Convey("Time for the first line if there's no year change in file", func() {
			reader := gzipedDataReader(
				`Mar 22 06:28:55 mail dovecot: Useless Payload
Mar 29 06:47:09 mail postfix/postscreen[17274]: Useless Payload`)
			date, err := guessInitialDateForFile(reader, testutil.MustParseTime(`2020-04-29 06:47:09 +0000`))
			So(err, ShouldBeNil)
			So(date, ShouldEqual, testutil.MustParseTime(`2020-03-22 06:28:55 +0000`))
		})

		Convey("Multiple lines with no change in year, last line in the modification time", func() {
			reader := gzipedDataReader(
				`Jan 22 06:28:55 mail dovecot: Useless Payload
Jan 23 06:28:55 mail dovecot: Useless Payload
Jan 31 06:47:09 mail postfix/postscreen[17274]: Useless Payload`)
			date, err := guessInitialDateForFile(reader, testutil.MustParseTime(`2020-01-31 06:47:09 +0000`))
			So(err, ShouldBeNil)
			So(date, ShouldEqual, testutil.MustParseTime(`2020-01-22 06:28:55 +0000`))
		})

		Convey("Year changes", func() {
			reader := gzipedDataReader(
				`Dec 31 23:59:55 mail dovecot: pop3-login:
Jan  1 00:00:01 mail postfix/postscreen[9183]: CONNECT from [18.88.247.65]:50082 to [170.68.1.1]:25
Jan  1 00:00:01 mail postfix/postscreen[17274]: DISCONNECT [224.35.90.202]:54744
Jan  1 00:00:02 mail postfix/postscreen[18660]: a`)
			date, err := guessInitialDateForFile(reader, testutil.MustParseTime(`2001-01-01 00:01:00 +0000`))
			So(err, ShouldBeNil)
			So(date, ShouldEqual, testutil.MustParseTime(`2000-12-31 23:59:55 +0000`))
		})

		Convey("The whole file in the year before file modification", func() {
			reader := gzipedDataReader(
				`Dec 31 23:59:50 mail postfix/postscreen[26735]: CONNECT
Dec 31 23:59:51 mail postfix/dnsblog[26740]: addr
Dec 31 23:59:55 mail dovecot: imap-login:
Dec 31 23:59:57 mail dovecot: imap-login:`)
			date, err := guessInitialDateForFile(reader, testutil.MustParseTime(`2001-01-01 00:00:02 +0000`))
			So(err, ShouldBeNil)
			So(date, ShouldEqual, testutil.MustParseTime(`2000-12-31 23:59:50 +0000`))
		})

		Convey("Return last year as year changed in the middle of the log", func() {
			reader := plainDataReader(
				`Dec 22 06:28:55 mail dovecot: Useless Payload
Dec 23 06:28:55 mail dovecot: Useless Payload
Mar 29 06:47:09 mail postfix/postscreen[17274]: Useless Payload`)
			date, err := guessInitialDateForFile(reader, testutil.MustParseTime(`2020-04-03 19:01:53 +0000`))
			So(err, ShouldBeNil)
			So(date, ShouldEqual, testutil.MustParseTime(`2019-12-22 06:28:55 +0000`))
		})

		Convey("First log when file start, no change in year", func() {
			reader := plainDataReader(
				`Dec 31 23:59:50 mail postfix/postscreen[26735]: CONNECT
Dec 31 23:59:51 mail postfix/dnsblog[26740]: addr
Dec 31 23:59:55 mail dovecot: imap-login:`)
			date, err := guessInitialDateForFile(reader, testutil.MustParseTime(`2000-12-31 23:59:55 +0000`))
			So(err, ShouldBeNil)
			So(date, ShouldEqual, testutil.MustParseTime(`2000-12-31 23:59:50 +0000`))
		})

		Convey("First log when file start, year changes", func() {
			reader := plainDataReader(
				`Dec 31 23:59:55 mail dovecot: pop3-login:
Jan  1 00:00:01 mail postfix/postscreen[9183]: CONNECT from [18.88.247.65]:50082 to [170.68.1.1]:25
Jan  1 00:00:01 mail postfix/postscreen[17274]: DISCONNECT [224.35.90.202]:54744`)
			date, err := guessInitialDateForFile(reader, testutil.MustParseTime(`2001-01-01 00:00:01 +0000`))
			So(err, ShouldBeNil)
			So(date, ShouldEqual, testutil.MustParseTime(`2000-12-31 23:59:55 +0000`))
		})

		Convey("Calendar order: B, M, E", func() {
			reader := plainDataReader(
				`Feb 10 23:59:55 mail dovecot: pop3-login:
Nov 19 00:00:01 mail postfix/postscreen[17274]: DISCONNECT [224.35.90.202]:54744`)
			date, err := guessInitialDateForFile(reader, testutil.MustParseTime(`2001-05-01 00:00:01 +0000`))
			So(err, ShouldBeNil)
			So(date, ShouldEqual, testutil.MustParseTime(`2000-02-10 23:59:55 +0000`))
		})

		Convey("Calendar order: E, B, M", func() {
			reader := plainDataReader(
				`Oct 11 23:59:55 mail dovecot: pop3-login:
Jan 19 00:00:01 mail postfix/postscreen[17274]: DISCONNECT [224.35.90.202]:54744`)
			date, err := guessInitialDateForFile(reader, testutil.MustParseTime(`2001-11-01 00:00:01 +0000`))
			So(err, ShouldBeNil)
			So(date, ShouldEqual, testutil.MustParseTime(`2000-10-11 23:59:55 +0000`))
		})

		Convey("Calendar order: M, E, B", func() {
			reader := plainDataReader(
				`Oct 11 23:59:55 mail dovecot: pop3-login:
Jul 19 00:00:01 mail postfix/postscreen[17274]: DISCONNECT [224.35.90.202]:54744`)
			date, err := guessInitialDateForFile(reader, testutil.MustParseTime(`2001-03-01 00:00:01 +0000`))
			So(err, ShouldBeNil)
			So(date, ShouldEqual, testutil.MustParseTime(`2000-10-11 23:59:55 +0000`))
		})
	})
}

func TestBuildingfileQueues(t *testing.T) {
	Convey("Build file Queues", t, func() {
		Convey("No files at all", func() {
			So(buildFilesToImport(fileEntryList{}, logPatterns{}, testutil.MustParseTime(`1970-01-01 12:00:00 +0000`)), ShouldResemble, fileQueues{})
		})

		Convey("No matching files", func() {
			f := fileEntryList{
				fileEntry{filename: "file.1"},
				fileEntry{filename: "another_file.2"},
			}

			So(buildFilesToImport(f, logPatterns{"mail.log"}, testutil.MustParseTime(`1970-01-01 12:00:00 +0000`)), ShouldResemble, fileQueues{"mail.log": {}})
		})

		Convey("One matching file in one queue", func() {
			f := fileEntryList{
				fileEntry{filename: "file.1", modificationTime: testutil.MustParseTime(`2020-02-15 11:35:44 +0200`)},
				fileEntry{filename: "mail.log", modificationTime: testutil.MustParseTime(`2020-02-16 11:35:44 +0200`)},
				fileEntry{filename: "not_mail.log", modificationTime: testutil.MustParseTime(`2020-02-17 11:35:44 +0200`)},
			}

			So(buildFilesToImport(f, logPatterns{"mail.log"}, testutil.MustParseTime(`1970-01-01 12:00:00 +0000`)), ShouldResemble,
				fileQueues{"mail.log": fileEntryList{
					fileEntry{
						filename: "mail.log", modificationTime: testutil.MustParseTime(`2020-02-16 11:35:44 +0200`),
					},
				}})
		})

		Convey("Current file is always used, even if it's older than the requested time (issue #309)", func() {
			f := fileEntryList{
				fileEntry{filename: "mail.log.1", modificationTime: testutil.MustParseTime(`2020-01-16 11:35:44 +0200`)},
				fileEntry{filename: "mail.log", modificationTime: testutil.MustParseTime(`2020-02-16 11:35:44 +0200`)},
			}

			So(buildFilesToImport(f, logPatterns{"mail.log"}, testutil.MustParseTime(`2020-03-01 12:00:00 +0000`)), ShouldResemble,
				fileQueues{"mail.log": fileEntryList{
					fileEntry{
						filename: "mail.log", modificationTime: testutil.MustParseTime(`2020-02-16 11:35:44 +0200`),
					},
				}})
		})

		Convey("No entry files", func() {
			f := fileEntryList{}
			So(buildFilesToImport(f, logPatterns{"mail.log"}, testutil.MustParseTime(`1970-01-01 12:00:00 +0000`)), ShouldResemble,
				fileQueues{"mail.log": fileEntryList{}})
		})

		Convey("Match Several files with several patterns", func() {
			f := fileEntryList{
				fileEntry{filename: "logs/mail.warn.4.gz", modificationTime: testutil.MustParseTime(`2020-03-08 07:43:48 +0200`)},
				fileEntry{filename: "logs/mail.info.1", modificationTime: testutil.MustParseTime(`2020-03-29 08:51:33 +0200`)},
				fileEntry{filename: "logs/mail.info.3.gz", modificationTime: testutil.MustParseTime(`2020-03-16 07:42:56 +0200`)},
				fileEntry{filename: "logs/mail.log.1", modificationTime: testutil.MustParseTime(`2020-04-03 08:36:24 +0200`)},
				fileEntry{filename: "logs/mail.warn", modificationTime: testutil.MustParseTime(`2020-04-03 18:42:48 +0200`)},
				fileEntry{filename: "logs/mail.info.4.gz", modificationTime: testutil.MustParseTime(`2020-03-08 07:43:48 +0200`)},
				fileEntry{filename: "logs/clamav.log", modificationTime: testutil.MustParseTime(`2020-02-14 11:35:44 +0200`)},
				fileEntry{filename: "logs/mail.info", modificationTime: testutil.MustParseTime(`2020-04-03 18:58:34 +0200`)},
				fileEntry{filename: "logs/mail.warn.2.gz", modificationTime: testutil.MustParseTime(`2020-03-22 07:25:05 +0200`)},
				fileEntry{filename: "logs/freshclam.log", modificationTime: testutil.MustParseTime(`2020-02-14 11:35:44 +0200`)},
				fileEntry{filename: "logs/mail.err", modificationTime: testutil.MustParseTime(`2020-03-23 07:39:09 +0200`)},
				fileEntry{filename: "logs/mail.warn.3.gz", modificationTime: testutil.MustParseTime(`2020-03-16 07:42:56 +0200`)},
				fileEntry{filename: "logs/mail.log", modificationTime: testutil.MustParseTime(`2020-04-03 18:58:34 +0200`)},
				fileEntry{filename: "logs/mail.err.3.gz", modificationTime: testutil.MustParseTime(`2020-03-11 07:39:14 +0200`)},
				fileEntry{filename: "logs/mail.err.4.gz", modificationTime: testutil.MustParseTime(`2020-02-16 07:54:10 +0200`)},
				fileEntry{filename: "logs/mail.warn.1", modificationTime: testutil.MustParseTime(`2020-03-29 08:51:33 +0200`)},
				fileEntry{filename: "logs/mail.info.2.gz", modificationTime: testutil.MustParseTime(`2020-03-22 07:25:05 +0200`)},
				fileEntry{filename: "logs/mail.err.2.gz", modificationTime: testutil.MustParseTime(`2020-03-15 07:39:37 +0200`)},
				fileEntry{filename: "logs/mail.err.1", modificationTime: testutil.MustParseTime(`2020-03-23 07:39:09 +0200`)},
			}

			// select all files modified after Mar 10, 00:00:00
			So(buildFilesToImport(f, logPatterns{"mail.log", "mail.err", "mail.warn"}, testutil.MustParseTime(`2020-03-10 00:00:00 +0200`)), ShouldResemble,
				fileQueues{
					"mail.log": fileEntryList{
						fileEntry{filename: "logs/mail.log.1", modificationTime: testutil.MustParseTime(`2020-04-03 08:36:24 +0200`)},
						fileEntry{filename: "logs/mail.log", modificationTime: testutil.MustParseTime(`2020-04-03 18:58:34 +0200`)},
					},
					"mail.err": fileEntryList{
						fileEntry{filename: "logs/mail.err.3.gz", modificationTime: testutil.MustParseTime(`2020-03-11 07:39:14 +0200`)},
						fileEntry{filename: "logs/mail.err.2.gz", modificationTime: testutil.MustParseTime(`2020-03-15 07:39:37 +0200`)},
						fileEntry{filename: "logs/mail.err.1", modificationTime: testutil.MustParseTime(`2020-03-23 07:39:09 +0200`)},
						fileEntry{filename: "logs/mail.err", modificationTime: testutil.MustParseTime(`2020-03-23 07:39:09 +0200`)},
					},
					"mail.warn": fileEntryList{
						fileEntry{filename: "logs/mail.warn.3.gz", modificationTime: testutil.MustParseTime(`2020-03-16 07:42:56 +0200`)},
						fileEntry{filename: "logs/mail.warn.2.gz", modificationTime: testutil.MustParseTime(`2020-03-22 07:25:05 +0200`)},
						fileEntry{filename: "logs/mail.warn.1", modificationTime: testutil.MustParseTime(`2020-03-29 08:51:33 +0200`)},
						fileEntry{filename: "logs/mail.warn", modificationTime: testutil.MustParseTime(`2020-04-03 18:42:48 +0200`)},
					},
				})
		})

		Convey("Match Several files with several patterns, alternative suffix containing date", func() {
			f := fileEntryList{
				fileEntry{filename: "logs/mail.err", modificationTime: testutil.MustParseTime(`2020-03-23 07:39:09 +0200`)},
				fileEntry{filename: "logs/mail.warn-20201004", modificationTime: testutil.MustParseTime(`2020-01-04 07:43:48 +0200`)},
				fileEntry{filename: "logs/mail.warn-20201001.gz", modificationTime: testutil.MustParseTime(`2020-01-01 08:51:33 +0200`)},
				fileEntry{filename: "logs/mail.err-20201001.gz", modificationTime: testutil.MustParseTime(`2020-01-01 07:39:09 +0200`)},
				fileEntry{filename: "logs/mail.info-20201002.gz", modificationTime: testutil.MustParseTime(`2020-01-02 07:25:05 +0200`)},
				fileEntry{filename: "logs/mail.warn-20201002.gz", modificationTime: testutil.MustParseTime(`2020-01-02 07:25:05 +0200`)},
				fileEntry{filename: "logs/mail.err-20201003.gz", modificationTime: testutil.MustParseTime(`2020-01-03 07:39:14 +0200`)},
				fileEntry{filename: "logs/mail.info-20201004", modificationTime: testutil.MustParseTime(`2020-01-04 07:43:48 +0200`)},
				fileEntry{filename: "logs/mail.info-20201001.gz", modificationTime: testutil.MustParseTime(`2020-01-01 08:51:33 +0200`)},
				fileEntry{filename: "logs/mail.err-20201004", modificationTime: testutil.MustParseTime(`2020-01-04 07:54:10 +0200`)},
				fileEntry{filename: "logs/mail.log-20201001", modificationTime: testutil.MustParseTime(`2020-01-01 08:36:24 +0200`)},
				fileEntry{filename: "logs/clamav.log", modificationTime: testutil.MustParseTime(`2020-02-14 11:35:44 +0200`)},
				fileEntry{filename: "logs/mail.info-20201003.gz", modificationTime: testutil.MustParseTime(`2020-01-03 07:42:56 +0200`)},
				fileEntry{filename: "logs/mail.err-20201002.gz", modificationTime: testutil.MustParseTime(`2020-01-02 07:39:37 +0200`)},
				fileEntry{filename: "logs/mail.info", modificationTime: testutil.MustParseTime(`2020-04-03 18:58:34 +0200`)},
				fileEntry{filename: "logs/mail.warn", modificationTime: testutil.MustParseTime(`2020-04-03 18:42:48 +0200`)},
				fileEntry{filename: "logs/mail.log", modificationTime: testutil.MustParseTime(`2020-04-03 18:58:34 +0200`)},
				fileEntry{filename: "logs/mail.warn-20201003.gz", modificationTime: testutil.MustParseTime(`2020-01-03 07:42:56 +0200`)},
				fileEntry{filename: "logs/freshclam.log", modificationTime: testutil.MustParseTime(`2020-02-14 11:35:44 +0200`)},
			}

			So(buildFilesToImport(f, logPatterns{"mail.log", "mail.err", "mail.warn"}, time.Time{}), ShouldResemble,
				fileQueues{
					"mail.log": fileEntryList{
						fileEntry{filename: "logs/mail.log-20201001", modificationTime: testutil.MustParseTime(`2020-01-01 08:36:24 +0200`)},
						fileEntry{filename: "logs/mail.log", modificationTime: testutil.MustParseTime(`2020-04-03 18:58:34 +0200`)},
					},
					"mail.err": fileEntryList{
						fileEntry{filename: "logs/mail.err-20201001.gz", modificationTime: testutil.MustParseTime(`2020-01-01 07:39:09 +0200`)},
						fileEntry{filename: "logs/mail.err-20201002.gz", modificationTime: testutil.MustParseTime(`2020-01-02 07:39:37 +0200`)},
						fileEntry{filename: "logs/mail.err-20201003.gz", modificationTime: testutil.MustParseTime(`2020-01-03 07:39:14 +0200`)},
						fileEntry{filename: "logs/mail.err-20201004", modificationTime: testutil.MustParseTime(`2020-01-04 07:54:10 +0200`)},
						fileEntry{filename: "logs/mail.err", modificationTime: testutil.MustParseTime(`2020-03-23 07:39:09 +0200`)},
					},
					"mail.warn": fileEntryList{
						fileEntry{filename: "logs/mail.warn-20201001.gz", modificationTime: testutil.MustParseTime(`2020-01-01 08:51:33 +0200`)},
						fileEntry{filename: "logs/mail.warn-20201002.gz", modificationTime: testutil.MustParseTime(`2020-01-02 07:25:05 +0200`)},
						fileEntry{filename: "logs/mail.warn-20201003.gz", modificationTime: testutil.MustParseTime(`2020-01-03 07:42:56 +0200`)},
						fileEntry{filename: "logs/mail.warn-20201004", modificationTime: testutil.MustParseTime(`2020-01-04 07:43:48 +0200`)},
						fileEntry{filename: "logs/mail.warn", modificationTime: testutil.MustParseTime(`2020-04-03 18:42:48 +0200`)},
					},
				})
		})

	})
}

func TestMultipleFiles(t *testing.T) {
	Convey("Finds start time among several files", t, func() {
		Convey("No files triggers errors", func() {
			_, err := findEarlierstTimeFromFiles([]fileDescriptor{})
			So(err, ShouldNotBeNil)
		})

		Convey("One file only with no change in year", func() {
			t, err := findEarlierstTimeFromFiles([]fileDescriptor{
				{modificationTime: testutil.MustParseTime(`2020-04-03 19:01:53 +0000`),
					reader: gzipedDataReader(
						`Mar 22 06:28:55 mail dovecot: Useless Payload
Mar 29 06:47:09 mail postfix/postscreen[17274]: Useless Payload`),
				},
			})
			So(err, ShouldBeNil)
			So(t, ShouldEqual, testutil.MustParseTime(`2020-03-22 06:28:55 +0000`))
		})

		Convey("One file only with a change in year", func() {
			t, err := findEarlierstTimeFromFiles([]fileDescriptor{
				{modificationTime: testutil.MustParseTime(`2020-04-03 19:01:53 +0000`),
					reader: gzipedDataReader(
						`Dec 22 06:28:55 mail dovecot: Useless Payload
Dec 23 06:28:55 mail dovecot: Useless Payload
Mar 29 06:47:09 mail postfix/postscreen[17274]: Useless Payload`),
				},
			})
			So(err, ShouldBeNil)
			So(t, ShouldEqual, testutil.MustParseTime(`2019-12-22 06:28:55 +0000`))
		})

		Convey("Multiple files, with one of them changing year", func() {
			t, err := findEarlierstTimeFromFiles([]fileDescriptor{
				{modificationTime: testutil.MustParseTime(`2020-03-30 19:01:53 +0000`),
					reader: gzipedDataReader(
						`Dec 22 06:28:55 mail dovecot: Useless Payload
Mar 29 06:47:09 mail postfix/postscreen[17274]: Useless Payload`),
				},
				{modificationTime: testutil.MustParseTime(`2020-02-28 18:18:10 +0000`),
					reader: plainDataReader(
						`Nov  3 01:33:20 mail dovecot: Useless Payload
Nov 23 06:28:55 mail dovecot: Useless Payload
Feb 28 06:47:09 mail postfix/postscreen[17274]: Useless Payload`),
				},
				{modificationTime: testutil.MustParseTime(`2020-01-31 19:01:53 +0000`),
					reader: plainDataReader(
						`Jan 22 06:28:55 mail dovecot: Useless Payload
Jan 23 06:28:55 mail dovecot: Useless Payload
Jan 31 06:47:09 mail postfix/postscreen[17274]: Useless Payload`),
				},
			})
			So(err, ShouldBeNil)
			So(t, ShouldEqual, testutil.MustParseTime(`2019-11-03 01:33:20 +0000`))
		})
	})

	Convey("Finds start time among several files from directory contents", t, func() {
		dirContent := FakeDirectoryContent{
			entries: fileEntryList{
				fileEntry{filename: "log/mail.warn.2.gz", modificationTime: testutil.MustParseTime(`2020-03-29 19:01:53 +0000`)},
				fileEntry{filename: "log/mail.warn", modificationTime: testutil.MustParseTime(`2020-03-30 19:01:53 +0000`)},
				fileEntry{filename: "log/mail.log", modificationTime: testutil.MustParseTime(`2020-02-28 18:18:10 +0000`)},
				fileEntry{filename: "log/mail.err", modificationTime: testutil.MustParseTime(`2020-01-31 19:01:53 +0000`)},
			},
			contents: map[string]fakeFileData{
				"log/mail.warn.2.gz": gzippedDataFile(`Dec 22 06:28:55 mail dovecot: Useless Payload
Mar 29 06:47:09 mail postfix/postscreen[17274]: Useless Payload`),
				"log/mail.warn": plainDataFile(`Mar 30 01:33:20 mail dovecot: Useless Payload`),
				"log/mail.log": plainDataFile(`Nov  3 01:33:20 mail dovecot: Useless Payload
Nov 23 06:28:55 mail dovecot: Useless Payload
Feb 28 06:47:09 mail postfix/postscreen[17274]: Useless Payload`),
				"log/mail.err": plainDataFile(`Jan 22 06:28:55 mail dovecot: Useless Payload
Jan 23 06:28:55 mail dovecot: Useless Payload
Jan 31 06:47:09 mail postfix/postscreen[17274]: Useless Payload`),
			},
		}

		t, err := FindInitialLogTime(dirContent)
		So(err, ShouldBeNil)
		So(t, ShouldEqual, testutil.MustParseTime(`2019-11-03 01:33:20 +0000`))
	})
}

func TestImportDirectoryOnly(t *testing.T) {
	Convey("Import Files from Directory", t, func() {
		Convey("Empty directory yields no logs", func() {
			dirContent := FakeDirectoryContent{entries: fileEntryList{}}
			pub := fakePublisher{}
			importer := NewDirectoryImporter(dirContent, &pub, testutil.MustParseTime(`1970-01-01 00:00:00 +0000`))
			err := importer.Run()
			So(err, ShouldNotBeNil)
			So(len(pub.logs), ShouldEqual, 0)
		})

		Convey("One file returns its contents", func() {
			dirContent := FakeDirectoryContent{
				entries: fileEntryList{
					fileEntry{filename: "mail.log.2.gz", modificationTime: testutil.MustParseTime(`2020-01-31 19:01:53 +0000`)},
					fileEntry{filename: "mail.log", modificationTime: testutil.MustParseTime(`2020-03-31 00:00:00 +0000`)},
				},
				contents: map[string]fakeFileData{
					"mail.log.2.gz": gzippedDataFile(`Jan 22 06:28:55 mail dovecot: Useless Payload
Jan 23 13:46:15 mail dovecot: Useless Payload
Jan 31 08:47:09 mail postfix/postscreen[17274]: Useless Payload`),
					"mail.log": plainCurrentDataFile(``, ``),
				},
			}
			pub := fakePublisher{}
			importer := NewDirectoryImporter(dirContent, &pub, testutil.MustParseTime(`1970-01-01 00:00:00 +0000`))
			err := importer.Run()
			So(err, ShouldBeNil)
			So(len(pub.logs), ShouldEqual, 3)
			So(pub.logs[0].Header.Time, ShouldResemble, parser.Time{Month: time.January, Day: 22, Hour: 6, Minute: 28, Second: 55})
			So(pub.logs[1].Header.Time, ShouldResemble, parser.Time{Month: time.January, Day: 23, Hour: 13, Minute: 46, Second: 15})
			So(pub.logs[2].Header.Time, ShouldResemble, parser.Time{Month: time.January, Day: 31, Hour: 8, Minute: 47, Second: 9})
		})

		Convey("Many files in the same queue, no new lines after import", func() {
			dirContent := FakeDirectoryContent{
				entries: fileEntryList{
					fileEntry{filename: "log/mail.log.4.gz", modificationTime: testutil.MustParseTime(`2020-03-13 04:01:53 +0000`)},
					fileEntry{filename: "log/mail.log.5.gz", modificationTime: testutil.MustParseTime(`2020-02-14 06:01:53 +0000`)},
					fileEntry{filename: "log/mail.log.1", modificationTime: testutil.MustParseTime(`2020-07-14 06:01:53 +0000`)},
					fileEntry{filename: "log/mail.log.2.gz", modificationTime: testutil.MustParseTime(`2020-06-18 07:01:53 +0000`)},
					fileEntry{filename: "log/mail.log", modificationTime: testutil.MustParseTime(`2020-08-19 12:00:00 +0000`)},
				},
				contents: map[string]fakeFileData{
					"log/mail.log.5.gz": gzippedDataFile(`Feb  1 12:00:00 mail dovecot: Useless Payload
Feb 14 06:01:53 mail postfix/postscreen[17274]: Useless Payload`),
					"log/mail.log.4.gz": gzippedDataFile(`Feb 14 06:01:54 mail dovecot: Useless Payload
Feb 23 13:46:15 mail dovecot: Useless Payload
Mar 13 04:00:09 mail postfix/postscreen[17274]: Useless Payload`),
					"log/mail.log.2.gz": gzippedDataFile(`Mar 14 06:28:55 mail dovecot: Useless Payload
Jun 18 06:28:55 mail someotherstuff: useless`),
					"log/mail.log.1": plainDataFile(`Jun 18 08:29:33 mail dovecot: Useless Payload
Jul 14 07:01:53 mail postfix/postscreen[17274]: Useless Payload`),
					"log/mail.log": plainCurrentDataFile(`Jul 18 00:00:00 mail dovecot: Useless Payload
Aug 10 00:00:40 mail postfix/postscreen[17274]: Useless Payload`, ``),
				},
			}
			pub := fakePublisher{}
			importer := NewDirectoryImporter(dirContent, &pub, testutil.MustParseTime(`1970-01-01 00:00:00 +0000`))
			err := importer.Run()
			So(err, ShouldBeNil)
			So(len(pub.logs), ShouldEqual, 11)

			So(pub.logs[0].Header.Time, ShouldResemble, parser.Time{Month: time.February, Day: 1, Hour: 12, Minute: 00, Second: 0})
			So(pub.logs[1].Header.Time, ShouldResemble, parser.Time{Month: time.February, Day: 14, Hour: 6, Minute: 1, Second: 53})

			So(pub.logs[2].Header.Time, ShouldResemble, parser.Time{Month: time.February, Day: 14, Hour: 6, Minute: 1, Second: 54})
			So(pub.logs[3].Header.Time, ShouldResemble, parser.Time{Month: time.February, Day: 23, Hour: 13, Minute: 46, Second: 15})
			So(pub.logs[4].Header.Time, ShouldResemble, parser.Time{Month: time.March, Day: 13, Hour: 4, Minute: 0, Second: 9})

			So(pub.logs[5].Header.Time, ShouldResemble, parser.Time{Month: time.March, Day: 14, Hour: 6, Minute: 28, Second: 55})
			So(pub.logs[6].Header.Time, ShouldResemble, parser.Time{Month: time.June, Day: 18, Hour: 6, Minute: 28, Second: 55})

			So(pub.logs[7].Header.Time, ShouldResemble, parser.Time{Month: time.June, Day: 18, Hour: 8, Minute: 29, Second: 33})
			So(pub.logs[8].Header.Time, ShouldResemble, parser.Time{Month: time.July, Day: 14, Hour: 7, Minute: 1, Second: 53})

			So(pub.logs[9].Header.Time, ShouldResemble, parser.Time{Month: time.July, Day: 18, Hour: 0, Minute: 0, Second: 0})
			So(pub.logs[10].Header.Time, ShouldResemble, parser.Time{Month: time.August, Day: 10, Hour: 0, Minute: 0, Second: 40})
		})

		Convey("Get only logs after a point in time in the middle of a file", func() {
			dirContent := FakeDirectoryContent{
				entries: fileEntryList{
					fileEntry{filename: "log/mail.log.4.gz", modificationTime: testutil.MustParseTime(`2020-03-13 04:01:53 +0000`)},
					fileEntry{filename: "log/mail.log.5.gz", modificationTime: testutil.MustParseTime(`2020-02-14 06:01:53 +0000`)},
					fileEntry{filename: "log/mail.log.1", modificationTime: testutil.MustParseTime(`2020-07-14 06:01:53 +0000`)},
					fileEntry{filename: "log/mail.log.2.gz", modificationTime: testutil.MustParseTime(`2020-06-18 07:01:53 +0000`)},
					fileEntry{filename: "log/mail.log", modificationTime: testutil.MustParseTime(`2020-08-19 12:00:00 +0000`)},
				},
				contents: map[string]fakeFileData{
					"log/mail.log.5.gz": gzippedDataFile(`Feb  1 12:00:00 mail dovecot: Useless Payload
Feb 14 06:01:53 mail postfix/postscreen[17274]: Useless Payload`),
					"log/mail.log.4.gz": gzippedDataFile(`Feb 14 06:01:54 mail dovecot: Useless Payload
Feb 23 13:46:15 mail dovecot: Useless Payload
Mar 13 04:00:09 mail postfix/postscreen[17274]: Useless Payload`),
					"log/mail.log.2.gz": gzippedDataFile(`Mar 14 06:28:55 mail dovecot: Useless Payload
Jun 18 06:28:55 mail someotherstuff: useless`),
					"log/mail.log.1": plainDataFile(`Jun 18 08:29:33 mail dovecot: Useless Payload
Jul 14 07:01:53 mail postfix/postscreen[17274]: Useless Payload`),
					"log/mail.log": plainCurrentDataFile(`Jul 18 00:00:00 mail dovecot: Useless Payload
Aug 10 00:00:40 mail postfix/postscreen[17274]: Useless Payload`, ``),
				},
			}
			pub := fakePublisher{}
			importer := NewDirectoryImporter(dirContent, &pub, testutil.MustParseTime(`2020-06-18 06:28:54 +0000`))
			err := importer.Run()
			So(err, ShouldBeNil)
			So(len(pub.logs), ShouldEqual, 5)

			So(pub.logs[0].Header.Time, ShouldResemble, parser.Time{Month: time.June, Day: 18, Hour: 6, Minute: 28, Second: 55})

			So(pub.logs[1].Header.Time, ShouldResemble, parser.Time{Month: time.June, Day: 18, Hour: 8, Minute: 29, Second: 33})
			So(pub.logs[2].Header.Time, ShouldResemble, parser.Time{Month: time.July, Day: 14, Hour: 7, Minute: 1, Second: 53})

			So(pub.logs[3].Header.Time, ShouldResemble, parser.Time{Month: time.July, Day: 18, Hour: 0, Minute: 0, Second: 0})
			So(pub.logs[4].Header.Time, ShouldResemble, parser.Time{Month: time.August, Day: 10, Hour: 0, Minute: 0, Second: 40})
		})

		Convey("Import only, not watching new log entries", func() {
			dirContent := FakeDirectoryContent{
				entries: fileEntryList{
					fileEntry{filename: "log/mail.log.1", modificationTime: testutil.MustParseTime(`2020-07-14 06:01:53 +0000`)},
					fileEntry{filename: "log/mail.log", modificationTime: testutil.MustParseTime(`2020-08-19 12:00:00 +0000`)},
				},
				contents: map[string]fakeFileData{
					"log/mail.log.1": plainDataFile(`Jun 18 08:29:33 mail dovecot: Useless Payload
Jul 14 07:01:53 mail postfix/postscreen[17274]: Useless Payload`),
					"log/mail.log": plainCurrentDataFile(`Jul 18 00:00:00 mail dovecot: Useless Payload`,
						`Aug 10 00:00:40 mail postfix/postscreen[17274]: Useless Payload`),
				},
			}
			pub := fakePublisher{}
			importer := NewDirectoryImporter(dirContent, &pub, testutil.MustParseTime(`1970-01-01 00:00:00 +0000`))
			err := importer.ImportOnly()
			So(err, ShouldBeNil)
			So(len(pub.logs), ShouldEqual, 3)

			So(pub.logs[0].Header.Time, ShouldResemble, parser.Time{Month: time.June, Day: 18, Hour: 8, Minute: 29, Second: 33})
			So(pub.logs[1].Header.Time, ShouldResemble, parser.Time{Month: time.July, Day: 14, Hour: 7, Minute: 1, Second: 53})
			So(pub.logs[2].Header.Time, ShouldResemble, parser.Time{Month: time.July, Day: 18, Hour: 0, Minute: 0, Second: 0})
		})

		Convey("Multiple files in multiple queues, no new lines after files are open", func() {
			dirContent := FakeDirectoryContent{
				entries: fileEntryList{
					fileEntry{filename: "mail.err", modificationTime: testutil.MustParseTime(`2001-01-01 00:00:03 +0000`)},
					fileEntry{filename: "mail.err.1", modificationTime: testutil.MustParseTime(`2001-01-01 00:00:01 +0000`)},
					fileEntry{filename: "mail.log", modificationTime: testutil.MustParseTime(`2001-01-01 00:00:30 +0000`)},
					fileEntry{filename: "mail.log.1", modificationTime: testutil.MustParseTime(`2000-12-31 23:59:56 +0000`)},
				},
				contents: map[string]fakeFileData{
					"mail.err": plainCurrentDataFile(`Jan  1 00:00:02 mail postfix/postscreen[18660]: a`, ``),

					"mail.err.1": plainDataFile(`Dec 31 23:59:55 mail dovecot: pop3-login:
Jan  1 00:00:01 mail postfix/postscreen[9183]: CONNECT from [18.88.247.65]:50082 to [170.68.1.1]:25
Jan  1 00:00:01 mail postfix/postscreen[17274]: DISCONNECT [224.35.90.202]:54744`),

					"mail.log": plainCurrentDataFile(`Dec 31 23:59:57 mail dovecot: imap-login:
Jan  1 00:00:00 mail dovecot: imap`, ``),

					"mail.log.1": plainDataFile(`Dec 31 23:59:50 mail postfix/postscreen[26735]: CONNECT
Dec 31 23:59:55 mail dovecot: imap-login:`),
				},
			}
			pub := fakePublisher{}
			importer := NewDirectoryImporter(dirContent, &pub, testutil.MustParseTime(`1970-01-01 00:00:00 +0000`))
			err := importer.Run()
			So(err, ShouldBeNil)
			So(len(pub.logs), ShouldEqual, 8)
			So(pub.logs[0].Header.Time, ShouldResemble, parser.Time{Month: time.December, Day: 31, Hour: 23, Minute: 59, Second: 50})
			So(pub.logs[1].Header.Time, ShouldResemble, parser.Time{Month: time.December, Day: 31, Hour: 23, Minute: 59, Second: 55})
			So(pub.logs[2].Header.Time, ShouldResemble, parser.Time{Month: time.December, Day: 31, Hour: 23, Minute: 59, Second: 55})
			So(pub.logs[3].Header.Time, ShouldResemble, parser.Time{Month: time.December, Day: 31, Hour: 23, Minute: 59, Second: 57})
			So(pub.logs[4].Header.Time, ShouldResemble, parser.Time{Month: time.January, Day: 1, Hour: 0, Minute: 0, Second: 0})
			So(pub.logs[5].Header.Time, ShouldResemble, parser.Time{Month: time.January, Day: 1, Hour: 0, Minute: 0, Second: 1})
			So(pub.logs[6].Header.Time, ShouldResemble, parser.Time{Month: time.January, Day: 1, Hour: 0, Minute: 0, Second: 1})
			So(pub.logs[7].Header.Time, ShouldResemble, parser.Time{Month: time.January, Day: 1, Hour: 0, Minute: 0, Second: 2})
		})

		Convey("Many files in many queues, long run, without new log lines added to the current files", func() {
			dirContent := FakeDirectoryContent{
				entries: fileEntryList{
					fileEntry{filename: "clamav.log", modificationTime: testutil.MustParseTime(`2020-04-03 19:01:53 +0200`)},
					fileEntry{filename: "freshclam.log", modificationTime: testutil.MustParseTime(`2020-04-03 19:01:53 +0200`)},
					fileEntry{filename: "mail.err", modificationTime: testutil.MustParseTime(`2020-04-03 19:01:53 +0200`)},
					fileEntry{filename: "mail.err.1", modificationTime: testutil.MustParseTime(`2020-04-03 19:01:53 +0200`)},
					fileEntry{filename: "mail.err.2.gz", modificationTime: testutil.MustParseTime(`2020-04-03 19:01:53 +0200`)},
					fileEntry{filename: "mail.err.3.gz", modificationTime: testutil.MustParseTime(`2020-04-03 19:01:53 +0200`)},
					fileEntry{filename: "mail.err.4.gz", modificationTime: testutil.MustParseTime(`2020-04-03 19:01:53 +0200`)},
					fileEntry{filename: "mail.log", modificationTime: testutil.MustParseTime(`2020-04-03 19:01:54 +0200`)},
					fileEntry{filename: "mail.log.1", modificationTime: testutil.MustParseTime(`2020-04-03 19:01:54 +0200`)},
					fileEntry{filename: "mail.warn", modificationTime: testutil.MustParseTime(`2020-04-03 19:01:54 +0200`)},
					fileEntry{filename: "mail.warn.1", modificationTime: testutil.MustParseTime(`2020-04-03 19:01:54 +0200`)},
					fileEntry{filename: "mail.warn.2.gz", modificationTime: testutil.MustParseTime(`2020-04-03 19:01:54 +0200`)},
					fileEntry{filename: "mail.warn.3.gz", modificationTime: testutil.MustParseTime(`2020-04-03 19:01:54 +0200`)},
					fileEntry{filename: "mail.warn.4.gz", modificationTime: testutil.MustParseTime(`2020-04-03 19:01:54 +0200`)},
				},
				contents: map[string]fakeFileData{
					"clamav.log":    plainDataFile(``),
					"freshclam.log": plainDataFile(``),
					"mail.err":      plainCurrentDataFile(``, ``),

					"mail.err.1": plainDataFile(`Mar 17 07:41:11 mail opendkim[225]: B35AF2C620BA
Mar 17 07:41:11 mail opendkim[225]: B35AF2C620BA: aaa`),

					"mail.err.2.gz": gzippedDataFile(`Mar 11 09:50:15 mail opendkim[225]: a
Mar 11 15:18:59 mail opendkim[225]: 6AB452C620C3: key retrieval
Mar 11 15:18:59 mail opendkim[225]: 6AB452C620C3: key`),

					"mail.err.3.gz": gzippedDataFile(`Mar 10 14:34:14 mail opendkim[225]: a
Mar 10 14:34:14 mail opendkim[225]: D76632C620BA: key`),

					"mail.err.4.gz": gzippedDataFile(`Feb 15 20:34:29 mail opendkim[225]: a
Feb 15 20:34:29 mail opendkim[225]: 1CDB02C620B5: key`),

					"mail.log": plainCurrentDataFile(`Apr  3 06:40:07 mail dovecot: imap-login:
Apr  3 06:41:08 mail postfix/qmgr[10471]: 2E4522C620DA:
Apr  3 16:58:34 mail dovecot: imap`, ``),

					"mail.log.1": plainDataFile(`Apr  2 06:46:27 mail postfix/postscreen[26735]: CONNECT
Apr  2 06:46:27 mail postfix/dnsblog[26740]: addr
Apr  3 06:18:09 mail dovecot: imap-login:`),

					"mail.warn": plainCurrentDataFile(`Mar 29 08:41:37 mail postfix/smtpd[8479]: warning:
Mar 29 09:33:36 mail postfix/smtpd[19194]: warning: TLS library problem:
Apr  3 16:42:48 mail postfix/smtpd[21096]: warning:`, ``),

					"mail.warn.1": plainDataFile(`Mar 22 14:41:06 mail postfix/smtpd[625]: warning: TLS
Mar 22 15:02:16 mail postfix/smtpd[4998]: warning: TLS library problem:
Mar 29 02:36:03 mail postfix/smtpd[30701]: warning: `),

					"mail.warn.2.gz": gzippedDataFile(`Mar 16 07:34:46 mail postfix/smtpd[4306]: warning:
Mar 16 07:56:32 mail postfix/submission/smtpd[8736]: warning: hostname
Mar 22 03:22:55 mail postfix/smtpd[25150]: warning: `),

					"mail.warn.3.gz": gzippedDataFile(`Mar  8 07:59:10 mail postfix/smtpd[25984]: warning:
Mar  8 08:07:45 mail postfix/smtpd[27740]: warning: TLS library problem:
Mar 16 04:22:45 mail postfix/submission/smtpd[30082]: warning:`),

					"mail.warn.4.gz": gzippedDataFile(`Mar  1 11:09:14 mail postfix/submission/smtpd[26975]: warning:
Mar  1 22:32:37 mail postfix/submission/smtpd[3506]: warning:
Mar  8 00:38:13 mail postfix/submission/smtpd[1392]: warning: hostname`),
				},
			}
			pub := fakePublisher{}
			importer := NewDirectoryImporter(dirContent, &pub, testutil.MustParseTime(`1970-01-01 00:00:00 +0000`))
			err := importer.Run()
			So(err, ShouldBeNil)
			So(len(pub.logs), ShouldEqual, 30)
		})
	})
}

func TestImportDirectoryAndWatchNewLines(t *testing.T) {
	Convey("Import Files from Directory", t, func() {
		Convey("Many files in the same queue, with new lines after the import starts, split happens on breakline", func() {
			dirContent := FakeDirectoryContent{
				entries: fileEntryList{
					fileEntry{filename: "log/mail.log.1", modificationTime: testutil.MustParseTime(`2020-07-14 06:01:53 +0000`)},
					fileEntry{filename: "log/mail.log", modificationTime: testutil.MustParseTime(`2020-08-19 12:00:00 +0000`)},
				},
				contents: map[string]fakeFileData{
					"log/mail.log.1": plainDataFile(`Jun 18 08:29:33 mail dovecot: Useless Payload`),
					"log/mail.log": plainCurrentDataFile(`Jul 18 00:00:00 mail dovecot: Useless Payload
`, // NOTE: the breakline is important here to mimic the real content of a file
						`Aug 10 00:00:40 mail postfix/postscreen[17274]: Useless Payload`),
				},
			}

			pub := fakePublisher{}
			importer := NewDirectoryImporter(dirContent, &pub, testutil.MustParseTime(`1970-01-01 00:00:00 +0000`))
			err := importer.Run()
			So(err, ShouldBeNil)
			So(len(pub.logs), ShouldEqual, 3)

			So(pub.logs[0].Header.Time, ShouldResemble, parser.Time{Month: time.June, Day: 18, Hour: 8, Minute: 29, Second: 33})
			So(pub.logs[1].Header.Time, ShouldResemble, parser.Time{Month: time.July, Day: 18, Hour: 0, Minute: 0, Second: 0})
			So(pub.logs[2].Header.Time, ShouldResemble, parser.Time{Month: time.August, Day: 10, Hour: 0, Minute: 0, Second: 40})
		})

		Convey("Multiple files in multiple queues", func() {
			dirContent := FakeDirectoryContent{
				entries: fileEntryList{
					fileEntry{filename: "log/mail.log.1", modificationTime: testutil.MustParseTime(`2020-07-14 06:01:53 +0000`)},
					fileEntry{filename: "log/mail.err", modificationTime: testutil.MustParseTime(`2020-08-19 00:00:00 +0000`)},
					fileEntry{filename: "log/mail.log", modificationTime: testutil.MustParseTime(`2020-08-19 12:00:00 +0000`)},
				},
				contents: map[string]fakeFileData{
					"log/mail.err": plainCurrentDataFile(`Jul 12 00:00:00 mail dovecot: Useless Payload
`,
						`Jul 19 00:00:00 mail dovecot: Useless Payload
Aug 12 00:00:00 mail dovecot: Useless Payload`),
					"log/mail.log.1": plainDataFile(`Jun 18 08:29:33 mail dovecot: Useless Payload`),
					"log/mail.log": plainCurrentDataFile(`Jul 18 00:00:00 mail dovecot: Useless Payload
`, // NOTE: the breakline is important here to mimic the real content of a file
						`Aug 10 00:00:40 mail postfix/postscreen[17274]: Useless Payload`),
				},
			}

			pub := fakePublisher{}
			importer := NewDirectoryImporter(dirContent, &pub, testutil.MustParseTime(`1970-01-01 00:00:00 +0000`))
			err := importer.Run()
			So(err, ShouldBeNil)
			So(len(pub.logs), ShouldEqual, 6)

			So(pub.logs[0].Header.Time, ShouldResemble, parser.Time{Month: time.June, Day: 18, Hour: 8, Minute: 29, Second: 33})
			So(pub.logs[0].Time, ShouldEqual, testutil.MustParseTime(`2020-06-18 08:29:33 +0000`))
			So(pub.logs[1].Header.Time, ShouldResemble, parser.Time{Month: time.July, Day: 12, Hour: 0, Minute: 0, Second: 0})
			So(pub.logs[1].Time, ShouldEqual, testutil.MustParseTime(`2020-07-12 00:00:00 +0000`))
			So(pub.logs[2].Header.Time, ShouldResemble, parser.Time{Month: time.July, Day: 18, Hour: 0, Minute: 0, Second: 0})
			So(pub.logs[2].Time, ShouldEqual, testutil.MustParseTime(`2020-07-18 00:00:00 +0000`))
			So(pub.logs[3].Header.Time, ShouldResemble, parser.Time{Month: time.July, Day: 19, Hour: 0, Minute: 0, Second: 0})
			So(pub.logs[3].Time, ShouldEqual, testutil.MustParseTime(`2020-07-19 00:00:00 +0000`))
			So(pub.logs[4].Header.Time, ShouldResemble, parser.Time{Month: time.August, Day: 10, Hour: 0, Minute: 0, Second: 40})
			So(pub.logs[4].Time, ShouldEqual, testutil.MustParseTime(`2020-08-10 00:00:40 +0000`))
			So(pub.logs[5].Header.Time, ShouldResemble, parser.Time{Month: time.August, Day: 12, Hour: 0, Minute: 0, Second: 0})
			So(pub.logs[5].Time, ShouldEqual, testutil.MustParseTime(`2020-08-12 00:00:00 +0000`))
		})
	})
}
