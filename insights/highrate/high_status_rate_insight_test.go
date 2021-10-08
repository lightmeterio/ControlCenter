// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package highrate

import (
	"context"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	mock_dashboard "gitlab.com/lightmeter/controlcenter/dashboard/mock"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
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

func TestHighRateDetectorInsight(t *testing.T) {
	Convey("Test Insights Generator", t, func() {
		ctrl := gomock.NewController(t)

		d := mock_dashboard.NewMockDashboard(ctrl)

		accessor, clear := insighttestsutil.NewFakeAccessor(t)
		defer clear()

		connPair := accessor.ConnPair

		baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)

		threeHours := time.Hour * 3
		baseInsightRange := time.Hour * 6

		Convey("Bounce rate is lower than threshhold", func() {
			clock := &insighttestsutil.FakeClock{Time: baseTime.Add(baseInsightRange)}

			d.EXPECT().DeliveryStatus(gomock.Any(), timeutil.TimeInterval{
				From: baseTime,
				To:   baseTime.Add(baseInsightRange),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 6},  // 30%
				dashboard.Pair{Key: "deferred", Value: 4}, // 20%
				dashboard.Pair{Key: "sent", Value: 10},    // 50%
			}, nil)

			detector := NewDetector(accessor, core.Options{"dashboard": d, "highrate": Options{BaseBounceRateThreshold: 0.4}}) // threshold 40%

			tx, err := connPair.RwConn.Begin()
			So(err, ShouldBeNil)
			So(detector.Step(clock, tx), ShouldBeNil)
			So(tx.Commit(), ShouldBeNil)

			So(len(accessor.Insights), ShouldEqual, 0)

			insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: timeutil.TimeInterval{
				From: baseTime,
				To:   baseTime.Add(baseInsightRange),
			}}, clock)

			So(err, ShouldBeNil)

			So(insights, ShouldBeNil)
		})

		Convey("Bounce rate is higher than threshhold", func() {
			clock := &insighttestsutil.FakeClock{Time: baseTime.Add(baseInsightRange)}

			interval := timeutil.TimeInterval{
				From: baseTime,
				To:   baseTime.Add(baseInsightRange),
			}

			d.EXPECT().DeliveryStatus(gomock.Any(), interval).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 6},  // 30%
				dashboard.Pair{Key: "deferred", Value: 4}, // 20%
				dashboard.Pair{Key: "sent", Value: 10},    // 50%
			}, nil)

			detector := NewDetector(accessor, core.Options{"dashboard": d, "highrate": Options{BaseBounceRateThreshold: 0.2}}) // threshold 20%

			tx, err := connPair.RwConn.Begin()
			So(err, ShouldBeNil)
			So(detector.Step(clock, tx), ShouldBeNil)
			So(tx.Commit(), ShouldBeNil)

			So(len(accessor.Insights), ShouldEqual, 1)

			insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: interval}, clock)

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 1)

			So(insights[0].ID(), ShouldEqual, 1)
			So(insights[0].ContentType(), ShouldEqual, HighBaseBounceRateContentType)
			So(insights[0].Time(), ShouldEqual, baseTime.Add(baseInsightRange))
			So(insights[0].Content(), ShouldResemble, &BounceRateContent{Value: 0.3, Interval: interval})
		})

		Convey("Generate a new high bounced rate insight for the past 6 hours after 3 hours not to spam the user", func() {
			clock := &insighttestsutil.FakeClock{Time: baseTime.Add(baseInsightRange)}

			d.EXPECT().DeliveryStatus(gomock.Any(), timeutil.TimeInterval{
				From: baseTime,
				To:   baseTime.Add(baseInsightRange),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 6},  // 30%
				dashboard.Pair{Key: "deferred", Value: 4}, // 20%
				dashboard.Pair{Key: "sent", Value: 10},    // 50%
			}, nil)

			// after three days, all good
			d.EXPECT().DeliveryStatus(gomock.Any(), timeutil.TimeInterval{
				From: baseTime.Add(threeHours * 3).Add(time.Second * 1),
				To:   baseTime.Add(threeHours * 3).Add(time.Second * 1).Add(baseInsightRange),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 5},  // 50%
				dashboard.Pair{Key: "deferred", Value: 2}, // 20%
				dashboard.Pair{Key: "sent", Value: 3},     // 30%
			}, nil)

			detector := NewDetector(accessor, core.Options{"dashboard": d, "highrate": Options{BaseBounceRateThreshold: 0.2}}) // threshold 20%

			{
				tx, err := connPair.RwConn.Begin()
				So(err, ShouldBeNil)

				So(tx.Commit(), ShouldBeNil)
			}

			cycle := func(c *insighttestsutil.FakeClock) {
				tx, err := connPair.RwConn.Begin()
				So(err, ShouldBeNil)
				So(detector.Step(c, tx), ShouldBeNil)
				So(tx.Commit(), ShouldBeNil)
			}

			// generate an insight
			cycle(clock)
			clock.Sleep(1 * time.Second)

			// do not generate
			cycle(clock)

			// generate an insight
			clock.Sleep(threeHours * 3)
			cycle(clock)

			So(len(accessor.Insights), ShouldEqual, 2)

			insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: timeutil.TimeInterval{
				From: baseTime,
				To:   baseTime.Add(threeHours * 3).Add(time.Second * 1).Add(baseInsightRange),
			}}, clock)

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 2)

			// more recent insights first
			{
				So(insights[0].ID(), ShouldEqual, 2)
				So(insights[0].ContentType(), ShouldEqual, HighBaseBounceRateContentType)
				So(insights[0].Time(), ShouldEqual, baseTime.Add(baseInsightRange).Add(threeHours*3).Add(time.Second*1))
				So(insights[0].Content(), ShouldResemble, &BounceRateContent{
					Value: 0.5,
					Interval: timeutil.TimeInterval{
						From: baseTime.Add(threeHours * 3).Add(time.Second * 1),
						To:   baseTime.Add(threeHours * 3).Add(time.Second * 1).Add(baseInsightRange),
					}})
			}

			{
				So(insights[1].ID(), ShouldEqual, 1)
				So(insights[1].ContentType(), ShouldEqual, HighBaseBounceRateContentType)
				So(insights[1].Time(), ShouldEqual, baseTime.Add(baseInsightRange))
				So(insights[1].Content(), ShouldResemble, &BounceRateContent{
					Value: 0.3,
					Interval: timeutil.TimeInterval{
						From: baseTime,
						To:   baseTime.Add(baseInsightRange),
					}})
			}
		})

		ctrl.Finish()
	})
}

func TestDescriptionFormatting(t *testing.T) {
	Convey("Description Formatting", t, func() {
		n := notification.Notification{
			ID: 1,
			Content: BounceRateContent{
				Interval: timeutil.TimeInterval{From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), To: testutil.MustParseTime(`2000-01-01 10:00:00 +0000`)},
				Value:    0.5,
			},
		}

		m, err := notificationCore.TranslateNotification(n, translator.DummyTranslator{})
		So(err, ShouldBeNil)
		So(m, ShouldResemble, notificationCore.Message{
			Title:       "High Bounce Rate",
			Description: "50 percent bounce rate between 2000-01-01 00:00:00 +0000 UTC and 2000-01-01 10:00:00 +0000 UTC",
			Metadata:    map[string]string{},
		})
	})
}
