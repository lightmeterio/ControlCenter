package highrate

import (
	"database/sql"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	mock_dashboard "gitlab.com/lightmeter/controlcenter/dashboard/mock"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

func tempDir() string {
	dir, e := ioutil.TempDir("", "lightmeter-tests-*")
	if e != nil {
		panic("error creating temp dir")
	}
	return dir
}

func parseTime(s string) time.Time {
	p, err := time.Parse(`2006-01-02 15:04:05 -0700`, s)

	if err != nil {
		panic("parsing time: " + err.Error())
	}

	return p.In(time.UTC)
}

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

type fakeClock struct {
	time.Time
}

func (t *fakeClock) Now() time.Time {
	return time.Time(t.Time)
}

func (t *fakeClock) Sleep(d time.Duration) {
	t.Time = t.Time.Add(d)
}

type fakeAcessor struct {
	*core.DBCreator
	core.Fetcher
	insights []int64
}

func (c *fakeAcessor) GenerateInsight(tx *sql.Tx, properties core.InsightProperties) error {
	id, err := core.GenerateInsight(tx, properties)

	if err != nil {
		return err
	}

	c.insights = append(c.insights, id)

	return nil
}

func TestHighRateDetectorInsight(t *testing.T) {
	Convey("Test Insights Generator", t, func() {
		dir := tempDir()
		defer os.RemoveAll(dir)

		ctrl := gomock.NewController(t)

		d := mock_dashboard.NewMockDashboard(ctrl)

		connPair, err := dbconn.NewConnPair(path.Join(dir, "insights.db"))
		So(err, ShouldBeNil)

		migrator.Run(connPair.RwConn.DB, "insights")

		defer func() {
			So(connPair.Close(), ShouldBeNil)
		}()

		accessor := func() *fakeAcessor {
			creator, err := core.NewCreator(connPair.RwConn)
			So(err, ShouldBeNil)
			fetcher, err := core.NewFetcher(connPair.RoConn)
			So(err, ShouldBeNil)
			return &fakeAcessor{DBCreator: creator, Fetcher: fetcher}
		}()

		Convey("Bounce rate is lower than threshhold", func() {
			clock := &fakeClock{parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24)}

			d.EXPECT().DeliveryStatus(data.TimeInterval{
				From: parseTime(`2000-01-01 00:00:00 +0000`),
				To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 6},  // 30%
				dashboard.Pair{Key: "deferred", Value: 4}, // 20%
				dashboard.Pair{Key: "sent", Value: 10},    // 50%
			})

			detector := NewDetector(accessor, core.Options{"dashboard": d, "highrate": Options{WeeklyBounceRateThreshold: 0.4}}) // threshold 40%

			tx, err := connPair.RwConn.Begin()
			So(err, ShouldBeNil)

			for _, s := range detector.Steppers() {
				So(s.Step(clock, tx), ShouldBeNil)
			}

			So(tx.Commit(), ShouldBeNil)

			So(len(accessor.insights), ShouldEqual, 0)

			insights, err := accessor.FetchInsights(core.FetchOptions{Interval: data.TimeInterval{
				From: parseTime(`2000-01-01 00:00:00 +0000`),
				To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24),
			}})

			So(err, ShouldBeNil)

			So(insights, ShouldResemble, []core.FetchedInsight{})
		})

		Convey("Bounce rate is higher than threshhold", func() {
			clock := &fakeClock{parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24)}

			interval := data.TimeInterval{
				From: parseTime(`2000-01-01 00:00:00 +0000`),
				To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24),
			}

			d.EXPECT().DeliveryStatus(interval).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 6},  // 30%
				dashboard.Pair{Key: "deferred", Value: 4}, // 20%
				dashboard.Pair{Key: "sent", Value: 10},    // 50%
			})

			detector := NewDetector(accessor, core.Options{"dashboard": d, "highrate": Options{WeeklyBounceRateThreshold: 0.2}}) // threshold 20%

			tx, err := connPair.RwConn.Begin()
			So(err, ShouldBeNil)

			for _, s := range detector.Steppers() {
				So(s.Step(clock, tx), ShouldBeNil)
			}

			So(tx.Commit(), ShouldBeNil)

			So(len(accessor.insights), ShouldEqual, 1)

			insights, err := accessor.FetchInsights(core.FetchOptions{Interval: interval})

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 1)

			So(insights[0].ID(), ShouldEqual, 1)
			So(insights[0].ContentType(), ShouldEqual, highWeeklyBounceRateContentType)
			So(insights[0].Time(), ShouldEqual, parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour*7*24))
			So(insights[0].Content(), ShouldResemble, &highWeeklyBounceRateInsightContent{Value: 0.3, Interval: interval})
		})

		Convey("Generate a new weekly high bounced rate insight after three day not to spam the user", func() {
			clock := &fakeClock{parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24)}

			d.EXPECT().DeliveryStatus(data.TimeInterval{
				From: parseTime(`2000-01-01 00:00:00 +0000`),
				To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 6},  // 30%
				dashboard.Pair{Key: "deferred", Value: 4}, // 20%
				dashboard.Pair{Key: "sent", Value: 10},    // 50%
			})

			// after three days, all good
			d.EXPECT().DeliveryStatus(data.TimeInterval{
				From: parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 24 * 3).Add(time.Second * 1),
				To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 24 * 3).Add(time.Second * 1).Add(time.Hour * 7 * 24),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 5},  // 50%
				dashboard.Pair{Key: "deferred", Value: 2}, // 20%
				dashboard.Pair{Key: "sent", Value: 3},     // 30%
			})

			detector := NewDetector(accessor, core.Options{"dashboard": d, "highrate": Options{WeeklyBounceRateThreshold: 0.2}}) // threshold 20%


			cycle := func(c *fakeClock) {
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

			So(len(accessor.insights), ShouldEqual, 2)

			insights, err := accessor.FetchInsights(core.FetchOptions{Interval: data.TimeInterval{
				From: parseTime(`2000-01-01 00:00:00 +0000`),
				To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 24 * 3).Add(time.Second * 1).Add(time.Hour * 7 * 24),
			}})

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 2)

			// more recent insights first
			{
				So(insights[0].ID(), ShouldEqual, 2)
				So(insights[0].ContentType(), ShouldEqual, highWeeklyBounceRateContentType)
				So(insights[0].Time(), ShouldEqual, parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour*7*24).Add(time.Hour*24*3).Add(time.Second*1))
				So(insights[0].Content(), ShouldResemble, &highWeeklyBounceRateInsightContent{
					Value: 0.5,
					Interval: data.TimeInterval{
						From: parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 24 * 3).Add(time.Second * 1),
						To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 24 * 3).Add(time.Second * 1).Add(time.Hour * 7 * 24),
					}})
			}

			{
				So(insights[1].ID(), ShouldEqual, 1)
				So(insights[1].ContentType(), ShouldEqual, highWeeklyBounceRateContentType)
				So(insights[1].Time(), ShouldEqual, parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour*7*24))
				So(insights[1].Content(), ShouldResemble, &highWeeklyBounceRateInsightContent{
					Value: 0.3,
					Interval: data.TimeInterval{
						From: parseTime(`2000-01-01 00:00:00 +0000`),
						To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 7 * 24),
					}})
			}
		})

		ctrl.Finish()
	})
}
