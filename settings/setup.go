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

var (
	ErrInvalidInintialSetupOption    = errors.New(`Invalid Initial Setup Option`)
	ErrFailedToSubscribeToNewsletter = errors.New(`Error Subscribing To Newsletter`)
)

type InitialOptions struct {
	SubscribeToNewsletter bool
	Email                 string
}

type InitialSetupSettings struct {
	newsletterSubscriber newsletter.Subscriber
}

func NewInitialSetupSettings(newsletterSubscriber newsletter.Subscriber) *InitialSetupSettings {
	return &InitialSetupSettings{newsletterSubscriber}
}

func (c *InitialSetupSettings) Set(ctx context.Context, writer *metadata.AsyncWriter, initialOptions InitialOptions) error {
	if initialOptions.SubscribeToNewsletter {
		if err := c.newsletterSubscriber.Subscribe(ctx, initialOptions.Email); err != nil {
			log.Error().Err(err).Msg("Failed to subscribe")
			return errorutil.Wrap(ErrFailedToSubscribeToNewsletter)
		}
	}

	result := writer.Store([]metadata.Item{
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
