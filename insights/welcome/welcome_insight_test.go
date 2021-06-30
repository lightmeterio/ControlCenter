// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package welcome

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/notification"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"testing"
	"time"
)

var (
	dummyContext = context.Background()
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestWelcomeInsights(t *testing.T) {
	Convey("Test Welcome Generator", t, func() {
		accessor, clear := insighttestsutil.NewFakeAccessor(t)
		defer clear()

		connPair := accessor.ConnPair

		Convey("Insight is generated only once", func() {
			clock := &insighttestsutil.FakeClock{Time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 24)}

			detector := NewDetector(accessor)

			cycle := func(c *insighttestsutil.FakeClock) {
				tx, err := connPair.RwConn.Begin()
				So(err, ShouldBeNil)
				So(detector.Step(c, tx), ShouldBeNil)
				So(tx.Commit(), ShouldBeNil)
			}

			// generate insights only once, in the first cycle
			for i := 0; i < 10; i++ {
				cycle(clock)
				clock.Sleep(time.Hour * 8)
			}

			So(accessor.Insights, ShouldResemble, []int64{1, 2})

			insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: timeutil.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 24).Add(time.Hour * 24),
			}, OrderBy: core.OrderByCreationAsc}, clock)

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 2)

			So(insights[0].ID(), ShouldEqual, 1)
			So(insights[0].ContentType(), ShouldEqual, "welcome_content")
			So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour*24))
			So(insights[0].Content(), ShouldResemble, &content{})

			So(insights[1].ID(), ShouldEqual, 2)
			So(insights[1].ContentType(), ShouldEqual, "insights_introduction_content")
			So(insights[1].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour*24))
			So(insights[1].Content(), ShouldResemble, &content{})
		})
	})
}

func TestDescriptionFormatting(t *testing.T) {
	Convey("Description Formatting", t, func() {
		n := notification.Notification{
			ID:      1,
			Content: content{},
		}

		m, err := notificationCore.TranslateNotification(n, translator.DummyTranslator{})
		So(err, ShouldBeNil)
		So(m, ShouldResemble, notificationCore.Message{
			Title:       "",
			Description: "",
			Metadata:    map[string]string{},
		})
	})
}
