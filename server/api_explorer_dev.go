// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// +build dev !release

package server

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
	_ "gitlab.com/lightmeter/controlcenter/api/docs"
)

func exposeApiExplorer(mux *http.ServeMux) {
	mux.Handle("/api/", httpSwagger.Handler(
		httpSwagger.URL("/api.json"),
	))
}
