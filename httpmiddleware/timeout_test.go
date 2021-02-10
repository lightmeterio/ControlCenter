// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httpmiddleware

import (
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type sleepHandler struct {
	sleepTime time.Duration
}

func (h sleepHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	time.Sleep(h.sleepTime)
	return nil
}

func withTimeout(timeout time.Duration, middleware ...Middleware) Chain {
	middleware = append(middleware, []Middleware{RequestWithTimeout(timeout)}...)
	return New(middleware...)
}

func TestTimeout(t *testing.T) {
	Convey("Test Request Interval", t, func() {
		Convey("No Timeout", func() {
			h := &sleepHandler{sleepTime: time.Microsecond * 1}
			c := withTimeout(time.Millisecond * 100)
			s := httptest.NewServer(c.WithEndpoint(h))
			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Timeout Happens", func() {
			h := &sleepHandler{sleepTime: time.Millisecond * 100}
			c := withTimeout(time.Millisecond * 3)
			s := httptest.NewServer(c.WithEndpoint(h))
			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusRequestTimeout)
		})
	})
}

type keepAliveTestHandler struct {
	maxTimeout     time.Duration
	defaultTimeout time.Duration
	reqTimeout     time.Duration
	err            error
}

func (h *keepAliveTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	h.reqTimeout, h.err = timeoutForRequest(r, h.defaultTimeout, h.maxTimeout)
	return nil
}

func TestTimeoutFromKeepAlive(t *testing.T) {
	Convey("Test Request Interval", t, func() {
		h := &keepAliveTestHandler{defaultTimeout: time.Second * 6, maxTimeout: time.Second * 120}
		c := New()
		s := httptest.NewServer(c.WithEndpoint(h))
		req, err := http.NewRequest(http.MethodGet, s.URL, nil)
		So(err, ShouldBeNil)
		client := http.Client{}

		Convey("No Keep-Alive", func() {
			_, err := client.Do(req)
			So(err, ShouldBeNil)
			So(h.err, ShouldBeNil)
			So(h.reqTimeout, ShouldEqual, h.defaultTimeout)
		})

		Convey("Invalid Keep-Alive, yield error", func() {
			req.Header["Keep-Alive"] = []string{"timeout=invalid_number, max=1000"}
			_, err := client.Do(req)
			So(errors.Is(h.err, ErrInvalidKeepAliveHeader), ShouldBeTrue)
			So(err, ShouldBeNil)
		})

		Convey("Clamp to max timeout if Keep-Alive is too long", func() {
			req.Header["Keep-Alive"] = []string{"timeout=124, max=1000"}
			_, err := client.Do(req)
			So(h.reqTimeout, ShouldEqual, time.Second*120)
			So(h.err, ShouldBeNil)
			So(err, ShouldBeNil)
		})

		Convey("With Valid Keep-Alive, no optional max param", func() {
			req.Header["Keep-Alive"] = []string{"timeout=42"}
			_, err := client.Do(req)
			So(err, ShouldBeNil)
			So(h.err, ShouldBeNil)
			So(h.reqTimeout, ShouldEqual, 42*time.Second)
		})

		Convey("With Valid Keep-Alive, with optional max param", func() {
			req.Header["Keep-Alive"] = []string{"timeout=35, max=100"}
			_, err := client.Do(req)
			So(err, ShouldBeNil)
			So(h.err, ShouldBeNil)
			So(h.reqTimeout, ShouldEqual, 35*time.Second)
		})
	})
}
