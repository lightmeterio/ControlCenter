// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package slack

import (
	"context"
	"errors"
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

type SettingsFetcher func() (*Settings, error)

type Notifier struct {
	// this mutex protects the access to the settings and the slack api client
	clientMutex     sync.Mutex
	client          *slack.Client
	currentSettings *Settings

	fetchSettings SettingsFetcher
	policy        core.Policy

	MessagePosterBuilder func(client *slack.Client) MessagePoster
}

type disabledFromSettingsPolicy struct {
	settingsFetcher SettingsFetcher
}

func (p *disabledFromSettingsPolicy) Reject(core.Notification) (bool, error) {
	s, err := p.settingsFetcher()
	if err != nil && errors.Is(err, meta.ErrNoSuchKey) {
		return true, nil
	}

	if err != nil {
		return true, errorutil.Wrap(err)
	}

	return !s.Enabled, nil
}

func New(policy core.Policy, reader *meta.Reader) *Notifier {
	fetchSettings := func() (*Settings, error) {
		s := Settings{}

		if err := reader.RetrieveJson(context.Background(), SettingKey, &s); err != nil {
			return nil, errorutil.Wrap(err)
		}

		return &s, nil
	}

	policies := core.Policies{policy, &disabledFromSettingsPolicy{settingsFetcher: fetchSettings}}

	return NewWithCustomSettingsFetcher(policies, fetchSettings)
}

func NewWithCustomSettingsFetcher(policy core.Policy, settingsFetcher func() (*Settings, error)) *Notifier {
	return &Notifier{
		fetchSettings:        settingsFetcher,
		policy:               policy,
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
	msg := core.Message{
		Title:       "",
		Description: "Lightmeter ControlCenter successfully connected to Slack!",
		Metadata:    map[string]string{},
	}

	if err := tryToNotifyMessage(m, msg); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func tryToNotifyMessage(m *Notifier, message core.Message) error {
	client, settings, err := clientAndSettingsForMessenger(m)

	if err != nil && errors.Is(err, meta.ErrNoSuchKey) {
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	if !settings.Enabled {
		return nil
	}

	poster := m.MessagePosterBuilder(client)

	_, _, err = poster.PostMessage(settings.Channel, slack.MsgOptionText(message.Description, false), slack.MsgOptionAsUser(true))
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

type Settings struct {
	BearerToken string `json:"bearer_token"`
	Channel     string `json:"channel"`
	Enabled     bool   `json:"enabled"`
}

func (m *Notifier) ValidateSettings(s core.Settings) error {
	settings, ok := s.(Settings)

	if !ok {
		return core.ErrInvalidSettings
	}

	d := m.deriveNotifierWithCustomSettingsFetcher(core.PassPolicy, func() (*Settings, error) {
		return &settings, nil
	})

	if err := d.SendTestNotification(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (m *Notifier) deriveNotifierWithCustomSettingsFetcher(policy core.Policy, settingsFetcher func() (*Settings, error)) *Notifier {
	return &Notifier{
		fetchSettings:        settingsFetcher,
		policy:               policy,
		MessagePosterBuilder: m.MessagePosterBuilder,
	}
}

// implement Notifier
func (m *Notifier) Notify(n core.Notification, translator translator.Translator) error {
	reject, err := m.policy.Reject(n)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if reject {
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
	if err := writer.StoreJsonSync(ctx, SettingKey, settings); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func GetSettings(ctx context.Context, reader *meta.Reader) (*Settings, error) {
	settings := &Settings{}

	err := reader.RetrieveJson(ctx, SettingKey, settings)
	if err != nil {
		return nil, errorutil.Wrap(err, "could not get slack settings")
	}

	return settings, nil
}
