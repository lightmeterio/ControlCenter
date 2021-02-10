// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dirlogsource

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/logeater/logsource"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"os"
	"path"
	"sync"
	"testing"
	"time"
)

type fakePublisher struct {
	sync.Mutex
	records []data.Record
}

func (f *fakePublisher) Publish(r data.Record) {
	f.Lock()
	defer f.Unlock()
	f.records = append(f.records, r)
}

func TestReadingFromDirectory(t *testing.T) {
	Convey("Read from Directory", t, func() {
		sampleTarball := path.Join("..", "..", "test_files", "postfix_logs", "complete.tar.gz")
		tempDir, clear := testutil.TempDir(t)
		defer clear()

		logDir := path.Join(tempDir, "logs_sample")

		tarball, err := os.Open(sampleTarball)
		So(err, ShouldBeNil)
		extractTarGz(tarball, tempDir)

		pub := fakePublisher{}

		Convey("Only import from beginning", func() {
			s, err := New(logDir, time.Time{}, false)
			So(err, ShouldBeNil)
			r := logsource.NewReader(s, &pub)
			So(r.Run(), ShouldBeNil)
			So(len(pub.records), ShouldEqual, 9069)
		})

		Convey("Import logs and watch for changes", func() {
			s, err := New(logDir, time.Time{}, true)
			So(err, ShouldBeNil)
			r := logsource.NewReader(s, &pub)

			// we keep forever watching for changes in the log files...
			go r.Run()

			time.Sleep(time.Second * 3)

			appendLineToFile(path.Join(logDir, "mail.log"), "Jan 28 16:58:00 mail postfix/smtpd[15319]: connect from localhost[127.0.0.1]")

			time.Sleep(time.Second * 2)

			pub.Lock()
			defer pub.Unlock()

			So(len(pub.records), ShouldEqual, 9070)
			So(pub.records[len(pub.records)-1].Header.Time, ShouldResemble, parser.Time{Month: time.January, Day: 28, Hour: 16, Minute: 58, Second: 0})
		})
	})
}

func appendLineToFile(filename string, line string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	So(err, ShouldBeNil)
	_, err = f.WriteString(line + "\n")
	So(err, ShouldBeNil)
	So(f.Close(), ShouldBeNil)
}
