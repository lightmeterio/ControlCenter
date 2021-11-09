// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package messagerblinsight

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/messagerbl"
	"gitlab.com/lightmeter/controlcenter/notification"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	parsertimeutil "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/timeutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"net"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

var (
	dummyContext = context.Background()
)

type fakeActions map[time.Time]func() messagerbl.Result

type fakeChecker struct {
	clock   core.Clock
	actions fakeActions
}

func (d *fakeChecker) Step(withResults func([]messagerbl.Results) error) error {
	results := messagerbl.Results{}

	if action, ok := d.actions[d.clock.Now()]; ok {
		results.Values[0] = action()
		results.Size = 1
	}

	return withResults([]messagerbl.Results{results})
}

func (d *fakeChecker) IPAddress(context.Context) net.IP {
	return net.ParseIP("127.0.0.2")
}

func parseLogLine(line string) (parser.Header, parser.Payload) {
	h, p, err := parser.Parse(line)
	So(err, ShouldBeNil)
	return h, p
}

func actionFromLog(converter *parsertimeutil.TimeConverter, host string, line string) func() messagerbl.Result {
	return func() messagerbl.Result {
		h, p := parseLogLine(line)

		return messagerbl.Result{
			Address: net.ParseIP("127.0.0.2"),
			Host:    host,
			Header:  h,
			Payload: p.(parser.SmtpSentStatus),
			Time:    converter.Convert(h.Time),
		}
	}
}

func TestMessageRBLInsight(t *testing.T) {
	Convey("Test Message RBL Insight", t, func() {
		accessor, clearAccessor := insighttestsutil.NewFakeAccessor(t)
		defer clearAccessor()

		baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)
		converter := parsertimeutil.NewTimeConverter(baseTime, func(int, parser.Time, parser.Time) {})
		clock := &insighttestsutil.FakeClock{Time: baseTime}

		actions := map[time.Time]func() messagerbl.Result{
			// this will generate an insight from Host 1
			baseTime.Add(time.Second * 2): actionFromLog(&converter, "Host 1", `Jan  1 00:00:02 node postfix/smtp[12357]: 375593D395: to=<recipient@example.com>, relay=relay.example.com[254.112.150.90]:25, delay=0.86, delays=0.1/0/0.71/0.05, dsn=5.7.606, status=bounced (Error for Host 1 insight 1)`),

			// this will generate an insight from Host 2
			baseTime.Add(time.Second * 30): actionFromLog(&converter, "Host 2", `Jan  1 00:00:30 node postfix/smtp[12357]: 375593D395: to=<recipient@example2.com>, relay=relay.example.com[254.112.150.90]:25, delay=0.86, delays=0.1/0/0.71/0.05, dsn=5.7.606, status=deferred (Error for Host 2 insight 1)`),

			// this will NOT generate an insight from Host 1, as the there was another from Host 1 only 48s ago
			baseTime.Add(time.Second * 50): actionFromLog(&converter, "Host 1", `Jan  1 00:00:50 node postfix/smtp[12357]: 375593D395: to=<recipient@example.com>, relay=relay.example.com[254.112.150.90]:25, delay=0.86, delays=0.1/0/0.71/0.05, dsn=5.7.606, status=bounced (Error for Host 1 no insight)`),

			// this will generate an insight from Host 1, as the time since last generated insight for it was 70s ago
			baseTime.Add(time.Second * 72): actionFromLog(&converter, "Host 1", `Jan  1 00:01:12 node postfix/smtp[12357]: 375593D395: to=<recipient@example.com>, relay=relay.example.com[254.112.150.90]:25, delay=0.86, delays=0.1/0/0.71/0.05, dsn=5.7.606, status=bounced (Error for Host 1 insight 2)`),
		}

		checker := &fakeChecker{clock: clock, actions: actions}

		detector := NewDetector(accessor, core.Options{
			"messagerbl": Options{
				Detector:                    checker,
				MinTimeToGenerateNewInsight: time.Second * 70,
			},
		})

		executeCyclesUntil := func(end time.Time, stepDuration time.Duration) {
			insighttestsutil.ExecuteCyclesUntil(detector, accessor, clock, end, stepDuration)
		}

		executeCyclesUntil(testutil.MustParseTime(`2000-01-01 00:30:00 +0000`), time.Second*2)

		So(accessor.Insights, ShouldResemble, []int64{1, 2, 3})

		insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: timeutil.TimeInterval{
			From: testutil.MustParseTime(`0000-01-01 00:00:00 +0000`),
			To:   testutil.MustParseTime(`4000-01-01 00:00:00 +0000`),
		},
			OrderBy: core.OrderByCreationAsc,
		}, clock)

		So(err, ShouldBeNil)

		{
			So(insights[0].ID(), ShouldEqual, 1)
			So(insights[0].ContentType(), ShouldEqual, ContentType)
			So(insights[0].Time(), ShouldEqual, baseTime.Add(time.Second*2))
			So(insights[0].Rating(), ShouldEqual, core.BadRating)
			So(insights[0].Category(), ShouldEqual, core.LocalCategory)

			c, ok := insights[0].Content().(*Content)
			So(ok, ShouldBeTrue)

			So(c.Host, ShouldEqual, "Host 1")
			So(c.Message, ShouldEqual, "(Error for Host 1 insight 1)")
			So(c.Status, ShouldEqual, "bounced")
			So(c.Recipient, ShouldEqual, "example.com")
			So(c.Time, ShouldEqual, baseTime.Add(time.Second*2))
		}

		{
			So(insights[1].ID(), ShouldEqual, 2)
			So(insights[1].ContentType(), ShouldEqual, ContentType)
			So(insights[1].Time(), ShouldEqual, baseTime.Add(time.Second*30))
			So(insights[1].Rating(), ShouldEqual, core.BadRating)
			So(insights[1].Category(), ShouldEqual, core.LocalCategory)

			c, ok := insights[1].Content().(*Content)
			So(ok, ShouldBeTrue)

			So(*c, ShouldResemble, Content{
				Host:      "Host 2",
				Address:   net.ParseIP("127.0.0.2"),
				Message:   "(Error for Host 2 insight 1)",
				Status:    "deferred",
				Recipient: "example2.com",
				Time:      baseTime.Add(time.Second * 30),
			})
		}

		{
			So(insights[2].ID(), ShouldEqual, 3)
			So(insights[2].ContentType(), ShouldEqual, ContentType)
			So(insights[2].Time(), ShouldEqual, baseTime.Add(time.Second*72))
			So(insights[2].Rating(), ShouldEqual, core.BadRating)
			So(insights[2].Category(), ShouldEqual, core.LocalCategory)

			c, ok := insights[2].Content().(*Content)
			So(ok, ShouldBeTrue)

			So(*c, ShouldResemble, Content{
				Host:      "Host 1",
				Address:   net.ParseIP("127.0.0.2"),
				Message:   "(Error for Host 1 insight 2)",
				Status:    "bounced",
				Recipient: "example.com",
				Time:      baseTime.Add(time.Second * 72),
			})
		}
	})
}

