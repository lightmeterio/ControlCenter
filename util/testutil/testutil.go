// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package testutil

import (
	"gitlab.com/lightmeter/controlcenter/util/temputil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
)

var (
	TempDir       = temputil.TempDir
	MustParseTime = timeutil.MustParseTime
)
