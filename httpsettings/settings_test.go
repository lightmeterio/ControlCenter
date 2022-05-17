// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httpsettings

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	slackAPI "github.com/slack-go/slack"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/deliverydb"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/insights/highrate"
	"gitlab.com/lightmeter/controlcenter/insights/mailinactivity"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/metadata"
	_ "gitlab.com/lightmeter/controlcenter/metadata/migrations"
	"gitlab.com/lightmeter/controlcenter/notification"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/notification/email"
	"gitlab.com/lightmeter/controlcenter/notification/slack"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	insightsSettings "gitlab.com/lightmeter/controlcenter/settings/insights"
	"gitlab.com/lightmeter/controlcenter/settings/walkthrough"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"golang.org/x/text/message/catalog"
)

var (
	dummyContext = context.Background()
)

type dummySubscriber struct{}

func (*dummySubscriber) Subscribe(ctx context.Context, email string) error {
	log.Info().Msgf("A dummy call that would otherwise subscribe email %v to Lightmeter newsletter :-)", email)
	return nil
}

type fakeFetcher struct{}

func (f *fakeFetcher) FetchInsights(c context.Context, o core.FetchOptions, t timeutil.Clock) ([]core.FetchedInsight, error) {
	return nil, nil
}

type fakeNotifier struct {
}

func (c *fakeNotifier) ValidateSettings(notificationCore.Settings) error {
	return nil
}

func (c *fakeNotifier) Notify(notification.Notification, translator.Translator) error {
	log.Info().Msg("send notification")
	return nil
}

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

type fakeSlackPoster struct {
	err error
}

var fakeSlackError = errors.New(`Some Slack Error`)

func (p *fakeSlackPoster) PostMessage(channelID string, options ...slackAPI.MsgOption) (string, string, error) {
	return "", "", p.err
}

func buildTestSetup(t *testing.T) (*Settings, *metadata.AsyncWriter, metadata.Reader, *notification.Center, *fakeSlackPoster, *insights.Engine, func()) {
	conn, closeConn := testutil.TempDBConnectionMigrated(t, "master")
	connInsights, closeConnInsights := testutil.TempDBConnectionMigrated(t, "insights")
	connLogs, closeConnLogs := testutil.TempDBConnectionMigrated(t, "logs")

	// NOTE: finish migration of logs
	_, err := deliverydb.New(connLogs, &domainmapping.DefaultMapping)
	So(err, ShouldBeNil)

	m, err := metadata.NewHandler(conn)
	So(err, ShouldBeNil)

	writeRunner := metadata.NewSerialWriteRunner(m)
	done, cancel := runner.Run(writeRunner)

	writer := writeRunner.Writer()

	initialSetupSettings := settings.NewInitialSetupSettings(&dummySubscriber{})

	fakeNotifier := &fakeNotifier{}

	slackNotifier := slack.New(notification.PassPolicy, m.Reader)

	fakeSlackPoster := &fakeSlackPoster{}

	// don't use slack api, mocking the PostMessage call
	slackNotifier.MessagePosterBuilder = func(client *slackAPI.Client) slack.MessagePoster {
		return fakeSlackPoster
	}

	emailNotifier := email.New(notification.PassPolicy, m.Reader)

	notifiers := map[string]notification.Notifier{
		slack.SettingsKey: slackNotifier,
		email.SettingsKey: emailNotifier,
		"fake":            fakeNotifier,
	}

	center := notification.New(m.Reader, translator.New(catalog.NewBuilder()), notification.PassPolicy, notifiers)

	insightsAccessor, err := insights.NewAccessor(connInsights)
	So(err, ShouldBeNil)

	dashboard, err := dashboard.New(connLogs.RoConnPool)
	So(err, ShouldBeNil)

	insightsEngine, err := insights.NewCustomEngine(
		&m.Reader,
		insightsAccessor,
		&fakeFetcher{},
		center,
		core.Options{
			"dashboard":    dashboard,
			"logsConnPool": connLogs.RoConnPool,
		},
		insights.SettingsDetectors,
		insights.NoAdditionalActions,
	)
	So(err, ShouldBeNil)

	setup := NewSettings(writer, m.Reader, initialSetupSettings, center, insightsEngine)

	return setup, writer, m.Reader, center, fakeSlackPoster, insightsEngine, func() {
		cancel()
		done()
		closeConn()
		closeConnInsights()
		closeConnLogs()
	}
}

