// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	"context"
	"database/sql"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/insights/importsummary"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/notification"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"golang.org/x/text/language"
	"golang.org/x/text/message/catalog"
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

type fakeContent struct {
	T string `json:"title"`
	D string `json:"description"`
}

func (c fakeContent) Title() notificationCore.ContentComponent {
	return fakeContentComponent(c.T)
}

func (c fakeContent) Description() notificationCore.ContentComponent {
	return fakeContentComponent(c.D)
}

func (c fakeContent) Metadata() notificationCore.ContentMetadata {
	return nil
}

type fakeContentComponent string

func (c fakeContentComponent) String() string {
	return string(c)
}

func (c fakeContentComponent) TplString() string {
	return string(c)
}

func (c fakeContentComponent) Args() []interface{} {
	return nil
}

type fakeNotifier struct {
	notifications []notification.Notification
}

func (*fakeNotifier) ValidateSettings(notificationCore.Settings) error {
	return nil
}

func (f *fakeNotifier) Notify(n notification.Notification, _ translator.Translator) error {
	f.notifications = append(f.notifications, n)
	return nil
}

type fakeValue struct {
	Category core.Category
	Rating   core.Rating
	Content  core.Content
}

type fakeDetector struct {
	t *testing.T
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

func (fakeDetector) IsHistoricalDetector() {
}

func (*fakeDetector) Close() error {
	return nil
}

func (d *fakeDetector) Setup(*sql.Tx) error {
	return nil
}

func (d *fakeDetector) GenerateSampleInsight(tx *sql.Tx, clock core.Clock) error {
	return d.creator.GenerateInsight(context.Background(), tx, core.InsightProperties{
		Time:        clock.Now(),
		Category:    core.IntelCategory,
		ContentType: "fake_insight_type",
		Content:     &fakeContent{T: "title", D: "description"},
		Rating:      core.BadRating,
	})
}

func init() {
	core.RegisterContentType("fake_insight_type", 200, core.DefaultContentTypeDecoder(&fakeContent{}))
}

func (d *fakeDetector) Step(clock core.Clock, tx *sql.Tx) error {
	p := d.value()

	if p == nil {
		return nil
	}

	v := *p

	d.t.Log("New Fake Insight at time ", clock.Now())

	if err := d.creator.GenerateInsight(context.Background(), tx, core.InsightProperties{
		Time:        clock.Now(),
		Category:    v.Category,
		ContentType: "fake_insight_type",
		Content:     v.Content,
		Rating:      v.Rating,
	}); err != nil {
		return err
	}

	// reset value to prevent it of being generated on every cycle
	d.setValue(nil)

	return nil
}

func TestEngine(t *testing.T) {
	Convey("Test Insights Generator", t, func() {
		dir, clearDir := testutil.TempDir(t)
		defer clearDir()

		c, err := NewAccessor(dir)
		So(err, ShouldBeNil)

		notifier := &fakeNotifier{}

		nc := notification.NewWithCustomLanguageFetcher(translator.New(catalog.NewBuilder()), c.NotificationPolicy(), func() (language.Tag, error) {
			return language.English, nil
		}, map[string]notification.Notifier{"fake": notifier})

		detector := &fakeDetector{t: t}

		noAdditionalActions := func([]core.Detector, dbconn.RwConn, core.Clock) error { return nil }

		Convey("Test Insights Generation", func() {
			e, err := NewCustomEngine(c, nc, core.Options{}, func(c *creator, o core.Options) []core.Detector {
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
			genInsight(fakeValue{Category: core.LocalCategory, Content: fakeContent{T: "42"}, Rating: core.BadRating})
			nopStep()
			nopStep()
			genInsight(fakeValue{Category: core.IntelCategory, Content: fakeContent{T: "35"}, Rating: core.OkRating})
			nopStep()
			genInsight(fakeValue{Category: core.ComparativeCategory, Content: fakeContent{T: "13"}, Rating: core.BadRating})

			fakeClock := &timeutil.FakeClock{Time: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)}

			// wrong rating value
			err = e.RateInsight("fake_insight_type", 1000, fakeClock)
			So(err, ShouldEqual, core.ErrWrongRatingValue)
			nopStep()

			// correct rating
			err = e.RateInsight("fake_insight_type", 1, fakeClock)
			So(err, ShouldBeNil)
			nopStep()

			// allowed to re-rate after two weeks
			fakeClock.Sleep(core.TwoWeeks + 5*time.Second)

			err = e.RateInsight("fake_insight_type", 1, fakeClock)
			So(err, ShouldBeNil)
			nopStep()

			// not allowed to re-rate immediately
			err = e.RateInsight("fake_insight_type", 0, fakeClock)
			So(err, ShouldEqual, core.ErrAlreadyRated)
			nopStep()

			// stop main loop
			close(e.txActions)

			_, ok := <-doneWithRun

			So(ok, ShouldBeTrue)

			// Notify only bad-rating insights
			So(len(notifier.notifications), ShouldEqual, 2)

			{
				n, ok := notifier.notifications[0].Content.(core.InsightProperties)
				So(ok, ShouldBeTrue)
				So(notifier.notifications[0].ID, ShouldEqual, 1)
				So(n.Content, ShouldResemble, fakeContent{T: "42"})
			}

			{
				n, ok := notifier.notifications[1].Content.(core.InsightProperties)
				So(ok, ShouldBeTrue)
				So(notifier.notifications[1].ID, ShouldEqual, 3)
				So(n.Content, ShouldResemble, fakeContent{T: "13"})
			}

			fetcher := e.Fetcher()

			Convey("fetch all insights with no filter, sorting by time, default (desc) order", func() {
				insights, err := fetcher.FetchInsights(dummyContext, core.FetchOptions{
					Interval: timeutil.TimeInterval{
						From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`2000-01-01 22:00:00 +0000`),
					},
				}, fakeClock)

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 3)

				So(insights[0].Category(), ShouldEqual, core.ComparativeCategory)
				So(insights[0].Content().(*fakeContent).T, ShouldEqual, "13")
				So(insights[0].ID(), ShouldEqual, 3)
				So(insights[0].Rating(), ShouldEqual, core.BadRating)
				So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:07 +0000`))
				So(*insights[0].UserRating(), ShouldEqual, 1)
				So(insights[0].UserRatingOld(), ShouldBeFalse)

				So(insights[1].Category(), ShouldEqual, core.IntelCategory)
				So(insights[1].Content().(*fakeContent).T, ShouldEqual, "35")
				So(insights[1].ID(), ShouldEqual, 2)
				So(insights[1].Rating(), ShouldEqual, core.OkRating)
				So(insights[1].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:05 +0000`))
				So(*insights[1].UserRating(), ShouldEqual, 1)
				So(insights[1].UserRatingOld(), ShouldBeFalse)

				So(insights[2].Category(), ShouldEqual, core.LocalCategory)
				So(insights[2].Content().(*fakeContent).T, ShouldEqual, "42")
				So(insights[2].ID(), ShouldEqual, 1)
				So(insights[2].Rating(), ShouldEqual, core.BadRating)
				So(insights[2].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:02 +0000`))
				So(*insights[2].UserRating(), ShouldEqual, 1)
				So(insights[2].UserRatingOld(), ShouldBeFalse)
			})

			Convey("fetch 2 most recent insights", func() {
				insights, err := fetcher.FetchInsights(dummyContext, core.FetchOptions{
					Interval: timeutil.TimeInterval{
						From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`2000-01-01 22:00:00 +0000`),
					},
					MaxEntries: 2,
				}, fakeClock)

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 2)

				So(insights[0].Category(), ShouldEqual, core.ComparativeCategory)
				So(insights[0].Content().(*fakeContent).T, ShouldEqual, "13")
				So(insights[0].ID(), ShouldEqual, 3)
				So(insights[0].Rating(), ShouldEqual, core.BadRating)
				So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:07 +0000`))

				So(insights[1].Category(), ShouldEqual, core.IntelCategory)
				So(insights[1].Content().(*fakeContent).T, ShouldEqual, "35")
				So(insights[1].ID(), ShouldEqual, 2)
				So(insights[1].Rating(), ShouldEqual, core.OkRating)
				So(insights[1].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:05 +0000`))
			})

			Convey("fetch all insights with no filter, sorting by time, asc order", func() {
				insights, err := fetcher.FetchInsights(dummyContext, core.FetchOptions{
					Interval: timeutil.TimeInterval{
						From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`2000-01-01 22:00:00 +0000`),
					},
					OrderBy: core.OrderByCreationAsc,
				}, fakeClock)

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 3)

				So(insights[0].Category(), ShouldEqual, core.LocalCategory)
				So(insights[0].Content().(*fakeContent).T, ShouldEqual, "42")
				So(insights[0].ID(), ShouldEqual, 1)
				So(insights[0].Rating(), ShouldEqual, core.BadRating)
				So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:02 +0000`))

				So(insights[1].Category(), ShouldEqual, core.IntelCategory)
				So(insights[1].Content().(*fakeContent).T, ShouldEqual, "35")
				So(insights[1].ID(), ShouldEqual, 2)
				So(insights[1].Rating(), ShouldEqual, core.OkRating)
				So(insights[1].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:05 +0000`))

				So(insights[2].Category(), ShouldEqual, core.ComparativeCategory)
				So(insights[2].Content().(*fakeContent).T, ShouldEqual, "13")
				So(insights[2].ID(), ShouldEqual, 3)
				So(insights[2].Rating(), ShouldEqual, core.BadRating)
				So(insights[2].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:07 +0000`))
			})

			Convey("fetch intel category, asc order", func() {
				insights, err := fetcher.FetchInsights(dummyContext, core.FetchOptions{
					Interval: timeutil.TimeInterval{
						From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`2000-01-01 22:00:00 +0000`),
					},
					OrderBy:  core.OrderByCreationAsc,
					FilterBy: core.FilterByCategory,
					Category: core.IntelCategory,
				}, fakeClock)

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 1)

				So(insights[0].Category(), ShouldEqual, core.IntelCategory)
				So(insights[0].Content().(*fakeContent).T, ShouldEqual, "35")
				So(insights[0].ID(), ShouldEqual, 2)
				So(insights[0].Rating(), ShouldEqual, core.OkRating)
				So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:05 +0000`))
			})
		})

		Convey("Test Insights Samples generated when the application starts", func() {
			e, err := NewCustomEngine(c, nc, core.Options{}, func(c *creator, o core.Options) []core.Detector {
				detector.creator = c
				return []core.Detector{detector}
			},
				addInsightsSamples,
			)

			So(err, ShouldBeNil)

			announcer.Skip(e.ImportAnnouncer())

			done, cancel := e.Run()

			time.Sleep(100 * time.Millisecond)

			cancel()
			So(done(), ShouldBeNil)

			fetcher := e.Fetcher()

			sampleInsights, err := fetcher.FetchInsights(dummyContext, core.FetchOptions{
				Interval: timeutil.TimeInterval{
					From: testutil.MustParseTime("0000-01-01 00:00:00 +0000"),
					To:   testutil.MustParseTime("4000-01-01 00:00:00 +0000"),
				},
			}, timeutil.RealClock{})

			So(err, ShouldBeNil)

			// 1 + the import summary insight
			So(len(sampleInsights), ShouldEqual, 2)
		})

		Convey("Test engine loop", func() {
			e, err := NewCustomEngine(c, nc, core.Options{}, func(c *creator, o core.Options) []core.Detector {
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
			detector.setValue(&fakeValue{Category: core.LocalCategory, Content: fakeContent{D: "content"}, Rating: core.BadRating})

			announcer.Skip(e.ImportAnnouncer())

			done, cancel := e.Run()

			time.Sleep(100 * time.Millisecond)

			cancel()
			So(done(), ShouldBeNil)

			So(len(notifier.notifications), ShouldEqual, 1)

			n, ok := notifier.notifications[0].Content.(core.InsightProperties)
			So(ok, ShouldBeTrue)
			So(n.Content, ShouldResemble, fakeContent{D: "content"})
		})

		Convey("Skip historical import", func() {
			e, err := NewCustomEngine(c, nc, core.Options{}, func(c *creator, o core.Options) []core.Detector {
				detector.creator = c
				return []core.Detector{detector}
			},
				noAdditionalActions,
			)

			So(err, ShouldBeNil)

			defer func() {
				So(e.Close(), ShouldBeNil)
			}()

			progressFetcher := e.ProgressFetcher()

			// Skip import
			announcer.Skip(e.ImportAnnouncer())

			done, cancel := e.Run()

			time.Sleep(100 * time.Millisecond)

			cancel()
			So(done(), ShouldBeNil)

			progress, err := progressFetcher.Progress(context.Background())
			So(err, ShouldBeNil)

			So(*progress.Value, ShouldEqual, 100)
			So(progress.Active, ShouldBeFalse)
		})

		Convey("Test importing Historical insights", func() {
			e, err := NewCustomEngine(c, nc, core.Options{}, func(c *creator, o core.Options) []core.Detector {
				detector.creator = c
				return []core.Detector{detector}
			},
				noAdditionalActions,
			)

			So(err, ShouldBeNil)

			defer func() {
				So(e.Close(), ShouldBeNil)
			}()

			progressFetcher := e.ProgressFetcher()

			{
				// importing hasn't started yet. No progress made
				p, err := progressFetcher.Progress(dummyContext)
				So(err, ShouldBeNil)
				So(p.Active, ShouldBeFalse)
			}

			control := make(chan struct{})

			go func() {
				notifier := announcer.NewNotifier(e.ImportAnnouncer(), 10)

				<-control
				notifier.Start(timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`))

				// first step
				<-control
				notifier.Step(timeutil.MustParseTime(`2000-01-15 00:00:00 +0000`))

				// second step
				<-control
				notifier.Step(timeutil.MustParseTime(`2000-01-20 00:00:00 +0000`))

				<-control
				notifier.End(timeutil.MustParseTime(`2000-01-31 23:59:59 +0000`))
			}()

			errChan := make(chan error)

			go func() {
				errChan <- runOnHistoricalData(e)
			}()

			// unlock Start()
			control <- struct{}{}

			// Generate one insight, still during import. It'll be set as archived upon creation time
			detector.setValue(&fakeValue{Category: core.LocalCategory, Content: fakeContent{D: "I am historical, therefore archived"}, Rating: core.BadRating})
			control <- struct{}{}

			{
				time.Sleep(50 * time.Millisecond)
				p, err := progressFetcher.Progress(dummyContext)
				So(err, ShouldBeNil)
				So(p.Active, ShouldBeTrue)
				So(*p.Value, ShouldEqual, 10)
				So(*p.Time, ShouldResemble, timeutil.MustParseTime(`2000-01-15 00:00:00 +0000`))
			}

			// Unlock the first step
			control <- struct{}{}

			{
				time.Sleep(50 * time.Millisecond)
				p, err := progressFetcher.Progress(dummyContext)
				So(err, ShouldBeNil)
				So(p.Active, ShouldBeTrue)
				So(*p.Value, ShouldEqual, 20)
				So(*p.Time, ShouldResemble, timeutil.MustParseTime(`2000-01-20 00:00:00 +0000`))
			}

			control <- struct{}{}

			// wait until historical import ends
			err = <-errChan

			So(err, ShouldBeNil)

			{
				// Import has finished
				p, err := progressFetcher.Progress(dummyContext)
				So(err, ShouldBeNil)
				So(p.Active, ShouldBeFalse)
				So(*p.Value, ShouldEqual, 100)
				So(*p.Time, ShouldResemble, timeutil.MustParseTime(`2000-01-31 23:59:59 +0000`))
			}

			mainLoopChain := make(chan struct{})

			go func() {
				<-control
				runDatabaseWriterLoop(e)
				mainLoopChain <- struct{}{}
			}()

			// generate a non historical insight
			detector.setValue(&fakeValue{Category: core.LocalCategory, Content: fakeContent{D: "A non historical insight"}, Rating: core.BadRating})
			control <- struct{}{}

			execOnDetectors(e.txActions, []core.Detector{detector}, &timeutil.FakeClock{Time: timeutil.MustParseTime(`2000-02-05 00:00:00 +0000`)})

			// stop main loop
			close(e.txActions)

			<-mainLoopChain

			So(len(notifier.notifications), ShouldEqual, 1)
			So(notifier.notifications[0].ID, ShouldEqual, 3)
			So(notifier.notifications[0].Content.Description(), ShouldEqual, "A non historical insight")

			Convey("Get active (non archived) insights", func() {
				insights, err := e.Fetcher().FetchInsights(context.Background(), core.FetchOptions{
					Interval: timeutil.TimeInterval{
						From: testutil.MustParseTime(`0000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`4000-01-01 00:00:00 +0000`),
					},
					OrderBy:  core.OrderByCreationAsc,
					FilterBy: core.FilterByCategory,
					Category: core.ActiveCategory,
				}, timeutil.RealClock{})

				So(err, ShouldBeNil)

				// one fakeInsight and one summary insight
				So(len(insights), ShouldEqual, 2)

				So(insights[0].Content().Description().String(), ShouldEqual, "A non historical insight")

				So(insights[1].Content().Title().String(), ShouldEqual, "Imported insights")
				So(insights[1].Content().Description().String(), ShouldEqual, "Mail activity imported successfully Events since 2000-01-01 00:00:00 +0000 UTC were analysed, producing 1 Insights")

				summary, ok := insights[1].Content().(*importsummary.Content)
				So(ok, ShouldBeTrue)
				expected := []importsummary.ImportedInsight{
					{
						ID:       1,
						Time:     timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
						Category: core.ArchivedCategory,
						Rating:   core.BadRating,
						Content: map[string]interface{}{
							"description": "I am historical, therefore archived",
							"title":       "",
						},
						ContentType: "fake_insight_type",
					}}
				So(summary.Insights, ShouldResemble, expected)
			})

			Convey("Get archived insights", func() {
				insights, err := e.Fetcher().FetchInsights(context.Background(), core.FetchOptions{
					Interval: timeutil.TimeInterval{
						From: testutil.MustParseTime(`0000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`4000-01-01 00:00:00 +0000`),
					},
					OrderBy:  core.OrderByCreationAsc,
					FilterBy: core.FilterByCategory,
					Category: core.ArchivedCategory,
				}, timeutil.RealClock{})

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 1)

				So(insights[0].Content().Description().String(), ShouldEqual, "I am historical, therefore archived")
				So(insights[0].Category(), ShouldEqual, core.ArchivedCategory)
			})

			Convey("Get all insights (archived and active)", func() {
				insights, err := e.Fetcher().FetchInsights(context.Background(), core.FetchOptions{
					Interval: timeutil.TimeInterval{
						From: testutil.MustParseTime(`0000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`4000-01-01 00:00:00 +0000`),
					},
					OrderBy: core.OrderByCreationAsc,
				}, timeutil.RealClock{})

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 3)

				So(insights[0].Content().Description().String(), ShouldEqual, "I am historical, therefore archived")
				So(insights[0].Category(), ShouldEqual, core.ArchivedCategory)

				So(insights[1].Content().Description().String(), ShouldEqual, "A non historical insight")
				So(insights[1].Category(), ShouldEqual, core.LocalCategory)

				So(insights[2].Content().Description().String(), ShouldEqual, "Mail activity imported successfully Events since 2000-01-01 00:00:00 +0000 UTC were analysed, producing 1 Insights")
				So(insights[2].Category(), ShouldEqual, core.LocalCategory)
			})

			Convey("Choosing a category should exclude the archived insights", func() {
				insights, err := e.Fetcher().FetchInsights(context.Background(), core.FetchOptions{
					Interval: timeutil.TimeInterval{
						From: testutil.MustParseTime(`0000-01-01 00:00:00 +0000`),
						To:   testutil.MustParseTime(`4000-01-01 00:00:00 +0000`),
					},
					OrderBy:  core.OrderByCreationAsc,
					FilterBy: core.FilterByCategory,
					Category: core.LocalCategory,
				}, timeutil.RealClock{})

				So(err, ShouldBeNil)

				So(len(insights), ShouldEqual, 2)

				So(insights[0].Content().Description().String(), ShouldEqual, "A non historical insight")
				So(insights[0].Category(), ShouldEqual, core.LocalCategory)

				So(insights[1].Content().Description().String(), ShouldEqual, "Mail activity imported successfully Events since 2000-01-01 00:00:00 +0000 UTC were analysed, producing 1 Insights")
				So(insights[1].Category(), ShouldEqual, core.LocalCategory)
			})

		})
	})
}
