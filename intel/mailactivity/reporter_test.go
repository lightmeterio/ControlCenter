// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package mailactivity

import (
	"bytes"
	"database/sql"
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/deliverydb"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	_ "gitlab.com/lightmeter/controlcenter/intel/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

type mappedResult = tracking.MappedResult

func publishResult(pub tracking.ResultPublisher, mp mappedResult) {
	pub.Publish(mp.Result())
}

func TestReporters(t *testing.T) {
	Convey("Test Reporters", t, func() {
		db, clear := testutil.TempDBConnectionMigrated(t, "logs")
		defer clear()

		baseTime := testutil.MustParseTime(`2000-01-01 10:00:00 +0000`)

		// fill delivery database with some values
		{
			delivery, err := deliverydb.New(db, &domainmapping.Mapper{})
			So(err, ShouldBeNil)

			done, cancel := runner.Run(delivery)

			pub := delivery.ResultsPublisher()

			// a sent message in the first interval
			publishResult(pub, mappedResult{
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("sender"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("sender.example.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("recipient"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("recipient.example.com"),
				tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.SentStatus)),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
				tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Unix()),
				tracking.QueueMessageIDKey:            tracking.ResultEntryText("msgid1"),
				tracking.QueueOriginalMessageSizeKey:  tracking.ResultEntryInt64(35),
				tracking.QueueProcessedMessageSizeKey: tracking.ResultEntryInt64(42),
				tracking.QueueNRCPTKey:                tracking.ResultEntryInt64(0),
				tracking.ResultDeliveryServerKey:      tracking.ResultEntryText("mail"),
				tracking.ResultDelayKey:               tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelaySMTPDKey:          tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelayCleanupKey:        tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelayQmgrKey:           tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelaySMTPKey:           tracking.ResultEntryFloat64(0.0),
				tracking.ResultDSNKey:                 tracking.ResultEntryText("2.0.0"),
				tracking.QueueBeginKey:                tracking.ResultEntryInt64(0),
				tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("A1"),
			})

			// another sent message in the first interval
			publishResult(pub, mappedResult{
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("sender"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("sender.example.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("recipient"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("recipient.example.com"),
				tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.SentStatus)),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
				tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Add(2 * time.Minute).Unix()),
				tracking.QueueMessageIDKey:            tracking.ResultEntryText("msgid2"),
				tracking.QueueOriginalMessageSizeKey:  tracking.ResultEntryInt64(35),
				tracking.QueueProcessedMessageSizeKey: tracking.ResultEntryInt64(42),
				tracking.QueueNRCPTKey:                tracking.ResultEntryInt64(0),
				tracking.ResultDeliveryServerKey:      tracking.ResultEntryText("mail"),
				tracking.ResultDelayKey:               tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelaySMTPDKey:          tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelayCleanupKey:        tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelayQmgrKey:           tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelaySMTPKey:           tracking.ResultEntryFloat64(0.0),
				tracking.ResultDSNKey:                 tracking.ResultEntryText("2.0.0"),
				tracking.QueueBeginKey:                tracking.ResultEntryInt64(0),
				tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("A2"),
			})

			// a deferred message in the first interval
			publishResult(pub, mappedResult{
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("sender"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("sender.example.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("recipient"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("recipient.example.com"),
				tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.DeferredStatus)),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
				tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Add(2 * time.Minute).Unix()),
				tracking.QueueMessageIDKey:            tracking.ResultEntryText("msgid3"),
				tracking.QueueOriginalMessageSizeKey:  tracking.ResultEntryInt64(35),
				tracking.QueueProcessedMessageSizeKey: tracking.ResultEntryInt64(42),
				tracking.QueueNRCPTKey:                tracking.ResultEntryInt64(0),
				tracking.ResultDeliveryServerKey:      tracking.ResultEntryText("mail"),
				tracking.ResultDelayKey:               tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelaySMTPDKey:          tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelayCleanupKey:        tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelayQmgrKey:           tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelaySMTPKey:           tracking.ResultEntryFloat64(0.0),
				tracking.ResultDSNKey:                 tracking.ResultEntryText("2.0.0"),
				tracking.QueueBeginKey:                tracking.ResultEntryInt64(0),
				tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("B2"),
			})

			// an expired message in the first interval
			publishResult(pub, mappedResult{
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("sender"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("sender.example.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("recipient"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("recipient.example.com"),
				tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.ExpiredStatus)),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
				tracking.MessageExpiredTime:           tracking.ResultEntryInt64(baseTime.Add(2 * time.Minute).Add(2 * time.Second).Unix()),
				tracking.QueueMessageIDKey:            tracking.ResultEntryText("msgid3"),
				tracking.QueueOriginalMessageSizeKey:  tracking.ResultEntryInt64(35),
				tracking.QueueProcessedMessageSizeKey: tracking.ResultEntryInt64(42),
				tracking.QueueNRCPTKey:                tracking.ResultEntryInt64(0),
				tracking.ResultDeliveryServerKey:      tracking.ResultEntryText("mail"),
				tracking.ResultDelayKey:               tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelaySMTPDKey:          tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelayCleanupKey:        tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelayQmgrKey:           tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelaySMTPKey:           tracking.ResultEntryFloat64(0.0),
				tracking.ResultDSNKey:                 tracking.ResultEntryText("2.0.0"),
				tracking.QueueBeginKey:                tracking.ResultEntryInt64(0),
				tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("B2"),
			})

			// an inbound message in the second interval
			publishResult(pub, mappedResult{
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("sender"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("sender.example.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("recipient"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("recipient.example.com"),
				tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.SentStatus)),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionIncoming)),
				tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Add(13 * time.Minute).Unix()),
				tracking.QueueMessageIDKey:            tracking.ResultEntryText("msgid4"),
				tracking.QueueOriginalMessageSizeKey:  tracking.ResultEntryInt64(35),
				tracking.QueueProcessedMessageSizeKey: tracking.ResultEntryInt64(42),
				tracking.QueueNRCPTKey:                tracking.ResultEntryInt64(0),
				tracking.ResultDeliveryServerKey:      tracking.ResultEntryText("mail"),
				tracking.ResultDelayKey:               tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelaySMTPDKey:          tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelayCleanupKey:        tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelayQmgrKey:           tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelaySMTPKey:           tracking.ResultEntryFloat64(0.0),
				tracking.ResultDSNKey:                 tracking.ResultEntryText("2.0.0"),
				tracking.QueueBeginKey:                tracking.ResultEntryInt64(0),
				tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("C1"),
			})

			// a bounced message in the second interval
			publishResult(pub, mappedResult{
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("sender"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("sender.example.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("recipient"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("recipient.example.com"),
				tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.BouncedStatus)),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
				tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Add(17 * time.Minute).Unix()),
				tracking.QueueMessageIDKey:            tracking.ResultEntryText("msgid5"),
				tracking.QueueOriginalMessageSizeKey:  tracking.ResultEntryInt64(35),
				tracking.QueueProcessedMessageSizeKey: tracking.ResultEntryInt64(42),
				tracking.QueueNRCPTKey:                tracking.ResultEntryInt64(0),
				tracking.ResultDeliveryServerKey:      tracking.ResultEntryText("mail"),
				tracking.ResultDelayKey:               tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelaySMTPDKey:          tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelayCleanupKey:        tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelayQmgrKey:           tracking.ResultEntryFloat64(0.0),
				tracking.ResultDelaySMTPKey:           tracking.ResultEntryFloat64(0.0),
				tracking.ResultDSNKey:                 tracking.ResultEntryText("2.0.0"),
				tracking.QueueBeginKey:                tracking.ResultEntryInt64(0),
				tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("C1"),
			})

			cancel()
			So(done(), ShouldBeNil)
		}

		intelDb, clear := testutil.TempDBConnectionMigrated(t, "intel-collector")
		defer clear()

		reporter := NewReporter(db.RoConnPool)

		clock := &timeutil.FakeClock{Time: baseTime}

		dispatcher := &fakeDispatcher{}

		err := intelDb.RwConn.Tx(func(tx *sql.Tx) error {
			clock.Sleep(10 * time.Minute)
			err := reporter.Step(tx, clock)
			So(err, ShouldBeNil)

			err = collector.TryToDispatchReports(tx, clock, dispatcher)
			So(err, ShouldBeNil)

			clock.Sleep(10 * time.Minute)
			err = reporter.Step(tx, clock)
			So(err, ShouldBeNil)

			err = collector.TryToDispatchReports(tx, clock, dispatcher)
			So(err, ShouldBeNil)

			// no activity here, therefore no reports sent
			clock.Sleep(10 * time.Minute)
			err = reporter.Step(tx, clock)
			So(err, ShouldBeNil)

			err = collector.TryToDispatchReports(tx, clock, dispatcher)
			So(err, ShouldBeNil)

			return nil
		})

		So(err, ShouldBeNil)

		So(dispatcher.reports, ShouldResemble, []collector.Report{
			{
				Interval: timeutil.TimeInterval{From: time.Time{}, To: baseTime.Add(10 * time.Minute)},
				Content: []collector.ReportEntry{
					{
						Time: baseTime.Add(10 * time.Minute),
						ID:   "mail_activity",
						Payload: map[string]interface{}{
							"time_interval": map[string]interface{}{
								"from": mustEncodeTimeJson(baseTime),
								"to":   mustEncodeTimeJson(baseTime.Add(10 * time.Minute)),
							},
							"sent_messages":     float64(2),
							"deferred_messages": float64(1),
							"bounced_messages":  float64(0),
							"received_messages": float64(0),
							"expired_messages":  float64(1),
						},
					},
				},
			},
			{
				Interval: timeutil.TimeInterval{From: baseTime.Add(10 * time.Minute), To: baseTime.Add(20 * time.Minute)},
				Content: []collector.ReportEntry{
					{
						Time: baseTime.Add(20 * time.Minute),
						ID:   "mail_activity",
						Payload: map[string]interface{}{
							"time_interval": map[string]interface{}{
								"from": mustEncodeTimeJson(baseTime.Add(10 * time.Minute)),
								"to":   mustEncodeTimeJson(baseTime.Add(20 * time.Minute)),
							},
							"sent_messages":     float64(0),
							"deferred_messages": float64(0),
							"bounced_messages":  float64(1),
							"received_messages": float64(1),
							"expired_messages":  float64(0),
						},
					},
				},
			},
		})
	})
}

type fakeDispatcher struct {
	reports []collector.Report
}

func (f *fakeDispatcher) Dispatch(r collector.Report) error {
	f.reports = append(f.reports, r)
	return nil
}

func mustEncodeTimeJson(v time.Time) string {
	s, err := json.Marshal(v)
	So(err, ShouldBeNil)

	// remove quotes, because reasons.
	return string(bytes.Trim(s, `"`))
}
