package httpmiddleware

import (
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"testing"

)

func withRequest(middleware ...Middleware) Chain {
	middleware = append([]Middleware{RequestWithID()}, middleware...)
	return New(middleware...)
}

func TestRequestID(t *testing.T) {
	Convey("Request with id", t, func() {
		Convey("Success", func() {

			handler := CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
				id := GetRequestID(r.Context())
				if id == "" {
					t.Fatal("id is missing")
				}
				return nil
			})

			c := withRequest()
			s := httptest.NewServer(c.WithEndpoint(handler))
			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)
		})
	})
}

