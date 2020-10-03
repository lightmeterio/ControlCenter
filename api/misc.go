package api

import (
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"gitlab.com/lightmeter/controlcenter/version"
	"net/http"
)

type appVersionHandler struct{}

type appVersion struct {
	Version     string
	Commit      string
	TagOrBranch string
}

// @Summary Control Center Version
// @Produce json
// @Success 200 {object} appVersion
// @Router /api/v0/appVersion [get]
func (appVersionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	return httputil.WriteJson(w, appVersion{Version: version.Version, Commit: version.Commit, TagOrBranch: version.TagOrBranch}, http.StatusOK)
}

func HttpMisc(mux *http.ServeMux) {
	chain := httpmiddleware.New()

	mux.Handle("/api/v0/appVersion", chain.WithError(appVersionHandler{}))
}
