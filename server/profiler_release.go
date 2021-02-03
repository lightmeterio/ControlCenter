// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// +build release

package server

import "net/http"

func exposeProfiler(mux *http.ServeMux) {
	// We don't expose the profiler on release builds
}
