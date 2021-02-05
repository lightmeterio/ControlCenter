// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package notification

import (
	"context"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/po"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"golang.org/x/text/language"
	"golang.org/x/text/message/catalog"
	"sync/atomic"
	"testing"
	"time"
)

var (
	dummyContext = context.Background()
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

type TimeInterval struct {
	From time.Time
	To   time.Time
}

type fakeContent struct {
	Interval TimeInterval
}

func (c fakeContent) String() string {
	return fmt.Sprintf("No emails were sent between %v and %v", c.Args()...)
}

func (c fakeContent) TplString() string {
	return "No emails were sent between %v and %v"
}

func (c fakeContent) Args() []interface{} {
	return []interface{}{c.Interval.From, c.Interval.To}
}

func TestSendNotification(t *testing.T) {

	Convey("Notification", t, func() {
		conn, closeConn := testutil.TempDBConnection(t)
		defer closeConn()

		m, err := meta.NewHandler(conn, "master")
		So(err, ShouldBeNil)

		runner := meta.NewRunner(m)
		done, cancel := runner.Run()
		defer func() { cancel(); done() }()
		writer := runner.Writer()

		defer func() { errorutil.MustSucceed(m.Close()) }()

		content := new(fakeContent)
		content.Interval.To = time.Now()
		content.Interval.From = time.Now()

		Convey("Success", func() {
			Convey("Do subscribe (german)", func() {

				slackSettings := settings.SlackNotificationsSettings{
					Channel:     "general",
					Kind:        "slack",
					BearerToken: "xoxb-1388191062644-1385067635637-iXfDIfcPO3HKHEjLZY2seVX6",
					Enabled:     true,
					Language:    "de",
				}

				err = settings.SetSlackNotificationsSettings(dummyContext, writer, slackSettings)
				So(err, ShouldBeNil)

				DefaultCatalog := catalog.NewBuilder()
				lang := language.MustParse("de")
				DefaultCatalog.SetString(lang, content.TplString(), `Zwischen %v und %v wurden keine E-Mails gesendet`)

				translators := translator.New(DefaultCatalog)
				center := New(m.Reader, translators)
				So(err, ShouldBeNil)

				notification := Notification{
					ID:      0,
					Content: content,
				}
				err := center.Notify(notification)
				So(err, ShouldBeNil)
			})

			Convey("Do subscribe (english)", func() {

				slackSettings := settings.SlackNotificationsSettings{
					Channel:     "general",
					Kind:        "slack",
					BearerToken: "xoxb-1388191062644-1385067635637-iXfDIfcPO3HKHEjLZY2seVX6",
					Enabled:     true,
					Language:    "en",
				}

				err = settings.SetSlackNotificationsSettings(dummyContext, writer, slackSettings)
				So(err, ShouldBeNil)

				DefaultCatalog := catalog.NewBuilder()
				lang := language.MustParse("en")
				DefaultCatalog.SetString(lang, content.TplString(), content.TplString())

				translators := translator.New(DefaultCatalog)
				center := New(m.Reader, translators)
				So(err, ShouldBeNil)

				notification := Notification{
					ID:      0,
					Content: content,
				}

				err := center.Notify(notification)
				So(err, ShouldBeNil)
			})

			Convey("Do subscribe (pt_BR)", func() {

				slackSettings := settings.SlackNotificationsSettings{
					Channel:     "general",
					Kind:        "slack",
					BearerToken: "xoxb-1388191062644-1385067635637-iXfDIfcPO3HKHEjLZY2seVX6",
					Enabled:     true,
					Language:    "pt_BR",
				}

				err = settings.SetSlackNotificationsSettings(dummyContext, writer, slackSettings)
				So(err, ShouldBeNil)

				DefaultCatalog := catalog.NewBuilder()
				lang := language.MustParse("pt_BR")
				DefaultCatalog.SetString(lang, content.TplString(), content.TplString())

				translators := translator.New(DefaultCatalog)
				center := New(m.Reader, translators)
				So(err, ShouldBeNil)

				notification := Notification{
					ID:      0,
					Content: content,
				}

				err := center.Notify(notification)
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestSendNotificationMissingConf(t *testing.T) {

	Convey("Notification", t, func() {
		conn, closeConn := testutil.TempDBConnection(t)
		defer closeConn()

		m, err := meta.NewHandler(conn, "master")
		So(err, ShouldBeNil)

		defer func() { errorutil.MustSucceed(m.Close()) }()

		translators := translator.New(po.DefaultCatalog)
		center := New(m.Reader, translators)

		So(err, ShouldBeNil)

		content := new(fakeContent)
		notification := Notification{
			ID:      0,
			Content: content,
		}

		Convey("Success", func() {
			Convey("Do subscribe", func() {
				err := center.Notify(notification)
				So(err, ShouldBeNil)
			})
		})
	})
}

type fakeapi struct {
	t *testing.T
	Counter int32
}

func (s *fakeapi) PostMessage(stringer Message) error {
	s.t.Log(stringer)
	atomic.AddInt32(&s.Counter, 1)
	return nil
}

func TestFakeSendNotification(t *testing.T) {

	Convey("Notification", t, func() {
		conn, closeConn := testutil.TempDBConnection(t)
		defer closeConn()

		m, err := meta.NewHandler(conn, "master")
		So(err, ShouldBeNil)

		runner := meta.NewRunner(m)
		done, cancel := runner.Run()
		defer func() { cancel(); done() }()
		writer := runner.Writer()

		defer func() { errorutil.MustSucceed(m.Close()) }()

		slackSettings := settings.SlackNotificationsSettings{
			Channel:     "general",
			Kind:        "slack",
			BearerToken: "xoxb-1388191062644-1385067635637-iXfDIfcPO3HKHEjLZY2seVX6",
			Enabled:     true,
			Language:    "de",
		}

		err = settings.SetSlackNotificationsSettings(dummyContext, writer, slackSettings)
		So(err, ShouldBeNil)

		fakeapi := &fakeapi{t: t}

		DefaultCatalog := catalog.NewBuilder()
		lang := language.MustParse("de")
		DefaultCatalog.SetString(lang, `%v bounce rate between %v and %v`, `%v bounce rate ist zwischen %v und %v`)
		translators := translator.New(DefaultCatalog)

		centerInterface := New(m.Reader, translators)
		c := centerInterface.(*center)
		c.slackapi = fakeapi

		content := new(fakeContent)
		Notification := Notification{
			ID:      0,
			Content: content,
		}

		Convey("Success", func() {
			Convey("Do subscribe", func() {
				err := c.Notify(Notification)
				So(err, ShouldBeNil)
				So(fakeapi.Counter, ShouldEqual, 1)
			})
		})
	})
}

func TestFakeSendNotificationDisabled(t *testing.T) {

	Convey("Notification", t, func() {
		conn, closeConn := testutil.TempDBConnection(t)
		defer closeConn()

		m, err := meta.NewHandler(conn, "master")
		So(err, ShouldBeNil)

		runner := meta.NewRunner(m)
		done, cancel := runner.Run()
		defer func() { cancel(); done() }()
		writer := runner.Writer()

		defer func() { errorutil.MustSucceed(m.Close()) }()

		slackSettings := settings.SlackNotificationsSettings{
			Channel:     "general",
			Kind:        "slack",
			BearerToken: "xoxb-1388191062644-1385067635637-iXfDIfcPO3HKHEjLZY2seVX6",
			Enabled:     false,
			Language:    "en",
		}

		err = settings.SetSlackNotificationsSettings(dummyContext, writer, slackSettings)
		So(err, ShouldBeNil)

		fakeapi := &fakeapi{}
		translators := translator.New(po.DefaultCatalog)
		centerInterface := New(m.Reader, translators)

		c := centerInterface.(*center)
		c.slackapi = fakeapi

		content := new(fakeContent)
		Notification := Notification{
			ID:      0,
			Content: content,
		}

		Convey("Success", func() {
			Convey("Do subscribe", func() {
				err := c.Notify(Notification)
				So(err, ShouldBeNil)
				So(fakeapi.Counter, ShouldEqual, 0)
			})
		})
	})
}
