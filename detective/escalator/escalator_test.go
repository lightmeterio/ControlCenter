// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package escalator

import (
	"context"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/detective"
	mock_detective "gitlab.com/lightmeter/controlcenter/detective/mock"
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

func mustParseTimeInterval(from, to string) timeutil.TimeInterval {
	i, err := timeutil.ParseTimeInterval(from, to, time.UTC)
	So(err, ShouldBeNil)
	return i
}

func TestEscalation(t *testing.T) {
	Convey("Test Escalation", t, func() {
		ctrl := gomock.NewController(t)
		d := mock_detective.NewMockDetective(ctrl)
		requester := &fakeEscalateRequester{}
		ctx := context.Background()

		Convey("No detective results. Do not escalate", func() {
			d.EXPECT().CheckMessageDelivery(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]detective.MessageDelivery{}, nil)
			err := TryToEscalateRequest(ctx, d, requester, "sender@example.com", "recipient@example.com", mustParseTimeInterval("2000-01-01", "2000-01-01"))
			So(err, ShouldBeNil)
			So(len(requester.requests), ShouldEqual, 0)
		})

		Convey("All messages were delived. Do not escalate", func() {
			interval := mustParseTimeInterval("2000-01-01", "2000-01-01")

			d.EXPECT().CheckMessageDelivery(gomock.Any(), "sender@example.com", "recipient@example.com", interval).Return([]detective.MessageDelivery{
				detective.MessageDelivery{
					Time:   timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
					Status: "sent",
					Dsn:    "2.0.0",
				},
				detective.MessageDelivery{
					Time:   timeutil.MustParseTime(`2000-01-01 12:00:00 +0000`),
					Status: "sent",
					Dsn:    "2.0.0",
				},
			}, nil)

			err := TryToEscalateRequest(ctx, d, requester, "sender@example.com", "recipient@example.com", interval)
			So(err, ShouldBeNil)
			So(len(requester.requests), ShouldEqual, 0)
		})

		Convey("Any of the messages was not delivered. Escalate one issue", func() {
			interval := mustParseTimeInterval("2000-01-01", "2000-01-01")

			d.EXPECT().CheckMessageDelivery(gomock.Any(), "sender@example.com", "recipient@example.com", interval).Return([]detective.MessageDelivery{
				detective.MessageDelivery{
					Time:   timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
					Status: "sent",
					Dsn:    "2.0.0",
				},
				detective.MessageDelivery{
					Time:   timeutil.MustParseTime(`2000-01-01 11:00:00 +0000`),
					Status: "bounced",
					Dsn:    "4.7.1",
				},
				detective.MessageDelivery{
					Time:   timeutil.MustParseTime(`2000-01-01 12:00:00 +0000`),
					Status: "sent",
					Dsn:    "2.0.0",
				},
			}, nil)

			err := TryToEscalateRequest(ctx, d, requester, "sender@example.com", "recipient@example.com", interval)
			So(err, ShouldBeNil)
			So(requester.requests, ShouldResemble, []Request{
				Request{Sender: "sender@example.com", Recipient: "recipient@example.com", Interval: interval},
			})
		})

	})
}
