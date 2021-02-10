// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package mailinactivity

import (
	"context"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	mock_dashboard "gitlab.com/lightmeter/controlcenter/dashboard/mock"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
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
		ctrl := gomock.NewController(t)

		d := mock_dashboard.NewMockDashboard(ctrl)

		accessor, clear := insighttestsutil.NewFakeAccessor(t)
		defer clear()

		connPair := accessor.ConnPair

		lookupRange := time.Hour * 24

		detector := NewDetector(accessor, core.Options{
			"dashboard":      d,
			"mailinactivity": Options{LookupRange: lookupRange, MinTimeGenerationInterval: time.Hour * 8},
		})

		cycle := func(c *insighttestsutil.FakeClock) {
			tx, err := connPair.RwConn.Begin()
			So(err, ShouldBeNil)
			So(detector.Step(c, tx), ShouldBeNil)
			So(tx.Commit(), ShouldBeNil)
		}

		Convey("Don't generate an insight when application starts with no log data", func() {
			clock := &insighttestsutil.FakeClock{Time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange)}

			// there was no data available two days prior, not enough data to generate an insight
			d.EXPECT().DeliveryStatus(gomock.Any(), data.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange * -1),
				To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 0},
				dashboard.Pair{Key: "deferred", Value: 0},
				dashboard.Pair{Key: "sent", Value: 0},
			}, nil)

			// no activity in the past day, no insight is to be generated, as it's caused by not data being available
			// during such time
			d.EXPECT().DeliveryStatus(gomock.Any(), data.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 0},
				dashboard.Pair{Key: "deferred", Value: 0},
				dashboard.Pair{Key: "sent", Value: 0},
			}, nil)

			// do not generate insight
			cycle(clock)

			So(accessor.Insights, ShouldResemble, []int64{})
		})

		Convey("Server stays inactive for one day", func() {
			clock := &insighttestsutil.FakeClock{Time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange)}

			// some activity, no insights should be generated
			d.EXPECT().DeliveryStatus(gomock.Any(), data.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 1},
				dashboard.Pair{Key: "deferred", Value: 2},
				dashboard.Pair{Key: "sent", Value: 3},
			}, nil)

			// 8 hours later, check and realized there's been no activity for the past 24h
			{
				// the required "previous range"
				d.EXPECT().DeliveryStatus(gomock.Any(), data.TimeInterval{
					From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 8).Add(lookupRange * -1),
					To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange).Add(time.Hour * 8).Add(lookupRange * -1),
				}).Return(dashboard.Pairs{
					dashboard.Pair{Key: "bounced", Value: 1},
					dashboard.Pair{Key: "deferred", Value: 1},
					dashboard.Pair{Key: "sent", Value: 1},
				}, nil)

				// actual check
				d.EXPECT().DeliveryStatus(gomock.Any(), data.TimeInterval{
					From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 8),
					To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange).Add(time.Hour * 8),
				}).Return(dashboard.Pairs{
					dashboard.Pair{Key: "bounced", Value: 0},
					dashboard.Pair{Key: "deferred", Value: 0},
					dashboard.Pair{Key: "sent", Value: 0},
				}, nil)
			}

			// 8 hours later, there's activity again
			d.EXPECT().DeliveryStatus(gomock.Any(), data.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 16),
				To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange).Add(time.Hour * 16),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 0},
				dashboard.Pair{Key: "deferred", Value: 0},
				dashboard.Pair{Key: "sent", Value: 2},
			}, nil)

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

			insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: data.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange).Add(lookupRange),
			}})

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 1)

			So(insights[0].ID(), ShouldEqual, 1)
			So(insights[0].ContentType(), ShouldEqual, ContentType)
			So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange).Add(time.Hour*8))
			So(insights[0].Content(), ShouldResemble, &content{
				Interval: data.TimeInterval{
					From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 8),
					To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange).Add(time.Hour * 8),
				}})
		})

		ctrl.Finish()
	})
}
