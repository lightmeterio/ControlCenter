// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	httpauth "gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"net/http"
)

func HttpInsightsProgress(auth *httpauth.Authenticator, mux *http.ServeMux) {
	mux.Handle("/api/v0/importProgress", httpmiddleware.New(
		httpmiddleware.RequestWithTimeout(httpmiddleware.DefaultTimeout), httpmiddleware.RequireAuthenticationOnlyAfterSystemHasAnyUser(auth),
	).WithEndpoint(importProgressHandler{}))
}

type importProgressHandler struct{}

// @Summary Fetch Insights
// @Produce json
// @Success 200 {object} core.Progress
// @Failure 422 {string} string "desc"
// @Router /api/v0/importProgress [get]
func (h importProgressHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	p, err := core.GetProgress(r.Context())
	if err != nil {
		return errorutil.Wrap(err, "Error obtaining import progress")
	}

	if err := httputil.WriteJson(w, p, http.StatusOK); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
