// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httpauth_test

import (
	"context"
	"encoding/json"
	uuid "github.com/satori/go.uuid"
	"gitlab.com/lightmeter/controlcenter/httpauth"
	httpauthsub "gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
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

func TestHTTPAuth(t *testing.T) {
	Convey("HTTP Authentication", t, func() {
		conn, closeConn := testutil.TempDBConnectionMigrated(t, "master")
		defer closeConn()

		m, err := metadata.NewHandler(conn)
		So(err, ShouldBeNil)

		writeRunner := metadata.NewSerialWriteRunner(m)
		done, cancel := runner.Run(writeRunner)
		defer func() {
			cancel()
			done()
		}()

		uuid := uuid.NewV4().String()
		writer := writeRunner.Writer()
		writer.StoreJsonSync(context.Background(), metadata.UuidMetaKey, uuid)

		failedAttempts := 0

		registrar := &fakeRegistrar{
			SessionKey:                        []byte("session_key_1_super_secret"),
			Authenticated:                     false,
			ShouldFailToRegister:              false,
			ShouldFailToAuthenticate:          false,
			ShouldFailToCheckIfThereIsAnyUser: false,
		}

		mux := http.NewServeMux()

		a := httpauthsub.NewAuthenticatorWithOptions(registrar)
		httpauth.HttpAuthenticator(mux, a, m.Reader)

		s := httptest.NewServer(mux)
		defer s.Close()

		Convey("Unauthenticated and unregistred user", func() {
			c := buildCookieClient()
			defer c.CloseIdleConnections()

			Convey("Login will fail on invalid method", func() {
				req, err := http.NewRequest(http.MethodDelete, s.URL+"/login", nil)
				So(err, ShouldBeNil)
				r, err := c.Do(req)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
				So(failedAttempts, ShouldEqual, 0)
			})

			Convey("Login will fail on wrong request mime", func() {
				r, err := c.Post(s.URL+"/login", "application/json", nil)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
				So(failedAttempts, ShouldEqual, 0)
			})

			Convey("Login will fail on invalid form data", func() {
				r, err := c.Post(s.URL+"/login", "application/x-www-form-urlencoded", strings.NewReader(`^^%`))
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
				So(failedAttempts, ShouldEqual, 0)
			})

			Convey("Login will fail due to some error with the authenticator", func() {
				registrar.AuthenticateYieldsError = true
				r, err := c.PostForm(s.URL+"/login", url.Values{"email": {"alice@example.com"}, "password": {"some_password"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
				So(failedAttempts, ShouldEqual, 0)
			})

			Convey("Login will fail as there is no registred user", func() {
				r, err := c.PostForm(s.URL+"/login", url.Values{"email": {"alice@example.com"}, "password": {"some_password"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)
				So(failedAttempts, ShouldEqual, 0)
				body, _ := ioutil.ReadAll(r.Body)

				response := struct{ Error string }{}
				So(json.Unmarshal(body, &response), ShouldBeNil)
				So(response.Error, ShouldEqual, "Invalid email address or password")
			})

			Convey("User registrations fails", func() {
				Convey("Invalid HTTP method", func() {
					req, err := http.NewRequest(http.MethodDelete, s.URL+"/register", nil)
					So(err, ShouldBeNil)
					r, err := c.Do(req)
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
					So(failedAttempts, ShouldEqual, 0)
				})

				Convey("Invalid form mime type", func() {
					r, err := c.Post(s.URL+"/register", "application/json", strings.NewReader(`{}`))
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
					So(failedAttempts, ShouldEqual, 0)
				})

				Convey("Invalid Form data", func() {
					r, err := c.Post(s.URL+"/register", "application/x-www-form-urlencoded", strings.NewReader(`^^%`))
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
					So(failedAttempts, ShouldEqual, 0)
				})

				Convey("No email, name and password provided", func() {
					r, err := c.PostForm(s.URL+"/register", url.Values{})
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
					So(failedAttempts, ShouldEqual, 0)
				})

				Convey("No email provided", func() {
					r, err := c.PostForm(s.URL+"/register", url.Values{
						"name":     {"donutloop"},
						"password": {"some_password"},
					})

					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
					So(failedAttempts, ShouldEqual, 0)
				})

				Convey("No password provided", func() {
					r, err := c.PostForm(s.URL+"/register", url.Values{
						"name":  {"donutloop"},
						"email": {"alice@example.com"},
					})

					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
					So(failedAttempts, ShouldEqual, 0)
				})

				Convey("No name provided", func() {
					r, err := c.PostForm(s.URL+"/register", url.Values{
						"name":     {"donutloop"},
						"password": {"poor password"},
					})

					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
					So(failedAttempts, ShouldEqual, 0)
				})

				Convey("Some validation makes the registring fail", func() {
					registrar.ShouldFailToRegister = true

					r, err := c.PostForm(s.URL+"/register", url.Values{
						"name":     {"donutloop"},
						"email":    {"alice@example.com"},
						"password": {"poor password"},
					})

					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)
					body, _ := ioutil.ReadAll(r.Body)

					response := struct {
						Error string
					}{}

					So(json.Unmarshal(body, &response), ShouldBeNil)

					So(response.Error, ShouldEqual, "Weak Password")
				})
			})

			Convey("User registrations succeeds", func() {
				r, err := c.PostForm(s.URL+"/register", url.Values{
					"email":    {"alice@example.com"},
					"name":     {"Alice"},
					"password": {"correcthorsebatterystable"},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)

				Convey("get fake user data", func() {
					// first user logs in
					r, err := c.PostForm(s.URL+"/login", url.Values{"email": {"alice@example.com"}, "password": {"correcthorsebatterystable"}})
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)

					// get fake user data
					r, err = c.Get(s.URL + "/api/v0/userInfo")
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)

					b, err := ioutil.ReadAll(r.Body)
					So(err, ShouldBeNil)

					v := httpauthsub.UserSystemData{}
					json.Unmarshal(b, &v)

					So(v.InstanceID, ShouldEqual, uuid)
				})

				Convey("get fake user data after registration", func() {
					// get fake user data
					r, err = c.Get(s.URL + "/api/v0/userInfo")
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)

					b, err := ioutil.ReadAll(r.Body)
					So(err, ShouldBeNil)
					t.Log("Response:", string(b))
				})

				Convey("User logs out, returning to the login page", func() {
					r, err := c.Get(s.URL + "/logout")
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)
					So(failedAttempts, ShouldEqual, 0)

					Convey("User can login again", func() {
						r, err := c.PostForm(s.URL+"/login", url.Values{"email": {"alice@example.com"}, "password": {"correcthorsebatterystable"}})
						So(err, ShouldBeNil)
						So(r.StatusCode, ShouldEqual, http.StatusOK)
						So(failedAttempts, ShouldEqual, 0)
					})

					Convey("User can login again with posting more complex mime-type", func() {
						formData := url.Values{"email": {"alice@example.com"}, "password": {"correcthorsebatterystable"}}
						r, err := c.Post(s.URL+"/login", "application/x-www-form-urlencoded;charset=UTF-8", strings.NewReader(formData.Encode()))
						So(err, ShouldBeNil)
						So(r.StatusCode, ShouldEqual, http.StatusOK)
						So(failedAttempts, ShouldEqual, 0)
					})

					Convey("user has login", func() {
						// first user logs in
						r, err = c.PostForm(s.URL+"/login", url.Values{"email": {"alice@example.com"}, "password": {"correcthorsebatterystable"}})
						So(err, ShouldBeNil)
						So(r.StatusCode, ShouldEqual, http.StatusOK)

						// check login
						r, err = c.Get(s.URL + "/auth/check")
						So(err, ShouldBeNil)
						So(r.StatusCode, ShouldEqual, http.StatusOK)
					})
				})
			})
		})

		Convey("user has not login", func() {
			c := buildCookieClient()

			registrar.Email = "user@example.com"
			registrar.Password = "654321"
			registrar.Name = "Sakura"

			// check login
			r, err := c.Get(s.URL + "/auth/check")
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)
		})

		Convey("user has not registerd", func() {
			c := buildCookieClient()
			defer c.CloseIdleConnections()

			// check registered
			r, err := c.Get(s.URL + "/auth/check")
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusForbidden)
		})
	})
}
