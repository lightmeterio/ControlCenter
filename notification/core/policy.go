// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type Policy interface {
	// Can a notification be notified according to this policy?
	Pass(Notification) (bool, error)
}

type Policies []Policy

func (policies Policies) Pass(n Notification) (bool, error) {
	for _, p := range policies {
		pass, err := p.Pass(n)

		if err != nil {
			return false, errorutil.Wrap(err)
		}

		if pass {
			return true, nil
		}
	}

	return false, nil
}
