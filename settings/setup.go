package settings

import (
	"context"
	"errors"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/newsletter"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"log"
)

type SetupMailKind string

const (
	MailKindDirect        SetupMailKind = "direct"
	MailKindTransactional SetupMailKind = "transactional"
	MailKindMarketing     SetupMailKind = "marketing"
)

var (
	ErrInvalidInintialSetupOption    = errors.New(`Invalid Initial Setup Option`)
	ErrFailedToSubscribeToNewsletter = errors.New(`Error Subscribing To Newsletter`)
	ErrInvalidMailKindOption         = errors.New(`Invalid Mail Kind`)
)

type SlackNotificationsSettings struct {
	BearerToken string `json:"bearer_token"`
	Kind        string `json:"-"`
	Channel     string `json:"channel"`
	Enabled     bool   `json:"enabled"`
}

type InitialOptions struct {
	SubscribeToNewsletter bool
	MailKind              SetupMailKind
	Email                 string
}

type InitialSetupSettings struct {
	newsletterSubscriber newsletter.Subscriber
}

func NewInitialSetupSettings(newsletterSubscriber newsletter.Subscriber) *InitialSetupSettings {
	return &InitialSetupSettings{newsletterSubscriber}
}

func validMailKind(k SetupMailKind) bool {
	return k == MailKindDirect ||
		k == MailKindMarketing ||
		k == MailKindTransactional
}

// TODO: check if context time-outs
func (c *InitialSetupSettings) Set(context context.Context, writer *meta.AsyncWriter, initialOptions InitialOptions) error {
	if !validMailKind(initialOptions.MailKind) {
		return ErrInvalidMailKindOption
	}

	if initialOptions.SubscribeToNewsletter {
		if err := c.newsletterSubscriber.Subscribe(context, initialOptions.Email); err != nil {
			log.Println("Failed to subscribe with error:", err)
			return errorutil.Wrap(ErrFailedToSubscribeToNewsletter)
		}
	}

	err := writer.Store([]meta.Item{
		{Key: "mail_kind", Value: initialOptions.MailKind},
		{Key: "subscribe_newsletter", Value: initialOptions.SubscribeToNewsletter},
	}).Wait()

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

// TODO: investigate if this method should be moved to httpsettings
// TODO: check if context time-outs
func SetSlackNotificationsSettings(ctx context.Context, writer *meta.AsyncWriter, slackNotificationsSettings SlackNotificationsSettings) error {
	err := writer.StoreJson("messenger_slack",
		SlackNotificationsSettings{
			BearerToken: slackNotificationsSettings.BearerToken,
			Channel:     slackNotificationsSettings.Channel,
			Enabled:     slackNotificationsSettings.Enabled,
		}).Wait()

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

// TODO: investigate if this method should be moved to httpsettings
func GetSlackNotificationsSettings(ctx context.Context, reader *meta.Reader) (*SlackNotificationsSettings, error) {
	slackSettings := &SlackNotificationsSettings{}

	err := reader.RetrieveJson(ctx, "messenger_slack", slackSettings)
	if err != nil {
		return nil, errorutil.Wrap(err, "could get slack settings")
	}

	return slackSettings, nil
}
