// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package blockedips

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/intel/blockedips"
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

type poolTimeChecker struct {
	intervals []timeutil.TimeInterval
}

func (c *poolTimeChecker) Step(interval timeutil.TimeInterval, withResults func(blockedips.SummaryResult) error) error {
	c.intervals = append(c.intervals, interval)
	return nil
}

func TestExecutionTimes(t *testing.T) {
	Convey("Test Execution times", t, func() {
		accessor, clear := insighttestsutil.NewFakeAccessor(t)
		defer clear()

		checker := &poolTimeChecker{}

		d := NewDetector(accessor, core.Options{
			"blockedips": Options{
				Checker:        checker,
				PollInterval:   time.Second * 30,
				EventsInterval: 24 * time.Hour,
			},
		})

		baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)
		clock := &insighttestsutil.FakeClock{Time: baseTime}

		Convey("Execute every 30 seconds", func() {
			insighttestsutil.ExecuteCyclesUntil(d, accessor, clock, baseTime.Add(time.Minute*3+time.Second*2), 2*time.Second)

			So(checker.intervals, ShouldResemble, []timeutil.TimeInterval{
				{From: baseTime.Add(-24 * time.Hour), To: baseTime},
				{From: baseTime.Add(time.Second * 30).Add(-24 * time.Hour), To: baseTime.Add(time.Second * 30)},
				{From: baseTime.Add(time.Second * 60).Add(-24 * time.Hour), To: baseTime.Add(time.Second * 60)},
				{From: baseTime.Add(time.Second * 90).Add(-24 * time.Hour), To: baseTime.Add(time.Second * 90)},
				{From: baseTime.Add(time.Second * 120).Add(-24 * time.Hour), To: baseTime.Add(time.Second * 120)},
				{From: baseTime.Add(time.Second * 150).Add(-24 * time.Hour), To: baseTime.Add(time.Second * 150)},
				{From: baseTime.Add(time.Second * 180).Add(-24 * time.Hour), To: baseTime.Add(time.Second * 180)},
			})
		})
	})
}

