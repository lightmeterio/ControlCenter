// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package connectionstats

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/connectionstats"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	_ "gitlab.com/lightmeter/controlcenter/intel/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/postfixutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestReporters(t *testing.T) {
	Convey("Test Reporters", t, func() {
		conn, closeConn := testutil.TempDBConnectionMigrated(t, "connections")
		defer closeConn()
		options := connectionstats.Options{RetentionDuration: (time.Hour * 24 * 30 * 3)}
		stats, err := connectionstats.New(conn, options)
		So(err, ShouldBeNil)

		pub := stats.Publisher()

		done, cancel := runner.Run(stats)

		postfixutil.ReadFromTestReader(strings.NewReader(`
Jan  1 00:00:30 mail postfix/smtpd[26098]: disconnect from unknown[11.22.33.44] ehlo=1 auth=8/14 mail=1 rcpt=0/1 data=0/1 rset=1 commands=3/19
Jan  1 00:04:00 mail postfix/smtpd[9715]: disconnect from localhost[127.0.0.1] ehlo=1 mail=1 rcpt=1 data=1 quit=1 commands=5
Jan  1 00:06:00 mail postfix/smtpd[26099]: disconnect from unknown[44.55.66.77] ehlo=1 auth=7/8 rset=1 quit=1 commands=10/11
Jan  1 00:13:00 mail postfix/smtpd[123456]: disconnect from unknown[12.34.56.78] ehlo=1 auth=0/1 commands=1/2
Jan  1 00:14:00 mail postfix/smtpd[123456]: disconnect from unknown[12.34.56.78] unknown=1 bdat=1 helo=1 starttls=1 noop=1 vrfy=1 etrn=1 xclient=1 xforward=1 commands=9
		`), pub, 2020, &timeutil.FakeClock{Time: timeutil.MustParseTime(`2020-01-01 10:00:00 +0000`)})

		cancel()
		So(done(), ShouldBeNil)

		baseTime := timeutil.MustParseTime(`2020-01-01 00:00:00 +0000`)

		intelDb, clear := testutil.TempDBConnectionMigrated(t, "intel-collector")
		defer clear()

		reporter := NewReporter(conn.RoConnPool)

		clock := &timeutil.FakeClock{Time: baseTime}

		dispatcher := &fakeDispatcher{}

		err = intelDb.RwConn.Tx(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
			clock.Sleep(10 * time.Minute)
			err := reporter.Step(tx, clock)
			So(err, ShouldBeNil)

			clock.Sleep(10 * time.Minute)
			err = reporter.Step(tx, clock)
			So(err, ShouldBeNil)

			// on this execution, no new connections have been done, so nothing is reported
			clock.Sleep(10 * time.Minute)
			err = reporter.Step(tx, clock)
			So(err, ShouldBeNil)

			// data collected
			err = collector.TryToDispatchReports(tx, clock, dispatcher)
			So(err, ShouldBeNil)

			return nil
		})

		So(err, ShouldBeNil)

		So(dispatcher.reports, ShouldResemble, []collector.Report{
			{
				Interval: timeutil.TimeInterval{From: time.Time{}, To: baseTime.Add(30 * time.Minute)},
				Content: []collector.ReportEntry{
					{
						Time: baseTime.Add(10 * time.Minute),
						ID:   "connection_stats_with_auth",
						Payload: map[string]interface{}{
							"time_interval": map[string]interface{}{
								"from": mustEncodeTimeJson(baseTime),
								"to":   mustEncodeTimeJson(baseTime.Add(10 * time.Minute)),
							},
							"entries": []interface{}{
								map[string]interface{}{
									"time": mustEncodeTimeJson(timeutil.MustParseTime(`2020-01-01 00:00:30 +0000`)),
									"ip":   "11.22.33.44",
									"commands": map[string]interface{}{
										"ehlo": map[string]interface{}{
											"success": float64(1),
											"total":   float64(1),
										},
										"auth": map[string]interface{}{
											"success": float64(8),
											"total":   float64(14),
										},
										"mail": map[string]interface{}{
											"success": float64(1),
											"total":   float64(1),
										},
										"rcpt": map[string]interface{}{
											"success": float64(0),
											"total":   float64(1),
										},
										"data": map[string]interface{}{
											"success": float64(0),
											"total":   float64(1),
										},
										"rset": map[string]interface{}{
											"success": float64(1),
											"total":   float64(1),
										},
									},
								},
								// NOTE: Notice that we skip the lines that do not try to authenticate. They are useless to us at the moment
								map[string]interface{}{
									"time": mustEncodeTimeJson(timeutil.MustParseTime(`2020-01-01 00:06:00 +0000`)),
									"ip":   "44.55.66.77",
									"commands": map[string]interface{}{
										"ehlo": map[string]interface{}{
											"success": float64(1),
											"total":   float64(1),
										},
										"auth": map[string]interface{}{
											"success": float64(7),
											"total":   float64(8),
										},
										"rset": map[string]interface{}{
											"success": float64(1),
											"total":   float64(1),
										},
										"quit": map[string]interface{}{
											"success": float64(1),
											"total":   float64(1),
										},
									},
								},
							},
						},
					},
					{
						Time: baseTime.Add(20 * time.Minute),
						ID:   "connection_stats_with_auth",
						Payload: map[string]interface{}{
							"time_interval": map[string]interface{}{
								"from": mustEncodeTimeJson(baseTime.Add(10 * time.Minute)),
								"to":   mustEncodeTimeJson(baseTime.Add(20 * time.Minute)),
							},
							"entries": []interface{}{
								map[string]interface{}{
									"time": mustEncodeTimeJson(timeutil.MustParseTime(`2020-01-01 00:13:00 +0000`)),
									"ip":   "12.34.56.78",
									"commands": map[string]interface{}{
										"ehlo": map[string]interface{}{
											"success": float64(1),
											"total":   float64(1),
										},
										"auth": map[string]interface{}{
											"success": float64(0),
											"total":   float64(1),
										},
									},
								},
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
