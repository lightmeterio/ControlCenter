package notification

import (
	"context"
	"errors"
	"fmt"
	"github.com/slack-go/slack"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/notification/bus"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type Content interface {
	fmt.Stringer
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

func New(settingsReader *meta.Reader) Center {
	cp := &center{
		bus:            bus.New(),
		settingsReader: settingsReader,
	}

	if err := cp.init(); err != nil {
		panic(err)
	}

	return cp
}

type center struct {
	bus            bus.Interface
	settingsReader *meta.Reader
	slackapi       Messenger
}

func (cp *center) init() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)

	defer cancel()

	slackSettings, err := settings.GetSlackNotificationsSettings(ctx, cp.settingsReader)
	if err != nil {
		if errors.Is(err, meta.ErrNoSuchKey) {
			return nil
		}

		return err
	}

	if !slackSettings.Enabled {
		return nil
	}

	cp.slackapi = newSlack(slackSettings.BearerToken, slackSettings.Channel)

	cp.bus.AddEventListener("slack", func(notification Notification) error {
		return cp.slackapi.PostMessage(notification.Content)
	})

	return nil
}

func (cp *center) AddSlackNotifier(slackSettings settings.SlackNotificationsSettings) error {
	cp.slackapi = newSlack(slackSettings.BearerToken, slackSettings.Channel)

	cp.bus.UpdateEventListener("slack", func(notification Notification) error {

		if !slackSettings.Enabled {
			return nil
		}

		return cp.slackapi.PostMessage(notification.Content)
	})

	return nil
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

type Messenger interface {
	PostMessage(stringer fmt.Stringer) error
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

func (s *slackapi) PostMessage(message fmt.Stringer) error {
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
