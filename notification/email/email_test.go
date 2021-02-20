// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package email

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"gitlab.com/lightmeter/controlcenter/version"
	"golang.org/x/text/language"
	"golang.org/x/text/message/catalog"
	"io/ioutil"
	"net"
	"strings"
	"testing"
	"time"
)

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

func (c fakeContent) Title() core.ContentComponent {
	return fakeContentComponent("some fake content")
}

func (c fakeContent) Description() core.ContentComponent {
	return fakeContentComponent("some fake description")
}

func (c fakeContent) Metadata() core.ContentMetadata {
	return core.ContentMetadata{
		"category": fakeContentComponent("Intel"),
		"priority": fakeContentComponent("Low"),
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
			LocalIP:     net.ParseIP("127.0.0.1"),
			PublicURL:   "https://example.com/lightmeter/",
			APPLanguage: "en",
		}

		Convey("Fails", func() {
			settings := Settings{
				Sender:       "sender@example.com",
				Recipients:   "recipient@example2.com, Someone else <recipient2@some.other.address.com>, a.third.one@lala.com",
				ServerName:   "localhost",
				ServerPort:   1026,
				Username:     "username@example.com",
				Password:     "wrong_password",
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
				Username:     "username@example.com",
				Password:     "super_password",
				SecurityType: SecurityTypeNone,
				AuthMethod:   AuthMethodPassword,
			}

			Convey("Succeds to Validate", func() {
				So(ValidateSettings(settings), ShouldBeNil)
			})

			Convey("Succeds to Send", func() {
				notifier := newWithCustomSettingsFetcherAndClock(core.PassPolicy, func() (*Settings, *globalsettings.Settings, error) {
					return &settings, &globalSettings, nil
				}, &clock)

				err := notifier.Notify(core.Notification{
					ID:      42,
					Content: fakeContent{},
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
Category: Intel
Priority: Low
DetailsURL: https://example.com/lightmeter/#/insight-42
PreferencesURL: https://example.com/lightmeter/#/settings
Version: 1.0.0

`

				So(strings.ReplaceAll(string(content), "\r\n", "\n"), ShouldEqual, expectedContent)
			})
		})
	})
}
