// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httpsettings

import (
	"encoding/json"
	slackAPI "github.com/slack-go/slack"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/meta"
	_ "gitlab.com/lightmeter/controlcenter/meta/migrations"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/notification/slack"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"golang.org/x/text/message/catalog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestSettingsPage(t *testing.T) {
	Convey("Retrieve all settings", t, func() {
		conn, closeConn := testutil.TempDBConnection(t)
		defer closeConn()

		m, err := meta.NewHandler(conn, "master")
		So(err, ShouldBeNil)

		defer func() { errorutil.MustSucceed(m.Close()) }()

		runner := meta.NewRunner(m)
		done, cancel := runner.Run()

		defer func() { cancel(); done() }()

		writer := runner.Writer()

		notifier := &fakeNotifier{}

		slackNotifier := slack.New(notification.AlwaysAllowPolicies, m.Reader)

		// don't use slack api, mocking the PostMessage call
		slackNotifier.MessagePosterBuilder = func(client *slackAPI.Client) slack.MessagePoster {
			return &fakeSlackPoster{}
		}

		nc := notification.New(m.Reader, translator.New(catalog.NewBuilder()), []notification.Notifier{notifier, slackNotifier})

		initialSetupSettings := settings.NewInitialSetupSettings(&dummySubscriber{})

		setup := NewSettings(writer, m.Reader, initialSetupSettings, nc, slackNotifier)

		// Approach: as for now we have independent endpoints, we instantiate one server per endpoint
		// But as soon as we unify them all in a single one, that'll not be needed anymore

		settingsServer := httptest.NewServer(httpmiddleware.New().WithError(httpmiddleware.CustomHTTPHandler(setup.SettingsForward)))

		c := &http.Client{}

		Convey("No settings set yields empty values", func() {
			r, err := c.Get(settingsServer.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			var body map[string]interface{}
			dec := json.NewDecoder(r.Body)
			err = dec.Decode(&body)
			So(err, ShouldBeNil)

			expected := map[string]interface{}{
				"slack_notifications": map[string]interface{}{"bearer_token": "", "channel": "", "enabled": nil, "language": ""},
				"general": map[string]interface{}{
					"postfix_public_ip": "",
					"app_language":      "",
				},
			}

			So(body, ShouldResemble, expected)
		})

		Convey("Change some settings", func() {
			// set public ip address
			{
				r, err := c.PostForm(settingsServer.URL+"?setting=general",
					url.Values{"postfixPublicIP": {"11.22.33.44"}, "app_language": {"en"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			}

			// set slack settings
			{
				r, err := c.PostForm(settingsServer.URL+"?setting=notification",
					url.Values{
						"messenger_kind":     {"slack"},
						"messenger_token":    {"some_token"},
						"messenger_channel":  {"some_channel"},
						"messenger_enabled":  {"true"},
						"messenger_language": {"en"},
					})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			}

			r, err := c.Get(settingsServer.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			var body map[string]interface{}
			dec := json.NewDecoder(r.Body)
			err = dec.Decode(&body)
			So(err, ShouldBeNil)

			expected := map[string]interface{}{
				"slack_notifications": map[string]interface{}{"bearer_token": "some_token", "channel": "some_channel", "enabled": true, "language": "en"},
				"general": map[string]interface{}{
					"postfix_public_ip": "11.22.33.44",
					"app_language":      "en",
				},
			}

			So(body, ShouldResemble, expected)
		})

		Convey("Slack notifications are disabled if the requests explicitely requests it", func() {
			// set slack settings
			{
				r, err := c.PostForm(settingsServer.URL+"?setting=notification",
					url.Values{
						"messenger_kind":     {"slack"},
						"messenger_token":    {"some_token"},
						"messenger_channel":  {"some_channel"},
						"messenger_enabled":  {"false"},
						"messenger_language": {"en"},
					})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			}

			r, err := c.Get(settingsServer.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			var body map[string]interface{}
			dec := json.NewDecoder(r.Body)
			err = dec.Decode(&body)
			So(err, ShouldBeNil)

			expected := map[string]interface{}{
				"slack_notifications": map[string]interface{}{"bearer_token": "some_token", "channel": "some_channel", "enabled": false, "language": "en"},
				"general": map[string]interface{}{
					"postfix_public_ip": "",
					"app_language":      "",
				},
			}

			So(body, ShouldResemble, expected)
		})
	})
}
