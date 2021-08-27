// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package topdomains

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
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

// TODO: move this to some place where it can be reused!
type mappedResult map[tracking.ResultEntryType]tracking.ResultEntry

// TODO: move this to some place where it can be reused!
func publishResult(pub tracking.ResultPublisher, mp mappedResult) {
	r := tracking.Result{}

	for k, v := range mp {
		r[k] = v
	}

	pub.Publish(r)
}

func publishResults(pub tracking.ResultPublisher, results ...mappedResult) {
	for _, r := range results {
		publishResult(pub, r)
	}
}

func TestReporter(t *testing.T) {
	Convey("Test Reporter", t, func() {
		dir, closeDatabases := testutil.TempDatabases(t)
		defer closeDatabases()

		baseTime := testutil.MustParseTime(`2000-01-01 10:00:00 +0000`)

		intelDb := dbconn.Db("intel")

		err := migrator.Run(intelDb.RwConn.DB, "intel")
		So(err, ShouldBeNil)

		clock := &timeutil.FakeClock{Time: baseTime}

		// In the tests, lightmeter is running on `alice`, and the remote servers are `bob`

		dispatcher := &fakeDispatcher{}

		delivery, err := deliverydb.New(dir, &domainmapping.Mapper{})
		So(err, ShouldBeNil)

		defer delivery.Close()

		done, cancel := delivery.Run()

		pub := delivery.ResultsPublisher()

		// First report: sending and receiving messages
		publishResults(pub,
			mappedResult{ // very old message
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("old_alice"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("alice1.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("bob"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("bob1.com"),
				tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.SentStatus)),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
				tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(timeutil.MustParseTime(`1998-07-01 11:11:11 +0000`).Unix()),
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
			},
			mappedResult{ // not so old message
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("bob"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("bob1.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("alice"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("alice2.com"),
				tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.SentStatus)),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionIncoming)),
				tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Add(1 * time.Minute).Unix()),
				tracking.QueueMessageIDKey:            tracking.ResultEntryText("msgid2.1"),
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
				tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("A21"),
			},
		)

		// Second report: sending messages only
		publishResults(pub,
			mappedResult{ // recent message
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("alice"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("alice2.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("bob"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("bob2.com"),
				tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.SentStatus)),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
				tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Add(2 * time.Hour).Add(time.Minute * 5).Unix()),
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
				tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("A3"),
			},
			mappedResult{ // recent message
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("alice"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("alice1.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("bob"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("bob1.com"),
				tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.SentStatus)),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
				tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Add(2 * time.Hour).Add(2 * time.Minute).Unix()),
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
				tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("A4"),
			},
			mappedResult{ // recent message
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("alice"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("alice2.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("bob"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("bob1.com"),
				tracking.ResultStatusKey:              tracking.ResultEntryInt64(int64(parser.SentStatus)),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
				tracking.ResultDeliveryTimeKey:        tracking.ResultEntryInt64(baseTime.Add(2 * time.Hour).Add(10 * time.Minute).Unix()),
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
				tracking.QueueDeliveryNameKey:         tracking.ResultEntryText("A5"),
			},
		)

		cancel()
		So(done(), ShouldBeNil)

		reporter := NewReporter(delivery.ConnPool())

		defer func() {
			So(reporter.Close(), ShouldBeNil)
		}()

		reporters := collector.Reporters{reporter}

		err = intelDb.RwConn.Tx(func(tx *sql.Tx) error {
			// first execution, nothing happens
			err := reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			// generate one report
			clock.Sleep(1 * time.Hour)
			err = reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			// after two hours, generate another one
			clock.Sleep(2 * time.Hour)
			err = reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			// finally, collect it
			err = collector.TryToDispatchReports(clock, dispatcher)
			So(err, ShouldBeNil)

			return nil
		})

		So(err, ShouldBeNil)

		So(dispatcher.reports, ShouldResemble, []collector.Report{
			{
				Interval: timeutil.TimeInterval{From: time.Time{}, To: baseTime.Add(3 * time.Hour)},
				Content: []collector.ReportEntry{
					{
						Time: baseTime.Add(1 * time.Hour),
						ID:   "top_domains",
						Payload: map[string]interface{}{
							"senders":    []interface{}{"alice1.com"},
							"recipients": []interface{}{"alice2.com"},
						},
					},
					{
						Time: baseTime.Add(3 * time.Hour),
						ID:   "top_domains",
						Payload: map[string]interface{}{
							"senders":    []interface{}{"alice2.com", "alice1.com"},
							"recipients": []interface{}{},
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
