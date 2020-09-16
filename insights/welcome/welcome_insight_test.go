package welcome

import (
	"database/sql"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"os"
	"path"
	"testing"
	"time"
)

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

// TODO: move this struct to a common place to be reused by other tests instead of copy&pasting it everywhere
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

func TestWelcomeInsights(t *testing.T) {
	Convey("Test Welcome Generator", t, func() {
		dir := testutil.TempDir()
		defer os.RemoveAll(dir)

		connPair, err := dbconn.NewConnPair(path.Join(dir, "insights.db"))
		So(err, ShouldBeNil)

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

		Convey("Insight is generated only once", func() {
			clock := &fakeClock{parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 24)}

			detector := NewDetector(accessor)

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

			// generate insights only once, in the first cycle
			for i := 0; i < 10; i++ {
				cycle(clock)
				clock.Sleep(time.Hour * 8)
			}

			So(accessor.insights, ShouldResemble, []int64{1, 2})

			insights, err := accessor.FetchInsights(core.FetchOptions{Interval: data.TimeInterval{
				From: parseTime(`2000-01-01 00:00:00 +0000`),
				To:   parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 24).Add(time.Hour * 24),
			}, OrderBy: core.OrderByCreationAsc})

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 2)

			So(insights[0].ID(), ShouldEqual, 1)
			So(insights[0].ContentType(), ShouldEqual, "welcome_content")
			So(insights[0].Time(), ShouldEqual, parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour*24))
			So(insights[0].Content(), ShouldResemble, &struct{}{})

			So(insights[1].ID(), ShouldEqual, 2)
			So(insights[1].ContentType(), ShouldEqual, "insights_introduction_content")
			So(insights[1].Time(), ShouldEqual, parseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour*24))
			So(insights[1].Content(), ShouldResemble, &struct{}{})
		})
	})
}
