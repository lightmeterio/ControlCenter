// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package deliverydb

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"net"
	"path"
	"testing"
	"time"
)

var fakeMapping domainmapping.Mapper

const databaseName = "logs"

func init() {
	var err error
	fakeMapping, err = domainmapping.Mapping(domainmapping.RawList{"grouped": []string{"domaintobegrouped.com", "domaintobegrouped.de"}})
	errorutil.MustSucceed(err)
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestDatabaseCreation(t *testing.T) {
	Convey("Creation succeeds", t, func() {
		conn, closeConn := testutil.TempDBConnectionMigrated(t, databaseName)
		defer closeConn()

		Convey("Insert some values", func() {
			db, err := New(conn, &fakeMapping, NoFilters)
			So(err, ShouldBeNil)

			done, cancel := runner.Run(db)

			pub := db.ResultsPublisher()

			pub.Publish(buildDefaultResult())
			pub.Publish(buildDefaultResult())

			cancel()
			done()
		})
	})
}

func buildDefaultResult() tracking.Result {
	return tracking.MappedResult{
		tracking.ConnectionBeginKey:               tracking.ResultEntryInt64(2),
		tracking.ConnectionEndKey:                 tracking.ResultEntryInt64(3),
		tracking.ConnectionClientHostnameKey:      tracking.ResultEntryText("client.host"),
		tracking.ConnectionClientIPKey:            tracking.ResultEntryBlob(net.ParseIP("127.0.0.1")),
		tracking.QueueBeginKey:                    tracking.ResultEntryInt64(2),
		tracking.QueueEndKey:                      tracking.ResultEntryInt64(3),
		tracking.QueueSenderLocalPartKey:          tracking.ResultEntryText("sender"),
		tracking.QueueSenderDomainPartKey:         tracking.ResultEntryText("sender.com"),
		tracking.QueueOriginalMessageSizeKey:      tracking.ResultEntryInt64(32),
		tracking.QueueProcessedMessageSizeKey:     tracking.ResultEntryInt64(80),
		tracking.QueueNRCPTKey:                    tracking.ResultEntryInt64(5),
		tracking.QueueMessageIDKey:                tracking.ResultEntryText("lala@caca.com"),
		tracking.ResultDeliveryTimeKey:            tracking.ResultEntryInt64(3),
		tracking.ResultRecipientLocalPartKey:      tracking.ResultEntryText("recipient"),
		tracking.ResultRecipientDomainPartKey:     tracking.ResultEntryText("recipient-domain.com"),
		tracking.ResultOrigRecipientLocalPartKey:  tracking.ResultEntryText("orig_recipient"),
		tracking.ResultOrigRecipientDomainPartKey: tracking.ResultEntryText("example.com"),
		tracking.ResultDelayKey:                   tracking.ResultEntryFloat64(1.0),
		tracking.ResultDelaySMTPDKey:              tracking.ResultEntryFloat64(1.0),
		tracking.ResultDelayCleanupKey:            tracking.ResultEntryFloat64(2.0),
		tracking.ResultDelayQmgrKey:               tracking.ResultEntryFloat64(3.0),
		tracking.ResultDelaySMTPKey:               tracking.ResultEntryFloat64(4.0),
		tracking.ResultDSNKey:                     tracking.ResultEntryText("2.0.0"),
		tracking.ResultStatusKey:                  tracking.ResultEntryInt64(int64(parser.SentStatus)),
		tracking.ResultRelayNameKey:               tracking.ResultEntryText("relay1.name"),
		tracking.ResultRelayIPKey:                 tracking.ResultEntryBlob(net.ParseIP("123.2.3.4")),
		tracking.ResultRelayPortKey:               tracking.ResultEntryInt64(42),
		tracking.ResultDeliveryServerKey:          tracking.ResultEntryText("server"),
		tracking.ResultMessageDirectionKey:        tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
		tracking.QueueDeliveryNameKey:             tracking.ResultEntryText("AAAAAA"),
		tracking.ResultDeliveryLineChecksum:       tracking.ResultEntryInt64(42),
	}.Result()
}

func parseTimeInterval(from, to string) timeutil.TimeInterval {
	interval, err := timeutil.ParseTimeInterval(from, to, time.UTC)
	if err != nil {
		panic("pasring interval")
	}
	return interval
}

var (
	dummyContext = context.Background()
)

func countByStatus(dashboard dashboard.Dashboard, status parser.SmtpStatus, interval timeutil.TimeInterval) int {
	v, err := dashboard.CountByStatus(dummyContext, status, interval)
	So(err, ShouldBeNil)
	return v
}

func topBusiestDomains(dashboard dashboard.Dashboard, interval timeutil.TimeInterval) dashboard.Pairs {
	pairs, err := dashboard.TopBusiestDomains(dummyContext, interval)
	So(err, ShouldBeNil)
	return pairs
}

func topBouncedDomains(dashboard dashboard.Dashboard, interval timeutil.TimeInterval) dashboard.Pairs {
	pairs, err := dashboard.TopBouncedDomains(dummyContext, interval)
	So(err, ShouldBeNil)
	return pairs
}

func topDeferredDomains(dashboard dashboard.Dashboard, interval timeutil.TimeInterval) dashboard.Pairs {
	pairs, err := dashboard.TopDeferredDomains(dummyContext, interval)
	So(err, ShouldBeNil)
	return pairs
}

func deliveryStatus(dashboard dashboard.Dashboard, interval timeutil.TimeInterval) dashboard.Pairs {
	pairs, err := dashboard.DeliveryStatus(dummyContext, interval)
	So(err, ShouldBeNil)
	return pairs
}

func buildTime(year int, month time.Month, day, hour, minute, second int) time.Time {
	return time.Date(year, month, day, hour, minute, second, 0, time.UTC)
}

func fakeMessageWithRecipient(status parser.SmtpStatus, t time.Time, recipientLocalPart, recipientDomainPart string, dir tracking.MessageDirection) tracking.Result {
	r := buildDefaultResult()
	r[tracking.ResultRecipientLocalPartKey] = tracking.ResultEntryText(recipientLocalPart)
	r[tracking.ResultRecipientDomainPartKey] = tracking.ResultEntryText(recipientDomainPart)
	r[tracking.ResultDeliveryTimeKey] = tracking.ResultEntryInt64(t.Unix())
	r[tracking.ResultStatusKey] = tracking.ResultEntryInt64(int64(status))
	r[tracking.ResultMessageDirectionKey] = tracking.ResultEntryInt64(int64(dir))
	return r
}

func fakeOutboundMessageWithRecipient(status parser.SmtpStatus, t time.Time, recipientLocalPart, recipientDomainPart string) tracking.Result {
	return fakeMessageWithRecipient(status, t, recipientLocalPart, recipientDomainPart, tracking.MessageDirectionOutbound)
}

func fakeIncomingMessageWithRecipient(status parser.SmtpStatus, t time.Time, recipientLocalPart, recipientDomainPart string) tracking.Result {
	return fakeMessageWithRecipient(status, t, recipientLocalPart, recipientDomainPart, tracking.MessageDirectionIncoming)
}

func fakeIncomingMessageWithSenderAndRecipient(status parser.SmtpStatus, t time.Time, senderLocalPart, senderDomainPart, recipientLocalPart, recipientDomainPart string) tracking.Result {
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

func smtpStatusRecord(status parser.SmtpStatus, t time.Time) tracking.Result {
	return fakeOutboundMessageWithRecipient(status, t, "recipient", "test.com")
}

func smtpStatusIncomingRecord(status parser.SmtpStatus, t time.Time) tracking.Result {
	return fakeIncomingMessageWithRecipient(status, t, "recipient", "test.com")
}

func TestEntriesInsertion(t *testing.T) {
	Convey("LogInsertion", t, func() {
		conn, closeConn := testutil.TempDBConnectionMigrated(t, databaseName)
		defer closeConn()

		buildWs := func() (*DB, func() error, func(), tracking.ResultPublisher, dashboard.Dashboard) {
			db, err := New(conn, &fakeMapping, NoFilters)
			So(err, ShouldBeNil)
			done, cancel := runner.Run(db)
			pub := db.ResultsPublisher()

			dashboard, err := dashboard.New(conn.RoConnPool)
			So(err, ShouldBeNil)

			return db, done, cancel, pub, dashboard
		}

		Convey("Inserting entries", func() {
			Convey("Inserts nothing", func() {
				db, done, cancel, _, dashboard := buildWs()
				cancel()
				So(done(), ShouldBeNil)

				interval := parseTimeInterval("1999-12-02", "2000-01-03")

				So(db.HasLogs(), ShouldBeFalse)
				So(countByStatus(dashboard, parser.BouncedStatus, interval), ShouldEqual, 0)
				So(countByStatus(dashboard, parser.DeferredStatus, interval), ShouldEqual, 0)
				So(countByStatus(dashboard, parser.SentStatus, interval), ShouldEqual, 0)
			})

			Convey("Incoming messages don't show in the dashboard", func() {
				db, done, cancel, pub, dashboard := buildWs()

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

				l, err := db.MostRecentLogTime()
				So(err, ShouldBeNil)
				So(l, ShouldResemble, testutil.MustParseTime(`1999-12-02 13:10:12 +0000`))
			})

			Convey("Local messages with same domain sender are shown", func() {
				db, done, cancel, pub, d := buildWs()

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
				db, done, cancel, pub, dashboard := buildWs()

				pub.Publish(smtpStatusRecord(parser.SentStatus, testutil.MustParseTime(`1999-12-02 13:10:10 +0000`)))
				cancel()
				So(done(), ShouldBeNil)

				interval := parseTimeInterval("1999-12-01", "2000-01-03")

				So(db.HasLogs(), ShouldBeTrue)
				So(countByStatus(dashboard, parser.BouncedStatus, interval), ShouldEqual, 0)
				So(countByStatus(dashboard, parser.DeferredStatus, interval), ShouldEqual, 0)
				So(countByStatus(dashboard, parser.SentStatus, interval), ShouldEqual, 1)
			})

			Convey("Many different smtp status", func() {
				_, done, cancel, pub, d := buildWs()

				interval := parseTimeInterval("1999-12-02", "2000-03-11")

				{
					s := parser.SentStatus
					d := parser.DeferredStatus
					b := parser.BouncedStatus

					// Something before the interval
					pub.Publish(fakeOutboundMessageWithRecipient(s, buildTime(1999, time.December, 1, 13, 0, 0), "recip", "domain"))

					// Inside the interval
					pub.Publish(fakeOutboundMessageWithRecipient(s, buildTime(1999, time.December, 2, 14, 1, 2), "r1", "ALALALA.COM"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, buildTime(1999, time.December, 2, 14, 1, 3), "r2", "abcdf.com"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, buildTime(1999, time.December, 2, 14, 1, 4), "r3", "alalala.com"))
					pub.Publish(fakeOutboundMessageWithRecipient(d, buildTime(1999, time.December, 3, 14, 1, 4), "r3", "EMAIL2.COM"))
					pub.Publish(fakeOutboundMessageWithRecipient(d, buildTime(1999, time.December, 5, 15, 1, 0), "r2", "email3.com"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, buildTime(1999, time.December, 6, 16, 1, 4), "r3", "ALALALA.COM"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, buildTime(2000, time.January, 3, 15, 1, 0), "r2", "abcdf.com"))
					pub.Publish(fakeOutboundMessageWithRecipient(d, buildTime(2000, time.January, 4, 15, 1, 0), "r2", "EMAIL1.COM"))
					pub.Publish(fakeOutboundMessageWithRecipient(s, buildTime(2000, time.January, 4, 16, 1, 0), "r2", "example1.com"))
					pub.Publish(fakeOutboundMessageWithRecipient(s, buildTime(2000, time.January, 4, 16, 2, 1), "r2", "example1.com"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, buildTime(2000, time.March, 11, 16, 2, 1), "r100", "email2.com"))

					// Incoming messages do not count
					pub.Publish(fakeIncomingMessageWithRecipient(b, buildTime(2000, time.March, 11, 16, 2, 1), "r100", "email2.com"))
					pub.Publish(fakeIncomingMessageWithRecipient(s, buildTime(2000, time.March, 11, 16, 2, 1), "r100", "email2.com"))
					pub.Publish(fakeIncomingMessageWithRecipient(d, buildTime(2000, time.March, 11, 16, 2, 1), "r100", "email2.com"))

					// Something after the interval
					pub.Publish(fakeOutboundMessageWithRecipient(d, buildTime(2000, time.March, 12, 13, 0, 0), "recip", "domain"))
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
				_, done, cancel, pub, d := buildWs()

				{
					s := parser.SentStatus
					d := parser.DeferredStatus
					b := parser.BouncedStatus

					pub.Publish(fakeOutboundMessageWithRecipient(d, buildTime(2020, time.January, 1, 1, 0, 0), "p1", "domaintobegrouped.de"))
					pub.Publish(fakeOutboundMessageWithRecipient(d, buildTime(2020, time.January, 2, 1, 0, 0), "p1", "another.de"))
					pub.Publish(fakeOutboundMessageWithRecipient(d, buildTime(2020, time.January, 2, 2, 0, 0), "p1", "domaintobegrouped.com"))

					pub.Publish(fakeOutboundMessageWithRecipient(b, buildTime(2020, time.January, 3, 1, 0, 0), "p1", "domaintobegrouped.de"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, buildTime(2020, time.January, 4, 1, 0, 0), "p1", "domaintobegrouped.com"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, buildTime(2020, time.January, 5, 1, 0, 0), "p1", "domaintobegrouped.de"))
					pub.Publish(fakeOutboundMessageWithRecipient(b, buildTime(2020, time.January, 6, 1, 0, 0), "p1", "another.de"))

					pub.Publish(fakeOutboundMessageWithRecipient(s, buildTime(2020, time.January, 6, 1, 0, 0), "p1", "domaintobegrouped.com"))
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

func TestCleaningOldEntries(t *testing.T) {
	Convey("Clean old entries", t, func() {
		conn, closeConn := testutil.TempDBConnectionMigrated(t, databaseName)
		defer closeConn()

		// TODO: do not duplicate this function!
		buildWs := func() (*DB, func() error, func(), tracking.ResultPublisher, dashboard.Dashboard) {
			db, err := New(conn, &fakeMapping, NoFilters)
			So(err, ShouldBeNil)
			done, cancel := runner.Run(db)
			pub := db.ResultsPublisher()

			dashboard, err := dashboard.New(conn.RoConnPool)
			So(err, ShouldBeNil)

			return db, done, cancel, pub, dashboard
		}

		db, done, cancel, pub, d := buildWs()

		// TODO: have some entries before the time, some with parenting relationship,
		// some expired

		baseTime := timeutil.MustParseTime(`2020-01-01 10:00:00 +0000`)

		pub.Publish(tracking.MappedResult{
			tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.SentStatus)),
			tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Unix()),
			tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
			tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("sender.example.com"),
			tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("sender1"),
			tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("recipient.example.com"),
			tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("recipient1"),
			tracking.QueueMessageIDKey:            tracking.ResultEntryText("message_id_1"),
			tracking.ConnectionBeginKey:           tracking.ResultEntryInt64(baseTime.Unix()),
			tracking.QueueBeginKey:                tracking.ResultEntryInt64(baseTime.Unix()),
			tracking.QueueOriginalMessageSizeKey:  tracking.ResultEntryInt64(42),
			tracking.QueueProcessedMessageSizeKey: tracking.ResultEntryInt64(100),
			tracking.QueueNRCPTKey:                tracking.ResultEntryInt64(1),
			tracking.ResultDeliveryServerKey:      tracking.ResultEntryText("mail"),
			tracking.ResultDelayKey:               tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPDKey:          tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayCleanupKey:        tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayQmgrKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultRelayNameKey:           tracking.ResultEntryText("relay1.example.com"),
			tracking.ResultRelayIPKey:             tracking.ResultEntryBlob([]byte{192, 168, 0, 1}),
			tracking.ResultRelayPortKey:           tracking.ResultEntryInt64(25),
			tracking.ConnectionClientHostnameKey:  tracking.ResultEntryText("some.host.com"),
			tracking.ConnectionClientIPKey:        tracking.ResultEntryBlob([]byte{192, 168, 0, 2}),
			tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("A1"),
			tracking.ResultDSNKey:                 tracking.ResultEntryText("2.0.0"),
			tracking.ResultDeliveryLineChecksum:   tracking.ResultEntryInt64(200),
		}.Result())

		pub.Publish(tracking.MappedResult{
			tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.SentStatus)),
			tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Add(time.Minute * 30).Unix()),
			tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
			tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("sender2.example.com"),
			tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("sender2"),
			tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("recipient3.example.com"),
			tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("recipient3"),
			tracking.QueueMessageIDKey:            tracking.ResultEntryText("message_id_1"), // NOTE: same messageid as the first message
			tracking.ConnectionBeginKey:           tracking.ResultEntryInt64(baseTime.Add(time.Minute * 30).Unix()),
			tracking.QueueBeginKey:                tracking.ResultEntryInt64(baseTime.Add(time.Minute * 30).Unix()),
			tracking.QueueOriginalMessageSizeKey:  tracking.ResultEntryInt64(42),
			tracking.QueueProcessedMessageSizeKey: tracking.ResultEntryInt64(100),
			tracking.QueueNRCPTKey:                tracking.ResultEntryInt64(1),
			tracking.ResultDeliveryServerKey:      tracking.ResultEntryText("mail"),
			tracking.ResultDelayKey:               tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPDKey:          tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayCleanupKey:        tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayQmgrKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultRelayNameKey:           tracking.ResultEntryText("relay2.example.com"),
			tracking.ResultRelayIPKey:             tracking.ResultEntryBlob([]byte{192, 168, 0, 1}),
			tracking.ResultRelayPortKey:           tracking.ResultEntryInt64(25),
			tracking.ConnectionClientHostnameKey:  tracking.ResultEntryText("some.host.com"),
			tracking.ConnectionClientIPKey:        tracking.ResultEntryBlob([]byte{192, 168, 0, 2}),
			tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("A1"),
			tracking.ResultDSNKey:                 tracking.ResultEntryText("2.0.0"),
			tracking.ResultDeliveryLineChecksum:   tracking.ResultEntryInt64(201),
		}.Result())

		// a bounced message, followed by a return message
		pub.Publish(tracking.MappedResult{
			tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.BouncedStatus)),
			tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Add(time.Minute * 30).Unix()),
			tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
			tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("sender2.example.com"),
			tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("sender2"),
			tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("recipient2.example.com"),
			tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("recipient2"),
			tracking.QueueMessageIDKey:            tracking.ResultEntryText("message_id_2"), // NOTE: same messageid as the first message
			tracking.ConnectionBeginKey:           tracking.ResultEntryInt64(baseTime.Add(time.Minute * 30).Unix()),
			tracking.QueueBeginKey:                tracking.ResultEntryInt64(baseTime.Add(time.Minute * 30).Unix()),
			tracking.QueueOriginalMessageSizeKey:  tracking.ResultEntryInt64(42),
			tracking.QueueProcessedMessageSizeKey: tracking.ResultEntryInt64(100),
			tracking.QueueNRCPTKey:                tracking.ResultEntryInt64(1),
			tracking.ResultDeliveryServerKey:      tracking.ResultEntryText("mail"),
			tracking.ResultDelayKey:               tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPDKey:          tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayCleanupKey:        tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayQmgrKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultRelayNameKey:           tracking.ResultEntryText("relay2.example.com"),
			tracking.ResultRelayIPKey:             tracking.ResultEntryBlob([]byte{192, 168, 0, 1}),
			tracking.ResultRelayPortKey:           tracking.ResultEntryInt64(25),
			tracking.ConnectionClientHostnameKey:  tracking.ResultEntryText("some.host.com"),
			tracking.ConnectionClientIPKey:        tracking.ResultEntryBlob([]byte{192, 168, 0, 2}),
			tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("A2"),
			tracking.ResultDSNKey:                 tracking.ResultEntryText("4.0.0"),
			tracking.ResultDeliveryLineChecksum:   tracking.ResultEntryInt64(202),
		}.Result())

		// returned here
		pub.Publish(tracking.MappedResult{
			tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.SentStatus)),
			tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Add(time.Minute * 31).Unix()),
			tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionIncoming)),
			tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("recipient2.example.com"),
			tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("recipient2"),
			tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("sender2.example.com"),
			tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("sender2"),
			tracking.QueueMessageIDKey:            tracking.ResultEntryText("message_id_3"),
			tracking.ConnectionBeginKey:           tracking.ResultEntryInt64(baseTime.Add(time.Minute * 31).Unix()),
			tracking.QueueBeginKey:                tracking.ResultEntryInt64(baseTime.Add(time.Minute * 31).Unix()),
			tracking.QueueOriginalMessageSizeKey:  tracking.ResultEntryInt64(42),
			tracking.QueueProcessedMessageSizeKey: tracking.ResultEntryInt64(100),
			tracking.QueueNRCPTKey:                tracking.ResultEntryInt64(1),
			tracking.ResultDeliveryServerKey:      tracking.ResultEntryText("mail"),
			tracking.ResultDelayKey:               tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPDKey:          tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayCleanupKey:        tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayQmgrKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultRelayNameKey:           tracking.ResultEntryText("relay2.example.com"),
			tracking.ResultRelayIPKey:             tracking.ResultEntryBlob([]byte{192, 168, 0, 1}),
			tracking.ResultRelayPortKey:           tracking.ResultEntryInt64(25),
			tracking.ConnectionClientHostnameKey:  tracking.ResultEntryText("some.host.com"),
			tracking.ConnectionClientIPKey:        tracking.ResultEntryBlob([]byte{192, 168, 0, 2}),
			tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("A3"),
			tracking.ParentQueueDeliveryNameKey:   tracking.ResultEntryText("A2"),
			tracking.ResultDSNKey:                 tracking.ResultEntryText("2.0.0"),
			tracking.ResultDeliveryLineChecksum:   tracking.ResultEntryInt64(203),
		}.Result())

		// Deferred once and then expired
		pub.Publish(tracking.MappedResult{
			tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.DeferredStatus)),
			tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Add(time.Hour * 1).Unix()),
			tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionIncoming)),
			tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("sender2.example.com"),
			tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("sender2"),
			tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("recipient2.example.com"),
			tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("recipient2"),
			tracking.QueueMessageIDKey:            tracking.ResultEntryText("message_id_4"),
			tracking.ConnectionBeginKey:           tracking.ResultEntryInt64(baseTime.Add(time.Hour * 1).Unix()),
			tracking.QueueBeginKey:                tracking.ResultEntryInt64(baseTime.Add(time.Hour * 1).Unix()),
			tracking.QueueOriginalMessageSizeKey:  tracking.ResultEntryInt64(42),
			tracking.QueueProcessedMessageSizeKey: tracking.ResultEntryInt64(100),
			tracking.QueueNRCPTKey:                tracking.ResultEntryInt64(1),
			tracking.ResultDeliveryServerKey:      tracking.ResultEntryText("mail"),
			tracking.ResultDelayKey:               tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPDKey:          tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayCleanupKey:        tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayQmgrKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultRelayNameKey:           tracking.ResultEntryText("relay2.example.com"),
			tracking.ResultRelayIPKey:             tracking.ResultEntryBlob([]byte{192, 168, 0, 1}),
			tracking.ResultRelayPortKey:           tracking.ResultEntryInt64(25),
			tracking.ConnectionClientHostnameKey:  tracking.ResultEntryText("some.host.com"),
			tracking.ConnectionClientIPKey:        tracking.ResultEntryBlob([]byte{192, 168, 0, 2}),
			tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("A4"),
			tracking.ResultDSNKey:                 tracking.ResultEntryText("5.0.0"),
			tracking.ResultDeliveryLineChecksum:   tracking.ResultEntryInt64(204),
		}.Result())

		// expires here
		pub.Publish(tracking.MappedResult{
			tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.ExpiredStatus)),
			tracking.MessageExpiredTime:           tracking.ResultEntryInt64(baseTime.Add(time.Hour * 2).Unix()),
			tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionIncoming)),
			tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("sender2.example.com"),
			tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("sender2"),
			tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("recipient2.example.com"),
			tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("recipient2"),
			tracking.QueueMessageIDKey:            tracking.ResultEntryText("message_id_4"),
			tracking.ConnectionBeginKey:           tracking.ResultEntryInt64(baseTime.Add(time.Hour * 2).Unix()),
			tracking.QueueBeginKey:                tracking.ResultEntryInt64(baseTime.Add(time.Hour * 2).Unix()),
			tracking.QueueOriginalMessageSizeKey:  tracking.ResultEntryInt64(42),
			tracking.QueueProcessedMessageSizeKey: tracking.ResultEntryInt64(100),
			tracking.QueueNRCPTKey:                tracking.ResultEntryInt64(1),
			tracking.ResultDeliveryServerKey:      tracking.ResultEntryText("mail"),
			tracking.ResultDelayKey:               tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPDKey:          tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayCleanupKey:        tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayQmgrKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultRelayNameKey:           tracking.ResultEntryText("relay2.example.com"),
			tracking.ResultRelayIPKey:             tracking.ResultEntryBlob([]byte{192, 168, 0, 1}),
			tracking.ResultRelayPortKey:           tracking.ResultEntryInt64(25),
			tracking.ConnectionClientHostnameKey:  tracking.ResultEntryText("some.host.com"),
			tracking.ConnectionClientIPKey:        tracking.ResultEntryBlob([]byte{192, 168, 0, 2}),
			tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("A4"),
			tracking.ResultDSNKey:                 tracking.ResultEntryText("5.0.0"),
			tracking.ResultDeliveryLineChecksum:   tracking.ResultEntryInt64(205),
		}.Result())

		pub.Publish(tracking.MappedResult{
			tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.BouncedStatus)),
			tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Add(time.Hour * 3).Unix()),
			tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
			tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("sender.example.com"),
			tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("sender1"),
			tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("recipient.example.com"),
			tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("recipient1"),
			tracking.QueueMessageIDKey:            tracking.ResultEntryText("message_id_5"),
			tracking.ConnectionBeginKey:           tracking.ResultEntryInt64(baseTime.Add(time.Hour * 3).Unix()),
			tracking.QueueBeginKey:                tracking.ResultEntryInt64(baseTime.Add(time.Hour * 3).Unix()),
			tracking.QueueOriginalMessageSizeKey:  tracking.ResultEntryInt64(42),
			tracking.QueueProcessedMessageSizeKey: tracking.ResultEntryInt64(100),
			tracking.QueueNRCPTKey:                tracking.ResultEntryInt64(1),
			tracking.ResultDeliveryServerKey:      tracking.ResultEntryText("mail"),
			tracking.ResultDelayKey:               tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPDKey:          tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayCleanupKey:        tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayQmgrKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultRelayNameKey:           tracking.ResultEntryText("relay1.example.com"),
			tracking.ResultRelayIPKey:             tracking.ResultEntryBlob([]byte{192, 168, 0, 1}),
			tracking.ResultRelayPortKey:           tracking.ResultEntryInt64(25),
			tracking.ConnectionClientHostnameKey:  tracking.ResultEntryText("some.host.com"),
			tracking.ConnectionClientIPKey:        tracking.ResultEntryBlob([]byte{192, 168, 0, 2}),
			tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("A5"),
			tracking.ResultDSNKey:                 tracking.ResultEntryText("2.0.0"),
			tracking.ResultDeliveryLineChecksum:   tracking.ResultEntryInt64(206),
		}.Result())

		// a normal outbound message
		pub.Publish(tracking.MappedResult{
			tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.SentStatus)),
			tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Add(time.Hour * 3).Add(time.Minute * 5).Unix()),
			tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
			tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("sender.example.com"),
			tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("sender1"),
			tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("recipient.example.com"),
			tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("recipient1"),
			tracking.QueueMessageIDKey:            tracking.ResultEntryText("message_id_6"),
			tracking.ConnectionBeginKey:           tracking.ResultEntryInt64(baseTime.Add(time.Hour * 3).Add(time.Minute * 5).Unix()),
			tracking.QueueBeginKey:                tracking.ResultEntryInt64(baseTime.Add(time.Hour * 3).Add(time.Minute * 5).Unix()),
			tracking.QueueOriginalMessageSizeKey:  tracking.ResultEntryInt64(42),
			tracking.QueueProcessedMessageSizeKey: tracking.ResultEntryInt64(100),
			tracking.QueueNRCPTKey:                tracking.ResultEntryInt64(1),
			tracking.ResultDeliveryServerKey:      tracking.ResultEntryText("mail"),
			tracking.ResultDelayKey:               tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPDKey:          tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayCleanupKey:        tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelayQmgrKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultDelaySMTPKey:           tracking.ResultEntryFloat64(0.1),
			tracking.ResultRelayNameKey:           tracking.ResultEntryText("relay1.example.com"),
			tracking.ResultRelayIPKey:             tracking.ResultEntryBlob([]byte{192, 168, 0, 1}),
			tracking.ResultRelayPortKey:           tracking.ResultEntryInt64(25),
			tracking.ConnectionClientHostnameKey:  tracking.ResultEntryText("some.host.com"),
			tracking.ConnectionClientIPKey:        tracking.ResultEntryBlob([]byte{192, 168, 0, 2}),
			tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("A6"),
			tracking.ResultDSNKey:                 tracking.ResultEntryText("2.0.0"),
			tracking.ResultDeliveryLineChecksum:   tracking.ResultEntryInt64(207),
		}.Result())

		// delete two messages older than 6min, but not all yet
		db.Actions <- makeCleanAction(time.Minute*6, 2)

		// finally delete the remaining 3 messages older than 6min
		db.Actions <- makeCleanAction(time.Minute*6, 3)

		cancel()
		So(done(), ShouldBeNil)

		So(deliveryStatus(d, timeutil.MustParseTimeInterval("2020-01-01", "2020-12-31")), ShouldResemble, dashboard.Pairs{
			dashboard.Pair{Key: "sent", Value: 1},
			dashboard.Pair{Key: "bounced", Value: 1},
		})

		var (
			deliveryQueueCount  int
			expiredQueuesCount  int
			queueParentingCount int
			deliveriesCount     int
			messageIdsCount     int
			logLinesRefCount    int
		)

		ro, release := conn.RoConnPool.Acquire()
		defer release()

		// NOTE: those assertions are bad, as they check implementation details,
		// but accessing them via user-facing interface would be a huge pain in the neck.
		// That means those tests are quite brittle and will break in case we change the database
		// schema. But having them is useful nevertheless.
		So(ro.QueryRow(`select count(*) from deliveries`).Scan(&deliveriesCount), ShouldBeNil)
		So(deliveriesCount, ShouldEqual, 2)

		So(ro.QueryRow(`select count(*) from delivery_queue`).Scan(&deliveryQueueCount), ShouldBeNil)
		So(deliveryQueueCount, ShouldEqual, 2)

		So(ro.QueryRow(`select count(*) from expired_queues`).Scan(&expiredQueuesCount), ShouldBeNil)
		So(expiredQueuesCount, ShouldEqual, 0)

		So(ro.QueryRow(`select count(*) from queue_parenting`).Scan(&queueParentingCount), ShouldBeNil)
		So(queueParentingCount, ShouldEqual, 0)

		So(ro.QueryRow(`select count(*) from messageids`).Scan(&messageIdsCount), ShouldBeNil)
		So(messageIdsCount, ShouldEqual, 2)

		So(ro.QueryRow(`select count(*) from log_lines_ref`).Scan(&logLinesRefCount), ShouldBeNil)
		So(logLinesRefCount, ShouldEqual, 2)
	})
}

