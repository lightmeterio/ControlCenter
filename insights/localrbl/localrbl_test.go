package localrblinsight

import (
	"context"
	"github.com/mrichman/godnsbl"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/localrbl"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"net"
	"path"
	"strings"
	"testing"
	"time"
)

var (
	dummyContext = context.Background()
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

func (c *fakeChecker) CheckedIP(context.Context) net.IP {
	return c.checkedIP
}

func TestLocalRBL(t *testing.T) {
	Convey("Test Local RBL", t, func() {
		dir, clearDir := testutil.TempDir()
		defer clearDir()

		connPair, err := dbconn.NewConnPair(path.Join(dir, "insights.db"))
		So(err, ShouldBeNil)

		cycleOnDetector := func(d core.Detector, c *insighttestsutil.FakeClock) {
			tx, err := connPair.RwConn.Begin()
			So(err, ShouldBeNil)
			So(d.Step(c, tx), ShouldBeNil)
			So(tx.Commit(), ShouldBeNil)
		}

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

		Convey("Use Fake Checker", func() {
			checker := &fakeChecker{
				timeToCompleteScan: time.Second * 10,
				checkedIP:          net.ParseIP("11.22.33.44"),
			}

			d := NewDetector(accessor, core.Options{
				"localrbl": Options{
					CheckInterval: time.Second * 10,
					Checker:       checker,
				},
			})

			checker.StartListening()

			defer func() {
				d.(*detector).Close()
			}()

			cycle := func(c *insighttestsutil.FakeClock) {
				cycleOnDetector(d, c)
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

				insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: data.TimeInterval{
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

			Convey("Use real DNS checker", func() {
				conn, closeConn := testutil.TempDBConnection()
				defer closeConn()

				m, err := meta.NewHandler(conn, "master")
				So(err, ShouldBeNil)

				defer func() { errorutil.MustSucceed(m.Close()) }()

				{
					settings := localrbl.Settings{
						LocalIP: net.ParseIP("11.22.33.44"),
					}

					_, err := m.Writer.StoreJson(dummyContext, localrbl.SettingsKey, &settings)

					So(err, ShouldBeNil)
				}

				checker := localrbl.NewChecker(m.Reader, localrbl.Options{
					NumberOfWorkers:  2,
					Lookup:           fakeLookup,
					RBLProvidersURLs: []string{"rbl1-blocked", "rbl2", "rbl3-blocked", "rbl4-blocked", "rbl5"},
				})

				checker.StartListening()

				d := NewDetector(accessor, core.Options{
					"localrbl": Options{
						CheckInterval: time.Millisecond * 300,
						Checker:       checker,
					},
				})

				baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)

				clock := &insighttestsutil.FakeClock{Time: baseTime}

				sleep := func(d time.Duration) {
					clock.Sleep(d)
					// we sleep a little bit more in the real world
					// to give time to things synchronize
					time.Sleep(d * 3)
				}

				cycle := func(c *insighttestsutil.FakeClock) {
					cycleOnDetector(d, c)
				}

				// do not generate insight, but trigger it to start
				cycle(clock)

				// After almost one second we should have something
				sleep(time.Millisecond * 700)
				cycle(clock)

				So(accessor.Insights, ShouldResemble, []int64{1})

				insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: data.TimeInterval{
					From: testutil.MustParseTime(`0000-01-01 00:00:00 +0000`),
					To:   testutil.MustParseTime(`4000-01-01 00:00:00 +0000`),
				}})

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 1)

				So(insights[0].ID(), ShouldEqual, 1)
				So(insights[0].ContentType(), ShouldEqual, ContentType)
				So(insights[0].Time(), ShouldEqual, baseTime)

				c, ok := insights[0].Content().(*content)
				So(ok, ShouldBeTrue)

				// We cannot know for sure when the scan finished, just when it started
				So(c.ScanInterval.From, ShouldResemble, baseTime)
				So(c.Address, ShouldResemble, net.ParseIP("11.22.33.44"))
				So(c.RBLs, ShouldResemble, []localrbl.ContentElement{
					localrbl.ContentElement{RBL: "rbl1-blocked", Text: "Some Error"},
					localrbl.ContentElement{RBL: "rbl3-blocked", Text: "Some Error"},
					localrbl.ContentElement{RBL: "rbl4-blocked", Text: "Some Error"}})
			})
		})
	})
}

func fakeLookup(rblList string, targetHost string) godnsbl.RBLResults {
	if !strings.HasSuffix(rblList, "-blocked") {
		return godnsbl.RBLResults{}
	}

	return godnsbl.RBLResults{
		Host:    targetHost,
		List:    rblList,
		Results: []godnsbl.Result{{Listed: true, Address: targetHost, Text: "Some Error", Rbl: rblList}},
	}
}
