// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"errors"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
)

type Settings interface{}

var ErrInvalidSettings = errors.New(`Invalid Settings`)

// TODO: a notifier should be notified asynchronously!!!
type Notifier interface {
	Notify(Notification, translator.Translator) error
	ValidateSettings(Settings) error
}

type Notification struct {
	ID      int64
	Content Content
}

type ContentMetadata = map[string]ContentComponent

type ContentComponent interface {
	fmt.Stringer
	translator.TranslatableStringer
}

type Content interface {
	Title() ContentComponent
	Description() ContentComponent
	Metadata() ContentMetadata
}
