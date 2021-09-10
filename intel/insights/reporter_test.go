// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/insights"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/notification"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
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

type fakeContent struct {
	T string `json:"title"`
	D string `json:"description"`
}

func (c fakeContent) Title() notificationCore.ContentComponent {
	return fakeContentComponent(c.T)
}

func (c fakeContent) Description() notificationCore.ContentComponent {
	return fakeContentComponent(c.D)
}

func (c fakeContent) Metadata() notificationCore.ContentMetadata {
	return nil
}

type fakeContentComponent string

func (c fakeContentComponent) String() string {
	return string(c)
}

func (c fakeContentComponent) TplString() string {
	return string(c)
}

func (c fakeContentComponent) Args() []interface{} {
	return nil
}

func init() {
	core.RegisterContentType("fake_type_1", 100, core.DefaultContentTypeDecoder(&fakeContent{}))
	core.RegisterContentType("fake_type_2", 200, core.DefaultContentTypeDecoder(&fakeContent{}))
	core.RegisterContentType("fake_type_3", 300, core.DefaultContentTypeDecoder(&fakeContent{}))
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

func TestReporter(t *testing.T) {
	Convey("Test Insights Count Reporter", t, func() {
		conn, closeConn := testutil.TempDBConnectionMigrated(t, "insights")
		defer closeConn()

		accessor, err := insights.NewAccessor(conn)
		So(err, ShouldBeNil)

		noAdditionalActions := func([]core.Detector, dbconn.RwConn, core.Clock) error { return nil }

		intelDb, clear := testutil.TempDBConnectionMigrated(t, "intel-collector")
		defer clear()

		e, err := insights.NewCustomEngine(
			accessor,
			&notification.Center{},
			core.Options{},
			insights.NoDetectors,
			noAdditionalActions)

		So(err, ShouldBeNil)

		announcer.Skip(e.ImportAnnouncer())

		baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)
		fifteenMin := 15 * time.Minute
		baseTimePlus15m := baseTime.Add(fifteenMin)
		baseTimePlus30m := baseTime.Add(30 * time.Minute)

		// generate 2 same-type reports at baseTime
		for i := range [2]int{} {
			e.GenerateInsight(dummyContext, core.InsightProperties{
				Time:        baseTime.Add(time.Duration(i) * time.Second),
				ContentType: "fake_type_1",
				Category:    core.LocalCategory,
			})
		}

		// generate 2 different-type reports at baseTime + 15min
		e.GenerateInsight(dummyContext, core.InsightProperties{
			Time:        baseTimePlus15m.Add(5 * time.Minute),
			ContentType: "fake_type_2",
			Category:    core.LocalCategory,
		})
		e.GenerateInsight(dummyContext, core.InsightProperties{
			Time:        baseTimePlus15m.Add(10 * time.Minute),
			ContentType: "fake_type_3",
			Category:    core.LocalCategory,
		})

		// generate a report with non-"local" category, which shouldn't be counted
		e.GenerateInsight(dummyContext, core.InsightProperties{
			Time:        baseTimePlus15m.Add(12 * time.Minute),
			ContentType: "fake_type_3",
			Category:    core.IntelCategory,
		})

		doneInsights, cancelInsights := e.Run()

		time.Sleep(100 * time.Millisecond)

		cancelInsights()
		So(doneInsights(), ShouldBeNil)

		defer func() {
			So(e.Close(), ShouldBeNil)
		}()

		clock := &insighttestsutil.FakeClock{Time: baseTime}

		So(err, ShouldBeNil)

		reporter := NewReporter(e.Fetcher())

		dispatcher := &fakeDispatcher{}

		err = intelDb.RwConn.Tx(func(tx *sql.Tx) error {
			clock.Sleep(fifteenMin)
			err := reporter.Step(tx, clock)
			So(err, ShouldBeNil)

			err = collector.TryToDispatchReports(tx, clock, dispatcher)
			So(err, ShouldBeNil)

			clock.Sleep(fifteenMin)
			err = reporter.Step(tx, clock)
			So(err, ShouldBeNil)

			err = collector.TryToDispatchReports(tx, clock, dispatcher)
			So(err, ShouldBeNil)

			// spend some time, with no insights being created. Those should not be reported
			clock.Sleep(fifteenMin)
			err = reporter.Step(tx, clock)
			So(err, ShouldBeNil)

			err = collector.TryToDispatchReports(tx, clock, dispatcher)
			So(err, ShouldBeNil)

			return nil
		})

		So(err, ShouldBeNil)

		So(dispatcher.reports, ShouldResemble, []collector.Report{
			{
				Interval: timeutil.TimeInterval{From: time.Time{}, To: baseTimePlus15m},
				Content: []collector.ReportEntry{
					{
						Time: baseTimePlus15m,
						ID:   "insights_count",
						Payload: map[string]interface{}{
							"time_interval": map[string]interface{}{
								"from": mustEncodeTimeJson(baseTime),
								"to":   mustEncodeTimeJson(baseTimePlus15m),
							},
							"insights": map[string]interface{}{
								"fake_type_1": float64(2),
							},
						},
					},
				},
			},
			{
				Interval: timeutil.TimeInterval{From: baseTimePlus15m, To: baseTimePlus30m},
				Content: []collector.ReportEntry{
					{
						Time: baseTimePlus30m,
						ID:   "insights_count",
						Payload: map[string]interface{}{
							"time_interval": map[string]interface{}{
								"from": mustEncodeTimeJson(baseTimePlus15m),
								"to":   mustEncodeTimeJson(baseTimePlus30m),
							},
							"insights": map[string]interface{}{
								"fake_type_2": float64(1),
								"fake_type_3": float64(1),
							},
						},
					},
				},
			},
		})
	})
}
