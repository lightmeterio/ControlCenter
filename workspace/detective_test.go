// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package workspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/detective"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/logeater/filelogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/logsource"
	"gitlab.com/lightmeter/controlcenter/logeater/transform"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"io"
	"os"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func buildDetective(t *testing.T, filename string, year int) (detective.Detective, func()) {
	f, err := os.Open(filename)
	So(err, ShouldBeNil)

	return buildDetectiveFromReader(t, f, year)
}

func buildDetectiveFromReader(t *testing.T, reader io.Reader, year int) (detective.Detective, func()) {
	dir, clearDir := testutil.TempDir(t)

	ws, err := NewWorkspace(dir)
	So(err, ShouldBeNil)

	builder, err := transform.Get("default", year)
	So(err, ShouldBeNil)

	// needed to prevent the insights execution of blocking
	importAnnouncer, err := ws.ImportAnnouncer()
	So(err, ShouldBeNil)
	announcer.Skip(importAnnouncer)

	logSource, err := filelogsource.New(reader, builder, importAnnouncer)
	So(err, ShouldBeNil)

	done, cancel := runner.Run(ws)

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
		)

		Convey("Empty input logs", func() {
			d, clear := buildDetectiveFromReader(t, bytes.NewReader(nil), year)
			defer clear()

			_, err := d.OldestAvailableTime(context.Background())
			So(errors.Is(err, detective.ErrNoAvailableLogs), ShouldBeTrue)
		})

		Convey("File with a successful delivery", func() {
			d, clear := buildDetective(t, "../test_files/postfix_logs/individual_files/3_local_delivery.log", year)
			defer clear()

			Convey("Message found", func() {
				messagesLowerCase, err := d.CheckMessageDelivery(context.Background(), "sender@example.com", "recipient@example.com", correctInterval, 1)
				So(err, ShouldBeNil)

				messagesMixedCase, err := d.CheckMessageDelivery(context.Background(), "Sender@eXamplE.com", "ReciPient@Example.COM", correctInterval, 1)
				So(err, ShouldBeNil)

				expectedTime := time.Date(year, time.January, 10, 16, 15, 30, 0, time.UTC)
				So(messagesLowerCase, ShouldResemble, &detective.MessagesPage{1, 1, 1, 1,
					detective.Messages{
						detective.Message{
							Queue: "400643011B47",
							Entries: []detective.MessageDelivery{
								{
									1,
									expectedTime.In(time.UTC),
									expectedTime.In(time.UTC),
									detective.Status(parser.SentStatus),
									"2.0.0",
									nil,
								},
							},
						},
					},
				})

				// Gitlab issue #526
				So(messagesMixedCase, ShouldResemble, messagesLowerCase)
			})

			noDeliveries := detective.Messages{}

			Convey("Page number too big", func() {
				messages, err := d.CheckMessageDelivery(context.Background(), "sender@example.com", "recipient@example.com", correctInterval, 2)
				So(err, ShouldBeNil)
				So(messages, ShouldResemble, &detective.MessagesPage{2, 1, 1, 0, noDeliveries})
			})

			Convey("Message out of interval", func() {
				wrongInterval := timeutil.TimeInterval{
					time.Date(year+1, time.January, 0, 0, 0, 0, 0, time.Local),
					time.Date(year+1, time.December, 31, 0, 0, 0, 0, time.Local),
				}

				messages, err := d.CheckMessageDelivery(context.Background(), "sender@example.com", "recipient@example.com", wrongInterval, 1)
				So(err, ShouldBeNil)
				So(messages, ShouldResemble, &detective.MessagesPage{1, 1, 1, 0, noDeliveries})
			})

			oldestTime, err := d.OldestAvailableTime(context.Background())
			So(err, ShouldBeNil)
			So(oldestTime, ShouldResemble, testutil.MustParseTime(`2020-01-10 16:15:30 +0000`))
		})

		Convey("File with an expired message", func() {
			d, clear := buildDetective(t, "../test_files/postfix_logs/individual_files/18_expired.log", year)
			defer clear()

			Convey("Message found", func() {
				messages, err := d.CheckMessageDelivery(context.Background(), "h-498b874f2bf0cf639807ad80e1@h-5e67b9b4406.com", "h-664d01@h-695da2287.com", correctInterval, 1)
				So(err, ShouldBeNil)

				expectedExpiredTime := testutil.MustParseTime(fmt.Sprint(year) + `-09-30 20:46:08 +0000`)

				So(messages, ShouldResemble, &detective.MessagesPage{
					PageNumber:   1,
					FirstPage:    1,
					LastPage:     1,
					TotalResults: 1,
					Messages: detective.Messages{
						detective.Message{
							Queue: "23EBE3D5C0",
							Entries: []detective.MessageDelivery{
								{
									5,
									time.Date(year, time.September, 25, 18, 26, 36, 0, time.UTC),
									time.Date(year, time.September, 30, 20, 46, 8, 0, time.UTC),
									detective.Status(parser.DeferredStatus),
									"4.1.1",
									&expectedExpiredTime,
								},
								{
									1,
									time.Date(year, time.September, 30, 20, 46, 8, 0, time.UTC),
									time.Date(year, time.September, 30, 20, 46, 8, 0, time.UTC),
									detective.Status(parser.ReturnedStatus),
									"2.0.0",
									&expectedExpiredTime,
								},
							},
						},
					},
				})
			})

			oldestTime, err := d.OldestAvailableTime(context.Background())
			So(err, ShouldBeNil)
			So(oldestTime, ShouldResemble, testutil.MustParseTime(`2020-09-25 18:26:36 +0000`))
		})

		Convey("File with 5 deliveries, some via postfix/local. Gitlab issue #516", func() {
			d, clear := buildDetective(t, "../test_files/postfix_logs/individual_files/21_deliveries_with_local_daemon.log", year)
			defer clear()

			Convey("Message found", func() {
				messages, err := d.CheckMessageDelivery(context.Background(), "h-195704c@h-b7bed8eb24c5049d9.com", "h-493fac8f3@h-ea3f4afa.com", correctInterval, 1)
				So(err, ShouldBeNil)

				So(messages, ShouldResemble, &detective.MessagesPage{
					PageNumber:   1,
					FirstPage:    1,
					LastPage:     1,
					TotalResults: 2,
					Messages: detective.Messages{
						detective.Message{
							Queue: "95154657C",
							Entries: []detective.MessageDelivery{
								{
									1,
									time.Date(year, time.June, 20, 5, 2, 7, 0, time.UTC),
									time.Date(year, time.June, 20, 5, 2, 7, 0, time.UTC),
									detective.Status(parser.SentStatus),
									"2.0.0",
									nil,
								},
							},
						},
						detective.Message{
							Queue: "D390B657C",
							Entries: []detective.MessageDelivery{
								{
									1,
									time.Date(year, time.June, 20, 5, 4, 7, 0, time.UTC),
									time.Date(year, time.June, 20, 5, 4, 7, 0, time.UTC),
									detective.Status(parser.SentStatus),
									"2.0.0",
									nil,
								},
							},
						},
					},
				})
			})
		})
	})
}
