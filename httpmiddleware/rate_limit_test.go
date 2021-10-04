// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httpmiddleware

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type addrPair struct {
	actual  string
	proxied string
}

func reqForAddress(url string, addr addrPair) *http.Request {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	So(err, ShouldBeNil)
	req.RemoteAddr = addr.actual

	if len(addr.proxied) > 0 {
		req.Header.Add(`X-Forwarded-For`, addr.proxied)
	}

	return req
}

func TestRateLimits(t *testing.T) {
	Convey("Test Rate Limits", t, func() {
		endpoint := func(w http.ResponseWriter, r *http.Request) error {
			// I am a teapot
			w.WriteHeader(http.StatusTeapot)
			return nil
		}

		clock := timeutil.FakeClock{
			Time: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
		}

		buildRequester := func(timeFrame time.Duration, numberOfTries int64, isBehindProxy bool) func(addr addrPair) *httptest.ResponseRecorder {
			handler := New(requestWithRateLimitAndWithCustomClock(&clock, timeFrame, numberOfTries, isBehindProxy, BlockQuery)).
				WithEndpoint(CustomHTTPHandler(endpoint))

			return func(addr addrPair) *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, reqForAddress("http://example.com", addr))
				return rec
			}
		}

		Convey("Blocked after 3 queries and then unblock after 5min", func() {
			const maxTries = 3
			timeFrame := 5 * time.Minute
			usingProxy := true
			rec := buildRequester(timeFrame, maxTries, usingProxy)

			ip1 := addrPair{"[::1]:5000", "1.2.3.4"}
			ip2 := addrPair{"[::1]:4000", "33.44.55.66"}
			ip3 := addrPair{"5.5.5.5:4000", ""}

			// the first three attempts for ip1 work well
			for i := 0; i < 3; i++ {
				So(rec(ip1).Code, ShouldEqual, http.StatusTeapot)
				clock.Sleep(30 * time.Second)
			}

			// 4th atempt on ip1. Block request.
			So(rec(ip1).Code, ShouldEqual, http.StatusTooManyRequests)

			// After 2 minutes ip1 is still blocked
			clock.Sleep(2 * time.Minute)
			So(rec(ip1).Code, ShouldEqual, http.StatusTooManyRequests)

			// ip3 makes only one connection
			clock.Sleep(1 * time.Second)
			So(rec(ip3).Code, ShouldEqual, http.StatusTeapot)

			// ip2 makes several requests and gets blocked
			for i := 0; i < 3; i++ {
				clock.Sleep(50 * time.Millisecond)
				So(rec(ip2).Code, ShouldEqual, http.StatusTeapot)
			}

			// After 3 minutes ip1 is free to make requests again
			clock.Sleep(3 * time.Minute)
			So(rec(ip1).Code, ShouldEqual, http.StatusTeapot)

			// But ip2 is still blocked
			So(rec(ip2).Code, ShouldEqual, http.StatusTooManyRequests)

			// 5min later ip1 is still free, and ip2 is free again
			clock.Sleep(5 * time.Minute)
			So(rec(ip1).Code, ShouldEqual, http.StatusTeapot)
			So(rec(ip2).Code, ShouldEqual, http.StatusTeapot)
		})

		Convey("Ignore IP from headers when not using a reverse proxy", func() {
			const maxTries = 3
			timeFrame := 5 * time.Minute
			usingProxy := false
			rec := buildRequester(timeFrame, maxTries, usingProxy)

			// the first three attempts for ip1 work well
			for i := 0; i < 3; i++ {
				So(rec(addrPair{"1.2.3.4:5000", "1.2.3.4"}).Code, ShouldEqual, http.StatusTeapot)
				clock.Sleep(30 * time.Second)
			}

			// the same IP can have any ip forward headers, which will be ignored. The RemoteAddr will be used instead
			So(rec(addrPair{"1.2.3.4:5000", "4.3.2.1"}).Code, ShouldEqual, http.StatusTooManyRequests)

			clock.Sleep(5 * time.Minute)

			So(rec(addrPair{"1.2.3.4:5000", "4.4.4.4"}).Code, ShouldEqual, http.StatusTeapot)
		})
	})
}
