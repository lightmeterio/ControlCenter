// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package slack

import (
	"context"
	"github.com/slack-go/slack"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"reflect"
	"sync"
)

// TODO: make the notifications asynchronous!
// Add context to PostMessage and to slack api call!

const SettingKey = "messenger_slack"

type MessagePoster interface {
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
}

func messagePosterBuilder(client *slack.Client) MessagePoster {
	return client
}

type Notifier struct {
	// this mutex protects the access to the settings and the slack api client
	clientMutex     sync.Mutex
	client          *slack.Client
	currentSettings *Settings

	fetchSettings func() (*Settings, error)
	policies      core.Policies

	MessagePosterBuilder func(client *slack.Client) MessagePoster
}

func New(policies core.Policies, reader *meta.Reader) *Notifier {
	return NewWithCustomSettingsFetcher(policies, func() (*Settings, error) {
		s := Settings{}

		if err := reader.RetrieveJson(context.Background(), SettingKey, &s); err != nil {
			return nil, errorutil.Wrap(err)
		}

		return &s, nil
	})
}

func NewWithCustomSettingsFetcher(policies core.Policies, settingsFetcher func() (*Settings, error)) *Notifier {
	return &Notifier{
		fetchSettings:        settingsFetcher,
		policies:             policies,
		MessagePosterBuilder: messagePosterBuilder,
	}
}

func clientAndSettingsForMessenger(m *Notifier) (*slack.Client, *Settings, error) {
	updatedSettings, err := m.fetchSettings()

	if err != nil {
		return nil, nil, errorutil.Wrap(err)
	}

	if updatedSettings == nil {
		panic("slack setting cannot be nil and this is a bug in your code!")
	}

	m.clientMutex.Lock()

	defer m.clientMutex.Unlock()

	// update/create client if needed
	if m.currentSettings == nil || !reflect.DeepEqual(*updatedSettings, *m.currentSettings) {
		m.client = slack.New(updatedSettings.BearerToken)
		m.currentSettings = updatedSettings
	}

	return m.client, updatedSettings, nil
}

func (m *Notifier) SendTestNotification() error {
	if err := tryToNotifyMessage(m, core.Message("Lightmeter ControlCenter successfully connected to Slack!")); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func tryToNotifyMessage(m *Notifier, message core.Message) error {
	client, settings, err := clientAndSettingsForMessenger(m)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if !settings.Enabled {
		return nil
	}

	poster := m.MessagePosterBuilder(client)

	_, _, err = poster.PostMessage(settings.Channel, slack.MsgOptionText(message.String(), false), slack.MsgOptionAsUser(true))
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

type Settings struct {
	BearerToken string `json:"bearer_token"`
	Kind        string `json:"-"`
	Channel     string `json:"channel"`
	Enabled     bool   `json:"enabled"`
	Language    string `json:"language"`
}

func (m *Notifier) DeriveNotifierWithCustomSettingsFetcher(policies core.Policies, settingsFetcher func() (*Settings, error)) *Notifier {
	return &Notifier{
		fetchSettings:        settingsFetcher,
		policies:             policies,
		MessagePosterBuilder: m.MessagePosterBuilder,
	}
}

// implement Notifier
func (m *Notifier) Notify(n core.Notification, translator translator.Translator) error {
	pass, err := m.policies.Pass(n)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if !pass {
		return nil
	}

	message, err := core.TranslateNotification(n, translator)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if err := tryToNotifyMessage(m, message); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func SetSettings(ctx context.Context, writer *meta.AsyncWriter, settings Settings) error {
	result := writer.StoreJson(SettingKey,
		Settings{
			BearerToken: settings.BearerToken,
			Channel:     settings.Channel,
			Enabled:     settings.Enabled,
			Language:    settings.Language,
		})

	select {
	case err := <-result.Done():
		if err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	case <-ctx.Done():
		return errorutil.Wrap(ctx.Err())
	}
}

func GetSettings(ctx context.Context, reader *meta.Reader) (*Settings, error) {
	slackSettings := &Settings{}

	err := reader.RetrieveJson(ctx, SettingKey, slackSettings)
	if err != nil {
		return nil, errorutil.Wrap(err, "could get slack settings")
	}

	return slackSettings, nil
}
