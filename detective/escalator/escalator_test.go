// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package escalator

import (
	"context"
	"errors"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/detective"
	mock_detective "gitlab.com/lightmeter/controlcenter/detective/mock"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"testing"
	"time"
)

type fakeEscalateRequester struct {
	requests []Request
}

func (e *fakeEscalateRequester) Request(r Request) {
	e.requests = append(e.requests, r)
}

var mustParseTimeInterval = timeutil.MustParseTimeInterval

func TestEscalation(t *testing.T) {
	Convey("Test Escalation", t, func() {
		ctrl := gomock.NewController(t)
		d := mock_detective.NewMockDetective(ctrl)
		requester := &fakeEscalateRequester{}
		ctx := context.Background()

		Convey("No detective results. Do not escalate", func() {
			d.EXPECT().CheckMessageDelivery(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&detective.MessagesPage{}, nil)
			err := TryToEscalateRequest(ctx, d, requester, "sender@example.com", "recipient@example.com", mustParseTimeInterval("2000-01-01", "2000-01-01"), "")
			So(err, ShouldBeNil)
			So(len(requester.requests), ShouldEqual, 0)
		})

		Convey("All messages were delived. Do not escalate", func() {
			interval := mustParseTimeInterval("2000-01-01", "2000-01-01")

			d.EXPECT().CheckMessageDelivery(gomock.Any(), "sender@example.com", "recipient@example.com", interval, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				&detective.MessagesPage{
					PageNumber:   1,
					FirstPage:    1,
					LastPage:     1,
					TotalResults: 2,
					Messages: detective.Messages{
						detective.Message{
							Queue: "AAAA",
							Entries: []detective.MessageDelivery{
								// initially deferred, but ultimately delivered. No issues here.
								{
									NumberOfAttempts: 4,
									TimeMin:          timeutil.MustParseTime(`2000-01-01 09:00:00 +0000`),
									TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
									Status:           detective.Status(parser.DeferredStatus),
									Dsn:              "2.0.0",
								},

								{
									NumberOfAttempts: 1,
									TimeMin:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
									TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
									Status:           detective.Status(parser.SentStatus),
									Dsn:              "2.0.0",
								},
							},
						},
						detective.Message{
							Queue: "BBBB",
							Entries: []detective.MessageDelivery{
								{
									NumberOfAttempts: 1,
									TimeMin:          timeutil.MustParseTime(`2000-01-01 12:00:00 +0000`),
									TimeMax:          timeutil.MustParseTime(`2000-01-01 12:00:00 +0000`),
									Status:           detective.Status(parser.SentStatus),
									Dsn:              "2.0.0",
								},
							},
						},
					},
				}, nil)

			err := TryToEscalateRequest(ctx, d, requester, "sender@example.com", "recipient@example.com", interval, "")
			So(err, ShouldBeNil)
			So(len(requester.requests), ShouldEqual, 0)
		})

		Convey("Any of the messages was not delivered. Escalate one issue", func() {
			interval := mustParseTimeInterval("2000-01-01", "2000-01-01")

			d.EXPECT().CheckMessageDelivery(gomock.Any(), "sender@example.com", "recipient@example.com", interval, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				&detective.MessagesPage{
					PageNumber:   1,
					FirstPage:    1,
					LastPage:     1,
					TotalResults: 3,
					Messages: detective.Messages{
						detective.Message{
							Queue: "AAA",
							Entries: []detective.MessageDelivery{
								{
									NumberOfAttempts: 1,
									TimeMin:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
									TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
									Status:           detective.Status(parser.SentStatus),
									Dsn:              "2.0.0",
								},
							},
						},
						detective.Message{
							Queue: "BBB",
							Entries: []detective.MessageDelivery{
								{
									NumberOfAttempts: 1,
									TimeMin:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
									TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
									Status:           detective.Status(parser.BouncedStatus),
									Dsn:              "3.0.0",
								},
							},
						},
						detective.Message{
							Queue: "CCC",
							Entries: []detective.MessageDelivery{
								{
									NumberOfAttempts: 1,
									TimeMin:          timeutil.MustParseTime(`2000-01-01 12:00:00 +0000`),
									TimeMax:          timeutil.MustParseTime(`2000-01-01 12:00:00 +0000`),
									Status:           detective.Status(parser.SentStatus),
									Dsn:              "2.0.0",
								},
							},
						},
					},
				}, nil)

			err := TryToEscalateRequest(ctx, d, requester, "sender@example.com", "recipient@example.com", interval, "")
			So(err, ShouldBeNil)
			So(requester.requests, ShouldResemble, []Request{
				{
					Sender:    "sender@example.com",
					Recipient: "recipient@example.com",
					Interval:  interval,
					Messages: detective.Messages{
						detective.Message{
							Queue: "BBB",
							Entries: []detective.MessageDelivery{
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
				},
			})
		})

		Convey("Escalate by queue name", func() {
			interval := mustParseTimeInterval("2000-01-01", "2000-01-01")

			d.EXPECT().CheckMessageDelivery(gomock.Any(), "", "", interval, gomock.Any(), "BBB", gomock.Any(), gomock.Any()).Return(
				&detective.MessagesPage{
					PageNumber:   1,
					FirstPage:    1,
					LastPage:     1,
					TotalResults: 1,
					Messages: detective.Messages{
						detective.Message{
							Queue: "BBB",
							Entries: []detective.MessageDelivery{
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
				}, nil)

			err := TryToEscalateRequest(ctx, d, requester, "", "", interval, "BBB")
			So(err, ShouldBeNil)
			So(requester.requests, ShouldResemble, []Request{
				{
					Sender:    "",
					Recipient: "",
					Interval:  interval,
					Messages: detective.Messages{
						detective.Message{
							Queue: "BBB",
							Entries: []detective.MessageDelivery{
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
				},
			})
		})

		Convey("Use the real requester", func() {
			requester := New()

			interval := mustParseTimeInterval("2000-01-01", "2000-01-01")

			d.EXPECT().CheckMessageDelivery(gomock.Any(), "sender@example.com", "recipient@example.com", interval, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				&detective.MessagesPage{
					PageNumber:   1,
					FirstPage:    1,
					LastPage:     1,
					TotalResults: 3,
					Messages: detective.Messages{
						detective.Message{
							Queue: "AAA",
							Entries: []detective.MessageDelivery{
								{
									NumberOfAttempts: 1,
									TimeMin:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
									TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
									Status:           detective.Status(parser.SentStatus),
									Dsn:              "2.0.0",
								},
							},
						},
						detective.Message{
							Queue: "ZZZ",
							Entries: []detective.MessageDelivery{
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
						detective.Message{
							Queue: "DDD",
							Entries: []detective.MessageDelivery{
								{
									NumberOfAttempts: 1,
									TimeMin:          timeutil.MustParseTime(`2000-01-01 12:00:00 +0000`),
									TimeMax:          timeutil.MustParseTime(`2000-01-01 12:00:00 +0000`),
									Status:           detective.Status(parser.SentStatus),
									Dsn:              "2.0.0",
								},
							},
						},
						detective.Message{
							Queue: "CCC",
							Entries: []detective.MessageDelivery{
								{
									NumberOfAttempts: 1,
									TimeMin:          timeutil.MustParseTime(`2000-01-01 14:00:00 +0000`),
									TimeMax:          timeutil.MustParseTime(`2000-01-01 14:00:00 +0000`),
									Status:           detective.Status(parser.BouncedStatus),
									Dsn:              "4.0.0",
								},
							},
						},
					},
				}, nil)

			requestsChan := make(chan Request)

			go func() {
				stopErr := errors.New(`stop now!`)
				for {
					err := requester.Step(func(r Request) error {
						requestsChan <- r
						return stopErr
					}, func() error { return nil })

					if err == stopErr {
						close(requestsChan)
						return
					}
				}
			}()

			time.Sleep(100 * time.Millisecond)

			err := TryToEscalateRequest(ctx, d, requester, "sender@example.com", "recipient@example.com", interval, "")
			So(err, ShouldBeNil)

			requests := []Request{}

			for r := range requestsChan {
				requests = append(requests, r)
			}

			So(requests, ShouldResemble, []Request{
				{
					Sender:    "sender@example.com",
					Recipient: "recipient@example.com",
					Interval:  interval,
					Messages: detective.Messages{
						detective.Message{
							Queue: "ZZZ",
							Entries: []detective.MessageDelivery{
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
						detective.Message{
							Queue: "CCC",
							Entries: []detective.MessageDelivery{
								{
									NumberOfAttempts: 1,
									TimeMin:          timeutil.MustParseTime(`2000-01-01 14:00:00 +0000`),
									TimeMax:          timeutil.MustParseTime(`2000-01-01 14:00:00 +0000`),
									Status:           detective.Status(parser.BouncedStatus),
									Dsn:              "4.0.0",
								},
							},
						},
					},
				},
			})
		})
	})
}
