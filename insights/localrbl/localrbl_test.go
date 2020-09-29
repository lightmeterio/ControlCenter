package localrbl

import (
	"github.com/mrichman/godnsbl"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"net"
	"os"
	"path"
	"strings"
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
}

func (c *fakeChecker) Close() error {
	return nil
}

func (c *fakeChecker) startListening() {
	So(c.listening, ShouldBeFalse)
	c.listening = true
}

func (c *fakeChecker) notifyNewScan(now time.Time) {
	c.scanStartTime = now
	So(c.listening, ShouldBeTrue)
}

func (c *fakeChecker) step(now time.Time, withResults func(checkResults) error, withoutResults func() error) error {
	if !c.scanStartTime.IsZero() && now.After(c.scanStartTime.Add(c.timeToCompleteScan)) && c.shouldGenerateOnScanCompletion {
		return withResults(checkResults{
			rbls: []contentElem{{RBL: "some.rbl.checker.com", Text: "Something Really Bad"}},
		})
	}

	return withoutResults()
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
		}

		detector := newDetector(accessor, core.Options{
			"localrbl": Options{CheckedAddress: net.ParseIP("11.22.33.44"), CheckInterval: time.Second * 10},
		}, checker)

		checker.startListening()

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
			So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:11 +0000`))
			So(insights[0].Content(), ShouldResemble, &content{
				Address: "11.22.33.44",
				RBLs:    []contentElem{{RBL: "some.rbl.checker.com", Text: "Something Really Bad"}},
			})
		})
	})
}

func TestDnsRBL(t *testing.T) {
	Convey("Test Local RBL", t, func() {
		lookup := func(rblList string, targetHost string) godnsbl.RBLResults {
			if !strings.HasSuffix(rblList, "-blocked") {
				return godnsbl.RBLResults{}
			}

			return godnsbl.RBLResults{
				Host:    targetHost,
				List:    rblList,
				Results: []godnsbl.Result{{Listed: true, Address: targetHost, Text: "Some Error", Rbl: rblList}},
			}
		}

		Convey("Not blocked in any lists", func() {
			checker := newDnsChecker(lookup, Options{
				CheckedAddress:   net.ParseIP("11.22.33.44"),
				RBLProvidersURLs: []string{"rbl1", "rbl2", "rbl3", "rbl4", "rbl5"},
			})

			defer checker.Close()

			checker.startListening()

			baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)

			checker.notifyNewScan(baseTime)

			time.Sleep(500 * time.Millisecond)

			select {
			case <-checker.checkerResultsChan:
				So(false, ShouldBeTrue)
			default:
			}
		})

		Convey("Blocked in some RBLs", func() {
			checker := newDnsChecker(lookup, Options{
				CheckedAddress:   net.ParseIP("11.22.33.44"),
				RBLProvidersURLs: []string{"rbl1-blocked", "rbl2", "rbl3-blocked", "rbl4-blocked", "rbl5"},
			})

			defer checker.Close()

			checker.startListening()

			baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)

			checker.notifyNewScan(baseTime)

			time.Sleep(500 * time.Millisecond)

			select {
			case r := <-checker.checkerResultsChan:
				So(r.rbls, ShouldResemble, []contentElem{
					{RBL: "rbl1-blocked", Text: "Some Error"},
					{RBL: "rbl3-blocked", Text: "Some Error"},
					{RBL: "rbl4-blocked", Text: "Some Error"},
				})
			default:
				So(false, ShouldBeTrue)
			}
		})
	})
}
