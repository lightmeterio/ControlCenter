package highrate

import (
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	mock_dashboard "gitlab.com/lightmeter/controlcenter/dashboard/mock"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"os"
	"path"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestHighRateDetectorInsight(t *testing.T) {
	Convey("Test Insights Generator", t, func() {
		dir := testutil.TempDir()
		defer os.RemoveAll(dir)

		ctrl := gomock.NewController(t)

		d := mock_dashboard.NewMockDashboard(ctrl)

		connPair, err := dbconn.NewConnPair(path.Join(dir, "insights_state.db"))
		So(err, ShouldBeNil)

		defer func() {
			So(connPair.Close(), ShouldBeNil)
		}()

		accessor := func() *insighttestsutil.FakeAcessor {
			creator, err := core.NewCreator(connPair.RwConn)
			So(err, ShouldBeNil)
			fetcher, err := core.NewFetcher(connPair.RoConn)
			So(err, ShouldBeNil)
			return &insighttestsutil.FakeAcessor{DBCreator: creator, Fetcher: fetcher}
		}()

		Convey("Bounce rate is lower than threshhold", func() {
			clock := &insighttestsutil.FakeClock{testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24)}

			d.EXPECT().DeliveryStatus(data.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 6},  // 30%
				dashboard.Pair{Key: "deferred", Value: 4}, // 20%
				dashboard.Pair{Key: "sent", Value: 10},    // 50%
			})

			detector := NewDetector(accessor, core.Options{"dashboard": d, "highrate": Options{WeeklyBounceRateThreshold: 0.4}}) // threshold 40%

			tx, err := connPair.RwConn.Begin()
			So(err, ShouldBeNil)

			So(core.SetupAuxTables(tx), ShouldBeNil)

			So(detector.Setup(tx), ShouldBeNil)

			for _, s := range detector.Steppers() {
				So(s.Step(clock, tx), ShouldBeNil)
			}

			So(tx.Commit(), ShouldBeNil)

			So(len(accessor.Insights), ShouldEqual, 0)

			insights, err := accessor.FetchInsights(core.FetchOptions{Interval: data.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24),
			}})

			So(err, ShouldBeNil)

			So(insights, ShouldResemble, []core.FetchedInsight{})
		})

		Convey("Bounce rate is higher than threshhold", func() {
			clock := &insighttestsutil.FakeClock{testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24)}

			interval := data.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24),
			}

			d.EXPECT().DeliveryStatus(interval).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 6},  // 30%
				dashboard.Pair{Key: "deferred", Value: 4}, // 20%
				dashboard.Pair{Key: "sent", Value: 10},    // 50%
			})

			detector := NewDetector(accessor, core.Options{"dashboard": d, "highrate": Options{WeeklyBounceRateThreshold: 0.2}}) // threshold 20%

			tx, err := connPair.RwConn.Begin()
			So(err, ShouldBeNil)

			So(core.SetupAuxTables(tx), ShouldBeNil)

			So(detector.Setup(tx), ShouldBeNil)

			for _, s := range detector.Steppers() {
				So(s.Step(clock, tx), ShouldBeNil)
			}

			So(tx.Commit(), ShouldBeNil)

			So(len(accessor.Insights), ShouldEqual, 1)

			insights, err := accessor.FetchInsights(core.FetchOptions{Interval: interval})

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 1)

			So(insights[0].ID(), ShouldEqual, 1)
			So(insights[0].ContentType(), ShouldEqual, highWeeklyBounceRateContentType)
			So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour*7*24))
			So(insights[0].Content(), ShouldResemble, &highWeeklyBounceRateInsightContent{Value: 0.3, Interval: interval})
		})

		Convey("Generate a new weekly high bounced rate insight after three day not to spam the user", func() {
			clock := &insighttestsutil.FakeClock{testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24)}

			d.EXPECT().DeliveryStatus(data.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 6},  // 30%
				dashboard.Pair{Key: "deferred", Value: 4}, // 20%
				dashboard.Pair{Key: "sent", Value: 10},    // 50%
			})

			// after three days, all good
			d.EXPECT().DeliveryStatus(data.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 24 * 3).Add(time.Second * 1),
				To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 24 * 3).Add(time.Second * 1).Add(time.Hour * 7 * 24),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 5},  // 50%
				dashboard.Pair{Key: "deferred", Value: 2}, // 20%
				dashboard.Pair{Key: "sent", Value: 3},     // 30%
			})

			detector := NewDetector(accessor, core.Options{"dashboard": d, "highrate": Options{WeeklyBounceRateThreshold: 0.2}}) // threshold 20%

			{
				tx, err := connPair.RwConn.Begin()
				So(err, ShouldBeNil)

				So(core.SetupAuxTables(tx), ShouldBeNil)

				So(detector.Setup(tx), ShouldBeNil)

				So(tx.Commit(), ShouldBeNil)
			}

			cycle := func(c *insighttestsutil.FakeClock) {
				tx, err := connPair.RwConn.Begin()
				So(err, ShouldBeNil)

				for _, s := range detector.Steppers() {
					So(s.Step(c, tx), ShouldBeNil)
				}

				So(tx.Commit(), ShouldBeNil)
			}

			// generate an insight
			cycle(clock)
			clock.Sleep(1 * time.Second)

			// do not generate
			cycle(clock)

			// generate an insight
			clock.Sleep(time.Hour * 24 * 3)
			cycle(clock)

			So(len(accessor.Insights), ShouldEqual, 2)

			insights, err := accessor.FetchInsights(core.FetchOptions{Interval: data.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 24 * 3).Add(time.Second * 1).Add(time.Hour * 7 * 24),
			}})

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 2)

			// more recent insights first
			{
				So(insights[0].ID(), ShouldEqual, 2)
				So(insights[0].ContentType(), ShouldEqual, highWeeklyBounceRateContentType)
				So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour*7*24).Add(time.Hour*24*3).Add(time.Second*1))
				So(insights[0].Content(), ShouldResemble, &highWeeklyBounceRateInsightContent{
					Value: 0.5,
					Interval: data.TimeInterval{
						From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 24 * 3).Add(time.Second * 1),
						To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 24 * 3).Add(time.Second * 1).Add(time.Hour * 7 * 24),
					}})
			}

			{
				So(insights[1].ID(), ShouldEqual, 1)
				So(insights[1].ContentType(), ShouldEqual, highWeeklyBounceRateContentType)
				So(insights[1].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour*7*24))
				So(insights[1].Content(), ShouldResemble, &highWeeklyBounceRateInsightContent{
					Value: 0.3,
					Interval: data.TimeInterval{
						From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24),
					}})
			}
		})

		ctrl.Finish()
	})
}
