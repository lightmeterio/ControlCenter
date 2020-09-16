package mailinactivity

import (
	"database/sql"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	mock_dashboard "gitlab.com/lightmeter/controlcenter/dashboard/mock"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
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

func TestMailInactivityDetectorInsight(t *testing.T) {
	Convey("Test Insights Generator", t, func() {
		dir := tempDir()
		defer os.RemoveAll(dir)

		ctrl := gomock.NewController(t)

		d := mock_dashboard.NewMockDashboard(ctrl)

		connPair, err := dbconn.NewConnPair(path.Join(dir, "insights_state.db"))
		So(err, ShouldBeNil)

		defer func() {
			So(connPair.Close(), ShouldBeNil)
		}()

		accessor := func() *fakeAcessor {
			creator, err := core.NewCreator(connPair.RwConn)
			So(err, ShouldBeNil)
			fetcher, err := core.NewFetcher(connPair.RoConn)
			So(err, ShouldBeNil)
			return &fakeAcessor{DBCreator: creator, Fetcher: fetcher, insights: []int64{}}
		}()

		lookupRange := time.Hour * 24

		detector := NewDetector(accessor, core.Options{
			"dashboard":      d,
			"mailinactivity": Options{LookupRange: lookupRange, MinTimeGenerationInterval: time.Hour * 8},
		})

		{
			tx, err := connPair.RwConn.Begin()
			So(err, ShouldBeNil)

			So(core.SetupAuxTables(tx), ShouldBeNil)

			So(detector.Setup(tx), ShouldBeNil)

			So(tx.Commit(), ShouldBeNil)
		}

		cycle := func(c *fakeClock) {
			tx, err := connPair.RwConn.Begin()
			So(err, ShouldBeNil)

			for _, s := range detector.Steppers() {
				So(s.Step(c, tx), ShouldBeNil)
			}

			So(tx.Commit(), ShouldBeNil)
		}

		Convey("Don't generate an insight when application starts with no log data", func() {
			clock := &fakeClock{parseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange)}

			// there was no data available two days prior, not enough data to generate an insight
			d.EXPECT().DeliveryStatus(data.TimeInterval{
				From: parseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange * -1),
				To:   parseTime(`2000-01-01 00:00:00 +0000`),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 0},
				dashboard.Pair{Key: "deferred", Value: 0},
				dashboard.Pair{Key: "sent", Value: 0},
			})

			// no activity in the past day, no insight is to be generated, as it's caused by not data being available
			// during such time
			d.EXPECT().DeliveryStatus(data.TimeInterval{
				From: parseTime(`2000-01-01 00:00:00 +0000`),
				To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 0},
				dashboard.Pair{Key: "deferred", Value: 0},
				dashboard.Pair{Key: "sent", Value: 0},
			})

			// do not generate insight
			cycle(clock)

			So(accessor.insights, ShouldResemble, []int64{})
		})

		Convey("Server stays inactive for one day", func() {
			clock := &fakeClock{parseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange)}

			// some activity, no insights should be generated
			d.EXPECT().DeliveryStatus(data.TimeInterval{
				From: parseTime(`2000-01-01 00:00:00 +0000`),
				To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 1},
				dashboard.Pair{Key: "deferred", Value: 2},
				dashboard.Pair{Key: "sent", Value: 3},
			})

			// 8 hours later, check and realized there's been no activity for the past 24h
			{
				// the required "previous range"
				d.EXPECT().DeliveryStatus(data.TimeInterval{
					From: parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 8).Add(lookupRange * -1),
					To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange).Add(time.Hour * 8).Add(lookupRange * -1),
				}).Return(dashboard.Pairs{
					dashboard.Pair{Key: "bounced", Value: 1},
					dashboard.Pair{Key: "deferred", Value: 1},
					dashboard.Pair{Key: "sent", Value: 1},
				})

				// actual check
				d.EXPECT().DeliveryStatus(data.TimeInterval{
					From: parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 8),
					To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange).Add(time.Hour * 8),
				}).Return(dashboard.Pairs{
					dashboard.Pair{Key: "bounced", Value: 0},
					dashboard.Pair{Key: "deferred", Value: 0},
					dashboard.Pair{Key: "sent", Value: 0},
				})
			}

			// 8 hours later, there's activity again
			d.EXPECT().DeliveryStatus(data.TimeInterval{
				From: parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 16),
				To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange).Add(time.Hour * 16),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 0},
				dashboard.Pair{Key: "deferred", Value: 0},
				dashboard.Pair{Key: "sent", Value: 2},
			})

			// do not generate insight
			cycle(clock)

			// Generate insight
			clock.Sleep(time.Hour * 8)
			cycle(clock)

			// do not generate insight
			clock.Sleep(time.Hour * 8)
			cycle(clock)

			So(accessor.insights, ShouldResemble, []int64{1})

			So(len(accessor.insights), ShouldEqual, 1)

			insights, err := accessor.FetchInsights(core.FetchOptions{Interval: data.TimeInterval{
				From: parseTime(`2000-01-01 00:00:00 +0000`),
				To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange).Add(lookupRange),
			}})

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 1)

			So(insights[0].ID(), ShouldEqual, 1)
			So(insights[0].ContentType(), ShouldEqual, ContentType)
			So(insights[0].Time(), ShouldEqual, parseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange).Add(time.Hour*8))
			So(insights[0].Content(), ShouldResemble, &content{
				Interval: data.TimeInterval{
					From: parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 8),
					To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(lookupRange).Add(time.Hour * 8),
				}})
		})

		ctrl.Finish()
	})
}
