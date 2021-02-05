// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build release

package server

import "net/http"

func wrap(h http.Handler) http.Handler {
	return h
}