// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httpsettings

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/httpauth"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
)

func buildCookieClient() *http.Client {
	jar, err := cookiejar.New(&cookiejar.Options{})
	So(err, ShouldBeNil)
	return &http.Client{Jar: jar}
}

func TestRegressions(t *testing.T) {
	Convey("Regressions", t, func() {
		dir, setup, _, _, _, clear := buildTestSetup(t)
		defer clear()

		registrar := &auth.FakeRegistrar{
			SessionKey: []byte("some_key"),
			Email:      "alice@example.com",
			Name:       "Alice",
			Password:   "super-secret",
		}

		auth := auth.NewAuthenticator(registrar, dir)
		mux := http.NewServeMux()

		setup.HttpSetup(mux, auth)

		httpauth.HttpAuthenticator(mux, auth)

		s := httptest.NewServer(mux)

		c := buildCookieClient()

		Convey("Issue 450, getting settings must require authentication", func() {
			r, err := c.Get(s.URL + "/settings")
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)

			Convey("Once we are logged in, /settings is accessible", func() {
				r, err = c.PostForm(s.URL+"/login", url.Values{"email": {"alice@example.com"}, "password": {"super-secret"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)

				{
					r, err := c.Get(s.URL + "/settings")
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)
				}
			})
		})
	})
}
