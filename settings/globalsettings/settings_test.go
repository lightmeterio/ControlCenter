// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package globalsettings

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"net"
	"testing"
)

var (
	dummyContext = context.Background()
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestInitialSetup(t *testing.T) {
	Convey("Test Initial Setup", t, func() {
		conn, closeConn := testutil.TempDBConnectionMigrated(t, "master")
		defer closeConn()

		Convey("Store and retrieve", func() {
			m, err := metadata.NewHandler(conn)
			So(err, ShouldBeNil)

			err = m.Writer.StoreJson(context.Background(), SettingKey, Settings{
				LocalIP:     NewIP(`22.33.44.55`),
				APPLanguage: "en",
				PublicURL:   "http://example.com",
			})

			So(err, ShouldBeNil)

			var s Settings
			err = m.Reader.RetrieveJson(context.Background(), SettingKey, &s)
			So(err, ShouldBeNil)

			So(s, ShouldResemble, Settings{
				LocalIP:     NewIP(`22.33.44.55`),
				APPLanguage: "en",
				PublicURL:   "http://example.com",
			})
		})

		Convey("Keep compatibility with old format stored in the database", func() {
			m, err := metadata.NewHandler(conn)
			So(err, ShouldBeNil)

			// notice we were using net.IP in the settings itself before
			type oldFormat struct {
				LocalIP     net.IP `json:"postfix_public_ip"`
				APPLanguage string `json:"app_language"`
				PublicURL   string `json:"public_url"`
			}

			Convey("read from old format", func() {
				valueInOldFormat := oldFormat{
					LocalIP:     net.ParseIP(`22.33.44.55`),
					APPLanguage: "en",
					PublicURL:   "http://example.com",
				}

				err = m.Writer.StoreJson(context.Background(), SettingKey, valueInOldFormat)
				So(err, ShouldBeNil)

				// we then retrieve the settings in the new format
				var s Settings
				err = m.Reader.RetrieveJson(context.Background(), SettingKey, &s)
				So(err, ShouldBeNil)

				So(s, ShouldResemble, Settings{
					LocalIP:     NewIP(`22.33.44.55`),
					APPLanguage: "en",
					PublicURL:   "http://example.com",
				})
			})

			Convey("save to old format", func() {
				value := Settings{
					LocalIP:     NewIP(`22.33.44.55`),
					APPLanguage: "en",
					PublicURL:   "http://example.com",
				}

				err = m.Writer.StoreJson(context.Background(), SettingKey, value)
				So(err, ShouldBeNil)

				var s oldFormat
				err = m.Reader.RetrieveJson(context.Background(), SettingKey, &s)
				So(err, ShouldBeNil)

				So(s, ShouldResemble, oldFormat{
					LocalIP:     net.ParseIP(`22.33.44.55`),
					APPLanguage: "en",
					PublicURL:   "http://example.com",
				})
			})
		})
	})
}

func TestSettingsFromDefaultValues(t *testing.T) {
	Convey("Retrieve from default values", t, func() {
		conn, closeConn := testutil.TempDBConnectionMigrated(t, "master")
		defer closeConn()

		m, err := metadata.NewDefaultedHandler(conn, metadata.DefaultValues{
			"global": map[string]interface{}{
				"localIP": map[string]interface{}{
					"value": "22.33.44.55",
				},
				"aPPLanguage": "de",
				"publicURL":   "http://example.com/lightmeter",
			},
		})

		So(err, ShouldBeNil)

		Convey("Retrieve", func() {
			var settings Settings
			err = m.Reader.RetrieveJson(context.Background(), SettingKey, &settings)
			So(err, ShouldBeNil)

			So(settings, ShouldResemble, Settings{
				LocalIP:     NewIP("22.33.44.55"),
				APPLanguage: "de",
				PublicURL:   "http://example.com/lightmeter",
			})
		})

		Convey("Empty values do not override defaults", func() {
			err = m.Writer.StoreJson(context.Background(), SettingKey, Settings{
				LocalIP:     NewIP(""),
				APPLanguage: "",
				PublicURL:   "https://another.url.example.com",
			})

			So(err, ShouldBeNil)

			var settings Settings
			err = m.Reader.RetrieveJson(context.Background(), SettingKey, &settings)
			So(err, ShouldBeNil)

			So(settings, ShouldResemble, Settings{
				LocalIP:     NewIP("22.33.44.55"),
				APPLanguage: "de",
				PublicURL:   "https://another.url.example.com",
			})
		})
	})
}
