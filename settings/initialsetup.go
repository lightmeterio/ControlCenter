package settings

import (
	"errors"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/newsletter"
	"gitlab.com/lightmeter/controlcenter/util"
	"log"
)

type SystemSetup interface {
	SetInitialOptions(InitialSetupOptions) error
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
)

type InitialSetupOptions struct {
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

func (c *MasterConf) SetInitialOptions(o InitialSetupOptions) error {
	if !validMailKind(o.MailKind) {
		return ErrInvalidInintialSetupOption
	}

	if o.SubscribeToNewsletter {
		if err := c.newsletterSubscriber.Subscribe(o.Email); err != nil {
			log.Println("Failed to subscribe with error:", err)
			return util.WrapError(ErrFailedToSubscribeToNewsletter)
		}
	}

	_, err := c.meta.Store([]meta.Item{
		{Key: "mail_kind", Value: o.MailKind},
		{Key: "subscribe_newsletter", Value: o.SubscribeToNewsletter},
	})

	if err != nil {
		return util.WrapError(err)
	}

	return nil
}

func (c *MasterConf) Close() error {
	return nil
}
