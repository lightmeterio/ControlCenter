// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package email

import (
	"context"
	"io/ioutil"
	"net"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/util/stringutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"gitlab.com/lightmeter/controlcenter/version"
	"golang.org/x/text/language"
	"golang.org/x/text/message/catalog"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
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
	category string
	priority string
}

func (c fakeContent) Title() core.ContentComponent {
	return fakeContentComponent("some fake content")
}

func (c fakeContent) Description() core.ContentComponent {
	return fakeContentComponent("some fake description")
}

func (c fakeContent) Metadata() core.ContentMetadata {
	return core.ContentMetadata{
		"category": fakeContentComponent(c.category),
		"priority": fakeContentComponent(c.priority),
	}
}

func init() {
	// fake application version
	version.Version = "1.0.0"
}

func TestSendEmail(t *testing.T) {
	Convey("Send Email", t, func() {
		clock := timeutil.FakeClock{Time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)}

		translators := translator.New(catalog.NewBuilder())
		translator := translators.Translator(language.MustParse(`en`))

		backend := &FakeMailBackend{
			ExpectedUser:     "username@example.com",
			ExpectedPassword: "super_password",
		}

		dtor := StartFakeServer(backend, ":1026")

		defer func() {
			So(dtor(), ShouldBeNil)
		}()

		globalSettings := globalsettings.Settings{
			LocalIP:     globalsettings.IP{net.ParseIP(`127.0.0.1`)},
			PublicURL:   "https://example.com/lightmeter/",
			AppLanguage: "en",
		}

		Convey("Fails", func() {
			settings := Settings{
				Sender:       "sender@example.com",
				Recipients:   "recipient@example2.com, Someone else <recipient2@some.other.address.com>, a.third.one@lala.com",
				ServerName:   "localhost",
				ServerPort:   1026,
				Username:     stringutil.MakeSensitive("username@example.com"),
				Password:     stringutil.MakeSensitive("wrong_password"),
				SecurityType: SecurityTypeNone,
				AuthMethod:   AuthMethodPassword,
			}

			Convey("Fail to Validate", func() {
				So(ValidateSettings(settings), ShouldNotBeNil)
			})

			Convey("Fail to Send", func() {
				notifier := newWithCustomSettingsFetcherAndClock(core.PassPolicy, func() (*Settings, *globalsettings.Settings, error) {
					return &settings, &globalSettings, nil
				}, &clock)

				err := notifier.Notify(core.Notification{
					ID:      1,
					Content: fakeContent{},
				}, translator)

				So(err, ShouldNotBeNil)
				So(len(backend.Messages), ShouldEqual, 0)
			})
		})

		Convey("Succeeds", func() {
			settings := Settings{
				Sender:       "sender@example.com",
				Recipients:   "recipient@example2.com, Someone else <recipient2@some.other.address.com>, a.third.one@lala.com",
				ServerName:   "localhost",
				ServerPort:   1026,
				Username:     stringutil.MakeSensitive("username@example.com"),
				Password:     stringutil.MakeSensitive("super_password"),
				SecurityType: SecurityTypeNone,
				AuthMethod:   AuthMethodPassword,
			}

			Convey("Succeeds to Validate", func() {
				So(ValidateSettings(settings), ShouldBeNil)
			})

			Convey("Succeeds to Send", func() {
				notifier := newWithCustomSettingsFetcherAndClock(core.PassPolicy, func() (*Settings, *globalsettings.Settings, error) {
					return &settings, &globalSettings, nil
				}, &clock)

				err := notifier.Notify(core.Notification{
					ID:      42,
					Content: fakeContent{category: "intel", priority: "bad"},
				}, translator)

				So(err, ShouldBeNil)
				So(len(backend.Messages), ShouldEqual, 1)

				msg := backend.Messages[0]

				date, err := msg.Header.Date()
				So(err, ShouldBeNil)
				So(date.In(time.UTC), ShouldResemble, testutil.MustParseTime(`2000-01-01 00:00:00 +0000`))

				So(msg.Header.Get("From"), ShouldEqual, "sender@example.com")
				So(msg.Header.Get("To"), ShouldEqual, "recipient@example2.com, Someone else <recipient2@some.other.address.com>, a.third.one@lala.com")

				content, err := ioutil.ReadAll(msg.Body)
				So(err, ShouldBeNil)

				expectedContent := `
Title: some fake content
Description: some fake description
Category: intel
Priority: bad
PriorityColor: rgb(255, 92, 111)
DetailsURL: https://example.com/lightmeter/#/insight-card/42
PreferencesURL: https://example.com/lightmeter/#/settings
PublicURL: https://example.com/lightmeter/
Version: 1.0.0

`

				So(strings.ReplaceAll(string(content), "\r\n", "\n"), ShouldEqual, expectedContent)
			})
		})
	})
}

func TestSettings(t *testing.T) {
	Convey("Test Settings", t, func() {
		conn, closeConn := testutil.TempDBConnectionMigrated(t, "master")
		defer closeConn()

		m, err := metadata.NewDefaultedHandler(conn, metadata.DefaultValues{
			"messenger_email": map[string]interface{}{
				"sender":        "sender@email.com",
				"serverName":    "mail.example.com",
				"serverPort":    "587",
				"securityType":  "STARTTLS",
				"authMethod":    "password",
				"username":      "user@example.com",
				"password":      "super_passwd",
				"skipCertCheck": false,
			},
		})

		So(err, ShouldBeNil)

		err = m.Writer.StoreJson(context.Background(), SettingKey, Settings{
			Enabled:    true,
			Recipients: "recipient@example.com",
		})

		So(err, ShouldBeNil)

		s, err := GetSettings(context.Background(), m.Reader)
		So(err, ShouldBeNil)

		So(s, ShouldResemble, &Settings{
			Enabled:       true,
			SkipCertCheck: false,
			Recipients:    "recipient@example.com",
			Sender:        "sender@email.com",
			ServerName:    "mail.example.com",
			ServerPort:    587,
			SecurityType:  SecurityTypeSTARTTLS,
			AuthMethod:    AuthMethodPassword,
			Username:      stringutil.MakeSensitive("user@example.com"),
			Password:      stringutil.MakeSensitive("super_passwd"),
		})
	})
}
