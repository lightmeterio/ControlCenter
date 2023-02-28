// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package globalsettings

import (
	"context"
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
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

			err = m.Writer.StoreJson(context.Background(), SettingsKey, Settings{
				LocalIP:     IP{net.ParseIP(`22.33.44.55`)},
				AppLanguage: "en",
				PublicURL:   "http://example.com",
			})

			So(err, ShouldBeNil)

			s, err := GetSettings(context.Background(), m.Reader)
			So(err, ShouldBeNil)

			So(s, ShouldResemble, &Settings{
				LocalIP:     IP{net.ParseIP(`22.33.44.55`)},
				AppLanguage: "en",
				PublicURL:   "http://example.com",
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
				"localIP":     "22.33.44.55",
				"appLanguage": "de",
				"publicURL":   "http://example.com/lightmeter",
			},
		})

		So(err, ShouldBeNil)

		Convey("Retrieve", func() {
			settings, err := GetSettings(context.Background(), m.Reader)
			So(err, ShouldBeNil)

			So(settings, ShouldResemble, &Settings{
				LocalIP:     IP{net.ParseIP("22.33.44.55")},
				AppLanguage: "de",
				PublicURL:   "http://example.com/lightmeter",
			})
		})

		Convey("Check in·valid settings", func() {
			writeRunner := metadata.NewSerialWriteRunner(m)
			done, cancel := runner.Run(writeRunner)
			w := writeRunner.Writer()

			defer func() {
				cancel()
				So(done(), ShouldBeNil)
			}()

			err := SetSettings(dummyContext, w, Settings{PublicURL: "https://example.com/lightmeter"})
			So(err, ShouldBeNil)

			err = SetSettings(dummyContext, w, Settings{PublicURL: "http://localhost"})
			So(err, ShouldBeNil)

			err = SetSettings(dummyContext, w, Settings{PublicURL: "http://localhost:80"})
			So(err, ShouldBeNil)

			err = SetSettings(dummyContext, w, Settings{PublicURL: "abc"})
			So(err, ShouldNotBeNil)
			So(errors.Is(err, ErrPublicURLInvalid), ShouldBeTrue)

			err = SetSettings(dummyContext, w, Settings{PublicURL: "http://abc"})
			So(err, ShouldNotBeNil)
			So(errors.Is(err, ErrPublicURLNoDNS), ShouldBeTrue)
		})

		Convey("Empty values do not override defaults", func() {
			err = m.Writer.StoreJson(context.Background(), SettingsKey, Settings{
				LocalIP:     IP{nil},
				AppLanguage: "",
				PublicURL:   "https://another.url.example.com",
			})

			So(err, ShouldBeNil)

			settings, err := GetSettings(context.Background(), m.Reader)
			So(err, ShouldBeNil)

			So(settings, ShouldResemble, &Settings{
				LocalIP:     IP{net.ParseIP("22.33.44.55")},
				AppLanguage: "de",
				PublicURL:   "https://another.url.example.com",
			})
		})

		Convey("Keep compatibility with old format stored in the database", func() {
			m, err := metadata.NewHandler(conn)
			So(err, ShouldBeNil)

			// notice we were using net.IP in the settings itself before
			type oldFormat struct {
				LocalIP     net.IP `json:"postfix_public_ip"`
				AppLanguage string `json:"app_language"`
				PublicURL   string `json:"public_url"`
			}

			Convey("read from old format", func() {
				valueInOldFormat := oldFormat{
					LocalIP:     net.ParseIP(`22.33.44.55`),
					AppLanguage: "en",
					PublicURL:   "http://example.com",
				}

				err = m.Writer.StoreJson(context.Background(), SettingsKey, valueInOldFormat)
				So(err, ShouldBeNil)

				// we then retrieve the settings in the new format
				s, err := GetSettings(context.Background(), m.Reader)
				So(err, ShouldBeNil)

				So(s, ShouldResemble, &Settings{
					LocalIP:     IP{net.ParseIP(`22.33.44.55`)},
					AppLanguage: "en",
					PublicURL:   "http://example.com",
				})
			})

			Convey("save to old format", func() {
				value := Settings{
					LocalIP:     IP{net.ParseIP(`22.33.44.55`)},
					AppLanguage: "en",
					PublicURL:   "http://example.com",
				}

				err = m.Writer.StoreJson(context.Background(), SettingsKey, value)
				So(err, ShouldBeNil)

				var s oldFormat
				err = m.Reader.RetrieveJson(context.Background(), SettingsKey, &s)
				So(err, ShouldBeNil)

				So(s, ShouldResemble, oldFormat{
					LocalIP:     net.ParseIP(`22.33.44.55`),
					AppLanguage: "en",
					PublicURL:   "http://example.com",
				})
			})
		})
	})
}
