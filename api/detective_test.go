// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/detective"
	"gitlab.com/lightmeter/controlcenter/detective/escalator"
	mock_detective "gitlab.com/lightmeter/controlcenter/detective/mock"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/emailutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

const limit = detective.ResultsPerPage

func TestDetectiveCheckMessageDeliveryHandler(t *testing.T) {
	Convey("CheckMessageDeliveryHandler", t, func() {
		ctrl := gomock.NewController(t)

		defer ctrl.Finish()

		m := mock_detective.NewMockDetective(ctrl)

		chain := httpmiddleware.New(httpmiddleware.RequestWithInterval(time.UTC))

		interval, err := timeutil.ParseTimeInterval("1999-01-01", "1999-12-31", time.UTC)
		So(err, ShouldBeNil)

		s := httptest.NewServer(chain.WithEndpoint(checkMessageDeliveryHandler{detective: m}))

		Convey("No Parameters", func() {
			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnprocessableEntity)
		})

		Convey("No Time Interval", func() {
			r, err := http.Get(fmt.Sprintf("%s?mail_from=user1@example.org&mail_to=user2@example.org&status=-1&some_id=&page=1", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnprocessableEntity)
		})

		emptyResult := detective.MessagesPage{}

		Convey("No Sender", func() {
			m.EXPECT().CheckMessageDelivery(gomock.Any(), "", "user2@example.org", interval, -1, "", 1, limit).Return(&emptyResult, emailutil.ErrInvalidEmail)
			r, err := http.Get(fmt.Sprintf("%s?from=1999-01-01&to=1999-12-31&mail_to=user2@example.org&status=-1&some_id=&page=1", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnprocessableEntity)
		})

		Convey("No Recipient", func() {
			m.EXPECT().CheckMessageDelivery(gomock.Any(), "user1@example.org", "", interval, -1, "", 1, limit).Return(&emptyResult, emailutil.ErrInvalidEmail)
			r, err := http.Get(fmt.Sprintf("%s?from=1999-01-01&to=1999-12-31&mail_from=user1@example.org&status=-1&some_id=&page=1", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnprocessableEntity)
		})

		Convey("Dates out of order", func() {
			// "from" comes after "to"
			r, err := http.Get(fmt.Sprintf("%s?to=1999-01-01&from=1999-12-31&mail_from=user1@example.org&mail_to=user2@example.org&status=-1&some_id=&page=1", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnprocessableEntity)
		})

		Convey("No page", func() {
			r, err := http.Get(fmt.Sprintf("%s?from=1999-01-01&to=1999-12-31&mail_from=user1@example.org&mail_to=user2@example.org&status=-1&some_id=", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnprocessableEntity)
		})

		Convey("Success", func() {
			messages := detective.MessagesPage{
				PageNumber:   1,
				FirstPage:    1,
				LastPage:     1,
				TotalResults: 1,
				Messages: detective.Messages{
					detective.Message{
						Queue: "AAAAA",
						Entries: []detective.MessageDelivery{{
							1,
							testutil.MustParseTime(`2009-02-14 00:31:30 +0000`),
							testutil.MustParseTime(`2009-02-14 00:31:30 +0000`),
							detective.Status(parser.SentStatus),
							"2.0.0",
							[]string{"google.com"},
							nil,
							"user1@example.org",
							[]string{"user2@example.org"},
							nil,
						},
						},
					},
				},
			}

			m.EXPECT().CheckMessageDelivery(gomock.Any(), "user1@example.org", "user2@example.org", interval, -1, "", 1, limit).Return(&messages, nil)

			r, err := http.Get(fmt.Sprintf("%s?from=1999-01-01&to=1999-12-31&mail_from=user1@example.org&mail_to=user2@example.org&status=-1&some_id=&page=1", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			var body detective.MessagesPage
			dec := json.NewDecoder(r.Body)
			err = dec.Decode(&body)

			So(err, ShouldBeNil)
			So(body, ShouldResemble, messages)
		})
	})
}

func TestDetectiveOldestAvailableTime(t *testing.T) {
	Convey("OldestAvailableTime", t, func() {
		ctrl := gomock.NewController(t)

		defer ctrl.Finish()

		m := mock_detective.NewMockDetective(ctrl)

		s := httptest.NewServer(httpmiddleware.New().WithEndpoint(oldestAvailableTimeHandler{detective: m}))

		Convey("No logs for use with message detective are available yet", func() {
			m.EXPECT().OldestAvailableTime(gomock.Any()).Return(time.Time{}, detective.ErrNoAvailableLogs)

			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			var body OldestAvailableTimeResponse
			dec := json.NewDecoder(r.Body)
			So(dec.Decode(&body), ShouldBeNil)
			So(body.Time, ShouldBeNil)
		})

		Convey("Logs are available. Return some value", func() {
			expectedTime := testutil.MustParseTime(`2000-01-01 12:30:40 +0000`)
			m.EXPECT().OldestAvailableTime(gomock.Any()).Return(expectedTime, nil)

			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			var body OldestAvailableTimeResponse
			dec := json.NewDecoder(r.Body)
			So(dec.Decode(&body), ShouldBeNil)
			So(body.Time, ShouldNotBeNil)
			So(*body.Time, ShouldResemble, expectedTime)
		})
	})
}

type fakeEscalateRequester struct {
	requests []escalator.Request
}

func (e *fakeEscalateRequester) Request(r escalator.Request) {
	e.requests = append(e.requests, r)
}

var mustParseTimeInterval = timeutil.MustParseTimeInterval

func TestEscalation(t *testing.T) {
	Convey("Test Detective Message Escalation", t, func() {
		ctrl := gomock.NewController(t)
		d := mock_detective.NewMockDetective(ctrl)
		e := &fakeEscalateRequester{}
		s := httptest.NewServer(httpmiddleware.New().WithEndpoint(detectiveEscalatorHandler{requester: e, detective: d}))

		Convey("No message escalated", func() {
			d.EXPECT().CheckMessageDelivery(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&detective.MessagesPage{}, nil)

			r, err := http.PostForm(s.URL, url.Values{
				"from":      []string{"2000-01-01"},
				"to":        []string{"2000-01-02"},
				"mail_from": []string{"user1@example.com"},
				"mail_to":   []string{"user2@example.com"},
			})

			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)
			So(e.requests, ShouldEqual, nil)
		})

		Convey("Internal error if detective check fails", func() {
			d.EXPECT().CheckMessageDelivery(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&detective.MessagesPage{}, errors.New(`Some error`))

			r, err := http.PostForm(s.URL, url.Values{
				"from":      []string{"2000-01-01"},
				"to":        []string{"2000-01-02"},
				"mail_from": []string{"user1@example.com"},
				"mail_to":   []string{"user2@example.com"},
			})

			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			So(e.requests, ShouldEqual, nil)
		})

		Convey("Invalid interval value", func() {
			r, err := http.PostForm(s.URL, url.Values{
				"from":      []string{"2000-01-01"},
				"to":        []string{"aaaaaaa"},
				"mail_from": []string{"user1@example.com"},
				"mail_to":   []string{"user2@example.com"},
			})

			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
			So(e.requests, ShouldEqual, nil)
		})

		Convey("Escalate issue", func() {
			d.EXPECT().CheckMessageDelivery(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				&detective.MessagesPage{
					PageNumber:   1,
					FirstPage:    1,
					LastPage:     1,
					TotalResults: 1,
					Messages: detective.Messages{
						detective.Message{
							Queue: "AAA",
							Entries: []detective.MessageDelivery{
								{
									TimeMin:  timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
									TimeMax:  timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
									Status:   detective.Status(parser.BouncedStatus),
									Dsn:      "3.4.6",
									Expired:  nil,
									MailFrom: "user1@example.com",
									MailTo:   []string{"user2@example.com"},
								},
							},
						},
					},
				}, nil)

			r, err := http.PostForm(s.URL, url.Values{
				"from":      []string{"2000-01-01"},
				"to":        []string{"2000-01-02"},
				"mail_from": []string{"user1@example.com"},
				"mail_to":   []string{"user2@example.com"},
			})

			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)
			So(e.requests, ShouldResemble, []escalator.Request{
				{
					Sender:    "user1@example.com",
					Recipient: "user2@example.com",
					Interval:  mustParseTimeInterval("2000-01-01", "2000-01-02"),
					Messages: detective.Messages{
						detective.Message{
							Queue: "AAA",
							Entries: []detective.MessageDelivery{
								{
									TimeMin:  timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
									TimeMax:  timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
									Status:   detective.Status(parser.BouncedStatus),
									Dsn:      "3.4.6",
									Expired:  nil,
									MailFrom: "user1@example.com",
									MailTo:   []string{"user2@example.com"},
								},
							},
						},
					},
				},
			})
		})
	})
}
