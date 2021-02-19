// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package localrblinsight

import (
	"context"
	"fmt"
	"github.com/mrichman/godnsbl"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/localrbl"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"net"
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
	scanFailed                     bool
	scanIsInProgress               bool
	scanResultsShouldChange        bool
	scanCount                      int
	t *testing.T
}

func (c *fakeChecker) Close() error {
	return nil
}

func (c *fakeChecker) StartListening() {
	So(c.listening, ShouldBeFalse)
	c.listening = true
}

func (c *fakeChecker) NotifyNewScan(now time.Time) {
	c.t.Logf("New Scan Executed at time %v", now)
	c.scanStartTime = now
	c.scanFailed = c.checkedIP == nil
	c.scanIsInProgress = true
	c.scanCount++
	So(c.listening, ShouldBeTrue)
}

func (c *fakeChecker) Step(now time.Time, withResults func(localrbl.Results) error, withoutResults func() error) error {
	timeWhenScanWillFinish := c.scanStartTime.Add(c.timeToCompleteScan)
	scanHasEnded := now.After(timeWhenScanWillFinish)

	hasResultsOfASuccessfulScan := !c.scanFailed &&
		c.scanIsInProgress &&
		scanHasEnded &&
		c.shouldGenerateOnScanCompletion

	hasResultsOfAFailedScan := c.scanFailed && c.scanIsInProgress

	// determines when scan stopped
	c.scanIsInProgress = !scanHasEnded

	if hasResultsOfASuccessfulScan {
		c.scanFailed = false

		url := func() string {
			if c.scanResultsShouldChange {
				return fmt.Sprintf("%d.some.other.checker.de", c.scanCount)
			}

			return "some.rbl.checker.com"
		}()

		r := withResults(localrbl.Results{
			Interval: data.TimeInterval{From: c.scanStartTime, To: now},
			RBLs:     []localrbl.ContentElement{{RBL: url, Text: "Something Really Bad"}},
		})

		return r
	}

	if hasResultsOfAFailedScan {
		c.scanFailed = false

		return withResults(localrbl.Results{
			Err: localrbl.ErrIPNotConfigured,
		})
	}

	return withoutResults()
}

func (c *fakeChecker) IPAddress(context.Context) net.IP {
	return c.checkedIP
}

