// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"fmt"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func Messagef(format string, a ...interface{}) Message {
	return Message(fmt.Sprintf(format, a...))
}

type Message string

func (s *Message) String() string {
	return string(*s)
}

func TranslateNotification(notification Notification, t translator.Translator) (Message, error) {
	transformed := translator.TransformTranslation(notification.Content.TplString())

	translatedMessage, err := t.Translate(transformed)
	if err != nil {
		return "", errorutil.Wrap(err)
	}

	args := notification.Content.Args()

	// TODO: restore this, or better, rely on the translator!
	// for i, arg := range args {
	// 	t, ok := arg.(time.Time)
	// 	if ok {
	// 		args[i] = timeutil.PrettyFormatTime(t, language)
	// 	}
	// }

	return Messagef(translatedMessage, args...), nil
}
