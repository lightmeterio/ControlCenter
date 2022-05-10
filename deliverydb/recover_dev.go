// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !release
// +build !release

package deliverydb

import (
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/tracking"
)

func recoverFromError(err *error, tr tracking.Result) {
	if r := recover(); r != nil {
		log.Error().Object("result", tr).Msg("Failed to store delivery message")

		// FIXME: horrendous workaround while we cannot figure out the cause of the issue!
		*err = nil
	}
}
