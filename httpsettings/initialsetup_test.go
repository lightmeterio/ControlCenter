package httpsettings

import (
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/settings"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type fakeSystemSetup struct {
	options           *settings.InitialSetupOptions
	shouldFailToSetup bool
}

func (f *fakeSystemSetup) SetInitialOptions(o settings.InitialSetupOptions) error {
	if f.shouldFailToSetup {
		return errors.New(`Some Unknwon Failure!`)
	}
	return nil
}

func TestInitialSetup(t *testing.T) {
	Convey("Initial Setup", t, func() {
		f := &fakeSystemSetup{}
		s := httptest.NewServer(NewInitialSetupHandler(f))
		c := &http.Client{}

		Convey("Fails", func() {
			Convey("Invalid Form data", func() {
				r, err := c.Post(s.URL, "application/x-www-form-urlencoded", strings.NewReader(`^^%`))
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})

			Convey("Invalid mime type", func() {
				r, err := c.Post(s.URL, "ksajdhfk*I&^&*^87678  $$343", nil)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})

			Convey("Subscribe is not a boolean", func() {
				r, err := c.PostForm(s.URL, url.Values{"email_kind": {string(settings.MailKindTransactional)}, "subscribe_newsletter": {"Falsch"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})

			Convey("Unsupported multiple subscribe options", func() {
				r, err := c.PostForm(s.URL, url.Values{"email_kind": {string(settings.MailKindTransactional)}, "subscribe_newsletter": {"on", "on"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})

			Convey("Incompatible mime type", func() {
				r, err := c.Post(s.URL, "application/json", nil)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})

			Convey("Incompatible Method", func() {
				r, err := c.Get(s.URL)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})

			Convey("Unknown setup failure", func() {
				f.shouldFailToSetup = true
				r, err := c.PostForm(s.URL, url.Values{"email_kind": {string(settings.MailKindDirect)}, "subscribe_newsletter": {"on"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})

			Convey("Subscribe without email", func() {
				r, err := c.PostForm(s.URL, url.Values{"email_kind": {string(settings.MailKindDirect)}, "subscribe_newsletter": {"on"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})

			Convey("Subscribe with zero email", func() {
				r, err := c.PostForm(s.URL, url.Values{"email": {}, "email_kind": {string(settings.MailKindDirect)}, "subscribe_newsletter": {"on"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})
		})

		Convey("Success", func() {
			Convey("Do subscribe", func() {
				r, err := c.PostForm(s.URL, url.Values{
					"email":                {"user@example.com"},
					"email_kind":           {string(settings.MailKindDirect)},
					"subscribe_newsletter": {"on"},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			})

			Convey("Do not subscribe", func() {
				r, err := c.PostForm(s.URL, url.Values{
					"email_kind": {string(settings.MailKindDirect)},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			})
		})
	})
}
