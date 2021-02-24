// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package logsource

import (
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
)

type Source interface {
	PublishLogs(postfix.Publisher) error
}
