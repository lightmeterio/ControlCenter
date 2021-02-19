// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package email

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"golang.org/x/text/language"
	"golang.org/x/text/message/catalog"
	"io/ioutil"
	"testing"
	"time"
)

type fakeContent struct {
}

func (c fakeContent) String() string {
	return "some fake content!"
}

func (c fakeContent) TplString() string {
	return c.String()
}

func (c fakeContent) Args() []interface{} {
	return nil
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
				notifier := newWithCustomSettingsFetcherAndClock(core.PassPolicy, func() (*Settings, error) {
					return &settings, nil
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
				notifier := newWithCustomSettingsFetcherAndClock(core.PassPolicy, func() (*Settings, error) {
					return &settings, nil
				}, &clock)

				err := notifier.Notify(core.Notification{
					ID:      1,
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
				So(len(content), ShouldBeGreaterThan, 0)
			})
		})
	})
}
