// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"encoding/json"
	"fmt"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/detective"
	mock_detective "gitlab.com/lightmeter/controlcenter/detective/mock"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/util/emailutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDetective(t *testing.T) {
	ctrl := gomock.NewController(t)

	m := mock_detective.NewMockDetective(ctrl)

	chain := httpmiddleware.New(httpmiddleware.RequestWithInterval(time.UTC))

	Convey("CheckMessageDeliveryHandler", t, func() {

		interval, err := timeutil.ParseTimeInterval("1999-01-01", "1999-12-31", time.UTC)
		So(err, ShouldBeNil)

		s := httptest.NewServer(chain.WithEndpoint(checkMessageDeliveryHandler{detective: m}))

		Convey("No Parameters", func() {
			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnprocessableEntity)
		})

		Convey("No Time Interval", func() {
			r, err := http.Get(fmt.Sprintf("%s?mail_from=user1@example.org&mail_to=user2@example.org&page=1", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnprocessableEntity)
		})

		emptyResult := detective.MessagesPage{}

		Convey("No Sender", func() {
			m.EXPECT().CheckMessageDelivery(gomock.Any(), "", "user2@example.org", interval, 1).Return(&emptyResult, emailutil.ErrInvalidEmail)
			r, err := http.Get(fmt.Sprintf("%s?from=1999-01-01&to=1999-12-31&mail_to=user2@example.org&page=1", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnprocessableEntity)
		})

		Convey("No Recipient", func() {
			m.EXPECT().CheckMessageDelivery(gomock.Any(), "user1@example.org", "", interval, 1).Return(&emptyResult, emailutil.ErrInvalidEmail)
			r, err := http.Get(fmt.Sprintf("%s?from=1999-01-01&to=1999-12-31&mail_from=user1@example.org&page=1", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnprocessableEntity)
		})

		Convey("Dates out of order", func() {
			// "from" comes after "to"
			r, err := http.Get(fmt.Sprintf("%s?to=1999-01-01&from=1999-12-31&mail_from=user1@example.org&mail_to=user2&page=1", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnprocessableEntity)
		})

		Convey("No page", func() {
			r, err := http.Get(fmt.Sprintf("%s?from=1999-01-01&to=1999-12-31&mail_from=user1@example.org&mail_to=user2@example.org", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnprocessableEntity)
		})

		Convey("Success", func() {
			messages := detective.MessagesPage{
				PageNumber:   1,
				FirstPage:    1,
				LastPage:     1,
				TotalResults: 1,
				Messages:     []detective.MessageDelivery{detective.MessageDelivery{time.Unix(1234567890, 0).In(time.UTC), "Sent", "2.0.0"}},
			}
			m.EXPECT().CheckMessageDelivery(gomock.Any(), "user1@example.org", "user2@example.org", interval, 1).Return(&messages, nil)

			r, err := http.Get(fmt.Sprintf("%s?from=1999-01-01&to=1999-12-31&mail_from=user1@example.org&mail_to=user2@example.org&page=1", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			var body detective.MessagesPage
			dec := json.NewDecoder(r.Body)
			err = dec.Decode(&body)

			So(err, ShouldBeNil)
			So(body, ShouldResemble, messages)
		})
	})

	ctrl.Finish()
}
