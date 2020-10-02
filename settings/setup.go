package settings

import (
	"context"
	"errors"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/newsletter"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"log"
)

type SystemSetup interface {
	SetOptions(context.Context, interface{}) error
}

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
}

type InitialOptions struct {
	SubscribeToNewsletter bool
	MailKind              SetupMailKind
	Email                 string
}

type MasterConf struct {
	meta                 *meta.MetadataHandler
	newsletterSubscriber newsletter.Subscriber
}

func NewMasterConf(meta *meta.MetadataHandler, newsletterSubscriber newsletter.Subscriber) (*MasterConf, error) {
	return &MasterConf{meta, newsletterSubscriber}, nil
}

func validMailKind(k SetupMailKind) bool {
	return k == MailKindDirect ||
		k == MailKindMarketing ||
		k == MailKindTransactional
}

func (c *MasterConf) SetOptions(context context.Context, o interface{}) error {
	switch v := o.(type) {
	case InitialOptions:
		return c.setInitialOptions(context, v)
	case SlackNotificationsSettings:
		return c.SetSlackNotificationsSettings(context, v)
	}

	return errorutil.Wrap(errors.New("options is not supported"))
}

func (c *MasterConf) setInitialOptions(context context.Context, initialOptions InitialOptions) error {
	if !validMailKind(initialOptions.MailKind) {
		return ErrInvalidMailKindOption
	}

	if initialOptions.SubscribeToNewsletter {
		if err := c.newsletterSubscriber.Subscribe(context, initialOptions.Email); err != nil {
			log.Println("Failed to subscribe with error:", err)
			return errorutil.Wrap(ErrFailedToSubscribeToNewsletter)
		}
	}

	_, err := c.meta.Store(context, []meta.Item{
		{Key: "mail_kind", Value: initialOptions.MailKind},
		{Key: "subscribe_newsletter", Value: initialOptions.SubscribeToNewsletter},
	})

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (c *MasterConf) SetSlackNotificationsSettings(ctx context.Context, slackNotificationsSettings SlackNotificationsSettings) error {
	_, err := c.meta.StoreJson(ctx, fmt.Sprintf("messenger_"+slackNotificationsSettings.Kind),
		SlackNotificationsSettings{
			BearerToken: slackNotificationsSettings.BearerToken,
			Channel:     slackNotificationsSettings.Channel,
		})

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (c *MasterConf) GetSlackNotificationsSettings(ctx context.Context) (*SlackNotificationsSettings, error) {
	slackSettings := &SlackNotificationsSettings{}

	err := c.meta.RetrieveJson(ctx, "messenger_slack", slackSettings)
	if err != nil {
		return nil, errorutil.Wrap(err, "could get slack settings")
	}

	return slackSettings, nil
}

func (c *MasterConf) Close() error {
	return c.meta.Close()
}
