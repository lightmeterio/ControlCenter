// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package logslinecount

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	_ "gitlab.com/lightmeter/controlcenter/intel/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/util/postfixutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestReporter(t *testing.T) {
	Convey("Test Reporter", t, func() {
		baseTime := testutil.MustParseTime(`2000-01-01 10:00:00 +0000`)

		pub := NewPublisher()

		reporter := NewReporter(pub)

		intelDb, clear := testutil.TempDBConnectionMigrated(t, "intel-collector")
		defer clear()

		clock := &timeutil.FakeClock{Time: baseTime}

		dispatcher := &fakeDispatcher{}

		err := intelDb.RwConn.Tx(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
			// fill publsher with some values
			postfixutil.ReadFromTestFile("../../test_files/postfix_logs/individual_files/1_bounce_simple.log", pub, 2020)

			err := reporter.Step(tx, clock)
			So(err, ShouldBeNil)

			clock.Sleep(10 * time.Minute)

			// reads from a different file, and the values from the previous file must have been erased prior to it
			postfixutil.ReadFromTestFile("../../test_files/postfix_logs/individual_files/2_multiple_recipients_some_bounces.log", pub, 2020)

			err = reporter.Step(tx, clock)
			So(err, ShouldBeNil)

			err = collector.TryToDispatchReports(tx, clock, dispatcher)
			So(err, ShouldBeNil)

			// no new lines read, therefore no reports are sent
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
						Time: baseTime,
						ID:   "log_lines_count",
						Payload: map[string]interface{}{
							"time_interval": map[string]interface{}{
								"from": mustEncodeTimeJson(baseTime.Add(-10 * time.Minute)),
								"to":   mustEncodeTimeJson(baseTime),
							},
							"counters": map[string]interface{}{
								"amavis":                         map[string]interface{}{"supported": float64(0), "unsupported": float64(1)},
								"dovecot":                        map[string]interface{}{"supported": float64(0), "unsupported": float64(4)},
								"opendkim":                       map[string]interface{}{"supported": float64(0), "unsupported": float64(1)},
								"postfix/bounce":                 map[string]interface{}{"supported": float64(1), "unsupported": float64(0)},
								"postfix/cleanup":                map[string]interface{}{"supported": float64(2), "unsupported": float64(0)},
								"postfix/lmtp":                   map[string]interface{}{"supported": float64(1), "unsupported": float64(0)},
								"postfix/qmgr":                   map[string]interface{}{"supported": float64(6), "unsupported": float64(0)},
								"postfix/sender-cleanup/cleanup": map[string]interface{}{"supported": float64(1), "unsupported": float64(1)},
								"postfix/smtp":                   map[string]interface{}{"supported": float64(2), "unsupported": float64(1)},
								"postfix/smtpd":                  map[string]interface{}{"supported": float64(3), "unsupported": float64(0)},
								"postfix/submission/smtpd":       map[string]interface{}{"supported": float64(3), "unsupported": float64(1)},
							},
						},
					},
					{
						Time: baseTime.Add(10 * time.Minute),
						ID:   "log_lines_count",
						Payload: map[string]interface{}{
							"time_interval": map[string]interface{}{
								"from": mustEncodeTimeJson(baseTime),
								"to":   mustEncodeTimeJson(baseTime.Add(10 * time.Minute)),
							},
							"counters": map[string]interface{}{
								"postfix/bounce":                 map[string]interface{}{"supported": float64(1), "unsupported": float64(0)},
								"postfix/cleanup":                map[string]interface{}{"supported": float64(2), "unsupported": float64(0)},
								"postfix/lmtp":                   map[string]interface{}{"supported": float64(1), "unsupported": float64(0)},
								"postfix/qmgr":                   map[string]interface{}{"supported": float64(6), "unsupported": float64(0)},
								"postfix/sender-cleanup/cleanup": map[string]interface{}{"supported": float64(1), "unsupported": float64(1)},
								"postfix/smtp":                   map[string]interface{}{"supported": float64(10), "unsupported": float64(4)},
								"postfix/smtpd":                  map[string]interface{}{"supported": float64(3), "unsupported": float64(0)},
								"postfix/submission/smtpd":       map[string]interface{}{"supported": float64(3), "unsupported": float64(1)},
							},
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
