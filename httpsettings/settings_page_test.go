package httpsettings

import (
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/meta"
	_ "gitlab.com/lightmeter/controlcenter/meta/migrations"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
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

		// Approach: as for now we have independent endpoints, we instantiate one server per endpoint
		// But as soon as we unify them all in a single one, that'll not be needed anymore

		allSettingsServer := httptest.NewServer(httpmiddleware.New().WithError(httpmiddleware.CustomHTTPHandler(setup.SettingsHandler)))
		generalSettingsServer := httptest.NewServer(httpmiddleware.New().WithError(httpmiddleware.CustomHTTPHandler(setup.GeneralSettingsHandler)))
		notificationsSettingsServer := httptest.NewServer(httpmiddleware.New().WithError(httpmiddleware.CustomHTTPHandler(setup.NotificationSettingsHandler)))

		c := &http.Client{}

		Convey("No settings set yields empty values", func() {
			r, err := c.Get(allSettingsServer.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			var body map[string]interface{}
			dec := json.NewDecoder(r.Body)
			err = dec.Decode(&body)
			So(err, ShouldBeNil)

			expected := map[string]interface{}{
				"slack_notifications": map[string]interface{}{"bearer_token": "", "channel": "", "enabled": false},
				"general": map[string]interface{}{
					"postfix_public_ip": "",
				},
			}

			So(body, ShouldResemble, expected)
		})

		Convey("Change some settings", func() {
			// set public ip address
			{
				r, err := c.PostForm(generalSettingsServer.URL,
					url.Values{"postfixPublicIP": {"11.22.33.44"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			}

			// set slack settings
			{
				r, err := c.PostForm(notificationsSettingsServer.URL,
					url.Values{
						"messenger_kind":    {"slack"},
						"messenger_token":   {"some_token"},
						"messenger_channel": {"some_channel"},
						"messenger_enabled": {"true"},
					})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
			}

			r, err := c.Get(allSettingsServer.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			var body map[string]interface{}
			dec := json.NewDecoder(r.Body)
			err = dec.Decode(&body)
			So(err, ShouldBeNil)

			expected := map[string]interface{}{
				"slack_notifications": map[string]interface{}{"bearer_token": "some_token", "channel": "some_channel", "enabled": true},
				"general": map[string]interface{}{
					"postfix_public_ip": "11.22.33.44",
				},
			}

			So(body, ShouldResemble, expected)
		})

	})
}
