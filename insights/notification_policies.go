// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/insights/detectiveescalation"
	"gitlab.com/lightmeter/controlcenter/notification"
)

type DefaultNotificationPolicy struct {
}

func (DefaultNotificationPolicy) Reject(n notification.Notification) (bool, error) {
	p, ok := n.Content.(core.InsightProperties)
	if !ok {
		return true, nil
	}

	if p.MustBeNotified {
		return false, nil
	}

	if _, ok = p.Content.(detectiveescalation.Content); ok {
		return false, nil
	}

	return p.Rating != core.BadRating, nil
}
