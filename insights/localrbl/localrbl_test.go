package localrblinsight

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/localrbl"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"net"
	"os"
	"path"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

type fakeChecker struct {
	listening                      bool
	shouldGenerateOnScanCompletion bool
	timeToCompleteScan             time.Duration
	scanStartTime                  time.Time
	checkedIP                      net.IP
}

func (c *fakeChecker) Close() error {
	return nil
}

func (c *fakeChecker) StartListening() {
	So(c.listening, ShouldBeFalse)
	c.listening = true
}

func (c *fakeChecker) NotifyNewScan(now time.Time) {
	c.scanStartTime = now
	So(c.listening, ShouldBeTrue)
}

func (c *fakeChecker) Step(now time.Time, withResults func(localrbl.Results) error, withoutResults func() error) error {
	if !c.scanStartTime.IsZero() && now.After(c.scanStartTime.Add(c.timeToCompleteScan)) && c.shouldGenerateOnScanCompletion {
		return withResults(localrbl.Results{
			Interval: data.TimeInterval{From: c.scanStartTime, To: now},
			RBLs:     []localrbl.ContentElement{{RBL: "some.rbl.checker.com", Text: "Something Really Bad"}},
		})
	}

	return withoutResults()
}

func (c *fakeChecker) CheckedIP() net.IP {
	return c.checkedIP
}

func TestLocalRBL(t *testing.T) {
	Convey("Test Local RBL", t, func() {
		dir := testutil.TempDir()
		defer os.RemoveAll(dir)

		connPair, err := dbconn.NewConnPair(path.Join(dir, "insights.db"))
		So(err, ShouldBeNil)

		defer func() {
			So(connPair.Close(), ShouldBeNil)
		}()

		migrator.Run(connPair.RwConn.DB, "insights")

		accessor := func() *insighttestsutil.FakeAcessor {
			creator, err := core.NewCreator(connPair.RwConn)
			So(err, ShouldBeNil)
			fetcher, err := core.NewFetcher(connPair.RoConn)
			So(err, ShouldBeNil)
			return &insighttestsutil.FakeAcessor{DBCreator: creator, Fetcher: fetcher, Insights: []int64{}}
		}()

		checker := &fakeChecker{
			timeToCompleteScan: time.Second * 10,
			checkedIP:          net.ParseIP("11.22.33.44"),
		}

		detector := NewDetector(accessor, core.Options{
			"localrbl": Options{
				CheckInterval: time.Second * 10,
				Checker:       checker,
			},
		})

		checker.StartListening()

		defer detector.Close()

		cycle := func(c *insighttestsutil.FakeClock) {
			tx, err := connPair.RwConn.Begin()
			So(err, ShouldBeNil)

			for _, s := range detector.Steppers() {
				So(s.Step(c, tx), ShouldBeNil)
			}

			So(tx.Commit(), ShouldBeNil)
		}

		Convey("Scan host, but generates no insight", func() {
			baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)

			clock := &insighttestsutil.FakeClock{Time: baseTime}

			// do not generate insight, but trigger it to start
			cycle(clock)

			// After 5s, the scan has't finished yet
			clock.Sleep(time.Second * 5)
			cycle(clock)

			// After 9s, the scan has't finished yet
			clock.Sleep(time.Second * 4)
			cycle(clock)

			// After 10s, the scan is finished, but no insight'll have been created
			clock.Sleep(time.Second * 4)
			cycle(clock)

			So(accessor.Insights, ShouldResemble, []int64{})
		})

		Convey("Scan host, and generate an insight after 10s", func() {
			checker.shouldGenerateOnScanCompletion = true

			baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)

			clock := &insighttestsutil.FakeClock{Time: baseTime}

			// do not generate insight, but trigger it to start
			cycle(clock)

			// After 5s, the scan has't finished yet
			clock.Sleep(time.Second * 5)
			cycle(clock)

			// After 9s, the scan has't finished yet
			clock.Sleep(time.Second * 4)
			cycle(clock)

			// After 11s, the scan is finished, resulting in an insight generated
			clock.Sleep(time.Second * 2)
			cycle(clock)

			So(accessor.Insights, ShouldResemble, []int64{1})

			insights, err := accessor.FetchInsights(core.FetchOptions{Interval: data.TimeInterval{
				From: testutil.MustParseTime(`0000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`4000-01-01 00:00:00 +0000`),
			}})

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 1)

			So(insights[0].ID(), ShouldEqual, 1)
			So(insights[0].ContentType(), ShouldEqual, ContentType)
			So(insights[0].Time(), ShouldEqual, baseTime.Add(time.Second*11))
			So(insights[0].Content(), ShouldResemble, &content{
				ScanInterval: data.TimeInterval{From: baseTime, To: baseTime.Add(time.Second * 11)},
				Address:      net.ParseIP("11.22.33.44"),
				RBLs:         []localrbl.ContentElement{{RBL: "some.rbl.checker.com", Text: "Something Really Bad"}},
			})
		})
	})
}
