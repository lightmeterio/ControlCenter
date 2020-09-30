package insights

import (
	"database/sql"
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"log"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

type fakeNotificationCenter struct {
	notifications []notification.Content
}

func (f *fakeNotificationCenter) Notify(n notification.Content) {
	f.notifications = append(f.notifications, n)
}

type fakeValue struct {
	Category core.Category
	Rating   core.Rating
	Content  string
}

type fakeDetector struct {
	creator *creator
	v       *fakeValue
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
		Content:     "hi",
		Rating:      core.BadRating,
	})
}

func init() {
	core.RegisterContentType("fake_insight_type", 200, func(b []byte) (interface{}, error) {
		var v string

		if err := json.Unmarshal(b, &v); err != nil {
			return nil, err
		}

		return &v, nil
	})
}

func (d *fakeDetector) Steppers() []core.Stepper {
	return []core.Stepper{d}
}

func (d *fakeDetector) Step(clock core.Clock, tx *sql.Tx) error {
	if d.v == nil {
		return nil
	}

	v := *d.v

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

	d.v = nil

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

			errChan := make(chan error)

			go func() {
				errChan <- runDatabaseWriterLoop(e)
			}()

			clock := &insighttestsutil.FakeClock{Time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)}

			step := func(v *fakeValue) {
				if v != nil {
					detector.v = v
				}
				execOnSteppers(e.txActions, e.core.Steppers, clock)
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
			genInsight(fakeValue{Category: core.LocalCategory, Content: "42", Rating: core.BadRating})
			nopStep()
			nopStep()
			genInsight(fakeValue{Category: core.IntelCategory, Content: "35", Rating: core.GoodRating})
			nopStep()
			genInsight(fakeValue{Category: core.ComparativeCategory, Content: "13", Rating: core.Unrated})

			// stop main loop
			close(e.txActions)

			err, ok := <-errChan

			So(ok, ShouldBeTrue)
			So(err, ShouldBeNil)

			So(nc.notifications, ShouldResemble, []notification.Content{
				InsightNotification{ID: 1},
				InsightNotification{ID: 2},
				InsightNotification{ID: 3},
			})

			fetcher := e.Fetcher()

			Convey("fetch all insights with no filter, sorting by time, default (desc) order", func() {
				insights, err := fetcher.FetchInsights(core.FetchOptions{
					Interval: data.TimeInterval{
						From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`2000-01-01 22:00:00 +0000`),
					},
				})

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 3)

				So(insights[0].Category(), ShouldEqual, core.ComparativeCategory)
				So(*insights[0].Content().(*string), ShouldEqual, "13")
				So(insights[0].ID(), ShouldEqual, 3)
				So(insights[0].Rating(), ShouldEqual, core.Unrated)
				So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:07 +0000`))

				So(insights[1].Category(), ShouldEqual, core.IntelCategory)
				So(*insights[1].Content().(*string), ShouldEqual, "35")
				So(insights[1].ID(), ShouldEqual, 2)
				So(insights[1].Rating(), ShouldEqual, core.GoodRating)
				So(insights[1].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:05 +0000`))

				So(insights[2].Category(), ShouldEqual, core.LocalCategory)
				So(*insights[2].Content().(*string), ShouldEqual, "42")
				So(insights[2].ID(), ShouldEqual, 1)
				So(insights[2].Rating(), ShouldEqual, core.BadRating)
				So(insights[2].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:02 +0000`))
			})

			Convey("fetch 2 most recent insights", func() {
				insights, err := fetcher.FetchInsights(core.FetchOptions{
					Interval: data.TimeInterval{
						From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`2000-01-01 22:00:00 +0000`),
					},
					MaxEntries: 2,
				})

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 2)

				So(insights[0].Category(), ShouldEqual, core.ComparativeCategory)
				So(*insights[0].Content().(*string), ShouldEqual, "13")
				So(insights[0].ID(), ShouldEqual, 3)
				So(insights[0].Rating(), ShouldEqual, core.Unrated)
				So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:07 +0000`))

				So(insights[1].Category(), ShouldEqual, core.IntelCategory)
				So(*insights[1].Content().(*string), ShouldEqual, "35")
				So(insights[1].ID(), ShouldEqual, 2)
				So(insights[1].Rating(), ShouldEqual, core.GoodRating)
				So(insights[1].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:05 +0000`))
			})

			Convey("fetch all insights with no filter, sorting by time, asc order", func() {
				insights, err := fetcher.FetchInsights(core.FetchOptions{
					Interval: data.TimeInterval{
						From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`2000-01-01 22:00:00 +0000`),
					},
					OrderBy: core.OrderByCreationAsc,
				})

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 3)

				So(insights[0].Category(), ShouldEqual, core.LocalCategory)
				So(*insights[0].Content().(*string), ShouldEqual, "42")
				So(insights[0].ID(), ShouldEqual, 1)
				So(insights[0].Rating(), ShouldEqual, core.BadRating)
				So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:02 +0000`))

				So(insights[1].Category(), ShouldEqual, core.IntelCategory)
				So(*insights[1].Content().(*string), ShouldEqual, "35")
				So(insights[1].ID(), ShouldEqual, 2)
				So(insights[1].Rating(), ShouldEqual, core.GoodRating)
				So(insights[1].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:05 +0000`))

				So(insights[2].Category(), ShouldEqual, core.ComparativeCategory)
				So(*insights[2].Content().(*string), ShouldEqual, "13")
				So(insights[2].ID(), ShouldEqual, 3)
				So(insights[2].Rating(), ShouldEqual, core.Unrated)
				So(insights[2].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:07 +0000`))
			})

			Convey("fetch intel category, asc order", func() {
				insights, err := fetcher.FetchInsights(core.FetchOptions{
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
				So(*insights[0].Content().(*string), ShouldEqual, "35")
				So(insights[0].ID(), ShouldEqual, 2)
				So(insights[0].Rating(), ShouldEqual, core.GoodRating)
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

			sampleInsights, err := fetcher.FetchInsights(core.FetchOptions{
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
			detector.v = &fakeValue{Category: core.LocalCategory, Content: "content", Rating: core.BadRating}

			done, cancel := e.Run()

			time.Sleep(time.Second * 3)

			cancel()
			done()

			So(nc.notifications, ShouldResemble, []notification.Content{
				InsightNotification{ID: 1},
			})
		})
	})
}
