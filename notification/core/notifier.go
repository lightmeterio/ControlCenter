// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"fmt"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
)

// TODO: a notifier should be notified asynchronously!!!
type Notifier interface {
	Notify(Notification, translator.Translator) error
}

type Notification struct {
	ID      int64
	Content Content
	Rating  int64
}

type Content interface {
	fmt.Stringer
	translator.TranslatableStringer
}
