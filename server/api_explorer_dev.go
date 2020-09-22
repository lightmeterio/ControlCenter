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
