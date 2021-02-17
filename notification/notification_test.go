// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package notification

import (
	"context"
	"errors"
	"fmt"
	slackAPI "github.com/slack-go/slack"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/notification/slack"
	"gitlab.com/lightmeter/controlcenter/po"
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

type dummyPolicy struct {
}

func (dummyPolicy) Pass(core.Notification) (bool, error) {
	return true, nil
}

type fakeSlackPoster struct {
	err error
}

var fakeSlackError = errors.New(`Some Slack Error`)

func (p *fakeSlackPoster) PostMessage(channelID string, options ...slackAPI.MsgOption) (string, string, error) {
	return "", "", p.err
}

func centerWithTranslatorsAndDummyPolicy(t *testing.T, translators translator.Translators, slackSettings *slack.Settings) *Center {
	notifiers := func() []core.Notifier {
		if slackSettings == nil {
			return []core.Notifier{}
		}

		slackNotifier := slack.NewWithCustomSettingsFetcher(core.Policies{&dummyPolicy{}}, func() (*slack.Settings, error) {
			return slackSettings, nil
		})

		// don't use slack api, mocking the PostMessage call
		slackNotifier.MessagePosterBuilder = func(client *slackAPI.Client) slack.MessagePoster {
			return &fakeSlackPoster{}
		}

		return []core.Notifier{slackNotifier}
	}()

	center := NewWithCustomLanguageFetcher(translators, func() (language.Tag, error) {
		if slackSettings != nil {
			return language.Parse(slackSettings.Language)
		}

		return language.English, nil
	}, notifiers)

	return center
}

func buildSlackSettings(lang string, enabled bool) slack.Settings {
	return slack.Settings{
		Channel:     "general",
		Kind:        "slack",
		BearerToken: "some_slack_key",
		Enabled:     enabled,
		Language:    lang,
	}
}

func TestSendNotification(t *testing.T) {
	Convey("Notification", t, func() {
		content := new(fakeContent)
		content.Interval.To = time.Now()
		content.Interval.From = time.Now()

		Convey("Success", func() {
			Convey("Do subscribe (german)", func() {
				cat := catalog.NewBuilder()
				lang := language.MustParse("de")
				cat.SetString(lang, content.TplString(), `Zwischen %v und %v wurden keine E-Mails gesendet`)

				translators := translator.New(cat)
				s := buildSlackSettings("de", true)

				center := centerWithTranslatorsAndDummyPolicy(t, translators, &s)

				notification := core.Notification{
					ID:      0,
					Content: content,
				}

				err := center.Notify(notification)
				So(err, ShouldBeNil)
			})

			Convey("Do subscribe (english)", func() {
				cat := catalog.NewBuilder()
				lang := language.MustParse("en")
				cat.SetString(lang, content.TplString(), content.TplString())

				translators := translator.New(cat)
				s := buildSlackSettings("en", true)
				center := centerWithTranslatorsAndDummyPolicy(t, translators, &s)

				notification := core.Notification{
					ID:      0,
					Content: content,
				}

				err := center.Notify(notification)
				So(err, ShouldBeNil)
			})

			Convey("Do subscribe (pt_BR)", func() {
				cat := catalog.NewBuilder()
				lang := language.MustParse("pt_BR")
				cat.SetString(lang, content.TplString(), content.TplString())

				translators := translator.New(cat)
				s := buildSlackSettings("pt_BR", true)
				center := centerWithTranslatorsAndDummyPolicy(t, translators, &s)

				notification := core.Notification{
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
		translators := translator.New(po.DefaultCatalog)
		center := centerWithTranslatorsAndDummyPolicy(t, translators, nil)

		content := new(fakeContent)
		notification := core.Notification{
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
	t       *testing.T
	Counter int32
}

func (s *fakeapi) Notify(n core.Notification, _ translator.Translator) error {
	s.t.Log(n)
	atomic.AddInt32(&s.Counter, 1)
	return nil
}

func TestFakeSendNotification(t *testing.T) {
	Convey("Notification", t, func() {
		fakeapi := &fakeapi{t: t}

		cat := catalog.NewBuilder()
		lang := language.MustParse("de")
		cat.SetString(lang, `%v bounce rate between %v and %v`, `%v bounce rate ist zwischen %v und %v`)
		translators := translator.New(cat)

		center := NewWithCustomLanguageFetcher(translators, func() (language.Tag, error) { return language.German, nil }, []core.Notifier{fakeapi})

		content := new(fakeContent)

		notification := core.Notification{
			ID:      0,
			Content: content,
		}

		Convey("Success", func() {
			Convey("Do subscribe", func() {
				err := center.Notify(notification)
				So(err, ShouldBeNil)
				So(fakeapi.Counter, ShouldEqual, 1)
			})
		})
	})
}
