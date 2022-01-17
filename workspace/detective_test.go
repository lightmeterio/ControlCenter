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

	ws, err := NewWorkspace(dir, nil)
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

var bg = context.Background()

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

			_, err := d.OldestAvailableTime(bg)
			So(errors.Is(err, detective.ErrNoAvailableLogs), ShouldBeTrue)
		})

		Convey("File with a successful delivery", func() {
			d, clear := buildDetective(t, "../test_files/postfix_logs/individual_files/3_local_delivery.log", year)
			defer clear()

			noDeliveries := detective.Messages{}
			noDeliveriesPage1 := &detective.MessagesPage{1, 1, 1, 0, noDeliveries}

			Convey("Message found", func() {
				messagesLowerCase, err := d.CheckMessageDelivery(bg, "sender@example.com", "recipient@example.com", correctInterval, -1, "", 1)
				So(err, ShouldBeNil)

				messagesMixedCase, err := d.CheckMessageDelivery(bg, "Sender@eXamplE.com", "ReciPient@Example.COM", correctInterval, -1, "", 1)
				So(err, ShouldBeNil)

				// working partial searches
				messagesPartialSearch1, err := d.CheckMessageDelivery(bg, "example.com", "recipient@example.com", correctInterval, -1, "", 1)
				So(err, ShouldBeNil)

				messagesPartialSearch2, err := d.CheckMessageDelivery(bg, "@example.com", "example.com", correctInterval, -1, "", 1)
				So(err, ShouldBeNil)

				messagesPartialSearch3, err := d.CheckMessageDelivery(bg, "", "@example.com", correctInterval, -1, "", 1)
				So(err, ShouldBeNil)

				messagesPartialSearch4, err := d.CheckMessageDelivery(bg, "", "", correctInterval, -1, "", 1)
				So(err, ShouldBeNil)

				// partial searches with no results
				messagesPartialSearch5, err := d.CheckMessageDelivery(bg, "@test.org", "", correctInterval, -1, "", 1)
				So(err, ShouldBeNil)

				messagesPartialSearch6, err := d.CheckMessageDelivery(bg, "", "@domain.org", correctInterval, -1, "", 1)
				So(err, ShouldBeNil)

				queueID := "400643011B47"
				wrongQueueID := "511754122C58"

				messageID := "414300fb-b063-fa96-4fc6-2d35b3168d61@example.com"
				wrongMessageID := "1234-abcd@example.com"

				// someID searches
				messagesMailFromToAndQueueID, err := d.CheckMessageDelivery(bg, "example.com", "recipient@example.com", correctInterval, -1, queueID, 1)
				So(err, ShouldBeNil)

				messagesQueueID, err := d.CheckMessageDelivery(bg, "", "", correctInterval, -1, queueID, 1)
				So(err, ShouldBeNil)

				messagesWrongQueueID, err := d.CheckMessageDelivery(bg, "", "", correctInterval, -1, wrongQueueID, 1)
				So(err, ShouldBeNil)

				messagesMessageID, err := d.CheckMessageDelivery(bg, "", "", correctInterval, -1, messageID, 1)
				So(err, ShouldBeNil)

				messagesWrongMessageID, err := d.CheckMessageDelivery(bg, "", "", correctInterval, -1, wrongMessageID, 1)
				So(err, ShouldBeNil)

				expectedTime := time.Date(year, time.January, 10, 16, 15, 30, 0, time.UTC)
				expectedResult := &detective.MessagesPage{1, 1, 1, 1,
					detective.Messages{
						detective.Message{
							Queue:     queueID,
							MessageID: messageID,
							Entries: []detective.MessageDelivery{
								{
									1,
									expectedTime.In(time.UTC),
									expectedTime.In(time.UTC),
									detective.Status(parser.SentStatus),
									"2.0.0",
									nil,
									"sender@example.com",
									"recipient@example.com",
								},
							},
						},
					},
				}
				So(messagesLowerCase, ShouldResemble, expectedResult)

				// Gitlab issue #526
				So(messagesMixedCase, ShouldResemble, messagesLowerCase)

				// Gitlab issue #566
				So(messagesPartialSearch1, ShouldResemble, messagesLowerCase)
				So(messagesPartialSearch2, ShouldResemble, messagesLowerCase)
				So(messagesPartialSearch3, ShouldResemble, messagesLowerCase)
				So(messagesPartialSearch4, ShouldResemble, messagesLowerCase)

				So(messagesPartialSearch5, ShouldResemble, noDeliveriesPage1)
				So(messagesPartialSearch6, ShouldResemble, noDeliveriesPage1)

				// Gitlab issue #572
				So(messagesMailFromToAndQueueID, ShouldResemble, expectedResult)
				So(messagesQueueID, ShouldResemble, expectedResult)
				So(messagesMessageID, ShouldResemble, expectedResult)

				So(messagesWrongQueueID, ShouldResemble, noDeliveriesPage1)
				So(messagesWrongMessageID, ShouldResemble, noDeliveriesPage1)
			})

			Convey("Page number too big", func() {
				messages, err := d.CheckMessageDelivery(bg, "sender@example.com", "recipient@example.com", correctInterval, -1, "", 2)
				So(err, ShouldBeNil)
				So(messages, ShouldResemble, &detective.MessagesPage{2, 1, 1, 0, noDeliveries})
			})

			Convey("Message out of interval", func() {
				wrongInterval := timeutil.TimeInterval{
					time.Date(year+1, time.January, 0, 0, 0, 0, 0, time.Local),
					time.Date(year+1, time.December, 31, 0, 0, 0, 0, time.Local),
				}

				messages, err := d.CheckMessageDelivery(bg, "sender@example.com", "recipient@example.com", wrongInterval, -1, "", 1)
				So(err, ShouldBeNil)
				So(messages, ShouldResemble, noDeliveriesPage1)
			})

			oldestTime, err := d.OldestAvailableTime(bg)
			So(err, ShouldBeNil)
			So(oldestTime, ShouldResemble, testutil.MustParseTime(`2020-01-10 16:15:30 +0000`))
		})

		Convey("File with an expired message", func() {
			d, clear := buildDetective(t, "../test_files/postfix_logs/individual_files/18_expired.log", year)
			defer clear()

			Convey("Message found", func() {
				messages, err := d.CheckMessageDelivery(bg, "h-498b874f2bf0cf639807ad80e1@h-5e67b9b4406.com", "h-664d01@h-695da2287.com", correctInterval, -1, "", 1)
				So(err, ShouldBeNil)

				expectedExpiredTime := testutil.MustParseTime(fmt.Sprint(year) + `-09-30 20:46:08 +0000`)

				So(messages, ShouldResemble, &detective.MessagesPage{
					PageNumber:   1,
					FirstPage:    1,
					LastPage:     1,
					TotalResults: 1,
					Messages: detective.Messages{
						detective.Message{
							Queue:     "23EBE3D5C0",
							MessageID: "h-dea85411b67a40a063ef58e0ab590721@h-daa2fe3dd7fc0b5c2017db90829038b.com",
							Entries: []detective.MessageDelivery{
								{
									5,
									time.Date(year, time.September, 25, 18, 26, 36, 0, time.UTC),
									time.Date(year, time.September, 30, 20, 46, 8, 0, time.UTC),
									detective.Status(parser.DeferredStatus),
									"4.1.1",
									&expectedExpiredTime,
									"h-498b874f2bf0cf639807ad80e1@h-5e67b9b4406.com",
									"h-664d01@h-695da2287.com",
								},
								{
									1,
									time.Date(year, time.September, 30, 20, 46, 8, 0, time.UTC),
									time.Date(year, time.September, 30, 20, 46, 8, 0, time.UTC),
									detective.Status(parser.ReturnedStatus),
									"2.0.0",
									&expectedExpiredTime,
									"h-498b874f2bf0cf639807ad80e1@h-5e67b9b4406.com",
									"h-664d01@h-695da2287.com",
								},
							},
						},
					},
				})
			})

			oldestTime, err := d.OldestAvailableTime(bg)
			So(err, ShouldBeNil)
			So(oldestTime, ShouldResemble, testutil.MustParseTime(`2020-09-25 18:26:36 +0000`))
		})

		Convey("File with 5 deliveries, some via postfix/local. Gitlab issue #516", func() {
			d, clear := buildDetective(t, "../test_files/postfix_logs/individual_files/21_deliveries_with_local_daemon.log", year)
			defer clear()

			Convey("Message found", func() {
				messages, err := d.CheckMessageDelivery(bg, "h-195704c@h-b7bed8eb24c5049d9.com", "h-493fac8f3@h-ea3f4afa.com", correctInterval, -1, "", 1)
				So(err, ShouldBeNil)

				So(messages, ShouldResemble, &detective.MessagesPage{
					PageNumber:   1,
					FirstPage:    1,
					LastPage:     1,
					TotalResults: 2,
					Messages: detective.Messages{
						detective.Message{
							Queue:     "95154657C",
							MessageID: "h-ec262eb25918e7678e9e8737f7b@h-e7d9fe256179482d76de1b3e83c.com",
							Entries: []detective.MessageDelivery{
								{
									1,
									time.Date(year, time.June, 20, 5, 2, 7, 0, time.UTC),
									time.Date(year, time.June, 20, 5, 2, 7, 0, time.UTC),
									detective.Status(parser.SentStatus),
									"2.0.0",
									nil,
									"h-195704c@h-b7bed8eb24c5049d9.com",
									"h-493fac8f3@h-ea3f4afa.com",
								},
							},
						},
						detective.Message{
							Queue:     "D390B657C",
							MessageID: "h-dfd067542de35f4b23673e0b3b3@h-e7d9fe256179482d76de1b3e83c.com",
							Entries: []detective.MessageDelivery{
								{
									1,
									time.Date(year, time.June, 20, 5, 4, 7, 0, time.UTC),
									time.Date(year, time.June, 20, 5, 4, 7, 0, time.UTC),
									detective.Status(parser.SentStatus),
									"2.0.0",
									nil,
									"h-195704c@h-b7bed8eb24c5049d9.com",
									"h-493fac8f3@h-ea3f4afa.com",
								},
							},
						},
					},
				})
			})
		})
	})
}
