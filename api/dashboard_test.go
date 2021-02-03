// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	mock_dashboard "gitlab.com/lightmeter/controlcenter/dashboard/mock"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDashboard(t *testing.T) {
	ctrl := gomock.NewController(t)

	m := mock_dashboard.NewMockDashboard(ctrl)

	chain := httpmiddleware.New(httpmiddleware.RequestWithInterval(time.UTC))

	Convey("CountByStatus", t, func() {
		Convey("No Time Interval", func() {
			s := httptest.NewServer(chain.WithEndpoint(countByStatusHandler{dashboard: m}))
			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnprocessableEntity)
		})

		Convey("Dates out of order", func() {
			s := httptest.NewServer(chain.WithEndpoint(countByStatusHandler{dashboard: m}))
			// "from" comes after "to"
			r, err := http.Get(fmt.Sprintf("%s?to=1999-01-01&from=1999-12-31", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnprocessableEntity)
		})

		Convey("Success", func() {
			interval, err := data.ParseTimeInterval("1999-01-01", "1999-12-31", time.UTC)
			So(err, ShouldBeNil)

			m.EXPECT().CountByStatus(gomock.Any(), parser.SentStatus, interval).Return(4, nil)
			m.EXPECT().CountByStatus(gomock.Any(), parser.DeferredStatus, interval).Return(3, nil)
			m.EXPECT().CountByStatus(gomock.Any(), parser.BouncedStatus, interval).Return(2, nil)

			s := httptest.NewServer(chain.WithEndpoint((countByStatusHandler{dashboard: m})))
			r, err := http.Get(fmt.Sprintf("%s?from=1999-01-01&to=1999-12-31", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			var body map[string]interface{}
			dec := json.NewDecoder(r.Body)
			err = dec.Decode(&body)
			So(err, ShouldBeNil)

			// NOTE: all numbers are decoded into an interface{} as float64, so we want to have float64 here, too.
			expected := map[string]interface{}{"bounced": float64(2), "deferred": float64(3), "sent": float64(4)}
			So(body, ShouldResemble, expected)
		})
	})

	Convey("DeliveryStatus", t, func() {
		s := httptest.NewServer(chain.WithEndpoint(deliveryStatusHandler{dashboard: m}))

		Convey("Success", func() {
			m.EXPECT().DeliveryStatus(gomock.Any(), data.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`2000-01-02 23:59:59 +0000`),
			}).Return(dashboard.Pairs{
				dashboard.Pair{Key: "bounced", Value: 4},
				dashboard.Pair{Key: "deferred", Value: 5},
				dashboard.Pair{Key: "sent", Value: 9},
			}, nil)

			r, err := http.Get(fmt.Sprintf("%s?from=2000-01-01&to=2000-01-02", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			var body []interface{}
			dec := json.NewDecoder(r.Body)
			err = dec.Decode(&body)
			So(err, ShouldBeNil)

			expected := []interface{}{
				map[string]interface{}{"key": "bounced", "value": float64(4)},
				map[string]interface{}{"key": "deferred", "value": float64(5)},
				map[string]interface{}{"key": "sent", "value": float64(9)},
			}

			So(body, ShouldResemble, expected)
		})

		Convey("Internal error", func() {
			m.EXPECT().DeliveryStatus(gomock.Any(), data.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`2000-01-02 23:59:59 +0000`),
			}).Return(dashboard.Pairs{}, errors.New("Some Internal Dashboard Error"))

			r, err := http.Get(fmt.Sprintf("%s?from=2000-01-01&to=2000-01-02", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)

		})
	})

	ctrl.Finish()
}
