// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package testutil

import (
	"gitlab.com/lightmeter/controlcenter/util/temputil"
)

var (
	TempDir       = temputil.TempDir
	MustParseTime = temputil.MustParseTime
)
