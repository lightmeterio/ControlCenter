// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type Policy interface {
	// Should a notification be rejected by a policy?
	Reject(Notification) (bool, error)
}

type Policies []Policy

func (policies Policies) Reject(n Notification) (bool, error) {
	for _, p := range policies {
		rejected, err := p.Reject(n)

		if err != nil {
			return true, errorutil.Wrap(err)
		}

		if rejected {
			return true, nil
		}
	}

	return false, nil
}

type alwaysAllowPolicy struct{}

func (alwaysAllowPolicy) Reject(Notification) (bool, error) {
	return false, nil
}

var PassPolicy = alwaysAllowPolicy{}
