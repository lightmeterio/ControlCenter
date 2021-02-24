// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

// Similar to Content{}, but already translated
type Message struct {
	Title       string
	Description string
	Metadata    map[string]string
}

func TranslateNotification(n Notification, t translator.Translator) (Message, error) {
	title, err := translator.Translate(t, n.Content.Title())
	if err != nil {
		return Message{}, errorutil.Wrap(err)
	}

	description, err := translator.Translate(t, n.Content.Description())
	if err != nil {
		return Message{}, errorutil.Wrap(err)
	}

	message := Message{
		Title:       title,
		Description: description,
		Metadata:    map[string]string{},
	}

	for k, m := range n.Content.Metadata() {
		v, err := translator.Translate(t, m)
		if err != nil {
			return Message{}, errorutil.Wrap(err)
		}

		message.Metadata[k] = v
	}

	return message, nil
}
