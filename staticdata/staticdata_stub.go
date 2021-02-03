// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// +build !include

package staticdata

import (
	"net/http"
)

var HttpAssets http.FileSystem
