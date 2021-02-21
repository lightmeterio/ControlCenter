// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package deliverydb

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"log"
	"net"
	"testing"
	"time"
)

var fakeMapping domainmapping.Mapper

func init() {
	var err error
	fakeMapping, err = domainmapping.Mapping(domainmapping.RawList{"grouped": []string{"domaintobegrouped.com", "domaintobegrouped.de"}})
	errorutil.MustSucceed(err)
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestDatabaseCreation(t *testing.T) {
	Convey("Creation succeceds", t, func() {
		ws, clearWs := testutil.TempDir(t)
		defer clearWs()

		log.Println("database name:", ws)

		Convey("Insert some values", func() {
			db, err := New(ws, &fakeMapping)
			So(err, ShouldBeNil)

			done, cancel := db.Run()

			pub := db.ResultsPublisher()

			pub.Publish(buildDefaultResult())
			pub.Publish(buildDefaultResult())

			cancel()
			done()

			So(db.Close(), ShouldBeNil)
		})
	})
}

func buildDefaultResult() tracking.Result {
	result := tracking.Result{}
	result[tracking.ConnectionBeginKey] = tracking.ResultEntryInt64(2)
	result[tracking.ConnectionEndKey] = tracking.ResultEntryInt64(3)
	result[tracking.ConnectionClientHostnameKey] = tracking.ResultEntryText("client.host")
	result[tracking.ConnectionClientIPKey] = tracking.ResultEntryBlob(net.ParseIP("127.0.0.1"))
	result[tracking.QueueBeginKey] = tracking.ResultEntryInt64(2)
	result[tracking.QueueEndKey] = tracking.ResultEntryInt64(3)
	result[tracking.QueueSenderLocalPartKey] = tracking.ResultEntryText("sender")
	result[tracking.QueueSenderDomainPartKey] = tracking.ResultEntryText("sender.com")
	result[tracking.QueueOriginalMessageSizeKey] = tracking.ResultEntryInt64(32)
	result[tracking.QueueProcessedMessageSizeKey] = tracking.ResultEntryInt64(80)
	result[tracking.QueueNRCPTKey] = tracking.ResultEntryInt64(5)
	result[tracking.QueueMessageIDKey] = tracking.ResultEntryText("lala@caca.com")
	result[tracking.ResultDeliveryTimeKey] = tracking.ResultEntryInt64(3)
	result[tracking.ResultRecipientLocalPartKey] = tracking.ResultEntryText("recipient")
	result[tracking.ResultRecipientDomainPartKey] = tracking.ResultEntryText("recipient-domain.com")
	result[tracking.ResultOrigRecipientLocalPartKey] = tracking.ResultEntryNone()
	result[tracking.ResultOrigRecipientDomainPartKey] = tracking.ResultEntryNone()
	result[tracking.ResultDelayKey] = tracking.ResultEntryFloat64(1.0)
	result[tracking.ResultDelaySMTPDKey] = tracking.ResultEntryFloat64(1.0)
	result[tracking.ResultDelayCleanupKey] = tracking.ResultEntryFloat64(2.0)
	result[tracking.ResultDelayQmgrKey] = tracking.ResultEntryFloat64(3.0)
	result[tracking.ResultDelaySMTPKey] = tracking.ResultEntryFloat64(4.0)
	result[tracking.ResultDSNKey] = tracking.ResultEntryText("2.0.0")
	result[tracking.ResultStatusKey] = tracking.ResultEntryInt64(int64(parser.SentStatus))
	result[tracking.ResultRelayNameKey] = tracking.ResultEntryText("relay1.name")
	result[tracking.ResultRelayIPKey] = tracking.ResultEntryBlob(net.ParseIP("123.2.3.4"))
	result[tracking.ResultRelayPortKey] = tracking.ResultEntryInt64(42)
	result[tracking.ResultDeliveryServerKey] = tracking.ResultEntryText("server")
	result[tracking.ResultRecipientDomainPartKey] = tracking.ResultEntryText("domain.name")
	result[tracking.ResultMessageDirectionKey] = tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound))
	return result
}

func parseTimeInterval(from, to string) data.TimeInterval {
	interval, err := data.ParseTimeInterval(from, to, time.UTC)
	if err != nil {
		panic("pasring interval")
	}
	return interval
}

var (
	dummyContext = context.Background()
)

