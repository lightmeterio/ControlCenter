// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httpmiddleware

import (
	"context"
	"github.com/gorilla/sessions"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)

func withAuth(auth *auth.Authenticator, middleware ...Middleware) Chain {
	middleware = append([]Middleware{RequestWithSession(auth)}, middleware...)
	return New(middleware...)
}

func buildCookieClient() *http.Client {
	jar, err := cookiejar.New(&cookiejar.Options{})
	So(err, ShouldBeNil)
	return &http.Client{Jar: jar}
}

func buildLoginHandler(t *testing.T, authenticator *auth.Authenticator) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := authenticator.Store.New(r, auth.SessionName)
		if err != nil {
			t.Log("Error creating new session:", errorutil.Wrap(err))
		}

		// Implicitly log in
		session.Values["auth"] = auth.SessionData{Email: "donutloop", Name: "donutloop"}

		if err := session.Save(r, w); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.Error(err)
		}
	}
}

func TestAuthSession(t *testing.T) {
	Convey("With session", t, func() {
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

		Convey("Success", func() {
			httpClient := buildCookieClient()
			c := withAuth(authenticator)
			mux := http.NewServeMux()
			mux.Handle("/fake/login", http.HandlerFunc(buildLoginHandler(t, authenticator)))
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

func TestRequireAuthenticationOnlyAfterSystemHasAnyUser(t *testing.T) {
	Convey("RequireAuthenticationOnlyAfterSystemHasAnyUser", t, func() {
		registrar := &auth.FakeRegistrar{
			SessionKey: []byte("AAAAAAAAAAAAAAAA"),
		}

		authenticator := &auth.Authenticator{
			Registrar: registrar,
			Store:     sessions.NewCookieStore([]byte("secret-key")),
		}

		chain := New(RequireAuthenticationOnlyAfterSystemHasAnyUser(authenticator))

		mux := http.NewServeMux()
		mux.Handle("/fake/login", http.HandlerFunc(buildLoginHandler(t, authenticator)))
		mux.Handle("/any", chain.WithEndpoint(CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
			w.Write([]byte("hello"))
			return nil
		})))

		s := httptest.NewServer(mux)

		httpClient := buildCookieClient()

		responseContent := func(r *http.Response) (string, error) {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				return "", errorutil.Wrap(err)
			}

			return string(b), nil
		}

		Convey("No user registred. Allow access", func() {
			r, err := httpClient.Get(s.URL + "/any")
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)
			content, err := responseContent(r)
			So(err, ShouldBeNil)
			So(content, ShouldEqual, "hello")
		})

		Convey("Has an user, but it's not authenticated. Deny access", func() {
			registrar.Register(context.Background(), "user@example.com", "User Name", "super_password")
			r, err := httpClient.Get(s.URL + "/any")
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)
		})

		Convey("Has an user and is authenticated. Allow access", func() {
			registrar.Register(context.Background(), "user@example.com", "User Name", "super_password")

			{
				r, err := httpClient.Get(s.URL + "/fake/login")
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			}

			r, err := httpClient.Get(s.URL + "/any")
			So(r.StatusCode, ShouldEqual, http.StatusOK)
			content, err := responseContent(r)
			So(err, ShouldBeNil)
			So(content, ShouldEqual, "hello")
		})
	})
}
