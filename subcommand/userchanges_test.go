// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package subcommand

import (
	"context"
	uuid "github.com/satori/go.uuid"
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/httpauth"
	httpauthsub "gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func buildCookieClient() *http.Client {
	jar, err := cookiejar.New(&cookiejar.Options{})
	So(err, ShouldBeNil)
	return &http.Client{Jar: jar}
}

const originalTestPassword = `(1Yow@byU]>`

var dummyContext = context.Background()

func TestChangeUserInfo(t *testing.T) {
	Convey("Change User Info", t, func() {
		dir, closeDatabases := testutil.TempDatabases(t)
		defer closeDatabases()

		a, err := auth.NewAuth(dir, auth.Options{})
		So(err, ShouldBeNil)

		_, err = a.Register(dummyContext, "email@example.com", `Nora`, originalTestPassword)
		So(err, ShouldBeNil)

		authenticator := httpauthsub.NewAuthenticator(a, dir)

		mux := http.NewServeMux()

		runner := meta.NewRunner(dbconn.Db("master"))
		done, cancel := runner.Run()
		defer func() {
			cancel()
			done()
		}()

		uuid := uuid.NewV4().String()
		writer := runner.Writer()
		writer.StoreJsonSync(dummyContext, meta.UuidMetaKey, uuid)

		httpauth.HttpAuthenticator(mux, authenticator)

		s := httptest.NewServer(mux)
		defer s.Close()

		c := buildCookieClient()
		defer c.CloseIdleConnections()

		// First, login, setting all the proper cookies
		{
			r, err := c.PostForm(s.URL+"/login", url.Values{"email": {"email@example.com"}, "password": {originalTestPassword}})
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)
		}

		{
			// Just to ensure the user is logged-in
			r, err := c.Get(s.URL + "/auth/check")
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)
		}

		Convey("When the e-mail changes, all sessions are reset", func() {
			PerformUserInfoChange(true, dir, "email@example.com", "new@example.com", "", ``)

			{
				// check login
				r, err := c.Get(s.URL + "/auth/check")
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)
			}

			// The user is able to login again with the new e-mail
			{
				r, err := c.PostForm(s.URL+"/login", url.Values{"email": {"new@example.com"}, "password": {originalTestPassword}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			}
		})

		Convey("When the password changes, all sessions are reset", func() {
			newPassword := "(786875656&*^*&^*&^======"
			PerformUserInfoChange(true, dir, "email@example.com", "", "", newPassword)

			// check login
			{
				r, err := c.Get(s.URL + "/auth/check")
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)
			}

			// The user is able to login again with the new e-mail
			{
				r, err := c.PostForm(s.URL+"/login", url.Values{"email": {"email@example.com"}, "password": {newPassword}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			}
		})
	})
}
