// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package logsource

import (
	"gitlab.com/lightmeter/controlcenter/data"
)

type Source interface {
	PublishLogs(data.Publisher) error
}
