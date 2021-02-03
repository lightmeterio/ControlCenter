// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

// +build dev !release

package server

import (
	"net/http"
	"net/http/pprof"
)

/*
 * Collect some data according to the documentation of net/http/pprof, like:
 * go tool pprof 'http://localhost:8080/debug/pprof/profile?seconds=30'
 *
 * go run github.com/google/pprof -http ":6061" pprof.lightmeter.samples.cpu.001.pb.gz
 */
func exposeProfiler(mux *http.ServeMux) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}
