// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package reader

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/logeater/transform"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"
)

type pub struct {
	logs []postfix.Record
}

func (pub *pub) Publish(r postfix.Record) {
	pub.logs = append(pub.logs, r)
}

type fakeDelayedLine struct {
	content string
	delay   time.Duration
}

type fakeDelayedReader struct {
	currentLine int
	lines       []fakeDelayedLine
}

func (r *fakeDelayedReader) Read(b []byte) (n int, err error) {
	if r.currentLine == len(r.lines) {
		return 0, io.EOF
	}

	delayedLine := r.lines[r.currentLine]
	lineReader := strings.NewReader(delayedLine.content + "\n")
	r.currentLine++
	time.Sleep(delayedLine.delay)

	return lineReader.Read(b)
}

func TestReader(t *testing.T) {
	Convey("Test Reader", t, func() {
		transformer, err := transform.Get("default", 2000)
		So(err, ShouldBeNil)

		fakeAnnouncer := &announcer.DummyImportAnnouncer{}

		pub := pub{}

		Convey("Do not crash when all logs are in the same time (Gitlab issue #508)", func() {
			clock := timeutil.FakeClock{Time: time.Date(2020, time.June, 16, 13, 53, 50, 500, time.UTC)}

			Convey("default log format", func() {
				transformer, err := transform.Get("default", 2020)
				So(err, ShouldBeNil)
				reader := strings.NewReader(`Jun 16 13:53:50 mailgate postfix/smtpd[20252]: connect from unknown[192.168.42.12]
Jun 16 13:53:50 mailgate postfix/smtpd[20252]: connect from unknown[192.168.42.12]`)
				ReadFromReader(reader, &pub, transformer, fakeAnnouncer, &clock, time.Hour*500)
				So(len(pub.logs), ShouldEqual, 2)
			})

			Convey("prepend-rfc3339 log format", func() {
				transformer, err := transform.Get("prepend-rfc3339")
				So(err, ShouldBeNil)
				reader := strings.NewReader(`2021-06-16T13:53:50+02:00 Jun 16 13:53:50 mailgate postfix/smtpd[20252]: connect from unknown[192.168.42.12]
2021-06-16T13:53:50+02:00 Jun 16 13:53:50 mailgate postfix/smtpd[20252]: connect from unknown[192.168.42.12]`)
				ReadFromReader(reader, &pub, transformer, fakeAnnouncer, &clock, time.Hour*500)
				So(len(pub.logs), ShouldEqual, 2)
			})
		})

		Convey("Empty reader should announce progress immediately", func() {
			clock := timeutil.FakeClock{Time: testutil.MustParseTime(`2000-08-24 10:00:00 +0000`)}
			reader := strings.NewReader(``)
			ReadFromReader(reader, &pub, transformer, fakeAnnouncer, &clock, time.Millisecond*500)
			So(len(pub.logs), ShouldEqual, 0)

			So(fakeAnnouncer.Start, ShouldEqual, testutil.MustParseTime(`2000-08-24 10:00:00 +0000`))
			So(fakeAnnouncer.Progress(), ShouldResemble, []announcer.Progress{
				announcer.Progress{Finished: true, Time: testutil.MustParseTime(`2000-08-24 10:00:00 +0000`), Progress: 100},
			})
		})

		Convey("Read without any delays", func() {
			clock := timeutil.FakeClock{Time: testutil.MustParseTime(`2000-08-24 10:00:00 +0000`)}

			reader := strings.NewReader(`Aug 20 02:03:04 mail banana: Useless Payload
Aug 21 03:03:04 mail dog: Useless Payload
Aug 22 03:03:04 mail monkey: Useless Payload
Aug 23 04:03:04 mail gorilla: Useless Payload
Aug 24 05:03:04 mail apple: Useless Payload
`)

			ReadFromReader(reader, &pub, transformer, fakeAnnouncer, &clock, time.Millisecond*500)

			So(len(pub.logs), ShouldEqual, 5)

			So(pub.logs[0], ShouldResemble, postfix.Record{
				Time: testutil.MustParseTime(`2000-08-20 02:03:04 +0000`),
				Header: parser.Header{
					Time:      parser.Time{Month: time.August, Day: 20, Hour: 2, Minute: 3, Second: 4},
					Host:      "mail",
					Process:   "banana",
					Daemon:    "",
					PID:       0,
					ProcessIP: nil,
				},
				Location: postfix.RecordLocation{Line: 1, Filename: "unknown"},
				Payload:  nil,
			})

			So(pub.logs[4], ShouldResemble, postfix.Record{
				Time: testutil.MustParseTime(`2000-08-24 05:03:04 +0000`),
				Header: parser.Header{
					Time:      parser.Time{Month: time.August, Day: 24, Hour: 5, Minute: 3, Second: 4},
					Host:      "mail",
					Process:   "apple",
					Daemon:    "",
					PID:       0,
					ProcessIP: nil,
				},
				Location: postfix.RecordLocation{Line: 5, Filename: "unknown"},
				Payload:  nil,
			})

			So(fakeAnnouncer.Start, ShouldResemble, testutil.MustParseTime(`2000-08-20 02:03:04 +0000`))
			So(fakeAnnouncer.Progress(), ShouldResemble, []announcer.Progress{
				announcer.Progress{Finished: false, Time: testutil.MustParseTime(`2000-08-20 02:03:04 +0000`), Progress: 0},
				announcer.Progress{Finished: false, Time: testutil.MustParseTime(`2000-08-21 03:03:04 +0000`), Progress: 24},
				announcer.Progress{Finished: false, Time: testutil.MustParseTime(`2000-08-22 03:03:04 +0000`), Progress: 47},
				announcer.Progress{Finished: false, Time: testutil.MustParseTime(`2000-08-23 04:03:04 +0000`), Progress: 71},
				announcer.Progress{Finished: false, Time: testutil.MustParseTime(`2000-08-24 05:03:04 +0000`), Progress: 95},
				announcer.Progress{Finished: true, Time: testutil.MustParseTime(`2000-08-24 05:03:04 +0000`), Progress: 100},
			})
		})

		Convey("Empty input timeouts immediately", func() {
			clock := timeutil.FakeClock{Time: testutil.MustParseTime(`2000-08-24 10:00:00 +0000`)}
			reader := &fakeDelayedReader{}
			ReadFromReader(reader, &pub, transformer, fakeAnnouncer, &clock, time.Millisecond*100)
			So(len(pub.logs), ShouldEqual, 0)

			So(fakeAnnouncer.Start, ShouldResemble, testutil.MustParseTime(`2000-08-24 10:00:00 +0000`))
			So(fakeAnnouncer.Progress(), ShouldResemble, []announcer.Progress{
				announcer.Progress{Finished: true, Time: testutil.MustParseTime(`2000-08-24 10:00:00 +0000`), Progress: 100},
			})
		})

		Convey("Read with delays", func() {
			clock := timeutil.FakeClock{Time: testutil.MustParseTime(`2000-08-24 10:00:00 +0000`)}

			reader := &fakeDelayedReader{
				lines: []fakeDelayedLine{
					{content: `Aug 20 02:03:04 mail banana: Useless Payload`, delay: time.Millisecond * 50},
					{content: `Aug 21 03:03:04 mail dog: Useless Payload`},
					{content: `Aug 22 03:03:04 mail monkey: Useless Payload`}, // as the next line timeouts, defined as progress 100%
					{content: `Aug 23 04:03:04 mail gorilla: Useless Payload`, delay: time.Millisecond * 500},
					{content: `Aug 24 05:03:04 mail apple: Useless Payload`},
				},
			}

			ReadFromReader(reader, &pub, transformer, fakeAnnouncer, &clock, time.Millisecond*100)

			So(len(pub.logs), ShouldEqual, 5)

			So(pub.logs[0], ShouldResemble, postfix.Record{
				Time: testutil.MustParseTime(`2000-08-20 02:03:04 +0000`),
				Header: parser.Header{
					Time:      parser.Time{Month: time.August, Day: 20, Hour: 2, Minute: 3, Second: 4},
					Host:      "mail",
					Process:   "banana",
					Daemon:    "",
					PID:       0,
					ProcessIP: nil,
				},
				Location: postfix.RecordLocation{Line: 1, Filename: "unknown"},
				Payload:  nil,
			})

			So(pub.logs[4], ShouldResemble, postfix.Record{
				Time: testutil.MustParseTime(`2000-08-24 05:03:04 +0000`),
				Header: parser.Header{
					Time:      parser.Time{Month: time.August, Day: 24, Hour: 5, Minute: 3, Second: 4},
					Host:      "mail",
					Process:   "apple",
					Daemon:    "",
					PID:       0,
					ProcessIP: nil,
				},
				Location: postfix.RecordLocation{Line: 5, Filename: "unknown"},
				Payload:  nil,
			})

			So(fakeAnnouncer.Start, ShouldResemble, testutil.MustParseTime(`2000-08-20 02:03:04 +0000`))
			So(fakeAnnouncer.Progress(), ShouldResemble, []announcer.Progress{
				announcer.Progress{Finished: false, Time: testutil.MustParseTime(`2000-08-20 02:03:04 +0000`), Progress: 0},
				announcer.Progress{Finished: false, Time: testutil.MustParseTime(`2000-08-21 03:03:04 +0000`), Progress: 24},
				announcer.Progress{Finished: false, Time: testutil.MustParseTime(`2000-08-22 03:03:04 +0000`), Progress: 47},
				announcer.Progress{Finished: true, Time: testutil.MustParseTime(`2000-08-22 03:03:04 +0000`), Progress: 100},
			})
		})

		Convey("Check for delay only after at least one line has been read!", func() {
			clock := timeutil.FakeClock{Time: testutil.MustParseTime(`2000-08-24 10:00:00 +0000`)}

			reader := &fakeDelayedReader{
				lines: []fakeDelayedLine{
					{content: `Aug 20 02:03:04 mail banana: Useless Payload`, delay: time.Millisecond * 500},
					{content: `Aug 21 03:03:04 mail dog: Useless Payload`},
					{content: `Aug 22 03:03:04 mail monkey: Useless Payload`},
					{content: `Aug 23 04:03:04 mail gorilla: Useless Payload`},
					{content: `Aug 24 05:03:04 mail apple: Useless Payload`},
				},
			}

			ReadFromReader(reader, &pub, transformer, fakeAnnouncer, &clock, time.Millisecond*100)

			So(len(pub.logs), ShouldEqual, 5)

			So(pub.logs[0], ShouldResemble, postfix.Record{
				Time: testutil.MustParseTime(`2000-08-20 02:03:04 +0000`),
				Header: parser.Header{
					Time:      parser.Time{Month: time.August, Day: 20, Hour: 2, Minute: 3, Second: 4},
					Host:      "mail",
					Process:   "banana",
					Daemon:    "",
					PID:       0,
					ProcessIP: nil,
				},
				Location: postfix.RecordLocation{Line: 1, Filename: "unknown"},
				Payload:  nil,
			})

			So(pub.logs[4], ShouldResemble, postfix.Record{
				Time: testutil.MustParseTime(`2000-08-24 05:03:04 +0000`),
				Header: parser.Header{
					Time:      parser.Time{Month: time.August, Day: 24, Hour: 5, Minute: 3, Second: 4},
					Host:      "mail",
					Process:   "apple",
					Daemon:    "",
					PID:       0,
					ProcessIP: nil,
				},
				Location: postfix.RecordLocation{Line: 5, Filename: "unknown"},
				Payload:  nil,
			})

			So(fakeAnnouncer.Start, ShouldResemble, testutil.MustParseTime(`2000-08-20 02:03:04 +0000`))
			So(fakeAnnouncer.Progress(), ShouldResemble, []announcer.Progress{
				announcer.Progress{Finished: false, Time: testutil.MustParseTime(`2000-08-20 02:03:04 +0000`), Progress: 0},
				announcer.Progress{Finished: false, Time: testutil.MustParseTime(`2000-08-21 03:03:04 +0000`), Progress: 24},
				announcer.Progress{Finished: false, Time: testutil.MustParseTime(`2000-08-22 03:03:04 +0000`), Progress: 47},
				announcer.Progress{Finished: false, Time: testutil.MustParseTime(`2000-08-23 04:03:04 +0000`), Progress: 71},
				announcer.Progress{Finished: false, Time: testutil.MustParseTime(`2000-08-24 05:03:04 +0000`), Progress: 95},
				announcer.Progress{Finished: true, Time: testutil.MustParseTime(`2000-08-24 05:03:04 +0000`), Progress: 100},
			})
		})
	})
}

const testFilesDir = "../../test_files/postfix_logs/individual_files"

func TestReadTestFiles(t *testing.T) {
	Convey("Read a bunch of files, for exercising", t, func() {
		testEntry := func(name string) {
			f, err := os.Open(path.Join(testFilesDir, name))
			So(err, ShouldBeNil)

			pub := &pub{}

			transformer, err := transform.Get("default", 2000)
			So(err, ShouldBeNil)

			fakeAnnouncer := &announcer.DummyImportAnnouncer{}
			clock := timeutil.FakeClock{Time: testutil.MustParseTime(`2000-08-24 10:00:00 +0000`)}

			readAll := func() {
				ReadFromReader(f, pub, transformer, fakeAnnouncer, &clock, time.Millisecond*500)
			}

			So(readAll, ShouldNotPanic)
		}

		entries, err := ioutil.ReadDir(testFilesDir)
		So(err, ShouldBeNil)

		for _, entry := range entries {
			testEntry(entry.Name())
		}
	})
}
