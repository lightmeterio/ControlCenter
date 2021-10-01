// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package settings

import (
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/newsletter"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
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

func (c *InitialSetupSettings) Set(ctx context.Context, writer *metadata.AsyncWriter, initialOptions InitialOptions) error {
	if !validMailKind(initialOptions.MailKind) {
		return ErrInvalidMailKindOption
	}

	if initialOptions.SubscribeToNewsletter {
		if err := c.newsletterSubscriber.Subscribe(ctx, initialOptions.Email); err != nil {
			log.Error().Err(err).Msg("Failed to subscribe")
			return errorutil.Wrap(ErrFailedToSubscribeToNewsletter)
		}
	}

	result := writer.Store([]metadata.Item{
		{Key: "mail_kind", Value: initialOptions.MailKind},
		{Key: "subscribe_newsletter", Value: initialOptions.SubscribeToNewsletter},
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
