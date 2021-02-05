// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package logeater

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"strings"
	"testing"
	"time"
)

type FakePublisher struct {
	logs []data.Record
}

func (f *FakePublisher) Publish(r data.Record) {
	f.logs = append(f.logs, r)
}

func TestReadingLogs(t *testing.T) {
	Convey("Read From Reader", t, func() {
		pub := FakePublisher{}

		firstSecondInJanuary := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)

		Convey("Read Nothing", func() {
			reader := strings.NewReader(``)
			ReadFromReader(reader, &pub, firstSecondInJanuary)
			So(len(pub.logs), ShouldEqual, 0)
		})

		Convey("Ignore Wrong Line", func() {
			reader := strings.NewReader(`Not a valid log line`)
			ReadFromReader(reader, &pub, firstSecondInJanuary)
			So(len(pub.logs), ShouldEqual, 0)
		})

		Convey("Accepts line with error on reading the payload (but header is okay)", func() {
			reader := strings.NewReader(`Mar  1 07:42:10 mail opendkim[225]: C11EA2C620C7: not authenticated`)
			ReadFromReader(reader, &pub, firstSecondInJanuary)
			So(len(pub.logs), ShouldEqual, 1)
			So(pub.logs[0].Payload, ShouldBeNil)
			So(pub.logs[0].Header.Time.Day, ShouldEqual, 1)
			So(pub.logs[0].Header.Time.Month, ShouldEqual, time.March)
			So(pub.logs[0].Header.Time.Hour, ShouldEqual, 7)
			So(pub.logs[0].Header.Time.Minute, ShouldEqual, 42)
			So(pub.logs[0].Header.Time.Second, ShouldEqual, 10)
			So(pub.logs[0].Time, ShouldEqual, testutil.MustParseTime(`2000-03-01 07:42:10 +0000`))
		})

		Convey("Read three lines, one of them with invalid payload", func() {
			reader := strings.NewReader(`

Sep 16 00:07:43 smtpnode07 postfix-10.20.30.40/smtp[3022]: 0C31D3D1E6: to=<a@b.c>, relay=a.net[1.2.3.4]:25, delay=1, delays=0/0.9/0.69/0.03, dsn=4.7.0, status=deferred Extra text)
Nov  1 07:42:10 mail opendkim[225]: C11EA2C620C7: not authenticated

Dec 16 14:08:45 smtpnode07 postfix-10.20.30.40/smtp[3022]: 0C31D3D1E6: to=<a@b.c>, relay=a.net[1.2.3.4]:25, delay=1, delays=0/0.9/0.69/0.03, dsn=4.7.0, status=deferred Extra text)
			`)
			ReadFromReader(reader, &pub, firstSecondInJanuary)
			So(len(pub.logs), ShouldEqual, 3)
			So(pub.logs[0].Payload, ShouldNotBeNil)
			So(pub.logs[1].Payload, ShouldBeNil)
			So(pub.logs[2].Payload, ShouldNotBeNil)
		})
	})
}
