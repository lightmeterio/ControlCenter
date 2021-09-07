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
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

type fakeRegistrar = httpauthsub.FakeRegistrar

func buildCookieClient() *http.Client {
	jar, err := cookiejar.New(&cookiejar.Options{})
	So(err, ShouldBeNil)
	return &http.Client{Jar: jar}
}

const originalTestPassword = `(1Yow@byU]>`

var dummyContext = context.Background()

func TestChangeUserInfo(t *testing.T) {
	Convey("Change User Info", t, func() {
		dir, clearDir := testutil.TempDir(t)
		defer clearDir()

		authDb, err := dbconn.Open(path.Join(dir, "auth.db"), 10)
		So(err, ShouldBeNil)
		defer authDb.Close()

		err = migrator.Run(authDb.RwConn.DB, "auth")
		So(err, ShouldBeNil)

		a, err := auth.NewAuth(authDb, auth.Options{})
		So(err, ShouldBeNil)

		_, err = a.Register(dummyContext, "email@example.com", `Nora`, originalTestPassword)
		So(err, ShouldBeNil)

		authenticator := httpauthsub.NewAuthenticator(a, dir)

		mux := http.NewServeMux()

		conn, closeConn := testutil.TempDBConnection(t, "master")
		defer closeConn()

		m, err := metadata.NewHandler(conn)
		So(err, ShouldBeNil)

		runner := metadata.NewSerialWriteRunner(m)
		done, cancel := runner.Run()
		defer func() {
			cancel()
			done()
		}()

		uuid := uuid.NewV4().String()
		writer := runner.Writer()
		writer.StoreJsonSync(dummyContext, metadata.UuidMetaKey, uuid)

		httpauth.HttpAuthenticator(mux, authenticator, m.Reader)

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
