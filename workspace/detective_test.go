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

const limit = detective.ResultsPerPage

func buildDetective(t *testing.T, filename string, year int) (detective.Detective, func()) {
	f, err := os.Open(filename)
	So(err, ShouldBeNil)

	return buildDetectiveFromReader(t, f, year)
}

func buildDetectiveFromReader(t *testing.T, reader io.Reader, year int) (detective.Detective, func()) {
	dir, clearDir := testutil.TempDir(t)

	var err error

	defer func() {
		if err != nil {
			clearDir()
		}
	}()

	ws, err := NewWorkspace(dir, nil)
	So(err, ShouldBeNil)

	clock := &timeutil.FakeClock{Time: timeutil.MustParseTime(`3000-01-01 00:00:00 +0000`)}

	builder, err := transform.Get("default", clock, year)
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
	noDeliveries := detective.Messages{}
	noDeliveriesPage1 := &detective.MessagesPage{1, 1, 1, 0, noDeliveries}

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

			Convey("Message found", func() {
				messagesLowerCase, err := d.CheckMessageDelivery(bg, "sender@example.com", "recipient@example.com", correctInterval, -1, "", 1, limit)
				So(err, ShouldBeNil)

				messagesMixedCase, err := d.CheckMessageDelivery(bg, "Sender@eXamplE.com", "ReciPient@Example.COM", correctInterval, -1, "", 1, limit)
				So(err, ShouldBeNil)

				// working partial searches
				messagesPartialSearch1, err := d.CheckMessageDelivery(bg, "example.com", "recipient@example.com", correctInterval, -1, "", 1, limit)
				So(err, ShouldBeNil)

				messagesPartialSearch2, err := d.CheckMessageDelivery(bg, "@example.com", "example.com", correctInterval, -1, "", 1, limit)
				So(err, ShouldBeNil)

				messagesPartialSearch3, err := d.CheckMessageDelivery(bg, "", "@example.com", correctInterval, -1, "", 1, limit)
				So(err, ShouldBeNil)

				messagesPartialSearch4, err := d.CheckMessageDelivery(bg, "", "", correctInterval, -1, "", 1, limit)
				So(err, ShouldBeNil)

				// partial searches with no results
				messagesPartialSearch5, err := d.CheckMessageDelivery(bg, "@test.org", "", correctInterval, -1, "", 1, limit)
				So(err, ShouldBeNil)

				messagesPartialSearch6, err := d.CheckMessageDelivery(bg, "", "@domain.org", correctInterval, -1, "", 1, limit)
				So(err, ShouldBeNil)

				queueID := "400643011B47"
				wrongQueueID := "511754122C58"

				messageID := "414300fb-b063-fa96-4fc6-2d35b3168d61@example.com"
				wrongMessageID := "1234-abcd@example.com"

				// someID searches
				messagesMailFromToAndQueueID, err := d.CheckMessageDelivery(bg, "example.com", "recipient@example.com", correctInterval, -1, queueID, 1, limit)
				So(err, ShouldBeNil)

				messagesQueueID, err := d.CheckMessageDelivery(bg, "", "", correctInterval, -1, queueID, 1, limit)
				So(err, ShouldBeNil)

				messagesWrongQueueID, err := d.CheckMessageDelivery(bg, "", "", correctInterval, -1, wrongQueueID, 1, limit)
				So(err, ShouldBeNil)

				messagesMessageID, err := d.CheckMessageDelivery(bg, "", "", correctInterval, -1, messageID, 1, limit)
				So(err, ShouldBeNil)

				messagesWrongMessageID, err := d.CheckMessageDelivery(bg, "", "", correctInterval, -1, wrongMessageID, 1, limit)
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
									detective.Status(parser.ReceivedStatus),
									"2.0.0",
									[]string{"outlook.com"},
									nil,
									"sender@example.com",
									[]string{"recipient@example.com"},
									[]string{`Jan 10 16:15:30 mail postfix/lmtp[11996]: 400643011B47: to=<recipient@example.com>, relay=example-com.mail.protection.outlook.com[1.2.3.4]:25, delay=0.06, delays=0.02/0.02/0.01/0.01, dsn=2.0.0, status=sent (250 2.0.0 <recipient@example.com> hz3kESIo+1/dLgAAWP5Hkg Saved)`},
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
				messages, err := d.CheckMessageDelivery(bg, "sender@example.com", "recipient@example.com", correctInterval, -1, "", 2, limit)
				So(err, ShouldBeNil)
				So(messages, ShouldResemble, &detective.MessagesPage{2, 1, 1, 0, noDeliveries})
			})

			Convey("Message out of interval", func() {
				wrongInterval := timeutil.TimeInterval{
					time.Date(year+1, time.January, 0, 0, 0, 0, 0, time.Local),
					time.Date(year+1, time.December, 31, 0, 0, 0, 0, time.Local),
				}

				messages, err := d.CheckMessageDelivery(bg, "sender@example.com", "recipient@example.com", wrongInterval, -1, "", 1, limit)
				So(err, ShouldBeNil)
				So(messages, ShouldResemble, noDeliveriesPage1)
			})

			oldestTime, err := d.OldestAvailableTime(bg)
			So(err, ShouldBeNil)
			So(oldestTime, ShouldResemble, testutil.MustParseTime(`2020-01-10 16:15:30 +0000`))
		})

		Convey("Multi-recipient email and search by relay name", func() {
			d, clear := buildDetective(t, "../test_files/postfix_logs/individual_files/26_two_recipients.log", year)
			defer clear()

			queueID := "B9996EABB6"

			expectedTime := time.Date(year, time.January, 20, 19, 48, 07, 0, time.UTC)
			expectedResult := &detective.MessagesPage{1, 1, 1, 1,
				detective.Messages{
					detective.Message{
						Queue:     queueID,
						MessageID: "h-74f3afb0208ad285a794d760c8feb0eee631@internal.org",
						Entries: []detective.MessageDelivery{
							{
								1,
								expectedTime.In(time.UTC),
								expectedTime.In(time.UTC),
								detective.Status(parser.SentStatus),
								"2.0.0",
								[]string{"outlook.com"},
								nil,
								"sender@internal.org",
								[]string{"recipient1@external.org", "recipient2@external.org"},
								[]string{
									`Jan 20 19:48:07 teupos postfix/smtp[2467312]: B9996EABB6: to=<recipient1@external.org>, relay=example-com.mail.protection.outlook.com[12.11.12.13]:25, delay=2.7, delays=1.3/0.06/0.33/1, dsn=2.0.0, status=sent (250 2.0.0 OK  1642704487 v125si7680590wme.216 - smtp)`,
									`Jan 20 19:48:07 teupos postfix/smtp[2467312]: B9996EABB6: to=<recipient2@external.org>, relay=example-com.mail.protection.outlook.com[13.11.12.13]:25, delay=2.7, delays=1.3/0.06/0.33/1, dsn=2.0.0, status=sent (250 2.0.0 OK  1642704487 v125si7680590wme.216 - smtp)`,
								},
							},
						},
					},
				},
			}

			correctInterval = timeutil.TimeInterval{
				time.Date(year, time.January, 0, 0, 0, 0, 0, time.UTC),
				time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC),
			}

			Convey("Multi-recipient someID search should yield correct number of delivery attempts, and recipients", func() {
				messages, err := d.CheckMessageDelivery(bg, "", "", correctInterval, -1, queueID, 1, limit)
				So(err, ShouldBeNil)
				So(messages, ShouldResemble, expectedResult)
			})

			Convey("Searching for relay name should find delivery as well", func() {
				messages, err := d.CheckMessageDelivery(bg, "", "outlook.com", correctInterval, -1, "", 1, limit)
				So(err, ShouldBeNil)
				So(messages, ShouldResemble, expectedResult)
			})

			Convey("Searching for wrong relay should yield empty result", func() {
				messages, err := d.CheckMessageDelivery(bg, "", "wrong.relay", correctInterval, -1, "", 1, limit)
				So(err, ShouldBeNil)
				So(messages, ShouldResemble, noDeliveriesPage1)
			})
		})

		Convey("Expired messages", func() {
			d, clear := buildDetective(t, "../test_files/postfix_logs/individual_files/18_expired.log", year)
			defer clear()

			expectedExpiredTime := testutil.MustParseTime(fmt.Sprint(year) + `-09-30 20:46:08 +0000`)
			expectedResult := &detective.MessagesPage{
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
								[]string{"google.com", "outlook.com"},
								&expectedExpiredTime,
								"h-498b874f2bf0cf639807ad80e1@h-5e67b9b4406.com",
								[]string{"h-664d01@h-695da2287.com"},
								[]string{
									`Sep 25 18:26:36 smtpnode16 postfix-239.58.50.50/smtp[5084]: 23EBE3D5C0: to=<h-664d01@h-695da2287.com>, relay=ALT2.ASPMX.L.GOOGLE.com[3.155.237.60]:25, delay=2, delays=0.18/0/1.6/0.19, dsn=4.1.1, status=deferred (host ALT2.ASPMX.L.GOOGLE.com[3.155.237.60] said: 452 4.1.1 <h-664d01@h-695da2287.com> user is over quota, please try again later (in reply to RCPT TO command))`,
									`Sep 25 19:01:05 smtpnode16 postfix-239.58.50.50/smtp[8810]: 23EBE3D5C0: to=<h-664d01@h-695da2287.com>, relay=ALT2.ASPMX.L.GOOGLE.com[3.155.237.60]:25, delay=2071, delays=2069/0.05/1.9/0.23, dsn=4.1.1, status=deferred (host ALT2.ASPMX.L.GOOGLE.com[3.155.237.60] said: 452 4.1.1 <h-664d01@h-695da2287.com> user is over quota, please try again later (in reply to RCPT TO command))`,
									`Sep 30 12:46:06 smtpnode16 postfix-239.58.50.50/smtp[2851]: 23EBE3D5C0: to=<h-664d01@h-695da2287.com>, relay=ALT2.ASPMX.L.GOOGLE.com[3.155.237.60]:25, delay=411573, delays=411571/0/1.5/0.2, dsn=4.1.1, status=deferred (host ALT2.ASPMX.L.GOOGLE.com[3.155.237.60] said: 452 4.1.1 <h-664d01@h-695da2287.com> user is over quota, please try again later (in reply to RCPT TO command))`,
									`Sep 30 16:46:07 smtpnode16 postfix-239.58.50.50/smtp[29711]: 23EBE3D5C0: to=<h-664d01@h-695da2287.com>, relay=ALT2.ASPMX.L.GOOGLE.com[3.155.237.60]:25, delay=425973, delays=425971/0.03/2/0.37, dsn=4.1.1, status=deferred (host ALT2.ASPMX.L.GOOGLE.com[3.155.237.60] said: 452 4.1.1 <h-664d01@h-695da2287.com> user is over quota, please try again later (in reply to RCPT TO command))`,
									`Sep 30 20:46:08 smtpnode16 postfix-239.58.50.50/smtp[23560]: 23EBE3D5C0: to=<h-664d01@h-695da2287.com>, relay=example-com.mail.protection.outlook.com[3.155.237.60]:25, delay=440374, delays=440372/0.04/1.6/0.84, dsn=4.1.1, status=deferred (host ALT2.ASPMX.L.GOOGLE.com[3.155.237.60] said: 452 4.1.1 <h-664d01@h-695da2287.com> user is over quota, please try again later (in reply to RCPT TO command))`,
								},
							},
							{
								1,
								time.Date(year, time.September, 30, 20, 46, 8, 0, time.UTC),
								time.Date(year, time.September, 30, 20, 46, 8, 0, time.UTC),
								detective.Status(parser.ReturnedStatus),
								"2.0.0",
								[]string{"h-213dce00be4cedefd"},
								&expectedExpiredTime,
								"h-498b874f2bf0cf639807ad80e1@h-5e67b9b4406.com",
								[]string{"h-664d01@h-695da2287.com"},
								[]string{`Sep 30 20:46:08 smtpnode16 postfix-239.58.50.50/smtp[23557]: A7E673C067: to=<h-498b874f2bf0cf639807ad80e1@h-5e67b9b4406.com>, relay=h-213dce00be4cedefd[199.170.45.30]:25, delay=0.14, delays=0.01/0/0.11/0.02, dsn=2.0.0, status=sent (250 2.0.0 Ok: queued as 46hvZ85lJRz1w8W)`},
							},
						},
					},
				},
			}

			Convey("Message found", func() {
				messages, err := d.CheckMessageDelivery(bg, "h-498b874f2bf0cf639807ad80e1@h-5e67b9b4406.com", "h-664d01@h-695da2287.com", correctInterval, -1, "", 1, limit)
				So(err, ShouldBeNil)
				So(messages, ShouldResemble, expectedResult)
			})

			Convey("Search for expired messages. Gitlab issue #616", func() {
				messages, err := d.CheckMessageDelivery(bg, "", "", correctInterval, int(parser.ExpiredStatus), "", 1, limit)
				So(err, ShouldBeNil)
				So(messages, ShouldResemble, expectedResult)
			})

			oldestTime, err := d.OldestAvailableTime(bg)
			So(err, ShouldBeNil)
			So(oldestTime, ShouldResemble, testutil.MustParseTime(`2020-09-25 18:26:36 +0000`))
		})

		Convey("File with 5 deliveries, some via postfix/local. Gitlab issue #516", func() {
			d, clear := buildDetective(t, "../test_files/postfix_logs/individual_files/21_deliveries_with_local_daemon.log", year)
			defer clear()

			Convey("Message found", func() {
				messages, err := d.CheckMessageDelivery(bg, "h-195704c@h-b7bed8eb24c5049d9.com", "h-493fac8f3@h-ea3f4afa.com", correctInterval, -1, "", 1, limit)
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
									[]string{"local"},
									nil,
									"h-195704c@h-b7bed8eb24c5049d9.com",
									[]string{"h-493fac8f3@h-ea3f4afa.com"},
									[]string{`Jun 20 05:02:07 ns4 postfix/local[16460]: 95154657C: to=<h-493fac8f3@h-ea3f4afa.com>, orig_to=<h-195704c@h-20b651e8120a33ec11.com>, relay=local, delay=0.1, delays=0.09/0/0/0.01, dsn=2.0.0, status=sent (delivered to command: procmail -a "$EXTENSION" DEFAULT=$HOME/Maildir/)`},
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
									[]string{"local"},
									nil,
									"h-195704c@h-b7bed8eb24c5049d9.com",
									[]string{"h-493fac8f3@h-ea3f4afa.com"},
									[]string{`Jun 20 05:04:07 ns4 postfix/local[16746]: D390B657C: to=<h-493fac8f3@h-ea3f4afa.com>, orig_to=<h-195704c@h-20b651e8120a33ec11.com>, relay=local, delay=0.11, delays=0.1/0.01/0/0.01, dsn=2.0.0, status=sent (delivered to command: procmail -a "$EXTENSION" DEFAULT=$HOME/Maildir/)`},
								},
							},
						},
					},
				})
			})
		})

		Convey("Search for sent/received messages", func() {
			d, clear := buildDetective(t, "../test_files/postfix_logs/individual_files/27_one_sent_one_received.log", year)
			defer clear()

			Convey("No status: return sent and received messages", func() {
				messages, err := d.CheckMessageDelivery(bg, "", "", correctInterval, -1, "", 1, limit)
				So(err, ShouldBeNil)
				So(messages.TotalResults, ShouldEqual, 2)
			})

			Convey("Sent: return only sent messages", func() {
				messages, err := d.CheckMessageDelivery(bg, "", "", correctInterval, int(parser.SentStatus), "", 1, limit)
				So(err, ShouldBeNil)
				So(messages.TotalResults, ShouldEqual, 1)
				So(messages.Messages[0].Queue, ShouldEqual, "4FA51DFCAD")
				So(messages.Messages[0].Entries[0].Status, ShouldEqual, parser.SentStatus)
			})

			Convey("Received: return only received messages", func() {
				messages, err := d.CheckMessageDelivery(bg, "", "", correctInterval, int(parser.ReceivedStatus), "", 1, limit)
				So(err, ShouldBeNil)
				So(messages.TotalResults, ShouldEqual, 1)
				So(messages.Messages[0].Queue, ShouldEqual, "DF1C3EB916")
				So(messages.Messages[0].Entries[0].Status, ShouldEqual, parser.ReceivedStatus)
			})
		})
	})

	Convey("CSV conversion", t, func() {
		expectedTime := time.Date(2020, time.January, 10, 16, 15, 30, 0, time.UTC)
		result := &detective.MessagesPage{1, 1, 1, 1,
			detective.Messages{
				detective.Message{
					Queue:     "1234",
					MessageID: "xf56",
					Entries: []detective.MessageDelivery{
						{
							1,
							expectedTime.In(time.UTC),
							expectedTime.In(time.UTC),
							detective.Status(parser.ReceivedStatus),
							"2.0.0",
							[]string{"host.com"},
							nil,
							"sender@example.com",
							[]string{"recipient@example.com"},
							[]string{`fake log line here`},
						},
					},
				},
			},
		}

		So(result.ExportCSV(), ShouldResemble, [][]string{{"1234", "xf56", "1", "2020-01-10T16:15:30Z", "2020-01-10T16:15:30Z", "received", "2.0.0", "", "sender@example.com", "recipient@example.com", "host.com", "fake log line here"}})
	})
}
