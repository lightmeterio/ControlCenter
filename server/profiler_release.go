// +build release

package server

import "net/http"

func exposeProfiler(mux *http.ServeMux) {
	// We don't expose the profiler on release builds
}
