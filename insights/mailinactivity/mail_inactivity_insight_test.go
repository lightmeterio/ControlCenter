// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package mailinactivity

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/deliverydb"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/notification"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"testing"
	"time"
)

var (
	dummyContext = context.Background()
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestMailInactivityDetectorInsight(t *testing.T) {
	Convey("Test Insights Generator", t, func() {
		conn, closeConn := testutil.TempDBConnectionMigrated(t, "logs")
		defer closeConn()

		buildWs := func() (*deliverydb.DB, func() error, func(), tracking.ResultPublisher) {
			db, err := deliverydb.New(conn, &domainmapping.DefaultMapping)
			So(err, ShouldBeNil)
			done, cancel := runner.Run(db)
			pub := db.ResultsPublisher()

			So(err, ShouldBeNil)

			return db, done, cancel, pub
		}

		_, done, cancel, pub := buildWs()

		accessor, clear := insighttestsutil.NewFakeAccessor(t)
		defer clear()

		connPair := accessor.ConnPair

		lookupRange := time.Hour * 24

		detector := NewDetector(accessor, core.Options{
			"logsConnPool":   conn.RoConnPool,
			"mailinactivity": Options{LookupRange: lookupRange, MinTimeGenerationInterval: time.Hour * 8},
		})

		cycle := func(c *insighttestsutil.FakeClock) {
			tx, err := connPair.RwConn.Begin()
			So(err, ShouldBeNil)
			So(detector.Step(c, tx), ShouldBeNil)
			So(tx.Commit(), ShouldBeNil)
		}

		Convey("Don't generate an insight when application starts with no log data", func() {
			cancel()
			So(done(), ShouldBeNil)

			clock := &insighttestsutil.FakeClock{Time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange)}

			// do not generate insight
			cycle(clock)

			So(accessor.Insights, ShouldResemble, []int64{})
		})

		Convey("Server stays inactive for one day", func() {
			baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)

			clock := &insighttestsutil.FakeClock{Time: baseTime.Add(lookupRange)}

			// There is some inbound activity in the first 8 hours
			result1 := tracking.Result{}
			result1[tracking.QueueSenderLocalPartKey] = tracking.ResultEntryText("sender")
			result1[tracking.QueueSenderDomainPartKey] = tracking.ResultEntryText("sender.example.com")
			result1[tracking.ResultRecipientLocalPartKey] = tracking.ResultEntryText("recipient")
			result1[tracking.ResultRecipientDomainPartKey] = tracking.ResultEntryText("recipient.example.com")
			result1[tracking.ResultStatusKey] = tracking.ResultEntryInt64(int64(parser.SentStatus))
			result1[tracking.ResultMessageDirectionKey] = tracking.ResultEntryInt64(int64(tracking.MessageDirectionIncoming))
			result1[tracking.ResultDeliveryTimeKey] = tracking.ResultEntryInt64(baseTime.Add(1 * time.Hour).Unix())
			result1[tracking.QueueMessageIDKey] = tracking.ResultEntryText("msgid1")
			result1[tracking.QueueOriginalMessageSizeKey] = tracking.ResultEntryInt64(35)
			result1[tracking.QueueProcessedMessageSizeKey] = tracking.ResultEntryInt64(42)
			result1[tracking.QueueNRCPTKey] = tracking.ResultEntryInt64(0)
			result1[tracking.ResultDeliveryServerKey] = tracking.ResultEntryText("mail")
			result1[tracking.ResultDelayKey] = tracking.ResultEntryFloat64(0.0)
			result1[tracking.ResultDelaySMTPDKey] = tracking.ResultEntryFloat64(0.0)
			result1[tracking.ResultDelayCleanupKey] = tracking.ResultEntryFloat64(0.0)
			result1[tracking.ResultDelayQmgrKey] = tracking.ResultEntryFloat64(0.0)
			result1[tracking.ResultDelaySMTPKey] = tracking.ResultEntryFloat64(0.0)
			result1[tracking.ResultDSNKey] = tracking.ResultEntryText("2.0.0")
			result1[tracking.QueueBeginKey] = tracking.ResultEntryInt64(0)
			result1[tracking.QueueDeliveryNameKey] = tracking.ResultEntryText("A1")
			pub.Publish(result1)

			// No activity in the next 8 hours...

			// Then there is some outbound activity in the final 8 hours
			result2 := tracking.Result{}
			result2[tracking.QueueSenderLocalPartKey] = tracking.ResultEntryText("sender")
			result2[tracking.QueueSenderDomainPartKey] = tracking.ResultEntryText("sender.example.com")
			result2[tracking.ResultRecipientLocalPartKey] = tracking.ResultEntryText("recipient")
			result2[tracking.ResultRecipientDomainPartKey] = tracking.ResultEntryText("recipient.example.com")
			result2[tracking.ResultStatusKey] = tracking.ResultEntryInt64(int64(parser.SentStatus))
			result2[tracking.ResultMessageDirectionKey] = tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound))
			result2[tracking.ResultDeliveryTimeKey] = tracking.ResultEntryInt64(baseTime.Add(lookupRange).Add(10 * time.Hour).Unix())
			result2[tracking.QueueMessageIDKey] = tracking.ResultEntryText("msgid2")
			result2[tracking.QueueOriginalMessageSizeKey] = tracking.ResultEntryInt64(35)
			result2[tracking.QueueProcessedMessageSizeKey] = tracking.ResultEntryInt64(42)
			result2[tracking.QueueNRCPTKey] = tracking.ResultEntryInt64(0)
			result2[tracking.ResultDeliveryServerKey] = tracking.ResultEntryText("mail")
			result2[tracking.ResultDelayKey] = tracking.ResultEntryFloat64(0.0)
			result2[tracking.ResultDelaySMTPDKey] = tracking.ResultEntryFloat64(0.0)
			result2[tracking.ResultDelayCleanupKey] = tracking.ResultEntryFloat64(0.0)
			result2[tracking.ResultDelayQmgrKey] = tracking.ResultEntryFloat64(0.0)
			result2[tracking.ResultDelaySMTPKey] = tracking.ResultEntryFloat64(0.0)
			result2[tracking.ResultDSNKey] = tracking.ResultEntryText("2.0.0")
			result2[tracking.QueueBeginKey] = tracking.ResultEntryInt64(0)
			result2[tracking.QueueDeliveryNameKey] = tracking.ResultEntryText("A2")
			pub.Publish(result2)

			cancel()
			So(done(), ShouldBeNil)

			// do not generate insight
			cycle(clock)

			// Generate insight
			clock.Sleep(time.Hour * 8)
			cycle(clock)

			// do not generate insight
			clock.Sleep(time.Hour * 8)
			cycle(clock)

			So(accessor.Insights, ShouldResemble, []int64{1})

			So(len(accessor.Insights), ShouldEqual, 1)

			insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: timeutil.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange).Add(lookupRange),
			}}, clock)

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 1)

			So(insights[0].ID(), ShouldEqual, 1)
			So(insights[0].ContentType(), ShouldEqual, ContentType)
			So(insights[0].Rating(), ShouldEqual, core.OkRating)
			So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange).Add(time.Hour*8))
			So(insights[0].Content(), ShouldResemble, &Content{
				Interval: timeutil.TimeInterval{
					From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 8),
					To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange).Add(time.Hour * 8),
				}})
		})
	})
}

func TestDescriptionFormatting(t *testing.T) {
	Convey("Description Formatting", t, func() {
		n := notification.Notification{
			ID: 1,
			Content: Content{
				Interval: timeutil.TimeInterval{From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), To: testutil.MustParseTime(`2000-01-01 10:00:00 +0000`)},
			},
		}

		m, err := notificationCore.TranslateNotification(n, translator.DummyTranslator{})
		So(err, ShouldBeNil)
		So(m, ShouldResemble, notificationCore.Message{
			Title:       "Mail Inactivity",
			Description: "No emails were sent or received between 2000-01-01 00:00:00 +0000 UTC and 2000-01-01 10:00:00 +0000 UTC",
			Metadata:    map[string]string{},
		})
	})
}
