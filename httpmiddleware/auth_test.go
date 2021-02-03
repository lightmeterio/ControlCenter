// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package httpmiddleware

import (
	"github.com/gorilla/sessions"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)


func withAuth(auth *auth.Authenticator, middleware ...Middleware) Chain {
	middleware = append([]Middleware{RequestWithSession(auth)}, middleware...)
	return New(middleware...)
}

func TestSession(t *testing.T) {
	Convey("With session", t, func() {
		Convey("Success", func() {

			authenticator := &auth.Authenticator{
				Store: sessions.NewCookieStore([]byte("secret-key")),
			}

			buildCookieClient := func() *http.Client {
				jar, err := cookiejar.New(&cookiejar.Options{})
				So(err, ShouldBeNil)
				return &http.Client{Jar: jar}
			}

			loginHandler := func(w http.ResponseWriter, r *http.Request)  {
				session, err := authenticator.Store.New(r, auth.SessionName)
				if err != nil {
					t.Log("Error creating new session:", errorutil.Wrap(err))
				}

				// Implicitly log in
				session.Values["auth"] = auth.SessionData{Email: "donutloop", Name: "donutloop"}
				if err := session.Save(r, w); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					t.Error(err)
					return
				}
			}

			handler := CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
				session := GetSession(r.Context())
				if _, ok := session.Values["auth"]; !ok {
					t.Error("session is missing")
				}

				return nil
			})

			httpClient := buildCookieClient()

			c := withAuth(authenticator)

			mux := http.NewServeMux()
			mux.Handle("/fake/login", http.HandlerFunc(loginHandler))
			mux.Handle("/any", c.WithEndpoint(handler))

			s := httptest.NewServer(mux)

			r, err := httpClient.Get(s.URL + "/fake/login")
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			r, err = httpClient.Get(s.URL + "/any")
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)
		})
		Convey("Fail", func() {

			authenticator := &auth.Authenticator{
				Store: sessions.NewCookieStore([]byte("secret-key")),
			}

			handler := CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
				session := GetSession(r.Context())
				if _, ok := session.Values["auth"]; !ok {
					t.Error("session is missing")
				}

				return nil
			})

			c := withAuth(authenticator)

			mux := http.NewServeMux()
			mux.Handle("/any", c.WithEndpoint(handler))

			s := httptest.NewServer(mux)

			r, err := http.DefaultClient.Get(s.URL + "/any")
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)
		})
	})
}


