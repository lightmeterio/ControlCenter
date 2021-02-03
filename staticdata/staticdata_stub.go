// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

// +build !include

package staticdata

import (
	"net/http"
)

var HttpAssets http.FileSystem
