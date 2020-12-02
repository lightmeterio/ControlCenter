package httpmiddleware

import (
	"github.com/gorilla/sessions"
	. "github.com/smartystreets/goconvey/convey"
	v2 "gitlab.com/lightmeter/controlcenter/httpauth/v2"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)


func withAuth(auth *v2.Authenticator, middleware ...Middleware) Chain {
	middleware = append([]Middleware{RequestWithSession(auth)}, middleware...)
	return New(middleware...)
}

func TestSession(t *testing.T) {
	Convey("With session", t, func() {
		Convey("Success", func() {

			auth := &v2.Authenticator{
				Store: sessions.NewCookieStore([]byte("secret-key")),
			}

			buildCookieClient := func() *http.Client {
				jar, err := cookiejar.New(&cookiejar.Options{})
				So(err, ShouldBeNil)
				return &http.Client{Jar: jar}
			}

			loginHandler := func(w http.ResponseWriter, r *http.Request)  {
				session, err := auth.Store.New(r, v2.SessionName)
				if err != nil {
					log.Println("Error creating new session:", errorutil.Wrap(err))
				}

				// Implicitly log in
				session.Values["auth"] = v2.SessionData{Email: "donutloop", Name: "donutloop"}
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

			c := withAuth(auth)

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

			auth := &v2.Authenticator{
				Store: sessions.NewCookieStore([]byte("secret-key")),
			}

			handler := CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
				session := GetSession(r.Context())
				if _, ok := session.Values["auth"]; !ok {
					t.Error("session is missing")
				}

				return nil
			})

			c := withAuth(auth)

			mux := http.NewServeMux()
			mux.Handle("/any", c.WithEndpoint(handler))

			s := httptest.NewServer(mux)

			r, err := http.DefaultClient.Get(s.URL + "/any")
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)
		})
	})
}

