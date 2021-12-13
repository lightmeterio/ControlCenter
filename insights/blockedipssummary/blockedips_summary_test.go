// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package blockedipssummary

import (
	"context"
	"database/sql"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/blockedips"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	intelblockedips "gitlab.com/lightmeter/controlcenter/intel/blockedips"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
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

// this is used to make the tests easier, as we need to
type composedDetector struct {
	blockedIPsDetector core.Detector
	summaryDetector    core.Detector
}

func (d *composedDetector) Step(c core.Clock, tx *sql.Tx) error {
	if err := d.blockedIPsDetector.Step(c, tx); err != nil {
		return err
	}

	return d.summaryDetector.Step(c, tx)
}

func (d *composedDetector) Close() error {
	return nil
}

func TestSummary(t *testing.T) {
	Convey("Test Summary", t, func() {
		accessor, clear := insighttestsutil.NewFakeAccessor(t)
		defer clear()

		checker := &blockedips.FakeChecker{}

		options := core.Options{
			"blockedips_summary": Options{
				TimeSpan:        time.Hour * 24 * 7,
				InsightsFetcher: accessor,
			},
			"blockedips": blockedips.Options{
				Checker:      checker,
				PollInterval: time.Second * 10,
			},
		}

		d := &composedDetector{
			blockedIPsDetector: blockedips.NewDetector(accessor, options),
			summaryDetector:    NewDetector(accessor, options),
		}

		baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)
		clock := &insighttestsutil.FakeClock{Time: baseTime}

		Convey("No new  created as no blockedips insights were created during a week", func() {
			insighttestsutil.ExecuteCyclesUntil(d, accessor, clock, baseTime.Add(time.Hour*24*9), 5*time.Minute)
			So(accessor.Insights, ShouldResemble, []int64{})

			insights, err := accessor.FetchInsights(context.Background(), core.FetchOptions{
				Interval: timeutil.MustParseTimeInterval(`2000-01-01`, `4000-01-01`),
				OrderBy:  core.OrderByCreationAsc,
			}, clock)

			So(err, ShouldBeNil)
			So(len(insights), ShouldEqual, 0)
		})

		Convey("One insight is created when 7 complete days have elapsed", func() {
			checker.Actions = map[time.Time]intelblockedips.SummaryResult{
				testutil.MustParseTime(`2000-01-01 00:20:00 +0000`): {
					TopIPs: []intelblockedips.BlockedIP{
						{Address: "11.22.33.44", Count: 10},
						{Address: "66.77.88.99", Count: 15},
					},
					TotalNumber: 42,
					Interval:    timeutil.MustParseTimeInterval(`2020-10-10`, `2020-10-10`),
					TotalIPs:    4,
				},
				testutil.MustParseTime(`2000-01-03 00:25:00 +0000`): {
					TopIPs: []intelblockedips.BlockedIP{
						{Address: "5.6.7.8", Count: 11},
					},
					TotalNumber: 12,
					Interval:    timeutil.MustParseTimeInterval(`2020-10-11`, `2020-10-11`),
					TotalIPs:    3,
				},
			}

			insighttestsutil.ExecuteCyclesUntil(d, accessor, clock, baseTime.Add(time.Hour*24*9), 1*time.Minute)
			So(accessor.Insights, ShouldResemble, []int64{1, 2, 3})

			insights, err := accessor.FetchInsights(context.Background(), core.FetchOptions{
				Interval: timeutil.MustParseTimeInterval(`2000-01-01`, `4000-01-01`),
				OrderBy:  core.OrderByCreationDesc,
				FilterBy: core.FilterByCategory,
				Category: core.ActiveCategory,
			}, clock)

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 2)
			content, ok := insights[0].Content().(*Content)
			So(ok, ShouldBeTrue)
			So(insights[0].Time(), ShouldResemble, testutil.MustParseTime(`2000-01-08 00:00:00 +0000`))
			So(insights[0].Category(), ShouldEqual, core.IntelCategory)
			So(content, ShouldResemble, &Content{
				Interval: timeutil.TimeInterval{From: baseTime, To: baseTime.Add(time.Hour * 24 * 7)},
				Summary: []Summary{
					{Interval: timeutil.MustParseTimeInterval(`2020-10-10`, `2020-10-10`), IPCount: 4, ConnectionsCount: 42, RefID: 1},
					{Interval: timeutil.MustParseTimeInterval(`2020-10-11`, `2020-10-11`), IPCount: 3, ConnectionsCount: 12, RefID: 2},
				},
			})
		})
	})
}

func TestDescriptionFormatting(t *testing.T) {
	Convey("Description Formatting", t, func() {
		n := notification.Notification{
			ID: 1,
			Content: Content{
				Interval: timeutil.MustParseTimeInterval(`2021-10-01`, `2021-10-06`),
				Summary: []Summary{
					{Interval: timeutil.MustParseTimeInterval(`2020-10-10`, `2020-10-10`), IPCount: 4, ConnectionsCount: 42, RefID: 1},
					{Interval: timeutil.MustParseTimeInterval(`2020-10-11`, `2020-10-11`), IPCount: 6, ConnectionsCount: 35, RefID: 2},
				},
			},
		}

		m, err := notificationCore.TranslateNotification(n, translator.DummyTranslator{})
		So(err, ShouldBeNil)
		So(m, ShouldResemble, notificationCore.Message{
			Title:       "Suspicious IPs banned last week",
			Description: "77 Connections from 10 IPs were blocked over 6 days",
			Metadata:    map[string]string{},
		})
	})
}
