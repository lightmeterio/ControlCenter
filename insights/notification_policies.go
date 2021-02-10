// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/notification"
)

type DefaultNotificationPolicy struct {
}

func (DefaultNotificationPolicy) Pass(n notification.Notification) (bool, error) {
	p, ok := n.Content.(core.InsightProperties)
	return ok && p.Rating == core.BadRating, nil
}