func countByStatus(dashboard dashboard.Dashboard, status parser.SmtpStatus, interval data.TimeInterval) int {
	v, err := dashboard.CountByStatus(dummyContext, status, interval)
	So(err, ShouldBeNil)
	return v
}

func topBusiestDomains(dashboard dashboard.Dashboard, interval data.TimeInterval) dashboard.Pairs {
	pairs, err := dashboard.TopBusiestDomains(dummyContext, interval)
	So(err, ShouldBeNil)
	return pairs
}

func topBouncedDomains(dashboard dashboard.Dashboard, interval data.TimeInterval) dashboard.Pairs {
	pairs, err := dashboard.TopBouncedDomains(dummyContext, interval)
	So(err, ShouldBeNil)
	return pairs
}

func topDeferredDomains(dashboard dashboard.Dashboard, interval data.TimeInterval) dashboard.Pairs {
	pairs, err := dashboard.TopDeferredDomains(dummyContext, interval)
	So(err, ShouldBeNil)
	return pairs
}

func deliveryStatus(dashboard dashboard.Dashboard, interval data.TimeInterval) dashboard.Pairs {
	pairs, err := dashboard.DeliveryStatus(dummyContext, interval)
	So(err, ShouldBeNil)
	return pairs
}

func TestEntriesInsertion(t *testing.T) {
	Convey("LogInsertion", t, func() {
		dir, clearDir := testutil.TempDir(t)
		defer clearDir()

		buildWs := func() (*DB, func() error, func(), tracking.ResultPublisher, dashboard.Dashboard, func()) {
			db, err := New(dir, &fakeMapping)
			So(err, ShouldBeNil)
			done, cancel := db.Run()
			pub := db.ResultsPublisher()

			dashboard, err := dashboard.New(db.ConnPool())
			So(err, ShouldBeNil)

			return db, done, cancel, pub, dashboard, func() {
				So(db.Close(), ShouldBeNil)
			}
		}

		fakeMessageWithRecipient := func(status parser.SmtpStatus, t time.Time, recipientLocalPart, recipientDomainPart string, dir tracking.MessageDirection) tracking.Result {
			r := buildDefaultResult()
			r[tracking.ResultRecipientLocalPartKey] = tracking.ResultEntryText(recipientLocalPart)
			r[tracking.ResultRecipientDomainPartKey] = tracking.ResultEntryText(recipientDomainPart)
			r[tracking.ResultDeliveryTimeKey] = tracking.ResultEntryInt64(t.Unix())
			r[tracking.ResultStatusKey] = tracking.ResultEntryInt64(int64(status))
			r[tracking.ResultMessageDirectionKey] = tracking.ResultEntryInt64(int64(dir))
			return r
		}

		fakeOutboundMessageWithRecipient := func(status parser.SmtpStatus, t time.Time, recipientLocalPart, recipientDomainPart string) tracking.Result {
			return fakeMessageWithRecipient(status, t, recipientLocalPart, recipientDomainPart, tracking.MessageDirectionOutbound)
		}

		fakeIncomingMessageWithRecipient := func(status parser.SmtpStatus, t time.Time, recipientLocalPart, recipientDomainPart string) tracking.Result {
			return fakeMessageWithRecipient(status, t, recipientLocalPart, recipientDomainPart, tracking.MessageDirectionIncoming)
		}

		fakeIncomingMessageWithSenderAndRecipient := func(status parser.SmtpStatus, t time.Time, senderLocalPart, senderDomainPart, recipientLocalPart, recipientDomainPart string) tracking.Result {
			r := buildDefaultResult()
			r[tracking.ResultRecipientLocalPartKey] = tracking.ResultEntryText(recipientLocalPart)
			r[tracking.QueueSenderLocalPartKey] = tracking.ResultEntryText(senderLocalPart)
			r[tracking.QueueSenderDomainPartKey] = tracking.ResultEntryText(senderDomainPart)
			r[tracking.ResultRecipientDomainPartKey] = tracking.ResultEntryText(recipientDomainPart)
			r[tracking.ResultDeliveryTimeKey] = tracking.ResultEntryInt64(t.Unix())
			r[tracking.ResultStatusKey] = tracking.ResultEntryInt64(int64(status))
			r[tracking.ResultMessageDirectionKey] = tracking.ResultEntryInt64(int64(tracking.MessageDirectionIncoming))
			return r
		}

		smtpStatusRecord := func(status parser.SmtpStatus, t time.Time) tracking.Result {
			return fakeOutboundMessageWithRecipient(status, t, "recipient", "test.com")
		}

		smtpStatusIncomingRecord := func(status parser.SmtpStatus, t time.Time) tracking.Result {
			return fakeIncomingMessageWithRecipient(status, t, "recipient", "test.com")
		}

		Convey("Inserting entries", func() {
			Convey("Inserts nothing", func() {
				db, done, cancel, _, dashboard, dtor := buildWs()
				defer dtor()
				cancel()
				So(done(), ShouldBeNil)

				interval := parseTimeInterval("1999-12-02", "2000-01-03")

				So(db.HasLogs(), ShouldBeFalse)
				So(countByStatus(dashboard, parser.BouncedStatus, interval), ShouldEqual, 0)
				So(countByStatus(dashboard, parser.DeferredStatus, interval), ShouldEqual, 0)
				So(countByStatus(dashboard, parser.SentStatus, interval), ShouldEqual, 0)
			})

			Convey("Incoming messages don't show in the dashboard", func() {
				db, done, cancel, pub, dashboard, dtor := buildWs()
				defer dtor()

				pub.Publish(smtpStatusIncomingRecord(parser.SentStatus, testutil.MustParseTime(`1999-12-02 13:10:10 +0000`)))
				pub.Publish(smtpStatusIncomingRecord(parser.DeferredStatus, testutil.MustParseTime(`1999-12-02 13:10:11 +0000`)))
				pub.Publish(smtpStatusIncomingRecord(parser.BouncedStatus, testutil.MustParseTime(`1999-12-02 13:10:12 +0000`)))

				cancel()
				So(done(), ShouldBeNil)

				interval := parseTimeInterval("1999-12-02", "2000-01-03")

				So(db.HasLogs(), ShouldBeTrue)
				So(countByStatus(dashboard, parser.BouncedStatus, interval), ShouldEqual, 0)
				So(countByStatus(dashboard, parser.DeferredStatus, interval), ShouldEqual, 0)
				So(countByStatus(dashboard, parser.SentStatus, interval), ShouldEqual, 0)

				So(db.MostRecentLogTime(), ShouldResemble, testutil.MustParseTime(`1999-12-02 13:10:12 +0000`))
			})

			Convey("Local messages with same domain sender are shown", func() {
				db, done, cancel, pub, d, dtor := buildWs()
				defer dtor()

				pub.Publish(smtpStatusIncomingRecord(parser.SentStatus, testutil.MustParseTime(`1999-12-02 13:10:10 +0000`)))
				pub.Publish(smtpStatusIncomingRecord(parser.DeferredStatus, testutil.MustParseTime(`1999-12-02 13:10:11 +0000`)))
				pub.Publish(fakeIncomingMessageWithSenderAndRecipient(parser.BouncedStatus, testutil.MustParseTime(`1999-12-02 13:10:12 +0000`), "sender", "example.com", "recipient", "example.com"))
				pub.Publish(smtpStatusIncomingRecord(parser.BouncedStatus, testutil.MustParseTime(`1999-12-02 13:10:20 +0000`)))
				pub.Publish(fakeIncomingMessageWithSenderAndRecipient(parser.DeferredStatus, testutil.MustParseTime(`1999-12-02 13:10:30 +0000`), "sender2", "example2.com", "recipient2", "example2.com"))
				pub.Publish(fakeIncomingMessageWithSenderAndRecipient(parser.BouncedStatus, testutil.MustParseTime(`1999-12-02 13:10:30 +0000`), "sender3", "example2.com", "recipient2", "example2.com"))

				cancel()
				So(done(), ShouldBeNil)

				interval := parseTimeInterval("1999-12-02", "2000-01-03")

				So(db.HasLogs(), ShouldBeTrue)
				So(countByStatus(d, parser.BouncedStatus, interval), ShouldEqual, 2)
				So(countByStatus(d, parser.DeferredStatus, interval), ShouldEqual, 1)
				So(countByStatus(d, parser.SentStatus, interval), ShouldEqual, 0)

				So(topBusiestDomains(d, interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "example2.com", Value: 2},
					dashboard.Pair{Key: "example.com", Value: 1},
				})

				So(topBouncedDomains(d, interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "example.com", Value: 1},
					dashboard.Pair{Key: "example2.com", Value: 1},
				})

				So(topDeferredDomains(d, interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "example2.com", Value: 1},
				})
			})

			Convey("Inserts one log entry", func() {
				db, done, cancel, pub, dashboard, dtor := buildWs()
				defer dtor()

				pub.Publish(smtpStatusRecord(parser.SentStatus, testutil.MustParseTime(`1999-12-02 13:10:10 +0000`)))
				cancel()
				So(done(), ShouldBeNil)

				interval := parseTimeInterval("1999-12-01", "2000-01-03")

				So(db.HasLogs(), ShouldBeTrue)
				So(countByStatus(dashboard, parser.BouncedStatus, interval), ShouldEqual, 0)
				So(countByStatus(dashboard, parser.DeferredStatus, interval), ShouldEqual, 0)
				So(countByStatus(dashboard, parser.SentStatus, interval), ShouldEqual, 1)
			})

			Convey("Insert, reopen, insert", func() {
				func() {
					_, done, cancel, pub, _, dtor := buildWs()
					defer dtor()

					// this one is before the time interval
					pub.Publish(smtpStatusRecord(parser.DeferredStatus, testutil.MustParseTime(`1999-11-02 13:10:10 +0000`)))

					pub.Publish(smtpStatusRecord(parser.SentStatus, testutil.MustParseTime(`1999-12-02 13:10:10 +0000`)))
					cancel()
					So(done(), ShouldBeNil)
				}()

				// reopen workspace and add another log
				db, done, cancel, pub, dashboard, dtor := buildWs()
				defer dtor()

				pub.Publish(smtpStatusRecord(parser.SentStatus, testutil.MustParseTime(`1999-12-04 13:10:10 +0000`)))
				pub.Publish(smtpStatusRecord(parser.DeferredStatus, testutil.MustParseTime(`1999-12-04 13:10:10 +0000`)))

				pub.Publish(smtpStatusRecord(parser.BouncedStatus, testutil.MustParseTime(`2000-03-10 13:10:10 +0000`)))

				// this one is after the time interval
				pub.Publish(smtpStatusRecord(parser.DeferredStatus, testutil.MustParseTime(`2000-05-02 13:10:10 +0000`)))

				cancel()
				So(done(), ShouldBeNil)

				interval := parseTimeInterval("1999-12-02", "2000-03-11")

				So(db.HasLogs(), ShouldBeTrue)

				So(countByStatus(dashboard, parser.BouncedStatus, interval), ShouldEqual, 1)
				So(countByStatus(dashboard, parser.DeferredStatus, interval), ShouldEqual, 1)
				So(countByStatus(dashboard, parser.SentStatus, interval), ShouldEqual, 2)
			})

			t := func(year int, month time.Month, day, hour, minute, second int) time.Time {
				return time.Date(year, month, day, hour, minute, second, 0, time.UTC)
			}

			Convey("Many different smtp status", func() {
				_, done, cancel, pub, d, dtor := buildWs()
				defer dtor()

				interval := parseTimeInterval("1999-12-02", "2000-03-11")

				{
					s := parser.SentStatus
					d := parser.DeferredStatus
					b := parser.BouncedStatus

					// Something before the interval
					pub.Publish(fakeOutboundMessageWithRecipient(s, t(1999, time.December, 1, 13, 0, 0), "recip", "domain"))

					// Inside the interval
					pub.Publish(fakeOutboundMessageWithRecipient(s, t(1999, time.December, 2, 14, 1, 2), "r1", "ALALALA.COM"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, t(1999, time.December, 2, 14, 1, 3), "r2", "abcdf.com"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, t(1999, time.December, 2, 14, 1, 4), "r3", "alalala.com"))
					pub.Publish(fakeOutboundMessageWithRecipient(d, t(1999, time.December, 3, 14, 1, 4), "r3", "EMAIL2.COM"))
					pub.Publish(fakeOutboundMessageWithRecipient(d, t(1999, time.December, 5, 15, 1, 0), "r2", "email3.com"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, t(1999, time.December, 6, 16, 1, 4), "r3", "ALALALA.COM"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, t(2000, time.January, 3, 15, 1, 0), "r2", "abcdf.com"))
					pub.Publish(fakeOutboundMessageWithRecipient(d, t(2000, time.January, 4, 15, 1, 0), "r2", "EMAIL1.COM"))
					pub.Publish(fakeOutboundMessageWithRecipient(s, t(2000, time.January, 4, 16, 1, 0), "r2", "example1.com"))
					pub.Publish(fakeOutboundMessageWithRecipient(s, t(2000, time.January, 4, 16, 2, 1), "r2", "example1.com"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, t(2000, time.March, 11, 16, 2, 1), "r100", "email2.com"))

					// Incoming messages do not count
					pub.Publish(fakeIncomingMessageWithRecipient(b, t(2000, time.March, 11, 16, 2, 1), "r100", "email2.com"))
					pub.Publish(fakeIncomingMessageWithRecipient(s, t(2000, time.March, 11, 16, 2, 1), "r100", "email2.com"))
					pub.Publish(fakeIncomingMessageWithRecipient(d, t(2000, time.March, 11, 16, 2, 1), "r100", "email2.com"))

					// Something after the interval
					pub.Publish(fakeOutboundMessageWithRecipient(d, t(2000, time.March, 12, 13, 0, 0), "recip", "domain"))
				}

				cancel()
				So(done(), ShouldBeNil)

				Convey("Busiest: used domain, regardless of the status", func() {
					So(topBusiestDomains(d, interval), ShouldResemble, dashboard.Pairs{
						dashboard.Pair{Key: "alalala.com", Value: 3},
						dashboard.Pair{Key: "abcdf.com", Value: 2},
						dashboard.Pair{Key: "email2.com", Value: 2},
						dashboard.Pair{Key: "example1.com", Value: 2},
						dashboard.Pair{Key: "email1.com", Value: 1},
						dashboard.Pair{Key: "email3.com", Value: 1},
					})
				})

				Convey("Bounced: status = bounced", func() {
					So(topBouncedDomains(d, interval), ShouldResemble, dashboard.Pairs{
						dashboard.Pair{Key: "abcdf.com", Value: 2},
						dashboard.Pair{Key: "alalala.com", Value: 2},
						dashboard.Pair{Key: "email2.com", Value: 1},
					})
				})

				Convey("Deferred: status = deferred", func() {
					So(topDeferredDomains(d, interval), ShouldResemble, dashboard.Pairs{
						dashboard.Pair{Key: "email1.com", Value: 1},
						dashboard.Pair{Key: "email2.com", Value: 1},
						dashboard.Pair{Key: "email3.com", Value: 1},
					})
				})

				Convey("Delivery Status", func() {
					So(deliveryStatus(d, interval), ShouldResemble, dashboard.Pairs{
						dashboard.Pair{Key: "sent", Value: 3},
						dashboard.Pair{Key: "bounced", Value: 5},
						dashboard.Pair{Key: "deferred", Value: 3},
					})
				})
			})

			Convey("Group According to Domain mapping", func() {
				_, done, cancel, pub, d, dtor := buildWs()
				defer dtor()

				{
					s := parser.SentStatus
					d := parser.DeferredStatus
					b := parser.BouncedStatus

					pub.Publish(fakeOutboundMessageWithRecipient(d, t(2020, time.January, 1, 1, 0, 0), "p1", "domaintobegrouped.de"))
					pub.Publish(fakeOutboundMessageWithRecipient(d, t(2020, time.January, 2, 1, 0, 0), "p1", "another.de"))
					pub.Publish(fakeOutboundMessageWithRecipient(d, t(2020, time.January, 2, 2, 0, 0), "p1", "domaintobegrouped.com"))

					pub.Publish(fakeOutboundMessageWithRecipient(b, t(2020, time.January, 3, 1, 0, 0), "p1", "domaintobegrouped.de"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, t(2020, time.January, 4, 1, 0, 0), "p1", "domaintobegrouped.com"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, t(2020, time.January, 5, 1, 0, 0), "p1", "domaintobegrouped.de"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, t(2020, time.January, 6, 1, 0, 0), "p1", "another.de"))

					pub.Publish(fakeOutboundMessageWithRecipient(s, t(2020, time.January, 6, 1, 0, 0), "p1", "domaintobegrouped.com"))
				}

				cancel()
				So(done(), ShouldBeNil)

				interval := parseTimeInterval(`2020-01-01`, `2020-12-31`)

				So(topBusiestDomains(d, interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "grouped", Value: 6},
					dashboard.Pair{Key: "another.de", Value: 2},
				})

				So(topBouncedDomains(d, interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "grouped", Value: 3},
					dashboard.Pair{Key: "another.de", Value: 1},
				})

				So(topDeferredDomains(d, interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "grouped", Value: 2},
					dashboard.Pair{Key: "another.de", Value: 1},
				})
			})
		})
	})
}
