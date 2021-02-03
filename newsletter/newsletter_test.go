// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package newsletter

import (
	"context"
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

type handler struct {
	shouldFail bool
	count      int
	method     string
	mediaType  string
	email      string
	values     url.Values
	subscribe  string
	htmlemail  string
	listMeta   string
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.values = r.URL.Query()

	h.count++

	if h.shouldFail {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.method = r.Method

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	errorutil.MustSucceed(err)
	h.mediaType = mediaType

	errorutil.MustSucceed(r.ParseForm())

	h.email = r.FormValue("email")
	h.subscribe = r.FormValue("subscribe")
	h.htmlemail = r.FormValue("htmlemail")
	h.listMeta = r.FormValue("list[11]")

	w.Write([]byte("Successfully subscribed! Yay!"))
}

func TestNewsletterSubscription(t *testing.T) {
	Convey("Newsletter Subscription", t, func() {
		context, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)

		Convey("Succeeds", func() {
			h := &handler{}
			s := httptest.NewServer(h)
			subscriber := HTTPSubscriber{URL: s.URL, HTTPClient: new(http.Client)}

			So(subscriber.Subscribe(context, "user@example.com"), ShouldBeNil)
			So(h.count, ShouldEqual, 1)

			So(h.values["p"], ShouldResemble, []string{"asubscribe"})
			So(h.values["id"], ShouldResemble, []string{"2"})

			So(h.mediaType, ShouldEqual, "application/x-www-form-urlencoded")
			So(h.email, ShouldEqual, "user@example.com")
			So(h.subscribe, ShouldEqual, "subscribe")
			So(h.htmlemail, ShouldEqual, "1")
			So(h.listMeta, ShouldEqual, "signup")
		})

		Convey("Fails due server error", func() {
			h := &handler{shouldFail: true}
			s := httptest.NewServer(h)
			subscriber := HTTPSubscriber{URL: s.URL, HTTPClient: new(http.Client)}

			So(errors.Is(subscriber.Subscribe(context, "user@example.com"), ErrSubscribingToNewsletter), ShouldBeTrue)
			So(h.count, ShouldEqual, 1)
		})
	})
}