func TestInitialSetup(t *testing.T) {
	Convey("Initial Setup", t, func() {
		setup, _, _, _, _, _, clear := buildTestSetup(t)
		defer clear()

		chain := httpmiddleware.New()
		handler := chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(setup.SettingsForward))

		s := httptest.NewServer(handler)
		c := &http.Client{}

		querySettingsParameter := "?setting=initSetup"
		settingsURL := s.URL + querySettingsParameter

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
				r, err := c.PostForm(settingsURL, url.Values{"subscribe_newsletter": {"Falsch"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
			})

			Convey("Unsupported multiple subscribe options", func() {
				r, err := c.PostForm(settingsURL, url.Values{"subscribe_newsletter": {"on", "on"}})
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
				r, err := c.PostForm(settingsURL, url.Values{"subscribe_newsletter": {"on"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
			})

			Convey("Subscribe with zero email", func() {
				r, err := c.PostForm(settingsURL, url.Values{"email": {}, "subscribe_newsletter": {"on"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
			})

			Convey("invalid ip", func() {
				r, err := c.PostForm(settingsURL, url.Values{
					"email":                {"user@example.com"},
					"subscribe_newsletter": {"on"},
					"app_language":         {"en"},
					"postfix_public_ip":    {"9.9.9.X"},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey("Success", func() {
			Convey("Do subscribe", func() {
				r, err := c.PostForm(settingsURL, url.Values{
					"email":                {"user@example.com"},
					"subscribe_newsletter": {"on"},
					"app_language":         {"en"},
					"postfix_public_ip":    {"9.9.9.9"},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			})

			Convey("Do not subscribe", func() {
				r, err := c.PostForm(settingsURL, url.Values{
					"postfix_public_ip": {"9.9.9.9"},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			})
		})
	})
}

func TestAppSettings(t *testing.T) {
	Convey("Settings Setup", t, func() {
		setup, writer, reader, _, _, _, clear := buildTestSetup(t)
		defer clear()

		chain := httpmiddleware.New()
		handler := chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(setup.SettingsForward))
		s := httptest.NewServer(handler)

		c := &http.Client{}

		Convey("Do not clean IP settings when updating the language", func() {
			// First set an IP address manually
			writer.StoreJson(globalsettings.SettingsKey, &globalsettings.Settings{
				LocalIP:     globalsettings.IP{net.ParseIP(`127.0.0.1`)},
				AppLanguage: "en",
			}).Wait()

			// Set the app language via http, not posting the ip address
			r, err := c.PostForm(s.URL+"?setting=general", url.Values{
				"app_language": {"de"},
			})

			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			// The IP address must be intact
			settings := globalsettings.Settings{}
			err = reader.RetrieveJson(context.Background(), globalsettings.SettingsKey, &settings)
			So(err, ShouldBeNil)

			So(settings.AppLanguage, ShouldEqual, "de")
			So(settings.LocalIP.String(), ShouldEqual, "127.0.0.1")
		})
	})

	Convey("Clear general settings", t, func() {
		setup, writer, reader, _, _, _, clear := buildTestSetup(t)
		defer clear()

		chain := httpmiddleware.New()
		handler := chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(setup.SettingsForward))
		s := httptest.NewServer(handler)

		c := &http.Client{}

		Convey("Do not reset language when we clear general settings", func() {
			writer.StoreJson(globalsettings.SettingsKey, &globalsettings.Settings{
				LocalIP:     globalsettings.IP{net.ParseIP(`127.0.0.1`)},
				PublicURL:   "http://localhost:8080",
				AppLanguage: "de",
			}).Wait()

			// Check that the settings are set
			settings := globalsettings.Settings{}
			err := reader.RetrieveJson(context.Background(), globalsettings.SettingsKey, &settings)
			So(err, ShouldBeNil)

			So(settings.LocalIP.String(), ShouldEqual, `127.0.0.1`)
			So(settings.PublicURL, ShouldEqual, "http://localhost:8080")
			So(settings.AppLanguage, ShouldEqual, "de")

			// Clear general settings
			r, err := c.PostForm(s.URL+"?setting=general", url.Values{"action": {"clear"}})

			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			// The IP address and postfix URL must be cleared, but the language should stay
			settings = globalsettings.Settings{}
			err = reader.RetrieveJson(context.Background(), globalsettings.SettingsKey, &settings)
			So(err, ShouldBeNil)

			So(settings.LocalIP.IP, ShouldBeNil)
			So(settings.PublicURL, ShouldEqual, "")
			So(settings.AppLanguage, ShouldEqual, "de")
		})
	})
}

type fakeContentComponent string

func (c fakeContentComponent) String() string {
	return string(c)
}

func (c fakeContentComponent) Args() []interface{} {
	return nil
}

func (c fakeContentComponent) TplString() string {
	return c.String()
}

type fakeContent struct {
}

func (c fakeContent) Title() notificationCore.ContentComponent {
	return fakeContentComponent("some fake content")
}

func (c fakeContent) Description() notificationCore.ContentComponent {
	return fakeContentComponent("some fake description")
}

func (c fakeContent) Metadata() notificationCore.ContentMetadata {
	return nil
}

func TestSlackNotifications(t *testing.T) {
	Convey("Integration Settings Setup", t, func() {
		setup, _, reader, center, fakeSlackPoster, _, clear := buildTestSetup(t)
		defer clear()

		chain := httpmiddleware.New()
		handler := chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(setup.SettingsForward))

		c := &http.Client{}

		s := httptest.NewServer(handler)
		querySettingsParameter := "?setting=notification"
		settingsURL := s.URL + querySettingsParameter

		Convey("Settings", func() {
			querySettingsParameter := "?setting=notification"
			settingsURL := s.URL + querySettingsParameter

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
						"messenger_kind":     {"slack"},
						"messenger_token":    {"sjdfklsjdfkljfs"},
						"messenger_channel":  {"donutloop"},
						"messenger_enabled":  {"true"},
						"messenger_language": {"de"},
					})

					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)

					mo := new(slack.Settings)
					err = reader.RetrieveJson(dummyContext, slack.SettingsKey, mo)
					So(err, ShouldBeNil)

					So(mo.Channel, ShouldEqual, "donutloop")
					So(*mo.BearerToken, ShouldEqual, "sjdfklsjdfkljfs")
				})
			})
		})

		Convey("Success", func() {
			Convey("send valid values", func() {
				r, err := c.PostForm(settingsURL, url.Values{
					"messenger_kind":     {"slack"},
					"messenger_token":    {"some_valid_key"},
					"messenger_channel":  {"general"},
					"messenger_enabled":  {"true"},
					"messenger_language": {"de"},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)

				r, err = c.PostForm(settingsURL, url.Values{
					"messenger_kind":     {"slack"},
					"messenger_token":    {"some_valid_key"},
					"messenger_channel":  {"general"},
					"messenger_enabled":  {"true"},
					"messenger_language": {"en"},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)

				mo := new(slack.Settings)
				err = reader.RetrieveJson(dummyContext, slack.SettingsKey, mo)
				So(err, ShouldBeNil)

				So(mo.Channel, ShouldEqual, "general")
				So(*mo.BearerToken, ShouldEqual, "some_valid_key")

				content := new(fakeContent)
				notification := notification.Notification{
					ID:      0,
					Content: content,
				}

				err = center.Notify(notification)
				So(err, ShouldBeNil)
			})

			Convey("Fails if slack validations fail", func() {
				fakeSlackPoster.err = fakeSlackError

				r, err := c.PostForm(settingsURL, url.Values{
					"messenger_kind":    {"slack"},
					"messenger_token":   {"some_invalid_key"},
					"messenger_channel": {"donutloop"},
					"messenger_enabled": {"true"},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)

				mo := new(slack.Settings)
				err = reader.RetrieveJson(dummyContext, slack.SettingsKey, mo)
				So(errors.Is(err, metadata.ErrNoSuchKey), ShouldBeTrue)
			})
		})
	})

	Convey("Reset slack settings", t, func() {
		setup, _, reader, _, _, _, clear := buildTestSetup(t)
		defer clear()

		chain := httpmiddleware.New()
		handler := chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(setup.SettingsForward))

		c := &http.Client{}

		s := httptest.NewServer(handler)
		querySettingsParameter := "?setting=notification"
		settingsURL := s.URL + querySettingsParameter

		Convey("Reset slack settings should clear all fields", func() {
			r, err := c.PostForm(settingsURL, url.Values{
				"messenger_kind":    {"slack"},
				"messenger_token":   {"some_valid_key"},
				"messenger_channel": {"general"},
				"messenger_enabled": {"true"},
			})

			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			mo := new(slack.Settings)
			err = reader.RetrieveJson(dummyContext, slack.SettingsKey, mo)
			So(err, ShouldBeNil)

			So(mo.Channel, ShouldEqual, "general")
			So(*mo.BearerToken, ShouldEqual, "some_valid_key")

			// Reset slack settings
			r, err = c.PostForm(s.URL+"?setting=notification", url.Values{"action": {"clear"}, "subsection": {"slack"}})

			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			// The slack fields should be cleared
			mo = new(slack.Settings)
			err = reader.RetrieveJson(dummyContext, slack.SettingsKey, mo)
			So(err, ShouldBeNil)

			So(mo.Channel, ShouldEqual, "")
			So(mo.BearerToken, ShouldBeNil)
		})
	})
}

func TestEmailNotifications(t *testing.T) {
	Convey("Email Notifications", t, func() {
		setup, w, _, center, _, _, clear := buildTestSetup(t)
		defer clear()

		// set some basic global settings required by the email notifier
		err := globalsettings.SetSettings(dummyContext, w, globalsettings.Settings{
			PublicURL: "https://example.com/lightmeter",
		})

		So(err, ShouldBeNil)

		emailBackend := &email.FakeMailBackend{ExpectedUser: "user@example.com", ExpectedPassword: "super_password"}

		stop := email.StartFakeServer(emailBackend, ":10027")
		defer stop()

		chain := httpmiddleware.New()
		handler := chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(setup.SettingsForward))

		c := &http.Client{}

		s := httptest.NewServer(handler)
		querySettingsParameter := "?setting=notification"
		settingsURL := s.URL + querySettingsParameter

		Convey("Fail due wrong configuration (username)", func() {
			r, err := c.PostForm(settingsURL, url.Values{
				"email_notification_enabled":         {"true"},
				"email_notification_skip_cert_check": {"false"},
				"email_notification_server_name":     {"localhost"},
				"email_notification_port":            {"10027"},
				"email_notification_security_type":   {"none"},
				"email_notification_auth_method":     {"password"},
				"email_notification_username":        {"wronguser@example.com"},
				"email_notification_password":        {"super_password"},
				"email_notification_sender":          {"sender@example.com"},
				"email_notification_recipients":      {"Some Person <some.person@example.com>, Someone Else <someone@else.example.com>"},
			})

			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusBadRequest)

			So(len(emailBackend.Messages), ShouldEqual, 0)
		})

		Convey("Succeeds, but it's disabled", func() {
			r, err := c.PostForm(settingsURL, url.Values{
				"email_notification_enabled":         {"false"},
				"email_notification_skip_cert_check": {"false"},
				"email_notification_server_name":     {"localhost"},
				"email_notification_port":            {"10027"},
				"email_notification_security_type":   {"none"},
				"email_notification_auth_method":     {"password"},
				"email_notification_username":        {"user@example.com"},
				"email_notification_password":        {"super_password"},
				"email_notification_sender":          {"sender@example.com"},
				"email_notification_recipients":      {"Some Person <some.person@example.com>, Someone Else <someone@else.example.com>"},
			})

			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			err = center.Notify(notification.Notification{
				ID:      0,
				Content: &fakeContent{},
			})

			So(err, ShouldBeNil)

			So(len(emailBackend.Messages), ShouldEqual, 0)
		})

		Convey("Succeeds to setup and sends one notification", func() {
			r, err := c.PostForm(settingsURL, url.Values{
				"email_notification_enabled":         {"true"},
				"email_notification_skip_cert_check": {"false"},
				"email_notification_server_name":     {"localhost"},
				"email_notification_port":            {"10027"},
				"email_notification_security_type":   {"none"},
				"email_notification_auth_method":     {"password"},
				"email_notification_username":        {"user@example.com"},
				"email_notification_password":        {"super_password"},
				"email_notification_sender":          {"sender@example.com"},
				"email_notification_recipients":      {"Some Person <some.person@example.com>, Someone Else <someone@else.example.com>"},
			})

			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			err = center.Notify(notification.Notification{
				ID:      0,
				Content: &fakeContent{},
			})

			So(err, ShouldBeNil)

			So(len(emailBackend.Messages), ShouldEqual, 1)
			msg := emailBackend.Messages[0]

			So(msg.Header.Get("From"), ShouldEqual, "sender@example.com")
			So(msg.Header.Get("To"), ShouldEqual, "Some Person <some.person@example.com>, Someone Else <someone@else.example.com>")
		})
	})

	Convey("Reset email settings", t, func() {
		setup, _, reader, _, _, _, clear := buildTestSetup(t)
		defer clear()

		emailBackend := &email.FakeMailBackend{ExpectedUser: "user@example.com", ExpectedPassword: "super_password"}

		stop := email.StartFakeServer(emailBackend, ":10027")
		defer stop()

		chain := httpmiddleware.New()
		handler := chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(setup.SettingsForward))

		c := &http.Client{}

		s := httptest.NewServer(handler)
		querySettingsParameter := "?setting=notification"
		settingsURL := s.URL + querySettingsParameter

		Convey("Reset email settings", func() {
			r, err := c.PostForm(settingsURL, url.Values{
				"email_notification_enabled":         {"true"},
				"email_notification_skip_cert_check": {"false"},
				"email_notification_server_name":     {"localhost"},
				"email_notification_port":            {"10027"},
				"email_notification_security_type":   {"none"},
				"email_notification_auth_method":     {"password"},
				"email_notification_username":        {"user@example.com"},
				"email_notification_password":        {"super_password"},
				"email_notification_sender":          {"sender@example.com"},
				"email_notification_recipients":      {"Some Person <some.person@example.com>, Someone Else <someone@else.example.com>"},
			})

			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			settings, err := email.GetSettings(context.Background(), reader)
			So(err, ShouldBeNil)

			err = reader.RetrieveJson(context.Background(), email.SettingsKey, &settings)
			So(err, ShouldBeNil)

			So(settings.Sender, ShouldEqual, "sender@example.com")
			So(settings.Recipients, ShouldEqual, "Some Person <some.person@example.com>, Someone Else <someone@else.example.com>")

			// Reset email settings
			r, err = c.PostForm(s.URL+"?setting=notification", url.Values{"action": {"clear"}, "subsection": {"email"}})

			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			settings, err = email.GetSettings(context.Background(), reader)
			So(err, ShouldBeNil)

			So(settings.Sender, ShouldEqual, "")
			So(settings.Recipients, ShouldEqual, "")
		})
	})
}

func TestWalkthroughSettings(t *testing.T) {
	Convey("Integration Settings Setup", t, func() {
		setup, _, reader, _, _, _, clear := buildTestSetup(t)
		defer clear()

		chain := httpmiddleware.New()
		handler := chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(setup.SettingsForward))

		w := &walkthrough.Settings{}
		So(errors.Is(reader.RetrieveJson(dummyContext, walkthrough.SettingsKey, w), metadata.ErrNoSuchKey), ShouldBeTrue)

		c := &http.Client{}

		s := httptest.NewServer(handler)

		Convey("Save", func() {
			querySettingsParameter := "?setting=walkthrough"
			settingsURL := s.URL + querySettingsParameter

			Convey("Fails", func() {
				Convey("Invalid Form data", func() {
					r, err := c.Post(settingsURL, "application/x-www-form-urlencoded", strings.NewReader(`^^%`))
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
				})

				Convey("Invalid mime type", func() {
					r, err := c.Post(settingsURL, "ksajdhfk*I&^&*^87678  $$343", nil)
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
				})

				Convey("Incompatible Method", func() {
					r, err := c.Head(settingsURL)
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusMethodNotAllowed)
				})

				Convey("Invalid option", func() {
					r, err := c.PostForm(settingsURL, url.Values{
						"completed": {"banana"},
					})

					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
				})
			})

			Convey("Success", func() {
				Convey("set as completed", func() {
					r, err := c.PostForm(settingsURL, url.Values{
						"completed": {"true"},
					})

					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)

					w := &walkthrough.Settings{}
					So(reader.RetrieveJson(dummyContext, walkthrough.SettingsKey, w), ShouldBeNil)
					So(w.Completed, ShouldBeTrue)

					Convey("Retrieve", func() {
						r, err := c.Get(s.URL)
						So(err, ShouldBeNil)
						So(r.StatusCode, ShouldEqual, http.StatusOK)

						body, err := decodeBodyAsJson(r.Body)
						So(err, ShouldBeNil)

						asMap, ok := body.(map[string]interface{})
						So(ok, ShouldBeTrue)
						So(asMap["walkthrough"], ShouldResemble, map[string]interface{}{"completed": true})
					})
				})
			})
		})
	})
}

