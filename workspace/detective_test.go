// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package workspace

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/detective"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/logeater/filelogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/logsource"
	"gitlab.com/lightmeter/controlcenter/logeater/transform"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"os"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func buildDetective(t *testing.T, filename string, year int) (detective.Detective, func()) {
	dir, clearDir := testutil.TempDir(t)

	ws, err := NewWorkspace(dir)
	So(err, ShouldBeNil)

	builder, err := transform.Get("default", year)
	So(err, ShouldBeNil)

	f, err := os.Open(filename)
	So(err, ShouldBeNil)

	logSource, err := filelogsource.New(f, builder, announcer.Skipper(ws.ImportAnnouncer()))
	So(err, ShouldBeNil)

	done, cancel := ws.Run()

	logReader := logsource.NewReader(logSource, ws.NewPublisher())

	err = logReader.Run()
	So(err, ShouldBeNil)

	cancel()
	err = done()
	So(err, ShouldBeNil)

	// actual Message Detective testing
	d := ws.Detective()

	return d, func() {
		ws.Close()
		clearDir()
	}
}

func TestDetective(t *testing.T) {
	Convey("Detective on real logs", t, func() {
		const year = 2020
		d, clear := buildDetective(t, "../test_files/postfix_logs/individual_files/3_local_delivery.log", year)
		defer clear()

		Convey("Message found", func() {
			interval := timeutil.TimeInterval{
				time.Date(year, time.January, 0, 0, 0, 0, 0, time.Local),
				time.Date(year, time.December, 31, 0, 0, 0, 0, time.Local),
			}
			messages, err := d.CheckMessageDelivery(context.Background(), "sender@example.com", "recipient@example.com", interval)
			So(err, ShouldBeNil)

			expectedTime := time.Date(year, time.January, 10, 16, 15, 30, 0, time.UTC)
			So(messages, ShouldResemble, []detective.MessageDelivery{detective.MessageDelivery{expectedTime.In(time.UTC), "sent", "2.0.0"}})
		})

		Convey("Message out of interval", func() {
			interval := timeutil.TimeInterval{
				time.Date(year+1, time.January, 0, 0, 0, 0, 0, time.Local),
				time.Date(year+1, time.December, 31, 0, 0, 0, 0, time.Local),
			}
			messages, err := d.CheckMessageDelivery(context.Background(), "sender@example.com", "recipient@example.com", interval)
			So(err, ShouldBeNil)

			So(messages, ShouldResemble, []detective.MessageDelivery{})
		})
	})
}
