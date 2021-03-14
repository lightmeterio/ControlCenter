// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package socketsource

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/logeater/logsource"
	"gitlab.com/lightmeter/controlcenter/logeater/transform"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"net"
	"path"
	"sync"
	"testing"
	"time"
)

type pub struct {
	// accessed from different threads
	sync.Mutex

	logs []postfix.Record
}

func (pub *pub) Publish(r postfix.Record) {
	pub.Lock()
	defer pub.Unlock()
	pub.logs = append(pub.logs, r)
}

type fakeAnnouncer = announcer.DummyImportAnnouncer

func TestListenLogsOnSocket(t *testing.T) {
	Convey("Get logs from socket", t, func() {
		dir, clear := testutil.TempDir(t)

		defer clear()

		pub := &pub{}

		transformer, err := transform.Get("default", 2000)
		So(err, ShouldBeNil)

		Convey("Wrong socket description", func() {
			_, err := New("something invalid", transformer, &fakeAnnouncer{})
			So(err, ShouldNotBeNil)
		})

		Convey("Invalid network type", func() {
			_, err := New("magic=/tmp/lalala", transformer, &fakeAnnouncer{})
			So(err, ShouldNotBeNil)
		})

		Convey("Error opening socket (permission denied)", func() {
			_, err := New("unix=/proc/something", transformer, &fakeAnnouncer{})
			So(err, ShouldNotBeNil)
		})

		Convey("Use unix socket", func() {
			source, err := New("unix="+path.Join(dir, "logs.sock"), transformer, &fakeAnnouncer{})
			So(err, ShouldBeNil)

			done := make(chan error)

			go func() {
				reader := logsource.NewReader(source, pub)
				done <- reader.Run()
			}()

			c, err := net.Dial("unix", path.Join(dir, "logs.sock"))
			So(err, ShouldBeNil)

			_, err = c.Write([]byte(`Aug 21 02:03:04 mail banana: Useless Payload
Aug 21 03:03:04 mail dog: Useless Payload
Aug 22 03:03:04 mail monkey: Useless Payload
Aug 22 04:03:04 mail gorilla: Useless Payload
Aug 22 05:03:04 mail apple: Useless Payload
`))

			So(err, ShouldBeNil)

			time.Sleep(500 * time.Millisecond)

			So(source.Close(), ShouldBeNil)

			<-done

			pub.Lock()
			defer pub.Unlock()

			So(len(pub.logs), ShouldEqual, 5)

			So(pub.logs[0], ShouldResemble, postfix.Record{
				Time: testutil.MustParseTime(`2000-08-21 02:03:04 +0000`),
				Header: parser.Header{
					Time:      parser.Time{Month: time.August, Day: 21, Hour: 2, Minute: 3, Second: 4},
					Host:      "mail",
					Process:   "banana",
					Daemon:    "",
					PID:       0,
					ProcessIP: nil,
				},
				Location: postfix.RecordLocation{Line: 1, Filename: "unknown"},
				Payload:  nil,
			})
		})
	})
}
