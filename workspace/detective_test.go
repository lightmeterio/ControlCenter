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
		var (
			correctInterval = timeutil.TimeInterval{
				time.Date(year, time.January, 0, 0, 0, 0, 0, time.Local),
				time.Date(year, time.December, 31, 0, 0, 0, 0, time.Local),
			}
			wrongInterval = timeutil.TimeInterval{
				time.Date(year+1, time.January, 0, 0, 0, 0, 0, time.Local),
				time.Date(year+1, time.December, 31, 0, 0, 0, 0, time.Local),
			}
		)

		Convey("File with a successful delivery", func() {
			d, clear := buildDetective(t, "../test_files/postfix_logs/individual_files/3_local_delivery.log", year)
			defer clear()

			Convey("Message found", func() {

				messages, err := d.CheckMessageDelivery(context.Background(), "sender@example.com", "recipient@example.com", correctInterval, 1)
				So(err, ShouldBeNil)

				expectedTime := time.Date(year, time.January, 10, 16, 15, 30, 0, time.UTC)
				So(messages, ShouldResemble, &detective.MessagesPage{1, 1, 1, 1,
					map[int][]detective.MessageDelivery{
						1: []detective.MessageDelivery{detective.MessageDelivery{
							1,
							expectedTime.In(time.UTC),
							expectedTime.In(time.UTC),
							"sent",
							"2.0.0",
						},
						}},
				})
			})

			noDeliveries := map[int][]detective.MessageDelivery{}

			Convey("Page number too big", func() {
				messages, err := d.CheckMessageDelivery(context.Background(), "sender@example.com", "recipient@example.com", correctInterval, 2)
				So(err, ShouldBeNil)
				So(messages, ShouldResemble, &detective.MessagesPage{2, 1, 1, 0, noDeliveries})
			})

			Convey("Message out of interval", func() {
				messages, err := d.CheckMessageDelivery(context.Background(), "sender@example.com", "recipient@example.com", wrongInterval, 1)
				So(err, ShouldBeNil)

				So(messages, ShouldResemble, &detective.MessagesPage{1, 1, 1, 0, noDeliveries})
			})
		})

		Convey("File with an expired message", func() {
			d, clear := buildDetective(t, "../test_files/postfix_logs/individual_files/18_expired.log", year)
			defer clear()

			Convey("Message found", func() {
				messages, err := d.CheckMessageDelivery(context.Background(), "h-498b874f2bf0cf639807ad80e1@h-5e67b9b4406.com", "h-664d01@h-695da2287.com", correctInterval, 1)
				So(err, ShouldBeNil)

				So(messages, ShouldResemble, &detective.MessagesPage{1, 1, 1, 1,
					map[int][]detective.MessageDelivery{
						1: []detective.MessageDelivery{
							detective.MessageDelivery{
								4,
								time.Date(year, time.September, 25, 18, 26, 36, 0, time.UTC),
								time.Date(year, time.September, 30, 16, 46, 7, 0, time.UTC),
								"deferred",
								"4.1.1",
							},
							detective.MessageDelivery{
								1,
								time.Date(year, time.September, 30, 20, 46, 8, 0, time.UTC),
								time.Date(year, time.September, 30, 20, 46, 8, 0, time.UTC),
								"expired",
								"4.1.1",
							},
						},
					}})
			})
		})
	})
}
