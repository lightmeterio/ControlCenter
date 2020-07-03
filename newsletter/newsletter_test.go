package newsletter

import (
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/util"
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"
)

type handler struct {
	shouldFail bool
	count      int
	method     string
	mediaType  string
	email      string
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.count++

	if h.shouldFail {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.method = r.Method

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	util.MustSucceed(err, "")
	h.mediaType = mediaType

	util.MustSucceed(r.ParseMultipartForm(1024), "")

	h.email = r.FormValue("item_meta[14]")
}

func TestNewsletterSubscription(t *testing.T) {
	Convey("Newsletter Subscription", t, func() {
		Convey("Succeeds", func() {
			h := &handler{}
			s := httptest.NewServer(h)
			subscriber := HTTPSubscriber{URL: s.URL}

			So(subscriber.Subscribe("user@example.com"), ShouldBeNil)
			So(h.count, ShouldEqual, 1)
			So(h.mediaType, ShouldEqual, "multipart/form-data")
			So(h.email, ShouldEqual, "user@example.com")
		})

		Convey("Fails due server error", func() {
			h := &handler{shouldFail: true}
			s := httptest.NewServer(h)
			subscriber := HTTPSubscriber{URL: s.URL}

			So(errors.Is(subscriber.Subscribe("user@example.com"), ErrSubscribingToNewsletter), ShouldBeTrue)
			So(h.count, ShouldEqual, 1)
		})
	})
}
