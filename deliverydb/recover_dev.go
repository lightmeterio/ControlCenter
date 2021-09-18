// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !release
// +build !release

package deliverydb

import (
	"gitlab.com/lightmeter/controlcenter/tracking"
)

//nolint:unused,deadcode
func recoverFromError(*error, tracking.Result) {
	// Do not recover from panic on dev build
}
