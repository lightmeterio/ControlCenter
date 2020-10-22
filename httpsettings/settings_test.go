package httpsettings

import (
	"context"
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/meta"
	_ "gitlab.com/lightmeter/controlcenter/meta/migrations"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"golang.org/x/text/message/catalog"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
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

type fakeNotificationCenter struct{
	shouldFailToAddSlackNotifier bool
}

func (c *fakeNotificationCenter) Notify(center notification.Notification) error {
	log.Println("send notification")
	return nil
}

func (c *fakeNotificationCenter) AddSlackNotifier(notificationsSettings settings.SlackNotificationsSettings) error {
	log.Println("Add slack")
	if c.shouldFailToAddSlackNotifier {
		return errors.New("Invalid slack notifier")
	}

	return nil
}

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestInitialSetup(t *testing.T) {
	Convey("Initial Setup", t, func() {
		conn, closeConn := testutil.TempDBConnection()
		defer closeConn()

		m, err := meta.NewHandler(conn, "master")
		So(err, ShouldBeNil)

		defer func() { errorutil.MustSucceed(m.Close()) }()

		runner := meta.NewRunner(m)
		done, cancel := runner.Run()

		defer func() { cancel(); done() }()

		writer := runner.Writer()

		fakeCenter := &fakeNotificationCenter{}
		initialSetupSettings := settings.NewInitialSetupSettings(&dummySubscriber{})

		setup := NewSettings(writer, m.Reader, initialSetupSettings, fakeCenter)

		chain := httpmiddleware.New()
		handler := chain.WithError(httpmiddleware.CustomHTTPHandler(setup.SettingsForward))

		s := httptest.NewServer(handler)
		c := &http.Client{}

		querySettingsParameter := "?setting=initSetup"
		settingsURL := s.URL+querySettingsParameter

		Convey("Fails", func() {
			Convey("Invalid Form data", func() {
				r, err := c.Post(settingsURL, "application/x-www-form-urlencoded", strings.NewReader(`^^%`))
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})

			Convey("Invalid mime type", func() {
				r, err := c.Post(settingsURL, "ksajdhfk*I&^&*^87678  $$343", nil)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})

			Convey("Subscribe is not a boolean", func() {
				r, err := c.PostForm(settingsURL, url.Values{"email_kind": {string(settings.MailKindTransactional)}, "subscribe_newsletter": {"Falsch"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
			})

			Convey("Unsupported multiple subscribe options", func() {
				r, err := c.PostForm(settingsURL, url.Values{"email_kind": {string(settings.MailKindTransactional)}, "subscribe_newsletter": {"on", "on"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
			})

			Convey("Incompatible mime type", func() {
				r, err := c.Post(settingsURL, "application/json", nil)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})

			Convey("Incompatible Method", func() {
				r, err := c.Head(settingsURL)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusMethodNotAllowed)
			})

			Convey("Subscribe without email", func() {
				r, err := c.PostForm(settingsURL, url.Values{"email_kind": {string(settings.MailKindDirect)}, "subscribe_newsletter": {"on"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
			})

			Convey("Subscribe with zero email", func() {
				r, err := c.PostForm(settingsURL, url.Values{"email": {}, "email_kind": {string(settings.MailKindDirect)}, "subscribe_newsletter": {"on"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey("Success", func() {
			Convey("Do subscribe", func() {
				r, err := c.PostForm(settingsURL, url.Values{
					"email":                {"user@example.com"},
					"email_kind":           {string(settings.MailKindDirect)},
					"subscribe_newsletter": {"on"},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			})

			Convey("Do not subscribe", func() {
				r, err := c.PostForm(settingsURL, url.Values{
					"email_kind": {string(settings.MailKindDirect)},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			})
		})
	})
}

func TestSettingsSetup(t *testing.T) {
	Convey("Settings Setup", t, func() {
		conn, closeConn := testutil.TempDBConnection()
		defer closeConn()

		m, err := meta.NewHandler(conn, "master")
		So(err, ShouldBeNil)

		defer func() { errorutil.MustSucceed(m.Close()) }()
		runner := meta.NewRunner(m)
		done, cancel := runner.Run()
		defer func() { cancel(); done() }()
		writer := runner.Writer()

		fakeCenter := &fakeNotificationCenter{}
		initialSetupSettings := settings.NewInitialSetupSettings(&dummySubscriber{})

		setup := NewSettings(writer, m.Reader, initialSetupSettings, fakeCenter)


		chain := httpmiddleware.New()
		handler := chain.WithError(httpmiddleware.CustomHTTPHandler(setup.SettingsForward))
		s := httptest.NewServer(handler)

		querySettingsParameter := "?setting=notification"
		settingsURL := s.URL+querySettingsParameter

		c := &http.Client{}

		Convey("Fails", func() {
			Convey("Invalid Form data", func() {
				r, err := c.Post(settingsURL, "application/x-www-form-urlencoded", strings.NewReader(`^^%`))
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})

			Convey("Invalid mime type", func() {
				r, err := c.Post(settingsURL, "ksajdhfk*I&^&*^87678  $$343", nil)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})

			Convey("Incompatible Method", func() {
				r, err := c.Head(settingsURL)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusMethodNotAllowed)
			})
		})

		Convey("Success", func() {
			Convey("send valid values", func() {
				r, err := c.PostForm(settingsURL, url.Values{
					"messenger_kind":    {"slack"},
					"messenger_token":   {"sjdfklsjdfkljfs"},
					"messenger_channel": {"donutloop"},
					"messenger_enabled": {"true"},
					"messenger_language": {"de"},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)

				mo := new(settings.SlackNotificationsSettings)
				err = m.Reader.RetrieveJson(dummyContext, "messenger_slack", mo)
				So(err, ShouldBeNil)

				So(mo.Channel, ShouldEqual, "donutloop")
				So(mo.BearerToken, ShouldEqual, "sjdfklsjdfkljfs")
			})
		})
	})
}

type fakeContent struct{}

func (c *fakeContent) String() string {
	return "Hell world!, Mister Donutloop 2"
}

func (c *fakeContent) Args() []interface{} {
	return nil
}

func (c *fakeContent) TplString() string {
	return "Hell world!, Mister Donutloop 2"
}

// todo(marcel) before we create a release stub out the slack api
func TestIntegrationSettingsSetup(t *testing.T) {
	Convey("Integration Settings Setup", t, func() {
		conn, closeConn := testutil.TempDBConnection()
		defer closeConn()

		m, err := meta.NewHandler(conn, "master")
		So(err, ShouldBeNil)

		defer func() { errorutil.MustSucceed(m.Close()) }()

		runner := meta.NewRunner(m)
		done, cancel := runner.Run()
		defer func() { cancel(); done() }()
		writer := runner.Writer()

		fakeCenter := &fakeNotificationCenter{}
		initialSetupSettings := settings.NewInitialSetupSettings(&dummySubscriber{})

		setup := NewSettings(writer, m.Reader, initialSetupSettings, fakeCenter)

		center := notification.New(m.Reader, translator.New(catalog.NewBuilder()))

		chain := httpmiddleware.New()
		handler := chain.WithError(httpmiddleware.CustomHTTPHandler(setup.SettingsForward))

		c := &http.Client{}

		s := httptest.NewServer(handler)
		querySettingsParameter := "?setting=notification"
		settingsURL := s.URL+querySettingsParameter

		Convey("Success", func() {
			Convey("send valid values", func() {
				r, err := c.PostForm(settingsURL, url.Values{
					"messenger_kind":    {"slack"},
					"messenger_token":   {"xoxb-1388191062644-1385067635637-5dvVTcz77UHTyFDwmjZY6sEz"},
					"messenger_channel": {"general"},
					"messenger_enabled": {"true"},
					"messenger_language": {"de"},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)

				r, err = c.PostForm(settingsURL, url.Values{
					"messenger_kind":    {"slack"},
					"messenger_token":   {"xoxb-1388191062644-1385067635637-5dvVTcz77UHTyFDwmjZY6sEz"},
					"messenger_channel": {"general"},
					"messenger_enabled": {"true"},
					"messenger_language": {"en"},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)

				mo := new(settings.SlackNotificationsSettings)
				err = m.Reader.RetrieveJson(dummyContext, "messenger_slack", mo)
				So(err, ShouldBeNil)

				So(mo.Channel, ShouldEqual, "general")
				So(mo.BearerToken, ShouldEqual, "xoxb-1388191062644-1385067635637-5dvVTcz77UHTyFDwmjZY6sEz")

				content := new(fakeContent)
				notification := notification.Notification{
					ID:      0,
					Content: content,
				}

				err = center.Notify(notification)
				So(err, ShouldBeNil)
			})

			Convey("Fails if slack validations fail", func() {
				fakeCenter.shouldFailToAddSlackNotifier = true

				r, err := c.PostForm(settingsURL, url.Values{
					"messenger_kind":    {"slack"},
					"messenger_token":   {"sjdfklsjdfkljfs"},
					"messenger_channel": {"donutloop"},
					"messenger_enabled": {"true"},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)

				mo := new(settings.SlackNotificationsSettings)
				err = m.Reader.RetrieveJson(dummyContext, "messenger_slack", mo)
				So(errors.Is(err, meta.ErrNoSuchKey), ShouldBeTrue)
			})

		})
	})
}
