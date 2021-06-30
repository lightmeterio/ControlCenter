// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"context"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/detective"
	mock_detective "gitlab.com/lightmeter/controlcenter/detective/mock"
	"gitlab.com/lightmeter/controlcenter/httpauth"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/meta"
	detectivesettings "gitlab.com/lightmeter/controlcenter/settings/detective"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func buildCookieClient() *http.Client {
	jar, err := cookiejar.New(&cookiejar.Options{})
	So(err, ShouldBeNil)
	return &http.Client{Jar: jar}
}

func buildTestEnv(t *testing.T) (*httptest.Server, *mock_detective.MockDetective, *meta.AsyncWriter, func()) {
	ctrl := gomock.NewController(t)

	dir, clearDir := testutil.TempDir(t)

	registrar := &auth.FakeRegistrar{
		SessionKey: []byte("some_key"),
		Email:      "alice@example.com",
		Name:       "Alice",
		Password:   "super-secret",
	}

	detective := mock_detective.NewMockDetective(ctrl)

	auth := auth.NewAuthenticator(registrar, dir)
	mux := http.NewServeMux()

	settingdDB, removeDB := testutil.TempDBConnection(t)

	handler, err := meta.NewHandler(settingdDB, "master")
	So(err, ShouldBeNil)

	runner := meta.NewRunner(handler)

	done, cancel := runner.Run()

	settingsWriter := runner.Writer()
	settingsReader := handler.Reader

	HttpDetective(auth, mux, time.UTC, detective, &fakeEscalateRequester{}, settingsReader)

	httpauth.HttpAuthenticator(mux, auth, settingsReader)

	s := httptest.NewServer(mux)

	return s, detective, settingsWriter, func() {
		cancel()
		So(done(), ShouldBeNil)
		removeDB()
		clearDir()
		ctrl.Finish()
	}
}

func TestDetectiveAuth(t *testing.T) {
	Convey("Detective auth", t, func() {
		detectiveURL := "/api/v0/checkMessageDeliveryStatus?mail_from=a@b.c&mail_to=d@e.f&from=2020-01-01&to=2020-12-31&page=1"

		c := buildCookieClient()

		s, d, settingsWriter, clear := buildTestEnv(t)
		defer clear()

		d.EXPECT().
			CheckMessageDelivery(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&detective.MessagesPage{}, nil)

		Convey("Detective API not accessible to non-authenticated user", func() {
			r, err := c.Get(s.URL + detectiveURL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)

			Convey("Once we are logged in, detective API is accessible", func() {
				r, err = c.PostForm(s.URL+"/login", url.Values{"email": {"alice@example.com"}, "password": {"super-secret"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
				{
					r, err := c.Get(s.URL + detectiveURL)
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)
				}
			})
		})

		Convey("Detective API only accessible to end-users if setting is enabled", func() {
			r, err := c.Get(s.URL + detectiveURL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)

			Convey("Once we enable the setting, detective API is accessible to end-users", func() {
				settings := detectivesettings.Settings{}
				settings.EndUsersEnabled = true
				detectivesettings.SetSettings(context.Background(), settingsWriter, settings)

				{
					r, err := c.Get(s.URL + detectiveURL)
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)
				}
			})
		})
	})
}
