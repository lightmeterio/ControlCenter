// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	mock_detective "gitlab.com/lightmeter/controlcenter/detective/mock"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestRateLimits(t *testing.T) {
	Convey("Test Rate Limits", t, func() {
		c := &http.Client{}

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mock_detective.NewMockDetective(ctrl)

		a := auth.NewAuthenticatorWithOptions(&auth.FakeRegistrar{
			SessionKey: []byte("some_key"),
			Email:      "alice@example.com",
			Name:       "Alice",
			Password:   "super-secret",
		})

		loginChain := httpmiddleware.New(httpmiddleware.RequestWithTimeout(httpmiddleware.DefaultTimeout))
		loginServer := httptest.NewServer(
			loginChain.WithEndpoint(
				httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
					return auth.HandleLogin(a, w, r)
				}),
			),
		)

		detectiveChain := httpmiddleware.New(httpmiddleware.RequestWithInterval(time.UTC))
		detectiveServer := httptest.NewServer(detectiveChain.WithEndpoint(checkMessageDeliveryHandler{detective: m}))

		Convey("Blocked on login and detective after X queries", func() {
			// Login
			maxLoginTries := httpmiddleware.GetMaxNumberOfTriesForEndpoint("/login")

			loginResults := map[int]int{} // http status => number
			for i := int64(0); i < maxLoginTries+1; i++ {

				r, err := c.PostForm(loginServer.URL+"/login", url.Values{"email": {"unknown@example.com"}, "password": {"wrong"}})
				So(err, ShouldBeNil)

				_, ok := loginResults[r.StatusCode]
				if !ok {
					loginResults[r.StatusCode] = 0
				}

				loginResults[r.StatusCode]++
			}

			So(loginResults[http.StatusUnauthorized], ShouldBeGreaterThanOrEqualTo, 1)    // there was at least one non-blocked attempt
			So(loginResults[http.StatusTooManyRequests], ShouldBeGreaterThanOrEqualTo, 1) // there was at least one blocked attempt

			// Detective
			maxDetectiveTries := httpmiddleware.GetMaxNumberOfTriesForEndpoint("/api/v0/checkMessageDeliveryStatus")

			detectiveResults := map[int]int{} // http status => number
			for i := int64(0); i < maxDetectiveTries+1; i++ {
				r, err := c.Get(detectiveServer.URL + "/api/v0/checkMessageDeliveryStatus")
				So(err, ShouldBeNil)

				_, ok := detectiveResults[r.StatusCode]
				if !ok {
					detectiveResults[r.StatusCode] = 0
				}

				detectiveResults[r.StatusCode]++
			}

			So(detectiveResults[http.StatusUnprocessableEntity], ShouldBeGreaterThanOrEqualTo, 1) // there was at least one non-blocked attempt
			So(detectiveResults[http.StatusTooManyRequests], ShouldBeGreaterThanOrEqualTo, 1)     // there was at least one blocked attempt
		})
	})
}
