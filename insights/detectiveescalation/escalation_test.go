// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package detectiveescalation

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/detective"
	"gitlab.com/lightmeter/controlcenter/detective/escalator"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/notification"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

var (
	dummyContext = context.Background()
)

type fakeRequests map[time.Time]escalator.Request

type fakeEscalator struct {
	clock    core.Clock
	requests fakeRequests
}

func (d *fakeEscalator) Step(withResult func(escalator.Request) error, withoutResult func() error) error {
	if r, ok := d.requests[d.clock.Now()]; ok {
		return withResult(r)
	}

	return withoutResult()
}

func parseTimeInterval(from, to string) timeutil.TimeInterval {
	i, err := timeutil.ParseTimeInterval(from, to, time.UTC)
	So(err, ShouldBeNil)
	return i
}

func TestDetectiveEscalation(t *testing.T) {
	Convey("Test Detective Escalation", t, func() {
		accessor, clearAccessor := insighttestsutil.NewFakeAccessor(t)
		defer clearAccessor()

		baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)
		clock := &insighttestsutil.FakeClock{Time: baseTime}

		requests := map[time.Time]escalator.Request{
			baseTime.Add(time.Second * 4): escalator.Request{
				Sender:    "sender1@example.com",
				Recipient: "recipient1@example.com",
				Interval:  parseTimeInterval("2000-02-03", "2000-02-04"),
				Messages: detective.Messages{
					"BBB": []detective.MessageDelivery{
						{
							NumberOfAttempts: 30,
							TimeMin:          timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
							TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
							Status:           detective.Status(parser.DeferredStatus),
							Dsn:              "2.0.0",
						},
						{
							NumberOfAttempts: 1,
							TimeMin:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
							TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
							Status:           detective.Status(parser.ExpiredStatus),
							Dsn:              "3.0.0",
						},
					},
				},
			},
			baseTime.Add(time.Second * 10): escalator.Request{
				Sender:    "sender2@example.com",
				Recipient: "recipient2@example.com",
				Interval:  parseTimeInterval("2000-05-04", "2000-05-04"),
				Messages: detective.Messages{
					"BBB": []detective.MessageDelivery{
						{
							NumberOfAttempts: 1,
							TimeMin:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
							TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
							Status:           detective.Status(parser.BouncedStatus),
							Dsn:              "3.0.0",
						},
					},
				},
			},
		}

		escalator := &fakeEscalator{clock: clock, requests: requests}

		detector := NewDetector(accessor, core.Options{
			"detective": Options{
				Escalator: escalator,
			},
		})

		executeCyclesUntil := func(end time.Time, stepDuration time.Duration) {
			insighttestsutil.ExecuteCyclesUntil(detector, accessor, clock, end, stepDuration)
		}

		executeCyclesUntil(testutil.MustParseTime(`2000-01-01 00:30:00 +0000`), time.Second*2)

		So(accessor.Insights, ShouldResemble, []int64{1, 2})

		insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: timeutil.TimeInterval{
			From: testutil.MustParseTime(`0000-01-01 00:00:00 +0000`),
			To:   testutil.MustParseTime(`4000-01-01 00:00:00 +0000`),
		},
			OrderBy: core.OrderByCreationAsc,
		})

		So(err, ShouldBeNil)

		{
			So(insights[0].ID(), ShouldEqual, 1)
			So(insights[0].ContentType(), ShouldEqual, ContentType)
			So(insights[0].Time(), ShouldEqual, baseTime.Add(time.Second*4))
			So(insights[0].Rating(), ShouldEqual, core.Unrated)
			So(insights[0].Category(), ShouldEqual, core.LocalCategory)

			So(insights[0].Content(), ShouldResemble, &Content{
				Sender:    "sender1@example.com",
				Recipient: "recipient1@example.com",
				Interval:  parseTimeInterval("2000-02-03", "2000-02-04"),
				Messages: detective.Messages{
					"BBB": []detective.MessageDelivery{
						{
							NumberOfAttempts: 30,
							TimeMin:          timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
							TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
							Status:           detective.Status(parser.DeferredStatus),
							Dsn:              "2.0.0",
						},
						{
							NumberOfAttempts: 1,
							TimeMin:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
							TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
							Status:           detective.Status(parser.ExpiredStatus),
							Dsn:              "3.0.0",
						},
					},
				},
			})
		}

		{
			So(insights[1].ID(), ShouldEqual, 2)
			So(insights[1].ContentType(), ShouldEqual, ContentType)
			So(insights[1].Time(), ShouldEqual, baseTime.Add(time.Second*10))
			So(insights[1].Rating(), ShouldEqual, core.Unrated)
			So(insights[1].Category(), ShouldEqual, core.LocalCategory)

			So(insights[1].Content(), ShouldResemble, &Content{
				Sender:    "sender2@example.com",
				Recipient: "recipient2@example.com",
				Interval:  parseTimeInterval("2000-05-04", "2000-05-04"),
				Messages: detective.Messages{
					"BBB": []detective.MessageDelivery{
						{
							NumberOfAttempts: 1,
							TimeMin:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
							TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
							Status:           detective.Status(parser.BouncedStatus),
							Dsn:              "3.0.0",
						},
					},
				},
			})
		}
	})
}

func TestDescriptionFormatting(t *testing.T) {
	Convey("Description Formatting", t, func() {
		n := notification.Notification{
			ID: 1,
			Content: Content{
				Sender:    "sender2@example.com",
				Recipient: "recipient2@example.com",
				Interval:  parseTimeInterval("2000-05-04", "2000-05-04"),
				Messages: detective.Messages{
					"BBB": []detective.MessageDelivery{
						{
							NumberOfAttempts: 1,
							TimeMin:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
							TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
							Status:           detective.Status(parser.BouncedStatus),
							Dsn:              "3.0.0",
						},
					},
				},
			},
		}

		m, err := notificationCore.TranslateNotification(n, translator.DummyTranslator{})
		So(err, ShouldBeNil)
		So(m, ShouldResemble, notificationCore.Message{
			Title:       "User request on non delivered message",
			Description: "Sender: sender2@example.com, recipient: recipient2@example.com",
			Metadata:    map[string]string{},
		})
	})
}
