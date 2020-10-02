package httpsettings

import (
	"context"
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/meta"
	_ "gitlab.com/lightmeter/controlcenter/meta/migrations"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"
)

var (
	dummyContext = context.Background()
)

type dummySubscriber struct{}

func (*dummySubscriber) Subscribe(ctx context.Context, email string) error {
	log.Println("A dummy call that would otherwise subscribe email", email, "to Lightmeter newsletter :-)")
	return nil
}

type fakeSystemSetup struct {
	options           *settings.InitialOptions
	shouldFailToSetup bool
}

func (f *fakeSystemSetup) SetOptions(context.Context, interface{}) error {
	if f.shouldFailToSetup {
		return errors.New(`Some Unknwon Failure!`)
	}
	return nil
}

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestInitialSetup(t *testing.T) {
	Convey("Initial Setup", t, func() {
		dir, clearDir := testutil.TempDir()
		defer clearDir()

		connPair, err := dbconn.NewConnPair(path.Join(dir, "master.db"))
		So(err, ShouldBeNil)

		defer func() {
			So(connPair.Close(), ShouldBeNil)
		}()

		m, err := meta.NewMetaDataHandler(connPair, "master")
		So(err, ShouldBeNil)

		mc, err := settings.NewMasterConf(m, &dummySubscriber{})
		So(err, ShouldBeNil)

		setup := NewSettings(mc)

		chain := httpmiddleware.New()
		handler := chain.WithError(httpmiddleware.CustomHTTPHandler(setup.InitialSetupHandler))

		s := httptest.NewServer(handler)
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
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
			})

			Convey("Unsupported multiple subscribe options", func() {
				r, err := c.PostForm(s.URL, url.Values{"email_kind": {string(settings.MailKindTransactional)}, "subscribe_newsletter": {"on", "on"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
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

			Convey("Subscribe without email", func() {
				r, err := c.PostForm(s.URL, url.Values{"email_kind": {string(settings.MailKindDirect)}, "subscribe_newsletter": {"on"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
			})

			Convey("Subscribe with zero email", func() {
				r, err := c.PostForm(s.URL, url.Values{"email": {}, "email_kind": {string(settings.MailKindDirect)}, "subscribe_newsletter": {"on"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
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

func TestFakeInitialSetup(t *testing.T) {
	Convey("Initial Setup", t, func() {
		f := &fakeSystemSetup{}

		testSettings := NewSettings(f)
		chain := httpmiddleware.New()
		handler := chain.WithError(httpmiddleware.CustomHTTPHandler(testSettings.InitialSetupHandler))

		s := httptest.NewServer(handler)
		c := &http.Client{}

		Convey("Fails", func() {
			Convey("Unknown setup failure", func() {
				f.shouldFailToSetup = true
				r, err := c.PostForm(s.URL, url.Values{"email_kind": {string(settings.MailKindDirect)}, "subscribe_newsletter": {"on"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}

func TestSettingsSetup(t *testing.T) {
	Convey("Settings Setup", t, func() {
		dir, clearDir := testutil.TempDir()
		defer clearDir()

		connPair, err := dbconn.NewConnPair(path.Join(dir, "master.db"))
		So(err, ShouldBeNil)

		defer func() {
			So(connPair.Close(), ShouldBeNil)
		}()

		m, err := meta.NewMetaDataHandler(connPair, "master")
		So(err, ShouldBeNil)

		mc, err := settings.NewMasterConf(m, &dummySubscriber{})
		So(err, ShouldBeNil)

		setup := NewSettings(mc)

		chain := httpmiddleware.New()
		handler := chain.WithError(httpmiddleware.CustomHTTPHandler(setup.NotificationSettingsHandler))
		s := httptest.NewServer(handler)
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

			Convey("Incompatible Method", func() {
				r, err := c.Get(s.URL)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})

			Convey("To many values", func() {
				r, err := c.PostForm(s.URL, url.Values{"value_1": {""}, "value_2": {""}, "value_3": {""}, "value_4": {""}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})

		})

		Convey("Success", func() {
			Convey("send valid values", func() {
				r, err := c.PostForm(s.URL, url.Values{
					"messenger_kind":    {"slack"},
					"messenger_token":   {"sjdfklsjdfkljfs"},
					"messenger_channel": {"donutloop"},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)

				mo := new(settings.SlackNotificationsSettings)
				err = m.RetrieveJson(dummyContext, "messenger_slack", mo)
				So(err, ShouldBeNil)

				So(mo.Channel, ShouldEqual, "donutloop")
				So(mo.BearerToken, ShouldEqual, "sjdfklsjdfkljfs")
			})
		})
	})
}
