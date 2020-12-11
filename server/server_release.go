// +build release

package server

import "net/http"

func wrap(h http.Handler) http.Handler {
	return h
}