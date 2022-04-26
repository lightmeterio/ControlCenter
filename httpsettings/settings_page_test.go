// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httpsettings

import (
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	_ "gitlab.com/lightmeter/controlcenter/metadata/migrations"
	"gitlab.com/lightmeter/controlcenter/notification/email"
	"gitlab.com/lightmeter/controlcenter/notification/slack"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func notificationValuesToPost(values url.Values) url.Values {
	// something always posted by the notifications form
	var defaultValues = url.Values{
		"messenger_enabled":                  {"false"},
		"messenger_token":                    {""},
		"messenger_channel":                  {""},
		"notification_language":              {""},
		"email_notification_server_name":     {""},
		"email_notification_skip_cert_check": {"false"},
		"email_notification_port":            {"0"},
		"email_notification_username":        {""},
		"email_notification_password":        {""},
		"email_notification_sender":          {""},
		"email_notification_recipients":      {""},
		"email_notification_security_type":   {"none"},
		"email_notification_auth_method":     {"none"},
		"email_notification_enabled":         {"false"},
	}

	for k, v := range defaultValues {
		if _, ok := values[k]; !ok {
			values[k] = v
		}
	}

	return values
}

func TestSettingsPage(t *testing.T) {
	Convey("Retrieve all settings", t, func() {
		setup, _, reader, _, _, _, clear := buildTestSetup(t)
		defer clear()

		// Approach: as for now we have independent endpoints, we instantiate one server per endpoint
		// But as soon as we unify them all in a single one, that'll not be needed anymore

		settingsServer := httptest.NewServer(httpmiddleware.New().WithEndpoint(httpmiddleware.CustomHTTPHandler(setup.SettingsForward)))

		c := &http.Client{}

		Convey("No settings set yields empty values", func() {
			r, err := c.Get(settingsServer.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			body, err := decodeBodyAsJson(r.Body)
			So(err, ShouldBeNil)

			expected := map[string]interface{}{
				"detective": map[string]interface{}{
					"end_users_enabled": false,
				},
				"email_notifications": map[string]interface{}{
					"skip_cert_check": false,
					"auth_method":     "none",
					"enabled":         false,
					"recipients":      "",
					"security_type":   "none",
					"sender":          "",
					"server_name":     "",
					"server_port":     float64(0),
				},
				"general": map[string]interface{}{
					"app_language":      "",
					"postfix_public_ip": "",
					"public_url":        "",
				},
				"insights": map[string]interface{}{
					"bounce_rate_threshold":        float64(5),
					"mail_inactivity_lookup_range": float64(24),
					"mail_inactivity_min_interval": float64(12),
				},
				"notifications": map[string]interface{}{
					"language": "",
				},
				"slack_notifications": map[string]interface{}{
					"channel": "",
					"enabled": false,
				},
				"walkthrough": map[string]interface{}{
					"completed": false,
				},
				"feature_flags": map[string]interface{}{
					"disable_v1_dashboard":  false,
					"enable_v2_dashboard":   false,
					"disable_insights_view": false,
					"disable_raw_logs":      false,
				},
			}

			So(body, ShouldResemble, expected)
		})

		Convey("Change some settings", func() {
			// set public ip address
			{
				r, err := c.PostForm(settingsServer.URL+"?setting=general",
					url.Values{
						"postfix_public_ip": {"11.22.33.44"},
						"app_language":      {"en"},
						"public_url":        {"https://example.com/lightmeter"},
					})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			}

			// set slack settings
			{
				r, err := c.PostForm(settingsServer.URL+"?setting=notification",
					notificationValuesToPost(url.Values{
						"messenger_token":       {"some_token"},
						"messenger_channel":     {"some_channel"},
						"messenger_enabled":     {"true"},
						"notification_language": {"en"},
					}))
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			}

			// set insights settings
			{
				r, err := c.PostForm(settingsServer.URL+"?setting=insights",
					notificationValuesToPost(url.Values{
						"bounce_rate_threshold":        {"80"},
						"mail_inactivity_lookup_range": {"10"},
						"mail_inactivity_min_interval": {"10"},
					}))
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			}

			r, err := c.Get(settingsServer.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			body, err := decodeBodyAsJson(r.Body)
			So(err, ShouldBeNil)

			expected := map[string]interface{}{
				"detective": map[string]interface{}{
					"end_users_enabled": false,
				},
				"email_notifications": map[string]interface{}{
					"skip_cert_check": false,
					"auth_method":     "none",
					"enabled":         false,
					"recipients":      "",
					"security_type":   "none",
					"sender":          "",
					"server_name":     "",
					"server_port":     float64(0),
				},
				"general": map[string]interface{}{
					"app_language":      "en",
					"postfix_public_ip": "11.22.33.44",
					"public_url":        "https://example.com/lightmeter",
				},
				"insights": map[string]interface{}{
					"bounce_rate_threshold":        float64(80),
					"mail_inactivity_lookup_range": float64(10),
					"mail_inactivity_min_interval": float64(10),
				},
				"notifications": map[string]interface{}{
					"language": "en",
				},
				"slack_notifications": map[string]interface{}{
					"channel": "some_channel",
					"enabled": true,
				},
				"walkthrough": map[string]interface{}{
					"completed": false,
				},
				"feature_flags": map[string]interface{}{
					"disable_v1_dashboard":  false,
					"enable_v2_dashboard":   false,
					"disable_insights_view": false,
					"disable_raw_logs":      false,
				},
			}

			So(body, ShouldResemble, expected)
		})

		Convey("Slack notifications are disabled if the requests explicitely requests it", func() {
			// set slack settings
			{
				r, err := c.PostForm(settingsServer.URL+"?setting=notification",
					notificationValuesToPost(url.Values{
						"messenger_token":       {"some_token"},
						"messenger_channel":     {"some_channel"},
						"messenger_enabled":     {"false"},
						"notification_language": {"en"},
					}))
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			}

			r, err := c.Get(settingsServer.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			body, err := decodeBodyAsJson(r.Body)
			So(err, ShouldBeNil)

			expected := map[string]interface{}{
				"detective": map[string]interface{}{
					"end_users_enabled": false,
				},
				"email_notifications": map[string]interface{}{
					"skip_cert_check": false,
					"auth_method":     "none",
					"enabled":         false,
					"recipients":      "",
					"security_type":   "none",
					"sender":          "",
					"server_name":     "",
					"server_port":     float64(0),
				},
				"general": map[string]interface{}{
					"app_language":      "",
					"postfix_public_ip": "",
					"public_url":        "",
				},
				"insights": map[string]interface{}{
					"bounce_rate_threshold":        float64(5),
					"mail_inactivity_lookup_range": float64(24),
					"mail_inactivity_min_interval": float64(12),
				},
				"notifications": map[string]interface{}{
					"language": "en",
				},
				"slack_notifications": map[string]interface{}{
					"channel": "some_channel",
					"enabled": false,
				},
				"walkthrough": map[string]interface{}{
					"completed": false,
				},
				"feature_flags": map[string]interface{}{
					"disable_v1_dashboard":  false,
					"enable_v2_dashboard":   false,
					"disable_insights_view": false,
					"disable_raw_logs":      false,
				},
			}

			So(body, ShouldResemble, expected)

			slackSettings, err := slack.GetSettings(dummyContext, reader)
			So(err, ShouldBeNil)

			// The slack settings are indeed in the database
			So(*slackSettings.BearerToken, ShouldEqual, "some_token")
		})

		Convey("Email notifications", func() {
			// sets some basic configs
			r, err := c.PostForm(settingsServer.URL+"?setting=general",
				url.Values{"postfix_public_ip": {"11.22.33.44"}, "app_language": {"en"}})
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			Convey("Fails to change email notification settings", func() {
				// fails to test connection
				r, err := c.PostForm(settingsServer.URL+"?setting=notification",
					notificationValuesToPost(url.Values{
						"email_notification_server_name":     {"mail.example.com"},
						"email_notification_skip_cert_check": {"false"},
						"email_notification_username":        {"user@mail.example.com"},
						"email_notification_password":        {"super_password"},
						"email_notification_sender":          {"sender@example.com"},
						"email_notification_recipients":      {"recipient@example.com"},
						"email_notification_security_type":   {"STARTTLS"},
						"email_notification_auth_method":     {"password"},
						"email_notification_port":            {"999"},
						"email_notification_enabled":         {"true"},
						"notification_language":              {"en"},
					}))
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
			})

			Convey("Succeeds to test connection", func() {
				stop := email.StartFakeServer(&email.FakeMailBackend{
					ExpectedUser:     "user@example.com",
					ExpectedPassword: "super_password",
				}, ":2055")

				defer stop()

				// fails to test connection
				{
					r, err := c.PostForm(settingsServer.URL+"?setting=notification",
						notificationValuesToPost(url.Values{
							"email_notification_server_name":     {"localhost"},
							"email_notification_skip_cert_check": {"false"},
							"email_notification_username":        {"user@example.com"},
							"email_notification_password":        {"super_password"},
							"email_notification_sender":          {"sender@example.com"},
							"email_notification_recipients":      {"recipient@example.com"},
							"email_notification_security_type":   {"none"},
							"email_notification_auth_method":     {"password"},
							"email_notification_port":            {"2055"},
							"email_notification_enabled":         {"true"},
							"notification_language":              {"de"},
						}))
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)
				}

				r, err := c.Get(settingsServer.URL)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)

				body, err := decodeBodyAsJson(r.Body)
				So(err, ShouldBeNil)

				expected := map[string]interface{}{
					"detective": map[string]interface{}{
						"end_users_enabled": false,
					},
					"email_notifications": map[string]interface{}{
						"skip_cert_check": false,
						"auth_method":     "password",
						"enabled":         true,
						"recipients":      "recipient@example.com",
						"security_type":   "none",
						"sender":          "sender@example.com",
						"server_name":     "localhost",
						"server_port":     float64(2055),
					},
					"general": map[string]interface{}{
						"app_language":      "en",
						"postfix_public_ip": "11.22.33.44",
						"public_url":        "",
					},
					"insights": map[string]interface{}{
						"bounce_rate_threshold":        float64(5),
						"mail_inactivity_lookup_range": float64(24),
						"mail_inactivity_min_interval": float64(12),
					},
					"notifications": map[string]interface{}{
						"language": "de",
					},
					"slack_notifications": map[string]interface{}{
						"channel": "",
						"enabled": false,
					},
					"walkthrough": map[string]interface{}{
						"completed": false,
					},
					"feature_flags": map[string]interface{}{
						"disable_v1_dashboard":  false,
						"enable_v2_dashboard":   false,
						"disable_insights_view": false,
						"disable_raw_logs":      false,
					},
				}

				So(body, ShouldResemble, expected)

				emailSettings, err := email.GetSettings(dummyContext, reader)
				So(err, ShouldBeNil)
				So(*emailSettings.Password, ShouldEqual, "super_password")
				So(*emailSettings.Username, ShouldEqual, "user@example.com")
			})
		})
	})
}

func decodeBodyAsJson(r io.Reader) (interface{}, error) {
	var body map[string]interface{}
	dec := json.NewDecoder(r)
	err := dec.Decode(&body)
	return body, err
}
