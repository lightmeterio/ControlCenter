// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

// +build dev !release

package server

import (
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
)

// Dev allows Cross Origin Resource Sharing from any origin.
// Don't do this without consideration in production systems.
func wrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
			preflightHandler(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// preflightHandler adds the necessary headers in order to serve
// CORS from any origin using the methods "GET", "HEAD", "POST", "PUT", "DELETE"
// We insist, don't do this without consideration in production systems.
func preflightHandler(w http.ResponseWriter, r *http.Request) {
	headers := []string{"Content-Type", "Accept", "Set-Cookie"}
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
	w.Header().Set("Access-Control-Expose-Headers", "Set-Cookie")

	methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"}
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
	log.Info().Msgf("preflight request for %s", r.URL.Path)
}