func TestLocalRBL(t *testing.T) {
	Convey("Test Local RBL", t, func() {
		accessor, clear := insighttestsutil.NewFakeAccessor(t)
		defer clear()

		connPair := accessor.ConnPair

		cycleOnDetector := func(d core.Detector, c *insighttestsutil.FakeClock) {
			tx, err := connPair.RwConn.Begin()
			So(err, ShouldBeNil)
			So(d.Step(c, tx), ShouldBeNil)
			So(tx.Commit(), ShouldBeNil)
		}

		ip := net.ParseIP("11.22.33.44")

		Convey("Use Fake Checker", func() {
			checker := &fakeChecker{
				timeToCompleteScan: time.Second * 10,
				t: t,
			}

			d := NewDetector(accessor, core.Options{
				"localrbl": Options{
					CheckInterval:               time.Second * 20,
					Checker:                     checker,
					RetryOnScanErrorInterval:    time.Second * 3,
					MinTimeToGenerateNewInsight: time.Second * 52,
				},
			})

			checker.StartListening()

			defer func() {
				d.(*detector).Close()
			}()

			// TODO: refactor behaviour of cycle to make it move forward in time
			// executing many cycles in in between, according to the specified interval
			// so that we can simulate something closer to the real behaviour of
			// the insights engine and duplicate less test code
			cycle := func(c *insighttestsutil.FakeClock) {
				cycleOnDetector(d, c)
			}

			baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)
			clock := &insighttestsutil.FakeClock{Time: baseTime}

			Convey("Missing IP address", func() {
				// when application starts without a configured IP and a scan fails, the next scan will be scheduled
				// sooner, so that the user does not need to wait a long time to see a scan being executed
				checker.checkedIP = nil
				checker.shouldGenerateOnScanCompletion = true

				// trigger a scan, but it fails as IP is not yet configured
				cycle(clock)

				// After two seconds, nothing really happens
				clock.Sleep(time.Second * 2)
				cycle(clock)

				// The user configures the IP address
				checker.checkedIP = ip

				clock.Sleep(time.Second * 1)
				cycle(clock)

				// With a correct IP address, a new scan starts shortly after the previous one failed
				clock.Sleep(time.Second * 1)
				cycle(clock)

				// scan in progress
				clock.Sleep(time.Second * 3)
				cycle(clock)

				clock.Sleep(time.Second * 5)
				cycle(clock)

				clock.Sleep(time.Second * 5)
				cycle(clock)

				// scan finished, insight created
				clock.Sleep(time.Second * 1)
				cycle(clock)

				So(accessor.Insights, ShouldResemble, []int64{1})

				insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: data.TimeInterval{
					From: testutil.MustParseTime(`0000-01-01 00:00:00 +0000`),
					To:   testutil.MustParseTime(`4000-01-01 00:00:00 +0000`),
				}})

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 1)

				So(insights[0].ContentType(), ShouldEqual, ContentType)
				So(insights[0].Time(), ShouldEqual, baseTime.Add(time.Second*18))
			})

			Convey("IP Configured", func() {
				checker.checkedIP = ip

				Convey("Scan host, but generates no insight", func() {
					// do not generate insight, but trigger it to start
					cycle(clock)

					// On time 5s, the scan has't finished yet
					clock.Sleep(time.Second * 5)
					cycle(clock)

					// On time 9s, the scan has't finished yet
					clock.Sleep(time.Second * 4)
					cycle(clock)

					// On time 10s, the scan is finished, but no insight'll have been created
					clock.Sleep(time.Second * 4)
					cycle(clock)

					So(accessor.Insights, ShouldResemble, []int64{})
				})

				Convey("Scan host, and generate insight", func() {
					checker.shouldGenerateOnScanCompletion = true

					// do not generate insight, but trigger it to start
					cycle(clock)

					// On time 5s, the scan has't finished yet
					clock.Sleep(time.Second * 5)
					cycle(clock)

					// On time 9s, the scan hasn't finished yet
					clock.Sleep(time.Second * 4)
					cycle(clock)

					// On time 11s, the scan is finished, resulting in an insight generated
					clock.Sleep(time.Second * 2)
					cycle(clock)

					Convey("A new scan happens, but does not generate insight as the content is equal the previous insight", func() {
						// On time 15s, nothing happens
						clock.Sleep(time.Second * 4)
						cycle(clock)

						// On time 21s, a new scan is triggered
						clock.Sleep(time.Second * 5)
						cycle(clock)

						// On time 25s, nothing happens
						clock.Sleep(time.Second * 4)
						cycle(clock)

						// On time 29s, nothing happens
						clock.Sleep(time.Second * 4)
						cycle(clock)

						clock.Sleep(time.Second * 5)
						cycle(clock)

						clock.Sleep(time.Second * 5)
						cycle(clock)

						clock.Sleep(time.Second * 10)
						cycle(clock)

						clock.Sleep(time.Second * 10)
						cycle(clock)

						clock.Sleep(time.Second * 10)
						cycle(clock)

						clock.Sleep(time.Second * 20)
						cycle(clock)

						clock.Sleep(time.Second * 20)
						cycle(clock)

						insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: data.TimeInterval{
							From: testutil.MustParseTime(`0000-01-01 00:00:00 +0000`),
							To:   testutil.MustParseTime(`4000-01-01 00:00:00 +0000`),
						}, OrderBy: core.OrderByCreationAsc})

						So(err, ShouldBeNil)

						So(len(insights), ShouldEqual, 2)

						// first insight is generated as there was no previous one to compare with
						So(insights[0].ID(), ShouldEqual, 1)
						So(insights[0].ContentType(), ShouldEqual, ContentType)
						So(insights[0].Time(), ShouldEqual, baseTime.Add(time.Second*11))
						So(insights[0].Content(), ShouldResemble, &content{
							ScanInterval: data.TimeInterval{From: baseTime, To: baseTime.Add(time.Second * 11)},
							Address:      net.ParseIP("11.22.33.44"),
							RBLs:         []localrbl.ContentElement{{RBL: "some.rbl.checker.com", Text: "Something Really Bad"}},
						})

						// this insight is generated with the same content as it's generated after
						// MinTimeToGenerateNewInsight after the first one
						So(insights[1].ID(), ShouldEqual, 2)
						So(insights[1].ContentType(), ShouldEqual, ContentType)
						So(insights[1].Time(), ShouldEqual, baseTime.Add(time.Second*68))
						So(insights[1].Content(), ShouldResemble, &content{
							ScanInterval: data.TimeInterval{From: baseTime.Add(time.Second * 48), To: baseTime.Add(time.Second * 68)},
							Address:      net.ParseIP("11.22.33.44"),
							RBLs:         []localrbl.ContentElement{{RBL: "some.rbl.checker.com", Text: "Something Really Bad"}},
						})
					})

					Convey("A new scan happens, with different contents, generating a new insight", func() {
						checker.scanResultsShouldChange = true

						// On time 15s, nothing happens
						clock.Sleep(time.Second * 4)
						cycle(clock)

						// On time 21s, a new scan is triggered
						clock.Sleep(time.Second * 5)
						cycle(clock)

						// On time 25s, nothing happens
						clock.Sleep(time.Second * 4)
						cycle(clock)

						// On time 29s, nothing happens
						clock.Sleep(time.Second * 4)
						cycle(clock)

						clock.Sleep(time.Second * 3)
						cycle(clock)

						clock.Sleep(time.Second * 3)
						cycle(clock)

						clock.Sleep(time.Second * 3)
						cycle(clock)

						insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: data.TimeInterval{
							From: testutil.MustParseTime(`0000-01-01 00:00:00 +0000`),
							To:   testutil.MustParseTime(`4000-01-01 00:00:00 +0000`),
						}, OrderBy: core.OrderByCreationAsc})

						So(err, ShouldBeNil)

						So(len(insights), ShouldEqual, 2)

						// generated due no previous insight to compare with
						So(insights[0].ID(), ShouldEqual, 1)
						So(insights[0].ContentType(), ShouldEqual, ContentType)
						So(insights[0].Time(), ShouldEqual, baseTime.Add(time.Second*11))
						So(insights[0].Content(), ShouldResemble, &content{
							ScanInterval: data.TimeInterval{From: baseTime, To: baseTime.Add(time.Second * 11)},
							Address:      net.ParseIP("11.22.33.44"),
							RBLs:         []localrbl.ContentElement{{RBL: "some.rbl.checker.com", Text: "Something Really Bad"}},
						})

						// generated as its content differs from the content from the previous insight
						// meaning it provides some useful information to the sysadmin without spammimg them
						So(insights[1].ID(), ShouldEqual, 2)
						So(insights[1].ContentType(), ShouldEqual, ContentType)
						So(insights[1].Time(), ShouldEqual, baseTime.Add(time.Second*37))
						So(insights[1].Content(), ShouldResemble, &content{
							ScanInterval: data.TimeInterval{From: baseTime.Add(time.Second * 24), To: baseTime.Add(time.Second * 37)},
							Address:      net.ParseIP("11.22.33.44"),
							RBLs:         []localrbl.ContentElement{{RBL: "2.some.other.checker.de", Text: "Something Really Bad"}},
						})
					})
				})

				// TODO: move this test to its own test function
				Convey("Use real DNS checker", func() {
					conn, closeConn := testutil.TempDBConnection(t)
					defer closeConn()

					m, err := meta.NewHandler(conn, "master")
					So(err, ShouldBeNil)

					defer func() { errorutil.MustSucceed(m.Close()) }()

					{
						settings := globalsettings.Settings{
							LocalIP: net.ParseIP("11.22.33.44"),
						}

						err := m.Writer.StoreJson(dummyContext, globalsettings.SettingKey, &settings)

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