func TestRegression(t *testing.T) {
	Convey("Clock is in the past, where requests are more recent", t, func() {
		// This happens during historical import, in case the "historical clock" starts much
		// earlier than the point in time the
		accessor, clearAccessor := insighttestsutil.NewFakeAccessor(t)
		defer clearAccessor()

		baseTime := testutil.MustParseTime(`2021-04-17 00:00:00 +0000`)
		converter := parsertimeutil.NewTimeConverter(baseTime, func(int, parser.Time, parser.Time) {})
		clock := &insighttestsutil.FakeClock{Time: baseTime}

		actions := map[time.Time]func() messagerbl.Result{
			// the clock is "behind" the log time, so the log time should be used for the insight instead
			testutil.MustParseTime(`2021-04-17 05:00:00 +0000`): actionFromLog(&converter, "Host 1", `Apr 17 06:26:00 node postfix/smtp[12357]: 375593D395: to=<recipient@example.com>, relay=relay.example.com[254.112.150.90]:25, delay=0.86, delays=0.1/0/0.71/0.05, dsn=5.7.606, status=bounced (Error for Host 1 insight 1)`),
		}

		checker := &fakeChecker{clock: clock, actions: actions}

		detector := NewDetector(accessor, core.Options{
			"messagerbl": Options{
				Detector:                    checker,
				MinTimeToGenerateNewInsight: time.Second * 60,
			},
		})

		executeCyclesUntil := func(end time.Time, stepDuration time.Duration) {
			insighttestsutil.ExecuteCyclesUntil(detector, accessor, clock, end, stepDuration)
		}

		executeCyclesUntil(testutil.MustParseTime(`2021-04-30 00:30:00 +0000`), time.Minute*1)

		So(accessor.Insights, ShouldResemble, []int64{1})

		insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: timeutil.TimeInterval{
			From: testutil.MustParseTime(`0000-01-01 00:00:00 +0000`),
			To:   testutil.MustParseTime(`4000-01-01 00:00:00 +0000`),
		},
			OrderBy: core.OrderByCreationAsc,
		}, clock)

		So(err, ShouldBeNil)

		{
			So(insights[0].ID(), ShouldEqual, 1)
			So(insights[0].ContentType(), ShouldEqual, ContentType)
			So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2021-04-17 06:26:00 +0000`))
			So(insights[0].Rating(), ShouldEqual, core.BadRating)
			So(insights[0].Category(), ShouldEqual, core.LocalCategory)
		}
	})
}

func TestDescriptionFormatting(t *testing.T) {
	Convey("Description Formatting", t, func() {
		n := notification.Notification{
			ID: 1,
			Content: Content{
				Address:   net.ParseIP(`127.0.0.1`),
				Recipient: "lala@caca.com",
				Host:      "google",
				Status:    "bounced",
				Message:   "some message",
				Time:      testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
			},
		}

		m, err := notificationCore.TranslateNotification(n, translator.DummyTranslator{})
		So(err, ShouldBeNil)
		So(m, ShouldResemble, notificationCore.Message{
			Title:       "IP blocked by google",
			Description: "The IP 127.0.0.1 cannot deliver to lala@caca.com (google)",
			Metadata:    map[string]string{},
		})
	})
}