func TestReopenDatabase(t *testing.T) {
	// This is a different test as we need to reuse the same database directory over two runs,
	// ensuring that all the connections are closed between the runs!
	Convey("Test Reopening the database", t, func() {
		dir, removeDir := testutil.TempDir(t)
		defer removeDir()

		buildWs := func() (*DB, func() error, func(), tracking.ResultPublisher, dashboard.Dashboard, func()) {
			conn, err := dbconn.Open(path.Join(dir, "logs.db"), 5)
			So(err, ShouldBeNil)
			So(migrator.Run(conn.RwConn.DB, databaseName), ShouldBeNil)
			db, err := New(conn, &fakeMapping, NoFilters)
			So(err, ShouldBeNil)
			pub := db.ResultsPublisher()
			dashboard, err := dashboard.New(conn.RoConnPool)
			So(err, ShouldBeNil)
			done, cancel := runner.Run(db)
			return db, done, cancel, pub, dashboard, func() { So(conn.Close(), ShouldBeNil) }
		}

		Convey("Insert, reopen, insert", func() {
			func() {
				_, done, cancel, pub, _, closeConn := buildWs()
				defer closeConn()

				// this one is before the time interval
				pub.Publish(smtpStatusRecord(parser.DeferredStatus, testutil.MustParseTime(`1999-11-02 13:10:10 +0000`)))

				pub.Publish(smtpStatusRecord(parser.SentStatus, testutil.MustParseTime(`1999-12-02 13:10:10 +0000`)))
				cancel()
				So(done(), ShouldBeNil)
			}()

			// reopen workspace and add another log
			db, done, cancel, pub, dashboard, closeConn := buildWs()
			defer closeConn()

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
	})
}