func TestInsightsSettings(t *testing.T) {
	Convey("Insights Settings", t, func() {
		setup, _, reader, _, _, insightsEngine, clear := buildTestSetup(t)
		defer clear()

		chain := httpmiddleware.New()
		handler := chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(setup.SettingsForward))

		w := &insightsSettings.Settings{}
		So(errors.Is(reader.RetrieveJson(dummyContext, insightsSettings.SettingsKey, w), metadata.ErrNoSuchKey), ShouldBeTrue)

		c := &http.Client{}

		s := httptest.NewServer(handler)

		Convey("Save", func() {
			querySettingsParameter := "?setting=insights"
			settingsURL := s.URL + querySettingsParameter

			Convey("Fails", func() {
				Convey("Invalid Form data", func() {
					r, err := c.Post(settingsURL, "application/x-www-form-urlencoded", strings.NewReader(`^^%`))
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
				})

				Convey("Invalid mime type", func() {
					r, err := c.Post(settingsURL, "ksajdhfk*I&^&*^87678  $$343", nil)
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
				})

				Convey("Incompatible Method", func() {
					r, err := c.Head(settingsURL)
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusMethodNotAllowed)
				})

				Convey("Invalid option for bounce_rate_threshold", func() {
					r, err := c.PostForm(settingsURL, url.Values{
						"bounce_rate_threshold":        {"abc"},
						"mail_inactivity_lookup_range": {"12"},
						"mail_inactivity_min_interval": {"8"},
					})
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)

					r, err = c.PostForm(settingsURL, url.Values{
						"bounce_rate_threshold":        {"1000"},
						"mail_inactivity_lookup_range": {"12"},
						"mail_inactivity_min_interval": {"8"},
					})
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)

					r, err = c.PostForm(settingsURL, url.Values{
						"bounce_rate_threshold":        {"-1"},
						"mail_inactivity_lookup_range": {"12"},
						"mail_inactivity_min_interval": {"8"},
					})
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)

					r, err = c.PostForm(settingsURL, url.Values{
						"bounce_rate_threshold":        {"0.5"},
						"mail_inactivity_lookup_range": {"12"},
						"mail_inactivity_min_interval": {"8"},
					})
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
				})

				Convey("Invalid option for mail_inactivity", func() {
					r, err := c.PostForm(settingsURL, url.Values{
						"bounce_rate_threshold":        {"50"},
						"mail_inactivity_lookup_range": {"12"},
						"mail_inactivity_min_interval": {"abc"},
					})
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)

					r, err = c.PostForm(settingsURL, url.Values{
						"bounce_rate_threshold":        {"50"},
						"mail_inactivity_lookup_range": {"12"},
						"mail_inactivity_min_interval": {"0"},
					})
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)

					r, err = c.PostForm(settingsURL, url.Values{
						"bounce_rate_threshold":        {"50"},
						"mail_inactivity_lookup_range": {"12"},
						"mail_inactivity_min_interval": {"1000"},
					})
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)

					r, err = c.PostForm(settingsURL, url.Values{
						"bounce_rate_threshold":        {"50"},
						"mail_inactivity_lookup_range": {"12"},
						"mail_inactivity_min_interval": {"10.12"},
					})
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
				})
			})

			Convey("Success", func() {
				Convey("Set to 53%", func() {
					r, err := c.PostForm(settingsURL, url.Values{
						"bounce_rate_threshold":        {"53"},
						"mail_inactivity_lookup_range": {"12"},
						"mail_inactivity_min_interval": {"8"},
					})

					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)

					i := &insightsSettings.Settings{}
					So(reader.RetrieveJson(dummyContext, insightsSettings.SettingsKey, i), ShouldBeNil)
					So(i.BounceRateThreshold, ShouldEqual, 53)

					Convey("Retrieve", func() {
						r, err := c.Get(s.URL)
						So(err, ShouldBeNil)
						So(r.StatusCode, ShouldEqual, http.StatusOK)

						body, err := decodeBodyAsJson(r.Body)
						So(err, ShouldBeNil)

						asMap, ok := body.(map[string]interface{})
						So(ok, ShouldBeTrue)
						So(asMap["insights"], ShouldResemble, map[string]interface{}{
							"bounce_rate_threshold":        float64(53),
							"mail_inactivity_lookup_range": float64(12),
							"mail_inactivity_min_interval": float64(8),
						})
					})

					Convey("Detectors should be updated", func() {
						foundDetectors := map[string]int{
							"hrd": 0,
							"mi":  0,
						}

						detectors := insightsEngine.GetCoreDetectors()
						So(len(detectors), ShouldEqual, 2)

						for _, d := range detectors {
							if hrd, ok := d.(*highrate.Detector); ok {
								So(hrd.GetBounceRateThreshold(), ShouldEqual, 0.53)
								foundDetectors["hrd"]++
							}

							if mi, ok := d.(*mailinactivity.Detector); ok {
								So(mi.GetOptions().LookupRange, ShouldEqual, time.Hour*12)
								So(mi.GetOptions().MinTimeGenerationInterval, ShouldEqual, time.Hour*8)
								foundDetectors["mi"]++
							}
						}

						So(foundDetectors["hrd"], ShouldEqual, 1)
						So(foundDetectors["mi"], ShouldEqual, 1)
					})
				})
			})
		})
	})
}
