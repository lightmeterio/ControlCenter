// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package notification

import (
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/notification/slack"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"golang.org/x/text/language"
	"time"
)

type (
	Notification = core.Notification
	Notifier     = core.Notifier
	Policy       = core.Policy
	Policies     = core.Policies
)

type alwaysAllowPolicy struct{}

func (alwaysAllowPolicy) Pass(core.Notification) (bool, error) {
	return true, nil
}

var AlwaysAllowPolicies = core.Policies{alwaysAllowPolicy{}}

func NewWithCustomLanguageFetcher(translators translator.Translators, languageFetcher func() (language.Tag, error), notifiers []core.Notifier) *Center {
	return &Center{
		translators:   translators,
		notifiers:     notifiers,
		fetchLanguage: languageFetcher,
	}
}

func New(reader *meta.Reader, translators translator.Translators, notifiers []core.Notifier) *Center {
	return NewWithCustomLanguageFetcher(translators, func() (language.Tag, error) {
		// TODO: get the settings from a "Notifications general settings" separated from Slack
		settings, err := slack.GetSettings(context.Background(), reader)
		if err != nil && errors.Is(err, meta.ErrNoSuchKey) {
			// setting not found
			return language.English, nil
		}

		if err != nil {
			return language.Tag{}, errorutil.Wrap(err)
		}

		tag, err := language.Parse(settings.Language)

		if err != nil {
			return language.Tag{}, errorutil.Wrap(err)
		}

		return tag, nil
	}, notifiers)
}

type Center struct {
	translators   translator.Translators
	notifiers     []core.Notifier
	fetchLanguage func() (language.Tag, error)
}

func (c *Center) Notify(notification core.Notification) error {
	languageTag, err := c.fetchLanguage()
	if err != nil {
		return errorutil.Wrap(err)
	}

	translator := c.translators.Translator(languageTag, time.Time{})

	for _, n := range c.notifiers {
		if err := n.Notify(notification, translator); err != nil {
			log.Warn().Msgf("Error notifying: (%v): %v", n, err)
		}
	}

	return nil
}
