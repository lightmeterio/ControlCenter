package insights

import (
	"context"
	"database/sql"
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"log"
	"sync"
	"testing"
	"time"
)

var (
	dummyContext = context.Background()
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

type content struct {
	V string `json:"v"`
}

func (c content) String() string {
	return c.V
}

func (c content) TplString() string {
	return c.V
}

func (c content) Args() []interface{} {
	return nil
}

type fakeNotificationCenter struct {
	notifications []notification.Notification
}

func (f *fakeNotificationCenter) Notify(n notification.Notification) error {
	f.notifications = append(f.notifications, n)
	return nil
}

func (f *fakeNotificationCenter) AddSlackNotifier(notificationsSettings settings.SlackNotificationsSettings) error {
	return nil
}

type fakeValue struct {
	Category core.Category
	Rating   core.Rating
	Content  core.Content
}

type fakeDetector struct {
	// added just to silent the race detector during tests
	sync.Mutex
	creator   *creator
	fakeValue *fakeValue
}

func (d *fakeDetector) value() *fakeValue {
	d.Lock()
	defer d.Unlock()
	return d.fakeValue
}

func (d *fakeDetector) setValue(v *fakeValue) {
	d.Lock()
	defer d.Unlock()
	d.fakeValue = v
}

func (*fakeDetector) Close() error {
	return nil
}

func (d *fakeDetector) Setup(*sql.Tx) error {
	return nil
}

func (d *fakeDetector) GenerateSampleInsight(tx *sql.Tx, clock core.Clock) error {
	return d.creator.GenerateInsight(tx, core.InsightProperties{
		Time:        clock.Now(),
		Category:    core.IntelCategory,
		ContentType: "fake_insight_type",
		Content:     &content{V: "hi"},
		Rating:      core.BadRating,
	})
}

func init() {
	core.RegisterContentType("fake_insight_type", 200, func(b []byte) (core.Content, error) {
		var v content

		if err := json.Unmarshal(b, &v); err != nil {
			return nil, err
		}

		return &v, nil
	})
}

func (d *fakeDetector) Step(clock core.Clock, tx *sql.Tx) error {
	p := d.value()

	if p == nil {
		return nil
	}

	v := *p

	log.Println("New Fake Insight at time", clock.Now())

	if err := d.creator.GenerateInsight(tx, core.InsightProperties{
		Time:        clock.Now(),
		Category:    v.Category,
		ContentType: "fake_insight_type",
		Content:     v.Content,
		Rating:      v.Rating,
	}); err != nil {
		return err
	}

	d.setValue(nil)

	return nil
}

func TestEngine(t *testing.T) {
	Convey("Test Insights Generator", t, func() {
		dir, clearDir := testutil.TempDir()
		defer clearDir()

		nc := &fakeNotificationCenter{}

		detector := &fakeDetector{}

		noAdditionalActions := func([]core.Detector, dbconn.RwConn) error { return nil }

		Convey("Test Insights Generation", func() {
			e, err := NewCustomEngine(dir, nc, core.Options{}, func(c *creator, o core.Options) []core.Detector {
				detector.creator = c
				return []core.Detector{detector}
			}, noAdditionalActions)

			So(err, ShouldBeNil)

			defer func() {
				So(e.Close(), ShouldBeNil)
			}()

			doneWithRun := make(chan struct{})

			go func() {
				runDatabaseWriterLoop(e)
				doneWithRun <- struct{}{}
			}()

			clock := &insighttestsutil.FakeClock{Time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)}

			step := func(v *fakeValue) {
				if v != nil {
					detector.setValue(v)
				}

				execOnDetectors(e.txActions, e.core.Detectors, clock)
				time.Sleep(time.Millisecond * 100)
				clock.Sleep(time.Second * 1)
			}

			genInsight := func(v fakeValue) {
				step(&v)
			}

			nopStep := func() {
				step(nil)
			}

			nopStep()
			nopStep()
			genInsight(fakeValue{Category: core.LocalCategory, Content: content{"42"}, Rating: core.BadRating})
			nopStep()
			nopStep()
			genInsight(fakeValue{Category: core.IntelCategory, Content: content{"35"}, Rating: core.BadRating})
			nopStep()
			genInsight(fakeValue{Category: core.ComparativeCategory, Content: content{"13"}, Rating: core.BadRating})

			// stop main loop
			close(e.txActions)

			_, ok := <-doneWithRun

			So(ok, ShouldBeTrue)

			So(nc.notifications, ShouldResemble, []notification.Notification{
				{ID: 1, Content: content{"42"}},
				{ID: 2, Content: content{"35"}},
				{ID: 3, Content: content{"13"}},
			})

			fetcher := e.Fetcher()

			Convey("fetch all insights with no filter, sorting by time, default (desc) order", func() {
				insights, err := fetcher.FetchInsights(dummyContext, core.FetchOptions{
					Interval: data.TimeInterval{
						From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`2000-01-01 22:00:00 +0000`),
					},
				})

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 3)

				So(insights[0].Category(), ShouldEqual, core.ComparativeCategory)
				So(insights[0].Content().(*content).V, ShouldEqual, "13")
				So(insights[0].ID(), ShouldEqual, 3)
				So(insights[0].Rating(), ShouldEqual, core.BadRating)
				So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:07 +0000`))

				So(insights[1].Category(), ShouldEqual, core.IntelCategory)
				So(insights[1].Content().(*content).V, ShouldEqual, "35")
				So(insights[1].ID(), ShouldEqual, 2)
				So(insights[1].Rating(), ShouldEqual, core.BadRating)
				So(insights[1].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:05 +0000`))

				So(insights[2].Category(), ShouldEqual, core.LocalCategory)
				So(insights[2].Content().(*content).V, ShouldEqual, "42")
				So(insights[2].ID(), ShouldEqual, 1)
				So(insights[2].Rating(), ShouldEqual, core.BadRating)
				So(insights[2].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:02 +0000`))
			})

			Convey("fetch 2 most recent insights", func() {
				insights, err := fetcher.FetchInsights(dummyContext, core.FetchOptions{
					Interval: data.TimeInterval{
						From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`2000-01-01 22:00:00 +0000`),
					},
					MaxEntries: 2,
				})

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 2)

				So(insights[0].Category(), ShouldEqual, core.ComparativeCategory)
				So(insights[0].Content().(*content).V, ShouldEqual, "13")
				So(insights[0].ID(), ShouldEqual, 3)
				So(insights[0].Rating(), ShouldEqual, core.BadRating)
				So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:07 +0000`))

				So(insights[1].Category(), ShouldEqual, core.IntelCategory)
				So(insights[1].Content().(*content).V, ShouldEqual, "35")
				So(insights[1].ID(), ShouldEqual, 2)
				So(insights[1].Rating(), ShouldEqual, core.BadRating)
				So(insights[1].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:05 +0000`))
			})

			Convey("fetch all insights with no filter, sorting by time, asc order", func() {
				insights, err := fetcher.FetchInsights(dummyContext, core.FetchOptions{
					Interval: data.TimeInterval{
						From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`2000-01-01 22:00:00 +0000`),
					},
					OrderBy: core.OrderByCreationAsc,
				})

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 3)

				So(insights[0].Category(), ShouldEqual, core.LocalCategory)
				So(insights[0].Content().(*content).V, ShouldEqual, "42")
				So(insights[0].ID(), ShouldEqual, 1)
				So(insights[0].Rating(), ShouldEqual, core.BadRating)
				So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:02 +0000`))

				So(insights[1].Category(), ShouldEqual, core.IntelCategory)
				So(insights[1].Content().(*content).V, ShouldEqual, "35")
				So(insights[1].ID(), ShouldEqual, 2)
				So(insights[1].Rating(), ShouldEqual, core.BadRating)
				So(insights[1].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:05 +0000`))

				So(insights[2].Category(), ShouldEqual, core.ComparativeCategory)
				So(insights[2].Content().(*content).V, ShouldEqual, "13")
				So(insights[2].ID(), ShouldEqual, 3)
				So(insights[2].Rating(), ShouldEqual, core.BadRating)
				So(insights[2].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:07 +0000`))
			})

			Convey("fetch intel category, asc order", func() {
				insights, err := fetcher.FetchInsights(dummyContext, core.FetchOptions{
					Interval: data.TimeInterval{
						From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`2000-01-01 22:00:00 +0000`),
					},
					OrderBy:  core.OrderByCreationAsc,
					FilterBy: core.FilterByCategory,
					Category: core.IntelCategory,
				})

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 1)

				So(insights[0].Category(), ShouldEqual, core.IntelCategory)
				So(insights[0].Content().(*content).V, ShouldEqual, "35")
				So(insights[0].ID(), ShouldEqual, 2)
				So(insights[0].Rating(), ShouldEqual, core.BadRating)
				So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:05 +0000`))
			})
		})

		Convey("Test Insights Samples generated when the application starts", func() {
			e, err := NewCustomEngine(dir, nc, core.Options{}, func(c *creator, o core.Options) []core.Detector {
				detector.creator = c
				return []core.Detector{detector}
			},
				addInsightsSamples,
			)

			So(err, ShouldBeNil)

			fetcher := e.Fetcher()

			sampleInsights, err := fetcher.FetchInsights(dummyContext, core.FetchOptions{
				Interval: data.TimeInterval{
					From: testutil.MustParseTime("0000-01-01 00:00:00 +0000"),
					To:   testutil.MustParseTime("4000-01-01 00:00:00 +0000"),
				},
			})

			So(err, ShouldBeNil)

			So(len(sampleInsights), ShouldEqual, 1)
		})

		Convey("Test engine loop", func() {
			e, err := NewCustomEngine(dir, nc, core.Options{}, func(c *creator, o core.Options) []core.Detector {
				detector.creator = c
				return []core.Detector{detector}
			},
				noAdditionalActions,
			)

			So(err, ShouldBeNil)

			defer func() {
				So(e.Close(), ShouldBeNil)
			}()

			// Generate one insight, on the first cycle
			detector.setValue(&fakeValue{Category: core.LocalCategory, Content: content{"content"}, Rating: core.BadRating})

			done, cancel := e.Run()

			time.Sleep(time.Second * 3)

			cancel()
			done()

			So(nc.notifications, ShouldResemble, []notification.Notification{
				{ID: 1, Content: content{"content"}},
			})
		})
	})
}
