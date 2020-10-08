package httpmiddleware

import (
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

func TestTimeout(t *testing.T) {
	Convey("Test Request Interval", t, func() {
		Convey("No Timeout", func() {
			h := &sleepHandler{sleepTime: time.Microsecond * 1}
			c := WithTimeout(time.Millisecond * 100)
			s := httptest.NewServer(c.WithEndpoint(h))
			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("Timeout Happens", func() {
			h := &sleepHandler{sleepTime: time.Millisecond * 100}
			c := WithTimeout(time.Millisecond * 3)
			s := httptest.NewServer(c.WithEndpoint(h))
			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusRequestTimeout)
		})
	})
}