func TestSummary(t *testing.T) {
	Convey("Test Summary", t, func() {
		accessor, clear := insighttestsutil.NewFakeAccessor(t)
		defer clear()

		checker := &FakeChecker{}

		d := NewDetector(accessor, core.Options{
			"blockedips": Options{
				Checker:        checker,
				PollInterval:   time.Second * 10,
				EventsInterval: 24 * time.Hour,
			},
		})

		baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)
		clock := &insighttestsutil.FakeClock{Time: baseTime}

		Convey("No new  created", func() {
			insighttestsutil.ExecuteCyclesUntil(d, accessor, clock, baseTime.Add(time.Hour*2), 2*time.Second)
			So(accessor.Insights, ShouldResemble, []int64{})
		})

		buildInterval := func(now time.Time) timeutil.TimeInterval {
			return timeutil.TimeInterval{From: now.Add(-24 * time.Hour), To: now}
		}

		Convey("One insight is created", func() {
			checker.Actions = map[timeutil.TimeInterval]blockedips.SummaryResult{
				buildInterval(testutil.MustParseTime(`2000-01-01 00:20:00 +0000`)): {
					TopIPs: []blockedips.BlockedIP{
						{Address: "11.22.33.44", Count: 10},
						{Address: "66.77.88.99", Count: 15},
					},
					TotalNumber: 42,
					Interval:    timeutil.MustParseTimeInterval(`2020-10-10`, `2020-10-10`),
					TotalIPs:    4,
				},
			}

			insighttestsutil.ExecuteCyclesUntil(d, accessor, clock, baseTime.Add(time.Hour*2), 2*time.Second)
			So(accessor.Insights, ShouldResemble, []int64{1})

			insights, err := accessor.Fetcher.FetchInsights(context.Background(), core.FetchOptions{
				Interval: timeutil.MustParseTimeInterval(`2000-01-01`, `4000-01-01`),
			}, clock)

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 1)
			So(insights[0].Time(), ShouldResemble, testutil.MustParseTime(`2000-01-01 00:20:00 +0000`))
			So(insights[0].Category(), ShouldEqual, core.IntelCategory)
			content, ok := insights[0].Content().(*Content)
			So(ok, ShouldBeTrue)
			So(content, ShouldResemble, &Content{
				TopIPs: []blockedips.BlockedIP{
					{Address: "11.22.33.44", Count: 10},
					{Address: "66.77.88.99", Count: 15},
				},
				TotalNumber: 42,
				Interval:    timeutil.MustParseTimeInterval(`2020-10-10`, `2020-10-10`),
				TotalIPs:    4,
			})
		})

		Convey("When a new insight is created, all the previous ones are archived", func() {
			checker.Actions = map[timeutil.TimeInterval]blockedips.SummaryResult{
				buildInterval(testutil.MustParseTime(`2000-01-01 01:00:00 +0000`)): {
					TopIPs: []blockedips.BlockedIP{
						{Address: "11.22.33.44", Count: 10},
						{Address: "55.66.77.88", Count: 15},
					},
					TotalNumber: 42,
					Interval:    timeutil.MustParseTimeInterval(`2020-10-10`, `2020-10-10`),
					TotalIPs:    4,
				},
				buildInterval(testutil.MustParseTime(`2000-01-01 01:30:00 +0000`)): {
					TopIPs: []blockedips.BlockedIP{
						{Address: "11.22.33.44", Count: 30},
					},
					TotalNumber: 30,
					Interval:    timeutil.MustParseTimeInterval(`2020-10-11`, `2020-10-11`),
					TotalIPs:    2,
				},
				buildInterval(testutil.MustParseTime(`2000-01-01 01:40:00 +0000`)): {
					TopIPs: []blockedips.BlockedIP{
						{Address: "1.1.1.1", Count: 67},
						{Address: "2.2.2.2", Count: 3},
					},
					TotalNumber: 70,
					Interval:    timeutil.MustParseTimeInterval(`2020-10-12`, `2020-10-12`),
					TotalIPs:    5,
				},
			}

			insighttestsutil.ExecuteCyclesUntil(d, accessor, clock, baseTime.Add(time.Hour*2), 2*time.Second)

			So(accessor.Insights, ShouldResemble, []int64{1, 2, 3})

			insights, err := accessor.Fetcher.FetchInsights(context.Background(), core.FetchOptions{
				Interval: timeutil.MustParseTimeInterval(`2000-01-01`, `4000-01-01`),
				OrderBy:  core.OrderByCreationAsc,
			}, clock)

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 3)

			{
				So(insights[0].Time(), ShouldResemble, testutil.MustParseTime(`2000-01-01 01:00:00 +0000`))
				So(insights[0].Category(), ShouldEqual, core.ArchivedCategory)
				content, ok := insights[0].Content().(*Content)
				So(ok, ShouldBeTrue)
				So(content, ShouldResemble, &Content{
					TopIPs: []blockedips.BlockedIP{
						{Address: "11.22.33.44", Count: 10},
						{Address: "55.66.77.88", Count: 15},
					},
					TotalNumber: 42,
					Interval:    timeutil.MustParseTimeInterval(`2020-10-10`, `2020-10-10`),
					TotalIPs:    4,
				})
			}

			{
				So(insights[1].Time(), ShouldResemble, testutil.MustParseTime(`2000-01-01 01:30:00 +0000`))
				So(insights[1].Category(), ShouldEqual, core.ArchivedCategory)
				content, ok := insights[1].Content().(*Content)
				So(ok, ShouldBeTrue)
				So(content, ShouldResemble, &Content{
					TopIPs: []blockedips.BlockedIP{
						{Address: "11.22.33.44", Count: 30},
					},
					TotalNumber: 30,
					Interval:    timeutil.MustParseTimeInterval(`2020-10-11`, `2020-10-11`),
					TotalIPs:    2,
				})
			}

			{
				So(insights[2].Time(), ShouldResemble, testutil.MustParseTime(`2000-01-01 01:40:00 +0000`))
				So(insights[2].Category(), ShouldEqual, core.IntelCategory)
				content, ok := insights[2].Content().(*Content)
				So(ok, ShouldBeTrue)
				So(content, ShouldResemble, &Content{
					TopIPs: []blockedips.BlockedIP{
						{Address: "1.1.1.1", Count: 67},
						{Address: "2.2.2.2", Count: 3},
					},
					TotalNumber: 70,
					Interval:    timeutil.MustParseTimeInterval(`2020-10-12`, `2020-10-12`),
					TotalIPs:    5,
				})
			}
		})
	})
}

func TestDescriptionFormatting(t *testing.T) {
	Convey("Description Formatting", t, func() {
		n := notification.Notification{
			ID: 1,
			Content: Content{
				TopIPs: []blockedips.BlockedIP{
					{Address: "11.11.11.11", Count: 42},
				},
				TotalNumber: 245,
				Interval:    timeutil.MustParseTimeInterval(`2020-10-12`, `2020-10-12`),
				TotalIPs:    5,
			},
		}

		m, err := notificationCore.TranslateNotification(n, translator.DummyTranslator{})
		So(err, ShouldBeNil)
		So(m, ShouldResemble, notificationCore.Message{
			Title:       "Blocked suspicious connection attempts",
			Description: "245 connections blocked from 5 banned IPs (peer network)",
			Metadata:    map[string]string{},
		})
	})
}
