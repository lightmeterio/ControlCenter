// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package notification

import (
	"context"
	"errors"
	"fmt"
	"github.com/slack-go/slack"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/notification/bus"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"golang.org/x/text/language"

	"time"
)

type Content interface {
	fmt.Stringer
	translator.TranslatableStringer
}

type Notification struct {
	ID      int64
	Content Content
	Rating  int64
}

type Center interface {
	Notify(Notification) error
	AddSlackNotifier(notificationsSettings settings.SlackNotificationsSettings) error
}

func New(settingsReader *meta.Reader, translators translator.Translators) Center {
	cp := &center{
		bus:            bus.New(),
		settingsReader: settingsReader,
		translators:    translators,
	}

	if err := cp.init(); err != nil {
		errorutil.LogErrorf(err, "init notifications")
	}

	return cp
}

type center struct {
	bus            bus.Interface
	settingsReader *meta.Reader
	slackapi       Messenger
	translators    translator.Translators
}

func (cp *center) init() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)

	defer cancel()

	slackSettings, err := settings.GetSlackNotificationsSettings(ctx, cp.settingsReader)
	if err != nil {
		if errors.Is(err, meta.ErrNoSuchKey) {
			return nil
		}

		return errorutil.Wrap(err)
	}

	if !slackSettings.Enabled {
		return nil
	}

	languageTag, err := language.Parse(slackSettings.Language)
	if err != nil {
		return errorutil.Wrap(err)
	}

	cp.slackapi = newSlack(slackSettings.BearerToken, slackSettings.Channel)
	translator := cp.translators.Translator(languageTag, time.Time{})

	err = cp.slackapi.PostMessage(newConnectContent())
	if err != nil {
		return errorutil.Wrap(err)
	}

	cp.bus.AddEventListener("slack", func(notification Notification) error {
		translatedMessage, args, err := cp.Translate(slackSettings.Language, translator, notification)
		if err != nil {
			return errorutil.Wrap(err)
		}
		return cp.slackapi.PostMessage(Messagef(translatedMessage, args...))
	})

	return nil
}

func newConnectContent() Message {
	return "Lightmeter ControlCenter successfully connected to Slack!"
}

func (cp *center) AddSlackNotifier(slackSettings settings.SlackNotificationsSettings) error {
	cp.slackapi = newSlack(slackSettings.BearerToken, slackSettings.Channel)

	if slackSettings.Enabled {
		err := cp.slackapi.PostMessage(newConnectContent())
		if err != nil {
			return errorutil.Wrap(err)
		}
	}

	languageTag := language.MustParse(slackSettings.Language)

	cp.slackapi = newSlack(slackSettings.BearerToken, slackSettings.Channel)
	translator := cp.translators.Translator(languageTag, time.Time{})

	cp.bus.UpdateEventListener("slack", func(notification Notification) error {
		if !slackSettings.Enabled {
			return nil
		}

		translatedMessage, args, err := cp.Translate(slackSettings.Language, translator, notification)
		if err != nil {
			return errorutil.Wrap(err)
		}

		return cp.slackapi.PostMessage(Messagef(translatedMessage, args...))
	})

	return nil
}

func (cp *center) Translate(language string, t translator.Translator, notification Notification) (string, []interface{}, error) {
	transformed := translator.TransformTranslation(notification.Content.TplString())

	translatedMessage, err := t.Translate(transformed)
	if err != nil {
		return "", nil, errorutil.Wrap(err)
	}

	args := notification.Content.Args()
	for i, arg := range args {
		t, ok := arg.(time.Time)
		if ok {
			args[i] = timeutil.PrettyFormatTime(t, language)
		}
	}

	return translatedMessage, args, nil
}

func (cp *center) Notify(notification Notification) error {
	err := cp.bus.Publish(notification)
	if err != nil {
		if errors.Is(err, bus.ErrNoListeners) {
			return nil
		}

		return errorutil.Wrap(err)
	}

	return nil
}

func Messagef(format string, a ...interface{}) Message {
	return Message(fmt.Sprintf(format, a...))
}

type Message string

func (s *Message) String() string {
	return string(*s)
}

type Messenger interface {
	PostMessage(stringer Message) error
}

func newSlack(token string, channel string) Messenger {
	client := slack.New(token)

	return &slackapi{
		client:  client,
		channel: channel,
	}
}

type slackapi struct {
	client  *slack.Client
	channel string
}

func (s *slackapi) PostMessage(message Message) error {
	_, _, err := s.client.PostMessage(
		s.channel,
		slack.MsgOptionText(message.String(), false),
		slack.MsgOptionAsUser(true),
	)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
